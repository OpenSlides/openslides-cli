package server

import (
	"context"
	"strings"

	"github.com/OpenSlides/openslides-cli/internal/k8s/actions"
	"github.com/OpenSlides/openslides-cli/internal/k8s/client"
	pb "github.com/OpenSlides/openslides-cli/proto/osmanage"
)

func (s *OsmanageServiceServer) GetNamespaceExists(ctx context.Context, req *pb.GetNamespaceExistsRequest) (*pb.GetNamespaceExistsResponse, error) {
	k8sClient, err := client.New(req.Kubeconfig)
	if err != nil {
		return &pb.GetNamespaceExistsResponse{Error: err.Error()}, nil
	}
	namespace := strings.ReplaceAll(req.InstanceUrl, ".", "")

	exists, err := actions.GetNamespaceExists(ctx, k8sClient.Clientset(), namespace)
	if err != nil {
		return &pb.GetNamespaceExistsResponse{Error: err.Error()}, nil
	}

	return &pb.GetNamespaceExistsResponse{Exists: exists}, nil
}
