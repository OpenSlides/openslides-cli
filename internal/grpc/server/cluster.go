package server

import (
	"context"
	"fmt"

	"github.com/OpenSlides/openslides-cli/internal/k8s/actions"
	"github.com/OpenSlides/openslides-cli/internal/k8s/client"
	pb "github.com/OpenSlides/openslides-cli/proto/cluster"
)

type ClusterServer struct {
	pb.UnimplementedClusterServiceServer
}

func NewClusterServer() *ClusterServer {
	return &ClusterServer{}
}

func (s *ClusterServer) GetClusterStatus(ctx context.Context, req *pb.ClusterStatusRequest) (*pb.ClusterStatusResponse, error) {
	k8sClient, err := client.New(req.Kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("creating k8s client: %w", err)
	}

	status, err := actions.CheckClusterStatus(ctx, k8sClient)
	if err != nil {
		return nil, fmt.Errorf("checking cluster status: %w", err)
	}

	statusMsg := fmt.Sprintf("Cluster: %d/%d nodes ready", status.ReadyNodes, status.TotalNodes)

	return &pb.ClusterStatusResponse{
		Status:     statusMsg,
		TotalNodes: int32(status.TotalNodes),
		ReadyNodes: int32(status.ReadyNodes),
	}, nil
}
