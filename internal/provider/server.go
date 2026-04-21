package provider

import (
	"context"
	"fmt"
	"log/slog"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/containeroo/terrascaler/internal/config"
	"github.com/containeroo/terrascaler/internal/externalgrpc"
	"github.com/containeroo/terrascaler/internal/gitlab"
)

type Server struct {
	externalgrpc.UnimplementedCloudProviderServer

	cfg    config.Config
	gitlab *gitlab.Client
	logger *slog.Logger
}

func New(cfg config.Config, gitlabClient *gitlab.Client, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	return &Server{cfg: cfg, gitlab: gitlabClient, logger: logger}
}

func (s *Server) NodeGroups(context.Context, *externalgrpc.NodeGroupsRequest) (*externalgrpc.NodeGroupsResponse, error) {
	s.logger.Debug("NodeGroups requested",
		"node_group", s.cfg.NodeGroupID,
		"min_size", s.cfg.MinSize,
		"max_size", s.cfg.MaxSize,
	)
	return &externalgrpc.NodeGroupsResponse{NodeGroups: []*externalgrpc.NodeGroup{s.nodeGroup()}}, nil
}

func (s *Server) NodeGroupForNode(_ context.Context, req *externalgrpc.NodeGroupForNodeRequest) (*externalgrpc.NodeGroupForNodeResponse, error) {
	s.logger.Debug("NodeGroupForNode requested",
		"node_group", s.cfg.NodeGroupID,
		"node_name", nodeName(req.Node),
		"provider_id", providerID(req.Node),
	)
	return &externalgrpc.NodeGroupForNodeResponse{NodeGroup: s.nodeGroup()}, nil
}

func (s *Server) GPULabel(context.Context, *externalgrpc.GPULabelRequest) (*externalgrpc.GPULabelResponse, error) {
	return &externalgrpc.GPULabelResponse{}, nil
}

func (s *Server) GetAvailableGPUTypes(context.Context, *externalgrpc.GetAvailableGPUTypesRequest) (*externalgrpc.GetAvailableGPUTypesResponse, error) {
	return &externalgrpc.GetAvailableGPUTypesResponse{}, nil
}

func (s *Server) Cleanup(context.Context, *externalgrpc.CleanupRequest) (*externalgrpc.CleanupResponse, error) {
	s.logger.Debug("Cleanup requested")
	return &externalgrpc.CleanupResponse{}, nil
}

func (s *Server) Refresh(context.Context, *externalgrpc.RefreshRequest) (*externalgrpc.RefreshResponse, error) {
	s.logger.Debug("Refresh requested")
	return &externalgrpc.RefreshResponse{}, nil
}

func (s *Server) NodeGroupTargetSize(ctx context.Context, req *externalgrpc.NodeGroupTargetSizeRequest) (*externalgrpc.NodeGroupTargetSizeResponse, error) {
	s.logger.DebugContext(ctx, "NodeGroupTargetSize requested", "node_group", req.Id)
	if err := s.validateNodeGroup(req.Id); err != nil {
		s.logger.WarnContext(ctx, "NodeGroupTargetSize rejected", "node_group", req.Id, "error", err)
		return nil, err
	}

	targetSize, err := s.gitlab.TargetSize(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "NodeGroupTargetSize failed", "node_group", req.Id, "error", err)
		return nil, err
	}
	s.logger.InfoContext(ctx, "NodeGroupTargetSize completed",
		"node_group", req.Id,
		"target_size", targetSize,
	)
	return &externalgrpc.NodeGroupTargetSizeResponse{TargetSize: int32(targetSize)}, nil
}

func (s *Server) NodeGroupIncreaseSize(ctx context.Context, req *externalgrpc.NodeGroupIncreaseSizeRequest) (*externalgrpc.NodeGroupIncreaseSizeResponse, error) {
	s.logger.InfoContext(ctx, "NodeGroupIncreaseSize requested",
		"node_group", req.Id,
		"delta", req.Delta,
		"min_size", s.cfg.MinSize,
		"max_size", s.cfg.MaxSize,
	)
	if err := s.validateNodeGroup(req.Id); err != nil {
		s.logger.WarnContext(ctx, "NodeGroupIncreaseSize rejected", "node_group", req.Id, "delta", req.Delta, "error", err)
		return nil, err
	}
	if req.Delta <= 0 {
		err := fmt.Errorf("delta must be positive, got %d", req.Delta)
		s.logger.WarnContext(ctx, "NodeGroupIncreaseSize rejected", "node_group", req.Id, "delta", req.Delta, "error", err)
		return nil, err
	}

	_, _, err := s.gitlab.IncreaseTargetSize(ctx, int(req.Delta), func(current int, next int) error {
		if int32(next) > s.cfg.MaxSize {
			return fmt.Errorf("requested target size %d is larger than max-size %d", next, s.cfg.MaxSize)
		}
		if int32(current) < s.cfg.MinSize {
			return fmt.Errorf("current target size %d is smaller than min-size %d", current, s.cfg.MinSize)
		}
		return nil
	})
	if err != nil {
		s.logger.ErrorContext(ctx, "NodeGroupIncreaseSize failed", "node_group", req.Id, "delta", req.Delta, "error", err)
		return nil, err
	}
	s.logger.InfoContext(ctx, "NodeGroupIncreaseSize completed", "node_group", req.Id, "delta", req.Delta)
	return &externalgrpc.NodeGroupIncreaseSizeResponse{}, nil
}

func (s *Server) NodeGroupNodes(ctx context.Context, req *externalgrpc.NodeGroupNodesRequest) (*externalgrpc.NodeGroupNodesResponse, error) {
	s.logger.DebugContext(ctx, "NodeGroupNodes requested", "node_group", req.Id)
	return &externalgrpc.NodeGroupNodesResponse{}, nil
}

func (s *Server) NodeGroupTemplateNodeInfo(ctx context.Context, req *externalgrpc.NodeGroupTemplateNodeInfoRequest) (*externalgrpc.NodeGroupTemplateNodeInfoResponse, error) {
	s.logger.DebugContext(ctx, "NodeGroupTemplateNodeInfo requested", "node_group", req.Id)
	if err := s.validateNodeGroup(req.Id); err != nil {
		s.logger.WarnContext(ctx, "NodeGroupTemplateNodeInfo rejected", "node_group", req.Id, "error", err)
		return nil, err
	}

	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "terrascaler-template",
			Labels: s.templateLabels(),
		},
		Status: corev1.NodeStatus{
			Capacity: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(s.cfg.TemplateCPU),
				corev1.ResourceMemory: resource.MustParse(s.cfg.TemplateMemory),
				corev1.ResourcePods:   *resource.NewQuantity(s.cfg.TemplatePods, resource.DecimalSI),
			},
			Allocatable: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(s.cfg.TemplateCPU),
				corev1.ResourceMemory: resource.MustParse(s.cfg.TemplateMemory),
				corev1.ResourcePods:   *resource.NewQuantity(s.cfg.TemplatePods, resource.DecimalSI),
			},
		},
	}

	nodeBytes, err := node.Marshal()
	if err != nil {
		s.logger.ErrorContext(ctx, "NodeGroupTemplateNodeInfo failed", "node_group", req.Id, "error", err)
		return nil, err
	}
	s.logger.DebugContext(ctx, "NodeGroupTemplateNodeInfo completed",
		"node_group", req.Id,
		"cpu", s.cfg.TemplateCPU,
		"memory", s.cfg.TemplateMemory,
		"pods", s.cfg.TemplatePods,
	)
	return &externalgrpc.NodeGroupTemplateNodeInfoResponse{
		NodeInfo:  node,
		NodeBytes: nodeBytes,
	}, nil
}

func (s *Server) NodeGroupGetOptions(_ context.Context, req *externalgrpc.NodeGroupAutoscalingOptionsRequest) (*externalgrpc.NodeGroupAutoscalingOptionsResponse, error) {
	s.logger.Debug("NodeGroupGetOptions requested", "node_group", req.Id)
	if err := s.validateNodeGroup(req.Id); err != nil {
		s.logger.Warn("NodeGroupGetOptions rejected", "node_group", req.Id, "error", err)
		return nil, err
	}
	return &externalgrpc.NodeGroupAutoscalingOptionsResponse{NodeGroupAutoscalingOptions: req.Defaults}, nil
}

func (s *Server) nodeGroup() *externalgrpc.NodeGroup {
	return &externalgrpc.NodeGroup{
		Id:      s.cfg.NodeGroupID,
		MinSize: s.cfg.MinSize,
		MaxSize: s.cfg.MaxSize,
		Debug: fmt.Sprintf(
			"terrascaler node group %q updates %s:%s %s %q.%v.%s",
			s.cfg.NodeGroupID,
			s.cfg.GitLabProject,
			s.cfg.GitLabBranch,
			s.cfg.FilePath,
			s.cfg.BlockType,
			s.cfg.Labels,
			s.cfg.Attribute,
		),
	}
}

func nodeName(node *externalgrpc.ExternalGrpcNode) string {
	if node == nil {
		return ""
	}
	return node.Name
}

func providerID(node *externalgrpc.ExternalGrpcNode) string {
	if node == nil {
		return ""
	}
	return node.ProviderID
}

func (s *Server) validateNodeGroup(id string) error {
	if id != s.cfg.NodeGroupID {
		return fmt.Errorf("unknown node group %q", id)
	}
	return nil
}

func (s *Server) templateLabels() map[string]string {
	labels := map[string]string{
		"kubernetes.io/arch":     "amd64",
		"kubernetes.io/os":       "linux",
		"terrascaler/node-group": s.cfg.NodeGroupID,
	}
	for key, value := range s.cfg.TemplateLabels {
		labels[key] = value
	}
	return labels
}
