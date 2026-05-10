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
	cfg                      config.Config
	kube                     kubernetes.Interface
	target                   TargetStore
	template                 Resources
	metrics                  *Metrics
	logger                   *slog.Logger
	lastScaleUp              time.Time
	lastScaleDown            time.Time
	scaleDownCandidateSince  time.Time
	scaleDownCandidateTarget int
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
		"remove_nodes", plan.RemoveNodes,
		"pending_pods", plan.PendingPods,
		"eligible_pending_pods", plan.UnscheduledPods,
		"unschedulable_pods", plan.UnschedulablePods,
		"scale_down_potential_nodes", plan.ScaleDownPotentialNodes,
		"unfit_pods", plan.UnfitPods,
		"nodes", len(nodes),
		"pods", len(pods),
	)
	r.metrics.ObservePlan(plan)

	if plan.DesiredTarget == currentTarget {
		r.resetScaleDownCandidate()
		success = true
		return nil
	}
	if plan.DesiredTarget > currentTarget {
		r.resetScaleDownCandidate()
		if !r.lastScaleUp.IsZero() && now.Sub(r.lastScaleUp) < r.cfg.ScaleUpCooldown {
			r.logger.InfoContext(ctx, "scale-up skipped during cooldown",
				"current_target", currentTarget,
				"desired_target", plan.DesiredTarget,
				"cooldown", r.cfg.ScaleUpCooldown.String(),
				"last_scale_up_time", r.lastScaleUp.Format(time.RFC3339),
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
			r.lastScaleUp = now
			r.metrics.IncScaleUpCommit()
		}
		success = true
		return nil
	}

	if !r.scaleDownCandidateStable(now, plan.DesiredTarget) {
		r.logger.InfoContext(ctx, "scale-down candidate observed; waiting for unneeded time",
			"current_target", currentTarget,
			"desired_target", plan.DesiredTarget,
			"remove_nodes", plan.RemoveNodes,
			"scale_down_unneeded_time", r.cfg.ScaleDownUnneededTime.String(),
			"candidate_since", r.scaleDownCandidateSince.Format(time.RFC3339),
		)
		success = true
		return nil
	}
	if !r.lastScaleDown.IsZero() && now.Sub(r.lastScaleDown) < r.cfg.ScaleDownCooldown {
		r.logger.InfoContext(ctx, "scale-down skipped during cooldown",
			"current_target", currentTarget,
			"desired_target", plan.DesiredTarget,
			"remove_nodes", plan.RemoveNodes,
			"cooldown", r.cfg.ScaleDownCooldown.String(),
			"last_scale_down_time", r.lastScaleDown.Format(time.RFC3339),
		)
		success = true
		return nil
	}
	if r.cfg.DryRun {
		r.logger.InfoContext(ctx, "dry-run scale-down skipped GitLab update",
			"current_target", currentTarget,
			"desired_target", plan.DesiredTarget,
			"remove_nodes", plan.RemoveNodes,
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
		return nil
	})
	if err != nil {
		return err
	}
	if next < currentTarget {
		r.lastScaleDown = now
		r.metrics.IncScaleDownCommit()
		r.resetScaleDownCandidate()
	}
	success = true
	return nil
}

func (r *Runner) scaleDownCandidateStable(now time.Time, desiredTarget int) bool {
	if r.scaleDownCandidateSince.IsZero() || r.scaleDownCandidateTarget != desiredTarget {
		r.scaleDownCandidateSince = now
		r.scaleDownCandidateTarget = desiredTarget
		return r.cfg.ScaleDownUnneededTime == 0
	}
	return now.Sub(r.scaleDownCandidateSince) >= r.cfg.ScaleDownUnneededTime
}

func (r *Runner) resetScaleDownCandidate() {
	r.scaleDownCandidateSince = time.Time{}
	r.scaleDownCandidateTarget = 0
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
