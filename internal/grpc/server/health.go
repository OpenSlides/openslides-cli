package server

import (
	"strings"
	"time"

	"github.com/OpenSlides/openslides-cli/internal/constants"
	"github.com/OpenSlides/openslides-cli/internal/k8s/actions"
	"github.com/OpenSlides/openslides-cli/internal/k8s/client"
	pb "github.com/OpenSlides/openslides-cli/proto/osmanage"
)

func (s *OsmanageServiceServer) GetInstanceHealth(
	req *pb.GetInstanceHealthRequest,
	stream pb.OsmanageService_GetInstanceHealthServer,
) error {
	namespace := strings.ReplaceAll(req.InstanceUrl, ".", "")

	k8sClient, err := client.New(req.Kubeconfig)
	if err != nil {
		return stream.Send(&pb.GetInstanceHealthResponse{
			Complete: true,
			Error:    err.Error(),
		})
	}

	ctx := stream.Context()

	if req.Wait {
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

			return stream.Send(healthStatusToHealthResponse(status, false))
		}

		err := actions.WaitForInstanceHealthy(ctx, k8sClient, namespace, timeout, streamCallback)

		if err != nil {
			return stream.Send(&pb.GetInstanceHealthResponse{
				Complete: true,
				Error:    err.Error(),
			})
		}

		return stream.Send(&pb.GetInstanceHealthResponse{
			Complete: true,
		})
	}

	status, err := actions.GetHealthStatus(ctx, k8sClient, namespace)
	if err != nil {
		return stream.Send(&pb.GetInstanceHealthResponse{
			Complete: true,
			Error:    err.Error(),
		})
	}

	return stream.Send(healthStatusToHealthResponse(status, true))
}

// Helper to convert internal type to proto
func healthStatusToHealthResponse(status *actions.HealthStatus, complete bool) *pb.GetInstanceHealthResponse {
	pods := make([]*pb.PodStatus, len(status.Pods))
	for i, pod := range status.Pods {
		pods[i] = &pb.PodStatus{
			Name:  pod.Name,
			Phase: string(pod.Status.Phase),
			Ready: actions.IsPodReady(&pod),
		}
	}

	return &pb.GetInstanceHealthResponse{
		Healthy:    status.Healthy,
		ReadyPods:  int32(status.Ready),
		TotalPods:  int32(status.Total),
		ActivePods: int32(status.ActivePods),
		Pods:       pods,
		Complete:   complete,
	}
}
