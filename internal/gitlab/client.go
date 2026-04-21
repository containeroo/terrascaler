package gitlab

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	gl "gitlab.com/gitlab-org/api/client-go"

	"github.com/containeroo/terrascaler/internal/terraform"
)

type Config struct {
	BaseURL string
	Token   string
	Project string
	Branch  string
	File    string
	Target  terraform.Target
}

type Client struct {
	cfg    Config
	api    *gl.Client
	logger *slog.Logger
	mu     sync.Mutex
}

func New(cfg Config, logger *slog.Logger) (*Client, error) {
	if logger == nil {
		logger = slog.Default()
	}

	options := []gl.ClientOptionFunc{}
	if cfg.BaseURL != "" {
		options = append(options, gl.WithBaseURL(cfg.BaseURL))
	}
	api, err := gl.NewClient(cfg.Token, options...)
	if err != nil {
		return nil, err
	}
	return &Client{cfg: cfg, api: api, logger: logger}, nil
}

func (c *Client) TargetSize(ctx context.Context) (int, error) {
	c.logger.DebugContext(ctx, "fetching target size",
		"project", c.cfg.Project,
		"branch", c.cfg.Branch,
		"file", c.cfg.File,
		"block_type", c.cfg.Target.BlockType,
		"block_labels", c.cfg.Target.Labels,
		"attribute", c.cfg.Target.Attribute,
	)

	content, _, err := c.fetch(ctx)
	if err != nil {
		return 0, err
	}
	targetSize, err := terraform.ReadInt(c.cfg.File, content, c.cfg.Target)
	if err != nil {
		return 0, err
	}

	c.logger.DebugContext(ctx, "fetched target size",
		"project", c.cfg.Project,
		"branch", c.cfg.Branch,
		"file", c.cfg.File,
		"attribute", c.cfg.Target.Attribute,
		"target_size", targetSize,
	)
	return targetSize, nil
}

func (c *Client) IncreaseTargetSize(ctx context.Context, delta int, validate func(current int, next int) error) (int, int, error) {
	if delta <= 0 {
		return 0, 0, fmt.Errorf("delta must be positive, got %d", delta)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	c.logger.InfoContext(ctx, "starting GitLab-backed scale-up",
		"project", c.cfg.Project,
		"branch", c.cfg.Branch,
		"file", c.cfg.File,
		"block_type", c.cfg.Target.BlockType,
		"block_labels", c.cfg.Target.Labels,
		"attribute", c.cfg.Target.Attribute,
		"delta", delta,
	)

	current, next, err := c.increaseTargetSizeOnce(ctx, delta, validate)
	if isRetryableCommitError(err) {
		c.logger.WarnContext(ctx, "GitLab commit conflict, retrying scale-up",
			"project", c.cfg.Project,
			"branch", c.cfg.Branch,
			"file", c.cfg.File,
			"attribute", c.cfg.Target.Attribute,
			"delta", delta,
			"error", err,
		)
		current, next, err = c.increaseTargetSizeOnce(ctx, delta, validate)
	}
	if err != nil {
		c.logger.ErrorContext(ctx, "GitLab-backed scale-up failed",
			"project", c.cfg.Project,
			"branch", c.cfg.Branch,
			"file", c.cfg.File,
			"attribute", c.cfg.Target.Attribute,
			"delta", delta,
			"current_size", current,
			"requested_size", next,
			"error", err,
		)
		return current, next, err
	}

	c.logger.InfoContext(ctx, "GitLab-backed scale-up completed",
		"project", c.cfg.Project,
		"branch", c.cfg.Branch,
		"file", c.cfg.File,
		"attribute", c.cfg.Target.Attribute,
		"delta", delta,
		"previous_size", current,
		"target_size", next,
	)
	return current, next, err
}

func (c *Client) increaseTargetSizeOnce(ctx context.Context, delta int, validate func(current int, next int) error) (int, int, error) {
	content, lastCommitID, err := c.fetch(ctx)
	if err != nil {
		return 0, 0, err
	}

	current, err := terraform.ReadInt(c.cfg.File, content, c.cfg.Target)
	if err != nil {
		return 0, 0, err
	}

	next := current + delta
	if validate != nil {
		if err := validate(current, next); err != nil {
			return current, next, err
		}
	}

	updated, err := terraform.SetInt(c.cfg.File, content, c.cfg.Target, next)
	if err != nil {
		return 0, 0, err
	}
	if string(updated) == string(content) {
		return current, next, nil
	}

	message := fmt.Sprintf("terrascaler: scale %s from %d to %d", c.cfg.Target.Attribute, current, next)
	c.logger.InfoContext(ctx, "committing Terraform target size update",
		"project", c.cfg.Project,
		"branch", c.cfg.Branch,
		"file", c.cfg.File,
		"attribute", c.cfg.Target.Attribute,
		"previous_size", current,
		"target_size", next,
		"last_commit_id", lastCommitID,
	)

	options := &gl.UpdateFileOptions{
		Branch:        gl.Ptr(c.cfg.Branch),
		Content:       gl.Ptr(string(updated)),
		CommitMessage: gl.Ptr(message),
		LastCommitID:  gl.Ptr(lastCommitID),
	}
	_, _, err = c.api.RepositoryFiles.UpdateFile(c.cfg.Project, c.cfg.File, options, gl.WithContext(ctx))
	if err != nil {
		return 0, 0, fmt.Errorf("update GitLab file %s: %w", c.cfg.File, err)
	}
	c.logger.InfoContext(ctx, "committed Terraform target size update",
		"project", c.cfg.Project,
		"branch", c.cfg.Branch,
		"file", c.cfg.File,
		"attribute", c.cfg.Target.Attribute,
		"previous_size", current,
		"target_size", next,
	)
	return current, next, nil
}

func (c *Client) fetch(ctx context.Context) ([]byte, string, error) {
	file, _, err := c.api.RepositoryFiles.GetFile(c.cfg.Project, c.cfg.File, &gl.GetFileOptions{
		Ref: gl.Ptr(c.cfg.Branch),
	}, gl.WithContext(ctx))
	if err != nil {
		return nil, "", fmt.Errorf("fetch GitLab file %s: %w", c.cfg.File, err)
	}

	content, err := base64.StdEncoding.DecodeString(file.Content)
	if err != nil {
		return nil, "", fmt.Errorf("decode GitLab file %s: %w", c.cfg.File, err)
	}
	return content, file.LastCommitID, nil
}

func isRetryableCommitError(err error) bool {
	if err == nil {
		return false
	}
	var responseErr *gl.ErrorResponse
	if errors.As(err, &responseErr) {
		return responseErr.Response != nil &&
			(responseErr.Response.StatusCode == http.StatusBadRequest ||
				responseErr.Response.StatusCode == http.StatusConflict)
	}
	return false
}
