package server

import (
	"time"

	"github.com/OpenSlides/openslides-cli/internal/constants"
	"github.com/OpenSlides/openslides-cli/internal/k8s/actions"
	"github.com/OpenSlides/openslides-cli/internal/k8s/client"
	pb "github.com/OpenSlides/openslides-cli/proto/osmanage"
)

func (s *OsmanageServiceServer) StopInstance(
	req *pb.StopInstanceRequest,
	stream pb.OsmanageService_StopInstanceServer,
) error {
	k8sClient, err := client.New(req.Kubeconfig)
	if err != nil {
		return stream.Send(&pb.StopInstanceResponse{
			Complete: true,
			Error:    err.Error(),
		})
	}

	ctx := stream.Context()
	timeout := time.Duration(req.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = constants.DefaultNamespaceTimeout
	}

	err = actions.StopInstance(ctx, k8sClient, req.InstanceDir, timeout,
		func(elapsedSeconds int) error {
			return stream.Send(&pb.StopInstanceResponse{
				Complete:       false,
				ElapsedSeconds: int32(elapsedSeconds),
			})
		},
	)
	if err != nil {
		return stream.Send(&pb.StopInstanceResponse{
			Complete: true,
			Error:    err.Error(),
		})
	}
	return stream.Send(&pb.StopInstanceResponse{
		Complete: true,
	})
}
