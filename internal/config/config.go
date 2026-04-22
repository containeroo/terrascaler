package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
)

type Config struct {
	Kubeconfig string

	CheckInterval    time.Duration
	ScaleUpCooldown  time.Duration
	PendingPodMinAge time.Duration
	MetricsAddress   string
	Once             bool
	DryRun           bool

	GitLabBaseURL string
	GitLabToken   string
	GitLabProject string
	GitLabBranch  string
	GitLabMR      MergeRequestConfig

	FilePath  string
	BlockType string
	Labels    []string
	Attribute string

	MinSize int
	MaxSize int

	NodeSelector map[string]string

	TemplateCPU    string
	TemplateMemory string
	TemplatePods   int64
	TemplateLabels map[string]string
}

type MergeRequestConfig struct {
	Enabled            bool
	BranchPrefix       string
	Title              string
	Description        string
	Labels             []string
	AssigneeUsernames  []string
	AssigneeIDs        []int64
	ReviewerUsernames  []string
	ReviewerIDs        []int64
	RemoveSourceBranch bool
}

func Load() (Config, error) {
	var cfg Config
	var blockLabels string
	var nodeSelector string
	var templateLabels string
	var mrLabels string
	var mrAssigneeUsernames string
	var mrAssigneeIDs string
	var mrReviewerUsernames string
	var mrReviewerIDs string

	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags.StringVar(&cfg.Kubeconfig, "kubeconfig", env("KUBECONFIG", ""), "Path to kubeconfig; leave empty to use in-cluster config")
	flags.DurationVar(&cfg.CheckInterval, "check-interval", envDuration("TERRASCALER_CHECK_INTERVAL", time.Minute), "Autoscaling check interval")
	flags.DurationVar(&cfg.ScaleUpCooldown, "scale-up-cooldown", envDuration("TERRASCALER_SCALE_UP_COOLDOWN", 5*time.Minute), "Minimum time between scale-up commits")
	flags.DurationVar(&cfg.PendingPodMinAge, "pending-pod-min-age", envDuration("TERRASCALER_PENDING_POD_MIN_AGE", 30*time.Second), "Minimum age for pending pods without an Unschedulable condition")
	flags.StringVar(&cfg.MetricsAddress, "metrics-address", env("TERRASCALER_METRICS_ADDRESS", ":8080"), "Prometheus metrics listen address; empty disables metrics HTTP server")
	flags.BoolVar(&cfg.Once, "once", envBool("TERRASCALER_ONCE", false), "Run one autoscaling check and exit")
	flags.BoolVar(&cfg.DryRun, "dry-run", envBool("TERRASCALER_DRY_RUN", false), "Log intended scaling actions without updating GitLab")
	flags.StringVar(&cfg.GitLabBaseURL, "gitlab-base-url", env("GITLAB_BASE_URL", ""), "GitLab base URL; leave empty for gitlab.com")
	flags.StringVar(&cfg.GitLabToken, "gitlab-token", env("GITLAB_TOKEN", ""), "GitLab API token")
	flags.StringVar(&cfg.GitLabProject, "gitlab-project", env("GITLAB_PROJECT", ""), "GitLab project ID or path, for example group/project")
	flags.StringVar(&cfg.GitLabBranch, "gitlab-branch", env("GITLAB_BRANCH", "main"), "Git branch to update")
	flags.BoolVar(&cfg.GitLabMR.Enabled, "gitlab-merge-request", envBool("TERRASCALER_GITLAB_MERGE_REQUEST", false), "Create or update a GitLab merge request instead of committing directly to the target branch")
	flags.StringVar(&cfg.GitLabMR.BranchPrefix, "gitlab-mr-branch-prefix", env("TERRASCALER_GITLAB_MR_BRANCH_PREFIX", "terrascaler/scale"), "Branch prefix for GitLab merge request mode")
	flags.StringVar(&cfg.GitLabMR.Title, "gitlab-mr-title", env("TERRASCALER_GITLAB_MR_TITLE", "terrascaler: scale worker count"), "GitLab merge request title")
	flags.StringVar(&cfg.GitLabMR.Description, "gitlab-mr-description", env("TERRASCALER_GITLAB_MR_DESCRIPTION", "Automated Terrascaler scale-up proposal."), "GitLab merge request description")
	flags.StringVar(&mrLabels, "gitlab-mr-labels", env("TERRASCALER_GITLAB_MR_LABELS", "terrascaler"), "Comma-separated GitLab merge request labels")
	flags.StringVar(&mrAssigneeUsernames, "gitlab-mr-assignees", env("TERRASCALER_GITLAB_MR_ASSIGNEES", ""), "Comma-separated GitLab usernames to assign to created merge requests")
	flags.StringVar(&mrAssigneeIDs, "gitlab-mr-assignee-ids", env("TERRASCALER_GITLAB_MR_ASSIGNEE_IDS", ""), "Comma-separated GitLab user IDs to assign to created merge requests")
	flags.StringVar(&mrReviewerUsernames, "gitlab-mr-reviewers", env("TERRASCALER_GITLAB_MR_REVIEWERS", ""), "Comma-separated GitLab usernames to request review from on created merge requests")
	flags.StringVar(&mrReviewerIDs, "gitlab-mr-reviewer-ids", env("TERRASCALER_GITLAB_MR_REVIEWER_IDS", ""), "Comma-separated GitLab user IDs to request review from on created merge requests")
	flags.BoolVar(&cfg.GitLabMR.RemoveSourceBranch, "gitlab-mr-remove-source-branch", envBool("TERRASCALER_GITLAB_MR_REMOVE_SOURCE_BRANCH", true), "Remove source branch when the GitLab merge request is merged")
	flags.StringVar(&cfg.FilePath, "file", env("TERRASCALER_FILE", ""), "Terraform file path in the repository")
	flags.StringVar(&cfg.BlockType, "block-type", env("TERRASCALER_BLOCK_TYPE", "module"), "Terraform block type that contains the target attribute")
	flags.StringVar(&blockLabels, "block-labels", env("TERRASCALER_BLOCK_LABELS", ""), "Comma-separated Terraform block labels, for example hostedcluster")
	flags.StringVar(&cfg.Attribute, "attribute", env("TERRASCALER_ATTRIBUTE", ""), "Terraform integer attribute to update")
	flags.IntVar(&cfg.MinSize, "min-size", envInt("TERRASCALER_MIN_SIZE", 0), "Minimum target size")
	flags.IntVar(&cfg.MaxSize, "max-size", envInt("TERRASCALER_MAX_SIZE", 100), "Maximum target size")
	flags.StringVar(&nodeSelector, "node-selector", env("TERRASCALER_NODE_SELECTOR", ""), "Comma-separated node labels, key=value, that identify Terraform-managed workers")
	flags.StringVar(&cfg.TemplateCPU, "template-cpu", env("TERRASCALER_TEMPLATE_CPU", "2"), "New worker CPU capacity")
	flags.StringVar(&cfg.TemplateMemory, "template-memory", env("TERRASCALER_TEMPLATE_MEMORY", "8Gi"), "New worker memory capacity")
	flags.Int64Var(&cfg.TemplatePods, "template-pods", int64(envInt("TERRASCALER_TEMPLATE_PODS", 110)), "New worker pod capacity")
	flags.StringVar(&templateLabels, "template-labels", env("TERRASCALER_TEMPLATE_LABELS", ""), "Comma-separated labels for new workers, key=value")

	if err := flags.Parse(os.Args[1:]); err != nil {
		return Config{}, err
	}

	cfg.Labels = splitCSV(blockLabels)
	cfg.NodeSelector = parseMap(nodeSelector)
	cfg.TemplateLabels = parseMap(templateLabels)
	cfg.GitLabMR.Labels = splitCSV(mrLabels)
	cfg.GitLabMR.AssigneeUsernames = splitCSV(mrAssigneeUsernames)
	cfg.GitLabMR.AssigneeIDs = parseInt64CSV(mrAssigneeIDs)
	cfg.GitLabMR.ReviewerUsernames = splitCSV(mrReviewerUsernames)
	cfg.GitLabMR.ReviewerIDs = parseInt64CSV(mrReviewerIDs)

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	var missing []string
	for name, value := range map[string]string{
		"gitlab-token":   c.GitLabToken,
		"gitlab-project": c.GitLabProject,
		"file":           c.FilePath,
		"block-type":     c.BlockType,
		"attribute":      c.Attribute,
	} {
		if strings.TrimSpace(value) == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required configuration: %s", strings.Join(missing, ", "))
	}
	if c.CheckInterval <= 0 {
		return errors.New("check-interval must be > 0")
	}
	if c.ScaleUpCooldown < 0 {
		return errors.New("scale-up-cooldown must be >= 0")
	}
	if c.PendingPodMinAge < 0 {
		return errors.New("pending-pod-min-age must be >= 0")
	}
	if c.MinSize < 0 {
		return errors.New("min-size must be >= 0")
	}
	if c.MaxSize < c.MinSize {
		return errors.New("max-size must be >= min-size")
	}
	if c.TemplatePods <= 0 {
		return errors.New("template-pods must be > 0")
	}
	if _, err := resource.ParseQuantity(c.TemplateCPU); err != nil {
		return fmt.Errorf("template-cpu is invalid: %w", err)
	}
	if _, err := resource.ParseQuantity(c.TemplateMemory); err != nil {
		return fmt.Errorf("template-memory is invalid: %w", err)
	}
	return nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func parseMap(value string) map[string]string {
	out := map[string]string{}
	for _, part := range splitCSV(value) {
		key, val, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if key != "" {
			out[key] = val
		}
	}
	return out
}

func parseInt64CSV(value string) []int64 {
	parts := splitCSV(value)
	out := make([]int64, 0, len(parts))
	for _, part := range parts {
		parsed, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			continue
		}
		out = append(out, parsed)
	}
	return out
}
