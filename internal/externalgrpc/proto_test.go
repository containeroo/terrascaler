package externalgrpc

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

type testServer struct {
	UnimplementedCloudProviderServer
}

func (testServer) NodeGroups(context.Context, *NodeGroupsRequest) (*NodeGroupsResponse, error) {
	return &NodeGroupsResponse{NodeGroups: []*NodeGroup{{Id: "default", MinSize: 1, MaxSize: 3}}}, nil
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
