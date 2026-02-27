package server

import (
	"time"

	"github.com/OpenSlides/openslides-cli/internal/constants"
	"github.com/OpenSlides/openslides-cli/internal/k8s/actions"
	"github.com/OpenSlides/openslides-cli/internal/k8s/client"
	pb "github.com/OpenSlides/openslides-cli/proto/osmanage"
)

func (s *OsmanageServiceServer) ScaleService(
	req *pb.ScaleServiceRequest,
	stream pb.OsmanageService_ScaleServiceServer,
) error {
	k8sClient, err := client.New(req.Kubeconfig)
	if err != nil {
		return stream.Send(&pb.ScaleServiceResponse{
			Complete: true,
			Error:    err.Error(),
		})
	}

	ctx := stream.Context()
	timeout := time.Duration(req.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = constants.DefaultDeploymentTimeout
	}

	err = actions.ScaleService(ctx, k8sClient, req.InstanceDir, req.Service, req.SkipReadyCheck, timeout,
		func(status *actions.DeploymentStatus) error {
			return stream.Send(&pb.ScaleServiceResponse{
				Complete:        false,
				ReadyReplicas:   int32(status.Ready),
				DesiredReplicas: int32(status.Desired),
			})
		},
	)
	if err != nil {
		return stream.Send(&pb.ScaleServiceResponse{
			Complete: true,
			Error:    err.Error(),
		})
	}
	return stream.Send(&pb.ScaleServiceResponse{
		Complete: true,
	})
}
