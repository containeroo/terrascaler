package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/containeroo/terrascaler/internal/autoscaler"
	"github.com/containeroo/terrascaler/internal/config"
	"github.com/containeroo/terrascaler/internal/gitlab"
	"github.com/containeroo/terrascaler/internal/terraform"
)

var Version = "dev"

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	kubeConfig, err := loadKubeConfig(cfg.Kubeconfig)
	if err != nil {
		log.Fatalf("load Kubernetes config: %v", err)
	}
	kubeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		log.Fatalf("create Kubernetes client: %v", err)
	}

	gitlabClient, err := gitlab.New(gitlab.Config{
		BaseURL: cfg.GitLabBaseURL,
		Token:   cfg.GitLabToken,
		Project: cfg.GitLabProject,
		Branch:  cfg.GitLabBranch,
		MR: gitlab.MergeRequestConfig{
			Enabled:            cfg.GitLabMR.Enabled,
			BranchPrefix:       cfg.GitLabMR.BranchPrefix,
			Title:              cfg.GitLabMR.Title,
			Description:        cfg.GitLabMR.Description,
			Labels:             cfg.GitLabMR.Labels,
			AssigneeUsernames:  cfg.GitLabMR.AssigneeUsernames,
			AssigneeIDs:        cfg.GitLabMR.AssigneeIDs,
			ReviewerUsernames:  cfg.GitLabMR.ReviewerUsernames,
			ReviewerIDs:        cfg.GitLabMR.ReviewerIDs,
			RemoveSourceBranch: cfg.GitLabMR.RemoveSourceBranch,
		},
		File: cfg.FilePath,
		Target: terraform.Target{
			BlockType: cfg.BlockType,
			Labels:    cfg.Labels,
			Attribute: cfg.Attribute,
		},
	}, logger.With("component", "gitlab"))
	if err != nil {
		log.Fatalf("create GitLab client: %v", err)
	}

	runner, err := autoscaler.NewRunner(cfg, kubeClient, gitlabClient, logger.With("component", "autoscaler"))
	if err != nil {
		log.Fatalf("create autoscaler: %v", err)
	}
	runner.SetMetrics(autoscaler.NewMetrics(nil))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger.Info("starting terrascaler",
		"version", Version,
		"check_interval", cfg.CheckInterval.String(),
		"scale_up_cooldown", cfg.ScaleUpCooldown.String(),
		"pending_pod_min_age", cfg.PendingPodMinAge.String(),
		"metrics_address", cfg.MetricsAddress,
		"min_size", cfg.MinSize,
		"max_size", cfg.MaxSize,
		"dry_run", cfg.DryRun,
		"once", cfg.Once,
		"gitlab_base_url", cfg.GitLabBaseURL,
		"gitlab_project", cfg.GitLabProject,
		"gitlab_branch", cfg.GitLabBranch,
		"gitlab_merge_request", cfg.GitLabMR.Enabled,
		"gitlab_mr_branch_prefix", cfg.GitLabMR.BranchPrefix,
		"terraform_file", cfg.FilePath,
		"terraform_block_type", cfg.BlockType,
		"terraform_block_labels", cfg.Labels,
		"terraform_attribute", cfg.Attribute,
		"node_selector", cfg.NodeSelector,
		"template_cpu", cfg.TemplateCPU,
		"template_memory", cfg.TemplateMemory,
		"template_pods", cfg.TemplatePods,
	)

	metricsServer := startMetricsServer(cfg.MetricsAddress, logger)
	if metricsServer != nil {
		defer func() {
			if err := metricsServer.Shutdown(context.Background()); err != nil {
				logger.Error("shutdown metrics server", "error", err)
			}
		}()
	}

	if err := runner.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("run autoscaler: %v", err)
	}
}

func startMetricsServer(address string, logger *slog.Logger) *http.Server {
	if address == "" {
		logger.Info("metrics server disabled")
		return nil
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	server := &http.Server{
		Addr:    address,
		Handler: mux,
	}
	go func() {
		logger.Info("starting metrics server", "address", address)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("metrics server failed", "error", err)
		}
	}()
	return server
}

func loadKubeConfig(path string) (*rest.Config, error) {
	if path != "" {
		return clientcmd.BuildConfigFromFlags("", path)
	}

	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		&clientcmd.ConfigOverrides{},
	)
	return clientConfig.ClientConfig()
}
