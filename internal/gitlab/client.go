package gitlab

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	gl "gitlab.com/gitlab-org/api/client-go"

	"github.com/containeroo/terrascaler/internal/terraform"
)

type Config struct {
	BaseURL string
	Token   string
	Project string
	Branch  string
	MR      MergeRequestConfig
	File    string
	Target  terraform.Target
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

func (c *Client) SetTargetSize(ctx context.Context, desired int, validate func(current int, next int) error) (int, int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	current, next, err := c.setTargetSizeOnce(ctx, desired, validate)
	if isRetryableCommitError(err) {
		c.logger.WarnContext(ctx, "GitLab commit conflict, retrying target size update",
			"project", c.cfg.Project,
			"branch", c.cfg.Branch,
			"file", c.cfg.File,
			"attribute", c.cfg.Target.Attribute,
			"desired_size", desired,
			"error", err,
		)
		current, next, err = c.setTargetSizeOnce(ctx, desired, validate)
	}
	if err != nil {
		c.logger.ErrorContext(ctx, "GitLab target size update failed",
			"project", c.cfg.Project,
			"branch", c.cfg.Branch,
			"file", c.cfg.File,
			"attribute", c.cfg.Target.Attribute,
			"current_size", current,
			"requested_size", next,
			"error", err,
		)
		return current, next, err
	}

	c.logger.InfoContext(ctx, "GitLab target size update completed",
		"project", c.cfg.Project,
		"branch", c.cfg.Branch,
		"file", c.cfg.File,
		"attribute", c.cfg.Target.Attribute,
		"previous_size", current,
		"target_size", next,
	)
	return current, next, nil
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
	return c.updateTargetSize(ctx, content, lastCommitID, current, next, validate)
}

func (c *Client) setTargetSizeOnce(ctx context.Context, desired int, validate func(current int, next int) error) (int, int, error) {
	content, lastCommitID, err := c.fetch(ctx)
	if err != nil {
		return 0, 0, err
	}

	current, err := terraform.ReadInt(c.cfg.File, content, c.cfg.Target)
	if err != nil {
		return 0, 0, err
	}

	return c.updateTargetSize(ctx, content, lastCommitID, current, desired, validate)
}

func (c *Client) updateTargetSize(ctx context.Context, content []byte, lastCommitID string, current int, next int, validate func(current int, next int) error) (int, int, error) {
	if validate != nil {
		if err := validate(current, next); err != nil {
			return current, next, err
		}
	}
	if next == current {
		return current, next, nil
	}

	updated, err := terraform.SetInt(c.cfg.File, content, c.cfg.Target, next)
	if err != nil {
		return 0, 0, err
	}
	if string(updated) == string(content) {
		return current, next, nil
	}

	message := fmt.Sprintf("terrascaler: scale %s from %d to %d", c.cfg.Target.Attribute, current, next)
	branch := c.cfg.Branch
	if c.cfg.MR.Enabled {
		branch = c.mergeRequestBranch(next)
		if err := c.ensureMergeRequestBranch(ctx, branch); err != nil {
			return 0, 0, err
		}
		content, lastCommitID, err = c.fetchRef(ctx, branch)
		if err != nil {
			return 0, 0, err
		}
		branchCurrent, err := terraform.ReadInt(c.cfg.File, content, c.cfg.Target)
		if err != nil {
			return 0, 0, err
		}
		if branchCurrent == next {
			if err := c.ensureMergeRequest(ctx, branch, current, next); err != nil {
				return 0, 0, err
			}
			c.logger.InfoContext(ctx, "GitLab merge request branch already has requested target size",
				"project", c.cfg.Project,
				"branch", branch,
				"target_branch", c.cfg.Branch,
				"file", c.cfg.File,
				"attribute", c.cfg.Target.Attribute,
				"base_size", current,
				"target_size", next,
			)
			return current, next, nil
		}
		updated, err = terraform.SetInt(c.cfg.File, content, c.cfg.Target, next)
		if err != nil {
			return 0, 0, err
		}
		if string(updated) == string(content) {
			return current, next, nil
		}
	}

	c.logger.InfoContext(ctx, "committing Terraform target size update",
		"project", c.cfg.Project,
		"branch", branch,
		"target_branch", c.cfg.Branch,
		"merge_request", c.cfg.MR.Enabled,
		"file", c.cfg.File,
		"attribute", c.cfg.Target.Attribute,
		"previous_size", current,
		"target_size", next,
		"last_commit_id", lastCommitID,
	)

	options := &gl.UpdateFileOptions{
		Branch:        gl.Ptr(branch),
		Content:       gl.Ptr(string(updated)),
		CommitMessage: gl.Ptr(message),
		LastCommitID:  gl.Ptr(lastCommitID),
	}
	_, _, err = c.api.RepositoryFiles.UpdateFile(c.cfg.Project, c.cfg.File, options, gl.WithContext(ctx))
	if err != nil {
		return 0, 0, fmt.Errorf("update GitLab file %s: %w", c.cfg.File, err)
	}
	if c.cfg.MR.Enabled {
		if err := c.ensureMergeRequest(ctx, branch, current, next); err != nil {
			return 0, 0, err
		}
	}
	c.logger.InfoContext(ctx, "committed Terraform target size update",
		"project", c.cfg.Project,
		"branch", branch,
		"target_branch", c.cfg.Branch,
		"merge_request", c.cfg.MR.Enabled,
		"file", c.cfg.File,
		"attribute", c.cfg.Target.Attribute,
		"previous_size", current,
		"target_size", next,
	)
	return current, next, nil
}

func (c *Client) ensureMergeRequestBranch(ctx context.Context, branch string) error {
	if _, _, err := c.api.Branches.GetBranch(c.cfg.Project, branch, gl.WithContext(ctx)); err == nil {
		return nil
	} else if !isNotFoundError(err) {
		return fmt.Errorf("get GitLab branch %s: %w", branch, err)
	}

	c.logger.InfoContext(ctx, "creating GitLab merge request branch",
		"project", c.cfg.Project,
		"branch", branch,
		"ref", c.cfg.Branch,
	)
	_, _, err := c.api.Branches.CreateBranch(c.cfg.Project, &gl.CreateBranchOptions{
		Branch: gl.Ptr(branch),
		Ref:    gl.Ptr(c.cfg.Branch),
	}, gl.WithContext(ctx))
	if err != nil && !isBranchAlreadyExistsError(err) {
		return fmt.Errorf("create GitLab branch %s: %w", branch, err)
	}
	return nil
}

func (c *Client) ensureMergeRequest(ctx context.Context, branch string, current int, next int) error {
	existing, _, err := c.api.MergeRequests.ListProjectMergeRequests(c.cfg.Project, &gl.ListProjectMergeRequestsOptions{
		State:        gl.Ptr("opened"),
		SourceBranch: gl.Ptr(branch),
		TargetBranch: gl.Ptr(c.cfg.Branch),
	}, gl.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("list GitLab merge requests: %w", err)
	}
	if len(existing) > 0 {
		c.logger.InfoContext(ctx, "GitLab merge request already exists",
			"project", c.cfg.Project,
			"source_branch", branch,
			"target_branch", c.cfg.Branch,
			"merge_request_iid", existing[0].IID,
		)
		return nil
	}

	title := c.cfg.MR.Title
	if title == "" {
		title = fmt.Sprintf("terrascaler: scale %s to %d", c.cfg.Target.Attribute, next)
	}
	description := c.mergeRequestDescription(current, next)
	removeSourceBranch := c.cfg.MR.RemoveSourceBranch
	options := &gl.CreateMergeRequestOptions{
		Title:              gl.Ptr(title),
		Description:        gl.Ptr(description),
		SourceBranch:       gl.Ptr(branch),
		TargetBranch:       gl.Ptr(c.cfg.Branch),
		RemoveSourceBranch: gl.Ptr(removeSourceBranch),
	}
	if len(c.cfg.MR.Labels) > 0 {
		labels := gl.LabelOptions(c.cfg.MR.Labels)
		options.Labels = &labels
	}
	assigneeIDs, err := c.mergeRequestUserIDs(ctx, "assignee", c.cfg.MR.AssigneeUsernames, c.cfg.MR.AssigneeIDs)
	if err != nil {
		return err
	}
	if len(assigneeIDs) > 0 {
		options.AssigneeIDs = gl.Ptr(assigneeIDs)
	}
	reviewerIDs, err := c.mergeRequestUserIDs(ctx, "reviewer", c.cfg.MR.ReviewerUsernames, c.cfg.MR.ReviewerIDs)
	if err != nil {
		return err
	}
	if len(reviewerIDs) > 0 {
		options.ReviewerIDs = gl.Ptr(reviewerIDs)
	}

	mergeRequest, _, err := c.api.MergeRequests.CreateMergeRequest(c.cfg.Project, options, gl.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("create GitLab merge request: %w", err)
	}
	c.logger.InfoContext(ctx, "created GitLab merge request",
		"project", c.cfg.Project,
		"source_branch", branch,
		"target_branch", c.cfg.Branch,
		"merge_request_iid", mergeRequest.IID,
		"merge_request_url", mergeRequest.WebURL,
	)
	return nil
}

func (c *Client) mergeRequestUserIDs(ctx context.Context, role string, usernames []string, ids []int64) ([]int64, error) {
	out := append([]int64(nil), ids...)
	for _, username := range usernames {
		username = normalizeUsername(username)
		if username == "" {
			continue
		}
		id, err := c.projectMemberID(ctx, username)
		if err != nil {
			return nil, fmt.Errorf("resolve GitLab merge request %s %q: %w", role, username, err)
		}
		out = append(out, id)
	}
	return uniqueInt64(out), nil
}

func (c *Client) projectMemberID(ctx context.Context, username string) (int64, error) {
	members, _, err := c.api.ProjectMembers.ListAllProjectMembers(c.cfg.Project, &gl.ListProjectMembersOptions{
		Query: gl.Ptr(username),
	}, gl.WithContext(ctx))
	if err != nil {
		return 0, fmt.Errorf("list GitLab project members: %w", err)
	}
	for _, member := range members {
		if member != nil && member.Username == username {
			return member.ID, nil
		}
	}
	return 0, fmt.Errorf("project member not found")
}

func normalizeUsername(username string) string {
	return strings.TrimPrefix(strings.TrimSpace(username), "@")
}

func uniqueInt64(values []int64) []int64 {
	seen := map[int64]struct{}{}
	out := make([]int64, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func (c *Client) mergeRequestBranch(next int) string {
	prefix := strings.Trim(strings.TrimSpace(c.cfg.MR.BranchPrefix), "/")
	if prefix == "" {
		prefix = "terrascaler/scale"
	}
	return fmt.Sprintf("%s-%s-to-%d", prefix, sanitizeBranchPart(c.cfg.Target.Attribute), next)
}

func (c *Client) mergeRequestDescription(current int, next int) string {
	description := strings.TrimSpace(c.cfg.MR.Description)
	if description == "" {
		description = "Automated Terrascaler scale-up proposal."
	}
	return fmt.Sprintf(`%s

Terrascaler proposes increasing %s from %d to %d.

- Target branch: %s
- Terraform file: %s
- Terraform target: %s %v %s
`, description, c.cfg.Target.Attribute, current, next, c.cfg.Branch, c.cfg.File, c.cfg.Target.BlockType, c.cfg.Target.Labels, c.cfg.Target.Attribute)
}

func sanitizeBranchPart(value string) string {
	value = strings.ToLower(value)
	var builder strings.Builder
	previousDash := false
	for _, char := range value {
		valid := (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9')
		if valid {
			builder.WriteRune(char)
			previousDash = false
			continue
		}
		if !previousDash {
			builder.WriteRune('-')
			previousDash = true
		}
	}
	return strings.Trim(builder.String(), "-")
}

func (c *Client) fetch(ctx context.Context) ([]byte, string, error) {
	return c.fetchRef(ctx, c.cfg.Branch)
}

func (c *Client) fetchRef(ctx context.Context, ref string) ([]byte, string, error) {
	file, _, err := c.api.RepositoryFiles.GetFile(c.cfg.Project, c.cfg.File, &gl.GetFileOptions{
		Ref: gl.Ptr(ref),
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

func isNotFoundError(err error) bool {
	var responseErr *gl.ErrorResponse
	return errors.As(err, &responseErr) &&
		responseErr.Response != nil &&
		responseErr.Response.StatusCode == http.StatusNotFound
}

func isBranchAlreadyExistsError(err error) bool {
	var responseErr *gl.ErrorResponse
	if !errors.As(err, &responseErr) || responseErr.Response == nil {
		return false
	}
	return responseErr.Response.StatusCode == http.StatusBadRequest &&
		strings.Contains(strings.ToLower(responseErr.Message), "branch already exists")
}
