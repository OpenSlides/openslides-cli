package server

import (
	"time"

	"github.com/OpenSlides/openslides-cli/internal/constants"
	"github.com/OpenSlides/openslides-cli/internal/k8s/actions"
	"github.com/OpenSlides/openslides-cli/internal/k8s/client"
	pb "github.com/OpenSlides/openslides-cli/proto/osmanage"
)

func (s *OsmanageServiceServer) StartInstance(
	req *pb.StartInstanceRequest,
	stream pb.OsmanageService_StartInstanceServer,
) error {
	k8sClient, err := client.New(req.Kubeconfig)
	if err != nil {
		return stream.Send(&pb.StartInstanceResponse{
			Complete: true,
			Error:    err.Error(),
		})
	}

	ctx := stream.Context()
	timeout := time.Duration(req.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = constants.DefaultInstanceTimeout
	}

	err = actions.StartInstance(ctx, k8sClient, req.InstanceDir, req.SkipReadyCheck, timeout,
		func(status *actions.HealthStatus) error {
			return stream.Send(&pb.StartInstanceResponse{
				Complete:  false,
				ReadyPods: int32(status.Ready),
				TotalPods: int32(status.Total),
			})
		},
	)
	if err != nil {
		return stream.Send(&pb.StartInstanceResponse{
			Complete: true,
			Error:    err.Error(),
		})
	}
	return stream.Send(&pb.StartInstanceResponse{
		Complete: true,
	})
}
