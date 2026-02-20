package server

import (
	"context"

	"github.com/OpenSlides/openslides-cli/internal/k8s/actions"
	"github.com/OpenSlides/openslides-cli/internal/k8s/client"
	"github.com/OpenSlides/openslides-cli/internal/utils"
	pb "github.com/OpenSlides/openslides-cli/proto/osmanage"
)

func (s *OsmanageServiceServer) GetServiceAddress(ctx context.Context, req *pb.ServiceAddressRequest) (*pb.ServiceAddressResponse, error) {
	k8sClient, err := client.New(req.Kubeconfig)
	if err != nil {
		return &pb.ServiceAddressResponse{Error: err.Error()}, nil
	}
	namespace := utils.ExtractNamespace(req.InstanceDirPath)

	address, err := actions.GetServiceAddress(ctx, k8sClient.Clientset(), namespace, req.ServiceName)
	if err != nil {
		return &pb.ServiceAddressResponse{Error: err.Error()}, nil
	}

	return &pb.ServiceAddressResponse{Address: address}, nil
}
