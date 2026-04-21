package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
)

type Config struct {
	ListenAddress string
	TLSCertFile   string
	TLSKeyFile    string
	TLSClientCA   string

	GitLabBaseURL string
	GitLabToken   string
	GitLabProject string
	GitLabBranch  string

	FilePath  string
	BlockType string
	Labels    []string
	Attribute string

	NodeGroupID string
	MinSize     int32
	MaxSize     int32

	TemplateCPU    string
	TemplateMemory string
	TemplatePods   int64
	TemplateLabels map[string]string
}

func Load() (Config, error) {
	var cfg Config
	var blockLabels string
	var templateLabels string
	var minSize int
	var maxSize int

	flags := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flags.StringVar(&cfg.ListenAddress, "listen", env("TERRASCALER_LISTEN", ":8080"), "gRPC listen address")
	flags.StringVar(&cfg.TLSCertFile, "tls-cert-file", env("TERRASCALER_TLS_CERT_FILE", ""), "Server TLS certificate file")
	flags.StringVar(&cfg.TLSKeyFile, "tls-key-file", env("TERRASCALER_TLS_KEY_FILE", ""), "Server TLS key file")
	flags.StringVar(&cfg.TLSClientCA, "tls-client-ca-file", env("TERRASCALER_TLS_CLIENT_CA_FILE", ""), "Client CA file for mTLS")
	flags.StringVar(&cfg.GitLabBaseURL, "gitlab-base-url", env("GITLAB_BASE_URL", ""), "GitLab base URL; leave empty for gitlab.com")
	flags.StringVar(&cfg.GitLabToken, "gitlab-token", env("GITLAB_TOKEN", ""), "GitLab API token")
	flags.StringVar(&cfg.GitLabProject, "gitlab-project", env("GITLAB_PROJECT", ""), "GitLab project ID or path, for example group/project")
	flags.StringVar(&cfg.GitLabBranch, "gitlab-branch", env("GITLAB_BRANCH", "main"), "Git branch to update")
	flags.StringVar(&cfg.FilePath, "file", env("TERRASCALER_FILE", ""), "Terraform file path in the repository")
	flags.StringVar(&cfg.BlockType, "block-type", env("TERRASCALER_BLOCK_TYPE", "module"), "Terraform block type that contains the target attribute")
	flags.StringVar(&blockLabels, "block-labels", env("TERRASCALER_BLOCK_LABELS", ""), "Comma-separated Terraform block labels, for example hostedcluster")
	flags.StringVar(&cfg.Attribute, "attribute", env("TERRASCALER_ATTRIBUTE", ""), "Terraform integer attribute to update")
	flags.StringVar(&cfg.NodeGroupID, "node-group-id", env("TERRASCALER_NODE_GROUP_ID", "default"), "Cluster Autoscaler node group ID")
	flags.IntVar(&minSize, "min-size", envInt("TERRASCALER_MIN_SIZE", 0), "Minimum node group size")
	flags.IntVar(&maxSize, "max-size", envInt("TERRASCALER_MAX_SIZE", 100), "Maximum node group size")
	flags.StringVar(&cfg.TemplateCPU, "template-cpu", env("TERRASCALER_TEMPLATE_CPU", "2"), "Template node CPU capacity")
	flags.StringVar(&cfg.TemplateMemory, "template-memory", env("TERRASCALER_TEMPLATE_MEMORY", "8Gi"), "Template node memory capacity")
	flags.Int64Var(&cfg.TemplatePods, "template-pods", int64(envInt("TERRASCALER_TEMPLATE_PODS", 110)), "Template node pod capacity")
	flags.StringVar(&templateLabels, "template-labels", env("TERRASCALER_TEMPLATE_LABELS", ""), "Comma-separated template node labels, key=value")

	if err := flags.Parse(os.Args[1:]); err != nil {
		return Config{}, err
	}

	cfg.MinSize = int32(minSize)
	cfg.MaxSize = int32(maxSize)
	cfg.Labels = splitCSV(blockLabels)
	cfg.TemplateLabels = parseMap(templateLabels)

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
		"node-group-id":  c.NodeGroupID,
	} {
		if strings.TrimSpace(value) == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required configuration: %s", strings.Join(missing, ", "))
	}
	if c.MinSize < 0 {
		return errors.New("min-size must be >= 0")
	}
	if c.MaxSize < c.MinSize {
		return errors.New("max-size must be >= min-size")
	}
	if (c.TLSCertFile == "") != (c.TLSKeyFile == "") {
		return errors.New("tls-cert-file and tls-key-file must be set together")
	}
	if c.TLSClientCA != "" && c.TLSCertFile == "" {
		return errors.New("tls-client-ca-file requires tls-cert-file and tls-key-file")
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
