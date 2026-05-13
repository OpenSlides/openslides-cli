package server

import (
	"time"

	"github.com/OpenSlides/openslides-cli/internal/constants"
	"github.com/OpenSlides/openslides-cli/internal/k8s/actions"
	"github.com/OpenSlides/openslides-cli/internal/k8s/client"
	pb "github.com/OpenSlides/openslides-cli/proto/osmanage"
)

func (s *OsmanageServiceServer) UpdateInstance(
	req *pb.UpdateInstanceRequest,
	stream pb.OsmanageService_UpdateInstanceServer,
) error {
	k8sClient, err := client.New(req.Kubeconfig)
	if err != nil {
		return stream.Send(&pb.UpdateInstanceResponse{
			Complete: true,
			Error:    err.Error(),
		})
	}

	ctx := stream.Context()
	timeout := time.Duration(req.TimeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = constants.DefaultInstanceTimeout
	}

	streamCallback := func(status *actions.HealthStatus) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		return stream.Send(healthStatusToUpdateResponse(status, false))
	}

	inactiveCallback := func() error {
		return stream.Send(&pb.UpdateInstanceResponse{
			Complete: true,
			Inactive: true,
		})
	}

	err = actions.UpdateInstance(ctx, k8sClient, req.InstanceDir, req.SkipReadyCheck, timeout, streamCallback, inactiveCallback)
	if err != nil {
		return stream.Send(&pb.UpdateInstanceResponse{
			Complete: true,
			Error:    err.Error(),
		})
	}
	return stream.Send(&pb.UpdateInstanceResponse{
		Complete: true,
	})
}

// Helper to convert internal type to proto
func healthStatusToUpdateResponse(status *actions.HealthStatus, complete bool) *pb.UpdateInstanceResponse {
	pods := make([]*pb.PodStatus, len(status.Pods))
	for i, pod := range status.Pods {
		pods[i] = &pb.PodStatus{
			Name:  pod.Name,
			Phase: string(pod.Status.Phase),
			Ready: actions.IsPodReady(&pod),
		}
	}

	return &pb.UpdateInstanceResponse{
		Healthy:    status.Healthy,
		ReadyPods:  int32(status.Ready),
		TotalPods:  int32(status.Total),
		ActivePods: int32(status.ActivePods),
		Pods:       pods,
		Complete:   complete,
	}
}
