package externalgrpc

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type testServer struct {
	UnimplementedCloudProviderServer
}

func (testServer) NodeGroups(context.Context, *NodeGroupsRequest) (*NodeGroupsResponse, error) {
	return &NodeGroupsResponse{NodeGroups: []*NodeGroup{{Id: "default", MinSize: 1, MaxSize: 3}}}, nil
}

func (testServer) NodeGroupGetOptions(_ context.Context, req *NodeGroupAutoscalingOptionsRequest) (*NodeGroupAutoscalingOptionsResponse, error) {
	return &NodeGroupAutoscalingOptionsResponse{NodeGroupAutoscalingOptions: req.Defaults}, nil
}

func TestGRPCBinding(t *testing.T) {
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	RegisterCloudProviderServer(server, testServer{})
	go func() {
		_ = server.Serve(listener)
	}()
	defer server.Stop()

	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("grpc.NewClient() error = %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Errorf("conn.Close() error = %v", err)
		}
	}()

	var response NodeGroupsResponse
	if err := conn.Invoke(context.Background(), "/clusterautoscaler.cloudprovider.v1.externalgrpc.CloudProvider/NodeGroups", &NodeGroupsRequest{}, &response); err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}
	if len(response.NodeGroups) != 1 || response.NodeGroups[0].Id != "default" {
		t.Fatalf("unexpected response: %#v", response.NodeGroups)
	}
}

func TestNodeGroupGetOptionsKeepsLegacyDurationFields(t *testing.T) {
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	RegisterCloudProviderServer(server, testServer{})
	go func() {
		_ = server.Serve(listener)
	}()
	defer server.Stop()

	conn, err := grpc.NewClient("passthrough:///bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("grpc.NewClient() error = %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Errorf("conn.Close() error = %v", err)
		}
	}()

	request := &NodeGroupAutoscalingOptionsRequest{
		Id: "default",
		Defaults: &NodeGroupAutoscalingOptions{
			ScaleDownUnneededTime: &metav1.Duration{Duration: 10 * time.Minute},
			ScaleDownUnreadyTime:  &metav1.Duration{Duration: 20 * time.Minute},
			MaxNodeProvisionTime:  &metav1.Duration{Duration: 15 * time.Minute},
		},
	}
	var response NodeGroupAutoscalingOptionsResponse
	if err := conn.Invoke(context.Background(), "/clusterautoscaler.cloudprovider.v1.externalgrpc.CloudProvider/NodeGroupGetOptions", request, &response); err != nil {
		t.Fatalf("Invoke() error = %v", err)
	}

	options := response.NodeGroupAutoscalingOptions
	if options == nil {
		t.Fatal("NodeGroupAutoscalingOptions = nil, want defaults")
		return
	}
	if options.ScaleDownUnneededTime == nil || options.ScaleDownUnneededTime.Duration != 10*time.Minute {
		t.Fatalf("ScaleDownUnneededTime = %#v, want 10m", options.ScaleDownUnneededTime)
	}
	if options.ScaleDownUnreadyTime == nil || options.ScaleDownUnreadyTime.Duration != 20*time.Minute {
		t.Fatalf("ScaleDownUnreadyTime = %#v, want 20m", options.ScaleDownUnreadyTime)
	}
	if options.MaxNodeProvisionTime == nil || options.MaxNodeProvisionTime.Duration != 15*time.Minute {
		t.Fatalf("MaxNodeProvisionTime = %#v, want 15m", options.MaxNodeProvisionTime)
	}
}
