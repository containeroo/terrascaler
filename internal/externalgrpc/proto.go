package externalgrpc

import (
	context "context"

	proto "github.com/golang/protobuf/proto" //nolint:staticcheck // The handwritten gRPC compatibility types implement the legacy proto.Message interface.
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	anypb "google.golang.org/protobuf/types/known/anypb"
	durationpb "google.golang.org/protobuf/types/known/durationpb"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

type NodeGroup struct {
	Id      string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	MinSize int32  `protobuf:"varint,2,opt,name=minSize,proto3" json:"minSize,omitempty"`
	MaxSize int32  `protobuf:"varint,3,opt,name=maxSize,proto3" json:"maxSize,omitempty"`
	Debug   string `protobuf:"bytes,4,opt,name=debug,proto3" json:"debug,omitempty"`
}

func (m *NodeGroup) Reset()         { *m = NodeGroup{} }
func (m *NodeGroup) String() string { return proto.CompactTextString(m) }
func (*NodeGroup) ProtoMessage()    {}

type ExternalGrpcNode struct {
	ProviderID  string            `protobuf:"bytes,1,opt,name=providerID,proto3" json:"providerID,omitempty"`
	Name        string            `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	Labels      map[string]string `protobuf:"bytes,3,rep,name=labels,proto3" json:"labels,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	Annotations map[string]string `protobuf:"bytes,4,rep,name=annotations,proto3" json:"annotations,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
}

func (m *ExternalGrpcNode) Reset()         { *m = ExternalGrpcNode{} }
func (m *ExternalGrpcNode) String() string { return proto.CompactTextString(m) }
func (*ExternalGrpcNode) ProtoMessage()    {}

type NodeGroupsRequest struct{}

func (m *NodeGroupsRequest) Reset()         { *m = NodeGroupsRequest{} }
func (m *NodeGroupsRequest) String() string { return proto.CompactTextString(m) }
func (*NodeGroupsRequest) ProtoMessage()    {}

type NodeGroupsResponse struct {
	NodeGroups []*NodeGroup `protobuf:"bytes,1,rep,name=nodeGroups,proto3" json:"nodeGroups,omitempty"`
}

func (m *NodeGroupsResponse) Reset()         { *m = NodeGroupsResponse{} }
func (m *NodeGroupsResponse) String() string { return proto.CompactTextString(m) }
func (*NodeGroupsResponse) ProtoMessage()    {}

type NodeGroupForNodeRequest struct {
	Node *ExternalGrpcNode `protobuf:"bytes,1,opt,name=node,proto3" json:"node,omitempty"`
}

func (m *NodeGroupForNodeRequest) Reset()         { *m = NodeGroupForNodeRequest{} }
func (m *NodeGroupForNodeRequest) String() string { return proto.CompactTextString(m) }
func (*NodeGroupForNodeRequest) ProtoMessage()    {}

type NodeGroupForNodeResponse struct {
	NodeGroup *NodeGroup `protobuf:"bytes,1,opt,name=nodeGroup,proto3" json:"nodeGroup,omitempty"`
}

func (m *NodeGroupForNodeResponse) Reset()         { *m = NodeGroupForNodeResponse{} }
func (m *NodeGroupForNodeResponse) String() string { return proto.CompactTextString(m) }
func (*NodeGroupForNodeResponse) ProtoMessage()    {}

type PricingNodePriceRequest struct {
	Node           *ExternalGrpcNode      `protobuf:"bytes,1,opt,name=node,proto3" json:"node,omitempty"`
	StartTimestamp *timestamppb.Timestamp `protobuf:"bytes,4,opt,name=startTimestamp,proto3" json:"startTimestamp,omitempty"`
	EndTimestamp   *timestamppb.Timestamp `protobuf:"bytes,5,opt,name=endTimestamp,proto3" json:"endTimestamp,omitempty"`
}

func (m *PricingNodePriceRequest) Reset()         { *m = PricingNodePriceRequest{} }
func (m *PricingNodePriceRequest) String() string { return proto.CompactTextString(m) }
func (*PricingNodePriceRequest) ProtoMessage()    {}

type PricingNodePriceResponse struct {
	Price float64 `protobuf:"fixed64,1,opt,name=price,proto3" json:"price,omitempty"`
}

func (m *PricingNodePriceResponse) Reset()         { *m = PricingNodePriceResponse{} }
func (m *PricingNodePriceResponse) String() string { return proto.CompactTextString(m) }
func (*PricingNodePriceResponse) ProtoMessage()    {}

type PricingPodPriceRequest struct {
	PodBytes       []byte                 `protobuf:"bytes,4,opt,name=pod_bytes,json=podBytes,proto3" json:"pod_bytes,omitempty"`
	StartTimestamp *timestamppb.Timestamp `protobuf:"bytes,5,opt,name=startTimestamp,proto3" json:"startTimestamp,omitempty"`
	EndTimestamp   *timestamppb.Timestamp `protobuf:"bytes,6,opt,name=endTimestamp,proto3" json:"endTimestamp,omitempty"`
}

func (m *PricingPodPriceRequest) Reset()         { *m = PricingPodPriceRequest{} }
func (m *PricingPodPriceRequest) String() string { return proto.CompactTextString(m) }
func (*PricingPodPriceRequest) ProtoMessage()    {}

type PricingPodPriceResponse struct {
	Price float64 `protobuf:"fixed64,1,opt,name=price,proto3" json:"price,omitempty"`
}

func (m *PricingPodPriceResponse) Reset()         { *m = PricingPodPriceResponse{} }
func (m *PricingPodPriceResponse) String() string { return proto.CompactTextString(m) }
func (*PricingPodPriceResponse) ProtoMessage()    {}

type GPULabelRequest struct{}

func (m *GPULabelRequest) Reset()         { *m = GPULabelRequest{} }
func (m *GPULabelRequest) String() string { return proto.CompactTextString(m) }
func (*GPULabelRequest) ProtoMessage()    {}

type GPULabelResponse struct {
	Label string `protobuf:"bytes,1,opt,name=label,proto3" json:"label,omitempty"`
}

func (m *GPULabelResponse) Reset()         { *m = GPULabelResponse{} }
func (m *GPULabelResponse) String() string { return proto.CompactTextString(m) }
func (*GPULabelResponse) ProtoMessage()    {}

type GetAvailableGPUTypesRequest struct{}

func (m *GetAvailableGPUTypesRequest) Reset()         { *m = GetAvailableGPUTypesRequest{} }
func (m *GetAvailableGPUTypesRequest) String() string { return proto.CompactTextString(m) }
func (*GetAvailableGPUTypesRequest) ProtoMessage()    {}

type GetAvailableGPUTypesResponse struct {
	GpuTypes map[string]*anypb.Any `protobuf:"bytes,1,rep,name=gpuTypes,proto3" json:"gpuTypes,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
}

func (m *GetAvailableGPUTypesResponse) Reset()         { *m = GetAvailableGPUTypesResponse{} }
func (m *GetAvailableGPUTypesResponse) String() string { return proto.CompactTextString(m) }
func (*GetAvailableGPUTypesResponse) ProtoMessage()    {}

type CleanupRequest struct{}

func (m *CleanupRequest) Reset()         { *m = CleanupRequest{} }
func (m *CleanupRequest) String() string { return proto.CompactTextString(m) }
func (*CleanupRequest) ProtoMessage()    {}

type CleanupResponse struct{}

func (m *CleanupResponse) Reset()         { *m = CleanupResponse{} }
func (m *CleanupResponse) String() string { return proto.CompactTextString(m) }
func (*CleanupResponse) ProtoMessage()    {}

type RefreshRequest struct{}

func (m *RefreshRequest) Reset()         { *m = RefreshRequest{} }
func (m *RefreshRequest) String() string { return proto.CompactTextString(m) }
func (*RefreshRequest) ProtoMessage()    {}

type RefreshResponse struct{}

func (m *RefreshResponse) Reset()         { *m = RefreshResponse{} }
func (m *RefreshResponse) String() string { return proto.CompactTextString(m) }
func (*RefreshResponse) ProtoMessage()    {}

type NodeGroupTargetSizeRequest struct {
	Id string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
}

func (m *NodeGroupTargetSizeRequest) Reset()         { *m = NodeGroupTargetSizeRequest{} }
func (m *NodeGroupTargetSizeRequest) String() string { return proto.CompactTextString(m) }
func (*NodeGroupTargetSizeRequest) ProtoMessage()    {}

type NodeGroupTargetSizeResponse struct {
	TargetSize int32 `protobuf:"varint,1,opt,name=targetSize,proto3" json:"targetSize,omitempty"`
}

func (m *NodeGroupTargetSizeResponse) Reset()         { *m = NodeGroupTargetSizeResponse{} }
func (m *NodeGroupTargetSizeResponse) String() string { return proto.CompactTextString(m) }
func (*NodeGroupTargetSizeResponse) ProtoMessage()    {}

type NodeGroupIncreaseSizeRequest struct {
	Delta int32  `protobuf:"varint,1,opt,name=delta,proto3" json:"delta,omitempty"`
	Id    string `protobuf:"bytes,2,opt,name=id,proto3" json:"id,omitempty"`
}

func (m *NodeGroupIncreaseSizeRequest) Reset()         { *m = NodeGroupIncreaseSizeRequest{} }
func (m *NodeGroupIncreaseSizeRequest) String() string { return proto.CompactTextString(m) }
func (*NodeGroupIncreaseSizeRequest) ProtoMessage()    {}

type NodeGroupIncreaseSizeResponse struct{}

func (m *NodeGroupIncreaseSizeResponse) Reset()         { *m = NodeGroupIncreaseSizeResponse{} }
func (m *NodeGroupIncreaseSizeResponse) String() string { return proto.CompactTextString(m) }
func (*NodeGroupIncreaseSizeResponse) ProtoMessage()    {}

type NodeGroupDeleteNodesRequest struct {
	Nodes []*ExternalGrpcNode `protobuf:"bytes,1,rep,name=nodes,proto3" json:"nodes,omitempty"`
	Id    string              `protobuf:"bytes,2,opt,name=id,proto3" json:"id,omitempty"`
}

func (m *NodeGroupDeleteNodesRequest) Reset()         { *m = NodeGroupDeleteNodesRequest{} }
func (m *NodeGroupDeleteNodesRequest) String() string { return proto.CompactTextString(m) }
func (*NodeGroupDeleteNodesRequest) ProtoMessage()    {}

type NodeGroupDeleteNodesResponse struct{}

func (m *NodeGroupDeleteNodesResponse) Reset()         { *m = NodeGroupDeleteNodesResponse{} }
func (m *NodeGroupDeleteNodesResponse) String() string { return proto.CompactTextString(m) }
func (*NodeGroupDeleteNodesResponse) ProtoMessage()    {}

type NodeGroupDecreaseTargetSizeRequest struct {
	Delta int32  `protobuf:"varint,1,opt,name=delta,proto3" json:"delta,omitempty"`
	Id    string `protobuf:"bytes,2,opt,name=id,proto3" json:"id,omitempty"`
}

func (m *NodeGroupDecreaseTargetSizeRequest) Reset()         { *m = NodeGroupDecreaseTargetSizeRequest{} }
func (m *NodeGroupDecreaseTargetSizeRequest) String() string { return proto.CompactTextString(m) }
func (*NodeGroupDecreaseTargetSizeRequest) ProtoMessage()    {}

type NodeGroupDecreaseTargetSizeResponse struct{}

func (m *NodeGroupDecreaseTargetSizeResponse) Reset()         { *m = NodeGroupDecreaseTargetSizeResponse{} }
func (m *NodeGroupDecreaseTargetSizeResponse) String() string { return proto.CompactTextString(m) }
func (*NodeGroupDecreaseTargetSizeResponse) ProtoMessage()    {}

type NodeGroupNodesRequest struct {
	Id string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
}

func (m *NodeGroupNodesRequest) Reset()         { *m = NodeGroupNodesRequest{} }
func (m *NodeGroupNodesRequest) String() string { return proto.CompactTextString(m) }
func (*NodeGroupNodesRequest) ProtoMessage()    {}

type NodeGroupNodesResponse struct {
	Instances []*Instance `protobuf:"bytes,1,rep,name=instances,proto3" json:"instances,omitempty"`
}

func (m *NodeGroupNodesResponse) Reset()         { *m = NodeGroupNodesResponse{} }
func (m *NodeGroupNodesResponse) String() string { return proto.CompactTextString(m) }
func (*NodeGroupNodesResponse) ProtoMessage()    {}

type Instance struct {
	Id     string          `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Status *InstanceStatus `protobuf:"bytes,2,opt,name=status,proto3" json:"status,omitempty"`
}

func (m *Instance) Reset()         { *m = Instance{} }
func (m *Instance) String() string { return proto.CompactTextString(m) }
func (*Instance) ProtoMessage()    {}

type InstanceStatus struct {
	InstanceState InstanceStatus_InstanceState `protobuf:"varint,1,opt,name=instanceState,proto3,enum=clusterautoscaler.cloudprovider.v1.externalgrpc.InstanceStatus_InstanceState" json:"instanceState,omitempty"`
	ErrorInfo     *InstanceErrorInfo           `protobuf:"bytes,2,opt,name=errorInfo,proto3" json:"errorInfo,omitempty"`
}

func (m *InstanceStatus) Reset()         { *m = InstanceStatus{} }
func (m *InstanceStatus) String() string { return proto.CompactTextString(m) }
func (*InstanceStatus) ProtoMessage()    {}

type InstanceStatus_InstanceState int32

const (
	InstanceStatus_unspecified      InstanceStatus_InstanceState = 0
	InstanceStatus_instanceRunning  InstanceStatus_InstanceState = 1
	InstanceStatus_instanceCreating InstanceStatus_InstanceState = 2
	InstanceStatus_instanceDeleting InstanceStatus_InstanceState = 3
)

type InstanceErrorInfo struct {
	ErrorCode          string `protobuf:"bytes,1,opt,name=errorCode,proto3" json:"errorCode,omitempty"`
	ErrorMessage       string `protobuf:"bytes,2,opt,name=errorMessage,proto3" json:"errorMessage,omitempty"`
	InstanceErrorClass int32  `protobuf:"varint,3,opt,name=instanceErrorClass,proto3" json:"instanceErrorClass,omitempty"`
}

func (m *InstanceErrorInfo) Reset()         { *m = InstanceErrorInfo{} }
func (m *InstanceErrorInfo) String() string { return proto.CompactTextString(m) }
func (*InstanceErrorInfo) ProtoMessage()    {}

type NodeGroupTemplateNodeInfoRequest struct {
	Id string `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
}

func (m *NodeGroupTemplateNodeInfoRequest) Reset()         { *m = NodeGroupTemplateNodeInfoRequest{} }
func (m *NodeGroupTemplateNodeInfoRequest) String() string { return proto.CompactTextString(m) }
func (*NodeGroupTemplateNodeInfoRequest) ProtoMessage()    {}

type NodeGroupTemplateNodeInfoResponse struct {
	NodeBytes []byte `protobuf:"bytes,2,opt,name=nodeBytes,proto3" json:"nodeBytes,omitempty"`
}

func (m *NodeGroupTemplateNodeInfoResponse) Reset()         { *m = NodeGroupTemplateNodeInfoResponse{} }
func (m *NodeGroupTemplateNodeInfoResponse) String() string { return proto.CompactTextString(m) }
func (*NodeGroupTemplateNodeInfoResponse) ProtoMessage()    {}

type NodeGroupAutoscalingOptionsRequest struct {
	Id       string                       `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Defaults *NodeGroupAutoscalingOptions `protobuf:"bytes,2,opt,name=defaults,proto3" json:"defaults,omitempty"`
}

func (m *NodeGroupAutoscalingOptionsRequest) Reset() {
	*m = NodeGroupAutoscalingOptionsRequest{}
}
func (m *NodeGroupAutoscalingOptionsRequest) String() string { return proto.CompactTextString(m) }
func (*NodeGroupAutoscalingOptionsRequest) ProtoMessage()    {}

type NodeGroupAutoscalingOptions struct {
	ScaleDownUtilizationThreshold    float64              `protobuf:"fixed64,1,opt,name=scaleDownUtilizationThreshold,proto3" json:"scaleDownUtilizationThreshold,omitempty"`
	ScaleDownGpuUtilizationThreshold float64              `protobuf:"fixed64,2,opt,name=scaleDownGpuUtilizationThreshold,proto3" json:"scaleDownGpuUtilizationThreshold,omitempty"`
	ZeroOrMaxNodeScaling             bool                 `protobuf:"varint,6,opt,name=zeroOrMaxNodeScaling,proto3" json:"zeroOrMaxNodeScaling,omitempty"`
	IgnoreDaemonSetsUtilization      bool                 `protobuf:"varint,7,opt,name=ignoreDaemonSetsUtilization,proto3" json:"ignoreDaemonSetsUtilization,omitempty"`
	ScaleDownUnneededDuration        *durationpb.Duration `protobuf:"bytes,8,opt,name=scaleDownUnneededDuration,proto3" json:"scaleDownUnneededDuration,omitempty"`
	ScaleDownUnreadyDuration         *durationpb.Duration `protobuf:"bytes,9,opt,name=scaleDownUnreadyDuration,proto3" json:"scaleDownUnreadyDuration,omitempty"`
	MaxNodeProvisionDuration         *durationpb.Duration `protobuf:"bytes,10,opt,name=MaxNodeProvisionDuration,proto3" json:"MaxNodeProvisionDuration,omitempty"`
}

func (m *NodeGroupAutoscalingOptions) Reset()         { *m = NodeGroupAutoscalingOptions{} }
func (m *NodeGroupAutoscalingOptions) String() string { return proto.CompactTextString(m) }
func (*NodeGroupAutoscalingOptions) ProtoMessage()    {}

type NodeGroupAutoscalingOptionsResponse struct {
	NodeGroupAutoscalingOptions *NodeGroupAutoscalingOptions `protobuf:"bytes,1,opt,name=nodeGroupAutoscalingOptions,proto3" json:"nodeGroupAutoscalingOptions,omitempty"`
}

func (m *NodeGroupAutoscalingOptionsResponse) Reset() {
	*m = NodeGroupAutoscalingOptionsResponse{}
}
func (m *NodeGroupAutoscalingOptionsResponse) String() string { return proto.CompactTextString(m) }
func (*NodeGroupAutoscalingOptionsResponse) ProtoMessage()    {}

type CloudProviderServer interface {
	NodeGroups(context.Context, *NodeGroupsRequest) (*NodeGroupsResponse, error)
	NodeGroupForNode(context.Context, *NodeGroupForNodeRequest) (*NodeGroupForNodeResponse, error)
	PricingNodePrice(context.Context, *PricingNodePriceRequest) (*PricingNodePriceResponse, error)
	PricingPodPrice(context.Context, *PricingPodPriceRequest) (*PricingPodPriceResponse, error)
	GPULabel(context.Context, *GPULabelRequest) (*GPULabelResponse, error)
	GetAvailableGPUTypes(context.Context, *GetAvailableGPUTypesRequest) (*GetAvailableGPUTypesResponse, error)
	Cleanup(context.Context, *CleanupRequest) (*CleanupResponse, error)
	Refresh(context.Context, *RefreshRequest) (*RefreshResponse, error)
	NodeGroupTargetSize(context.Context, *NodeGroupTargetSizeRequest) (*NodeGroupTargetSizeResponse, error)
	NodeGroupIncreaseSize(context.Context, *NodeGroupIncreaseSizeRequest) (*NodeGroupIncreaseSizeResponse, error)
	NodeGroupDeleteNodes(context.Context, *NodeGroupDeleteNodesRequest) (*NodeGroupDeleteNodesResponse, error)
	NodeGroupDecreaseTargetSize(context.Context, *NodeGroupDecreaseTargetSizeRequest) (*NodeGroupDecreaseTargetSizeResponse, error)
	NodeGroupNodes(context.Context, *NodeGroupNodesRequest) (*NodeGroupNodesResponse, error)
	NodeGroupTemplateNodeInfo(context.Context, *NodeGroupTemplateNodeInfoRequest) (*NodeGroupTemplateNodeInfoResponse, error)
	NodeGroupGetOptions(context.Context, *NodeGroupAutoscalingOptionsRequest) (*NodeGroupAutoscalingOptionsResponse, error)
}

type UnimplementedCloudProviderServer struct{}

func (UnimplementedCloudProviderServer) NodeGroups(context.Context, *NodeGroupsRequest) (*NodeGroupsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method NodeGroups not implemented")
}
func (UnimplementedCloudProviderServer) NodeGroupForNode(context.Context, *NodeGroupForNodeRequest) (*NodeGroupForNodeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method NodeGroupForNode not implemented")
}
func (UnimplementedCloudProviderServer) PricingNodePrice(context.Context, *PricingNodePriceRequest) (*PricingNodePriceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PricingNodePrice not implemented")
}
func (UnimplementedCloudProviderServer) PricingPodPrice(context.Context, *PricingPodPriceRequest) (*PricingPodPriceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PricingPodPrice not implemented")
}
func (UnimplementedCloudProviderServer) GPULabel(context.Context, *GPULabelRequest) (*GPULabelResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GPULabel not implemented")
}
func (UnimplementedCloudProviderServer) GetAvailableGPUTypes(context.Context, *GetAvailableGPUTypesRequest) (*GetAvailableGPUTypesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetAvailableGPUTypes not implemented")
}
func (UnimplementedCloudProviderServer) Cleanup(context.Context, *CleanupRequest) (*CleanupResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Cleanup not implemented")
}
func (UnimplementedCloudProviderServer) Refresh(context.Context, *RefreshRequest) (*RefreshResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Refresh not implemented")
}
func (UnimplementedCloudProviderServer) NodeGroupTargetSize(context.Context, *NodeGroupTargetSizeRequest) (*NodeGroupTargetSizeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method NodeGroupTargetSize not implemented")
}
func (UnimplementedCloudProviderServer) NodeGroupIncreaseSize(context.Context, *NodeGroupIncreaseSizeRequest) (*NodeGroupIncreaseSizeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method NodeGroupIncreaseSize not implemented")
}
func (UnimplementedCloudProviderServer) NodeGroupDeleteNodes(context.Context, *NodeGroupDeleteNodesRequest) (*NodeGroupDeleteNodesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method NodeGroupDeleteNodes not implemented")
}
func (UnimplementedCloudProviderServer) NodeGroupDecreaseTargetSize(context.Context, *NodeGroupDecreaseTargetSizeRequest) (*NodeGroupDecreaseTargetSizeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method NodeGroupDecreaseTargetSize not implemented")
}
func (UnimplementedCloudProviderServer) NodeGroupNodes(context.Context, *NodeGroupNodesRequest) (*NodeGroupNodesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method NodeGroupNodes not implemented")
}
func (UnimplementedCloudProviderServer) NodeGroupTemplateNodeInfo(context.Context, *NodeGroupTemplateNodeInfoRequest) (*NodeGroupTemplateNodeInfoResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method NodeGroupTemplateNodeInfo not implemented")
}
func (UnimplementedCloudProviderServer) NodeGroupGetOptions(context.Context, *NodeGroupAutoscalingOptionsRequest) (*NodeGroupAutoscalingOptionsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method NodeGroupGetOptions not implemented")
}

func RegisterCloudProviderServer(s grpc.ServiceRegistrar, srv CloudProviderServer) {
	s.RegisterService(&CloudProvider_ServiceDesc, srv)
}

var CloudProvider_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "clusterautoscaler.cloudprovider.v1.externalgrpc.CloudProvider",
	HandlerType: (*CloudProviderServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "NodeGroups", Handler: _CloudProvider_NodeGroups_Handler},
		{MethodName: "NodeGroupForNode", Handler: _CloudProvider_NodeGroupForNode_Handler},
		{MethodName: "PricingNodePrice", Handler: _CloudProvider_PricingNodePrice_Handler},
		{MethodName: "PricingPodPrice", Handler: _CloudProvider_PricingPodPrice_Handler},
		{MethodName: "GPULabel", Handler: _CloudProvider_GPULabel_Handler},
		{MethodName: "GetAvailableGPUTypes", Handler: _CloudProvider_GetAvailableGPUTypes_Handler},
		{MethodName: "Cleanup", Handler: _CloudProvider_Cleanup_Handler},
		{MethodName: "Refresh", Handler: _CloudProvider_Refresh_Handler},
		{MethodName: "NodeGroupTargetSize", Handler: _CloudProvider_NodeGroupTargetSize_Handler},
		{MethodName: "NodeGroupIncreaseSize", Handler: _CloudProvider_NodeGroupIncreaseSize_Handler},
		{MethodName: "NodeGroupDeleteNodes", Handler: _CloudProvider_NodeGroupDeleteNodes_Handler},
		{MethodName: "NodeGroupDecreaseTargetSize", Handler: _CloudProvider_NodeGroupDecreaseTargetSize_Handler},
		{MethodName: "NodeGroupNodes", Handler: _CloudProvider_NodeGroupNodes_Handler},
		{MethodName: "NodeGroupTemplateNodeInfo", Handler: _CloudProvider_NodeGroupTemplateNodeInfo_Handler},
		{MethodName: "NodeGroupGetOptions", Handler: _CloudProvider_NodeGroupGetOptions_Handler},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "cloudprovider/externalgrpc/protos/externalgrpc.proto",
}

func _CloudProvider_NodeGroups_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(NodeGroupsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CloudProviderServer).NodeGroups(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/clusterautoscaler.cloudprovider.v1.externalgrpc.CloudProvider/NodeGroups"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CloudProviderServer).NodeGroups(ctx, req.(*NodeGroupsRequest))
	})
}

func _CloudProvider_NodeGroupForNode_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(NodeGroupForNodeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CloudProviderServer).NodeGroupForNode(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/clusterautoscaler.cloudprovider.v1.externalgrpc.CloudProvider/NodeGroupForNode"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CloudProviderServer).NodeGroupForNode(ctx, req.(*NodeGroupForNodeRequest))
	})
}

func _CloudProvider_PricingNodePrice_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PricingNodePriceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CloudProviderServer).PricingNodePrice(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/clusterautoscaler.cloudprovider.v1.externalgrpc.CloudProvider/PricingNodePrice"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CloudProviderServer).PricingNodePrice(ctx, req.(*PricingNodePriceRequest))
	})
}

func _CloudProvider_PricingPodPrice_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PricingPodPriceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CloudProviderServer).PricingPodPrice(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/clusterautoscaler.cloudprovider.v1.externalgrpc.CloudProvider/PricingPodPrice"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CloudProviderServer).PricingPodPrice(ctx, req.(*PricingPodPriceRequest))
	})
}

func _CloudProvider_GPULabel_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GPULabelRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CloudProviderServer).GPULabel(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/clusterautoscaler.cloudprovider.v1.externalgrpc.CloudProvider/GPULabel"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CloudProviderServer).GPULabel(ctx, req.(*GPULabelRequest))
	})
}

func _CloudProvider_GetAvailableGPUTypes_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetAvailableGPUTypesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CloudProviderServer).GetAvailableGPUTypes(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/clusterautoscaler.cloudprovider.v1.externalgrpc.CloudProvider/GetAvailableGPUTypes"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CloudProviderServer).GetAvailableGPUTypes(ctx, req.(*GetAvailableGPUTypesRequest))
	})
}

func _CloudProvider_Cleanup_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CleanupRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CloudProviderServer).Cleanup(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/clusterautoscaler.cloudprovider.v1.externalgrpc.CloudProvider/Cleanup"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CloudProviderServer).Cleanup(ctx, req.(*CleanupRequest))
	})
}

func _CloudProvider_Refresh_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RefreshRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CloudProviderServer).Refresh(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/clusterautoscaler.cloudprovider.v1.externalgrpc.CloudProvider/Refresh"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CloudProviderServer).Refresh(ctx, req.(*RefreshRequest))
	})
}

func _CloudProvider_NodeGroupTargetSize_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(NodeGroupTargetSizeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CloudProviderServer).NodeGroupTargetSize(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/clusterautoscaler.cloudprovider.v1.externalgrpc.CloudProvider/NodeGroupTargetSize"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CloudProviderServer).NodeGroupTargetSize(ctx, req.(*NodeGroupTargetSizeRequest))
	})
}

func _CloudProvider_NodeGroupIncreaseSize_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(NodeGroupIncreaseSizeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CloudProviderServer).NodeGroupIncreaseSize(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/clusterautoscaler.cloudprovider.v1.externalgrpc.CloudProvider/NodeGroupIncreaseSize"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CloudProviderServer).NodeGroupIncreaseSize(ctx, req.(*NodeGroupIncreaseSizeRequest))
	})
}

func _CloudProvider_NodeGroupDeleteNodes_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(NodeGroupDeleteNodesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CloudProviderServer).NodeGroupDeleteNodes(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/clusterautoscaler.cloudprovider.v1.externalgrpc.CloudProvider/NodeGroupDeleteNodes"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CloudProviderServer).NodeGroupDeleteNodes(ctx, req.(*NodeGroupDeleteNodesRequest))
	})
}

func _CloudProvider_NodeGroupDecreaseTargetSize_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(NodeGroupDecreaseTargetSizeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CloudProviderServer).NodeGroupDecreaseTargetSize(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/clusterautoscaler.cloudprovider.v1.externalgrpc.CloudProvider/NodeGroupDecreaseTargetSize"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CloudProviderServer).NodeGroupDecreaseTargetSize(ctx, req.(*NodeGroupDecreaseTargetSizeRequest))
	})
}

func _CloudProvider_NodeGroupNodes_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(NodeGroupNodesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CloudProviderServer).NodeGroupNodes(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/clusterautoscaler.cloudprovider.v1.externalgrpc.CloudProvider/NodeGroupNodes"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CloudProviderServer).NodeGroupNodes(ctx, req.(*NodeGroupNodesRequest))
	})
}

func _CloudProvider_NodeGroupTemplateNodeInfo_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(NodeGroupTemplateNodeInfoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CloudProviderServer).NodeGroupTemplateNodeInfo(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/clusterautoscaler.cloudprovider.v1.externalgrpc.CloudProvider/NodeGroupTemplateNodeInfo"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CloudProviderServer).NodeGroupTemplateNodeInfo(ctx, req.(*NodeGroupTemplateNodeInfoRequest))
	})
}

func _CloudProvider_NodeGroupGetOptions_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(NodeGroupAutoscalingOptionsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(CloudProviderServer).NodeGroupGetOptions(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: "/clusterautoscaler.cloudprovider.v1.externalgrpc.CloudProvider/NodeGroupGetOptions"}
	return interceptor(ctx, in, info, func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(CloudProviderServer).NodeGroupGetOptions(ctx, req.(*NodeGroupAutoscalingOptionsRequest))
	})
}
