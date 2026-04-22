package autoscaler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/containeroo/terrascaler/internal/config"
)

type TargetStore interface {
	TargetSize(ctx context.Context) (int, error)
	SetTargetSize(ctx context.Context, desired int, validate func(current int, next int) error) (int, int, error)
}

type Runner struct {
	cfg       config.Config
	kube      kubernetes.Interface
	target    TargetStore
	template  Resources
	metrics   *Metrics
	logger    *slog.Logger
	lastScale time.Time
}

func NewRunner(cfg config.Config, kube kubernetes.Interface, target TargetStore, logger *slog.Logger) (*Runner, error) {
	if logger == nil {
		logger = slog.Default()
	}
	template, err := TemplateResources(cfg.TemplateCPU, cfg.TemplateMemory, cfg.TemplatePods)
	if err != nil {
		return nil, err
	}
	return &Runner{
		cfg:      cfg,
		kube:     kube,
		target:   target,
		template: template,
		logger:   logger,
	}, nil
}

func (r *Runner) SetMetrics(metrics *Metrics) {
	r.metrics = metrics
}

func (r *Runner) Run(ctx context.Context) error {
	if r.cfg.Once {
		return r.RunOnce(ctx)
	}

	ticker := time.NewTicker(r.cfg.CheckInterval)
	defer ticker.Stop()

	for {
		if err := r.RunOnce(ctx); err != nil {
			r.logger.ErrorContext(ctx, "autoscaling check failed", "error", err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (r *Runner) RunOnce(ctx context.Context) error {
	now := time.Now()
	success := false
	defer func() {
		r.metrics.ObserveCheck(success, time.Now())
	}()

	currentTarget, err := r.target.TargetSize(ctx)
	if err != nil {
		return err
	}

	nodes, pods, err := r.snapshot(ctx)
	if err != nil {
		return err
	}

	plan := BuildPlan(
		now,
		nodes,
		pods,
		currentTarget,
		r.cfg.MinSize,
		r.cfg.MaxSize,
		r.cfg.NodeSelector,
		r.template,
		r.cfg.PendingPodMinAge,
	)

	r.logger.InfoContext(ctx, "autoscaling plan computed",
		"reason", plan.Reason,
		"current_target", plan.CurrentTarget,
		"desired_target", plan.DesiredTarget,
		"new_nodes", plan.NewNodes,
		"pending_pods", plan.PendingPods,
		"eligible_pending_pods", plan.UnscheduledPods,
		"unschedulable_pods", plan.UnschedulablePods,
		"unfit_pods", plan.UnfitPods,
		"nodes", len(nodes),
		"pods", len(pods),
	)
	r.metrics.ObservePlan(plan)

	if plan.DesiredTarget <= currentTarget {
		success = true
		return nil
	}
	if !r.lastScale.IsZero() && now.Sub(r.lastScale) < r.cfg.ScaleUpCooldown {
		r.logger.InfoContext(ctx, "scale-up skipped during cooldown",
			"current_target", currentTarget,
			"desired_target", plan.DesiredTarget,
			"cooldown", r.cfg.ScaleUpCooldown.String(),
			"last_scale_time", r.lastScale.Format(time.RFC3339),
		)
		success = true
		return nil
	}
	if r.cfg.DryRun {
		r.logger.InfoContext(ctx, "dry-run scale-up skipped GitLab update",
			"current_target", currentTarget,
			"desired_target", plan.DesiredTarget,
		)
		success = true
		return nil
	}

	_, next, err := r.target.SetTargetSize(ctx, plan.DesiredTarget, func(current int, desired int) error {
		if desired < r.cfg.MinSize {
			return fmt.Errorf("desired target size %d is smaller than min-size %d", desired, r.cfg.MinSize)
		}
		if desired > r.cfg.MaxSize {
			return fmt.Errorf("desired target size %d is larger than max-size %d", desired, r.cfg.MaxSize)
		}
		if desired < current {
			return fmt.Errorf("desired target size %d is smaller than current target size %d", desired, current)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if next > currentTarget {
		r.lastScale = now
		r.metrics.IncScaleUpCommit()
	}
	success = true
	return nil
}

func (r *Runner) snapshot(ctx context.Context) ([]corev1.Node, []corev1.Pod, error) {
	nodes, err := r.kube.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("list nodes: %w", err)
	}

	pods, err := r.kube.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("list pods: %w", err)
	}

	return nodes.Items, pods.Items, nil
}
