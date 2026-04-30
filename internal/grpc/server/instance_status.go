package server

import (
	"context"
	"fmt"

	"github.com/OpenSlides/openslides-cli/internal/k8s/actions"
	"github.com/OpenSlides/openslides-cli/internal/k8s/client"
	"github.com/OpenSlides/openslides-cli/internal/utils"
	pb "github.com/OpenSlides/openslides-cli/proto/osmanage"
)

func (s *OsmanageServiceServer) GetInstanceStatus(ctx context.Context, req *pb.GetInstanceStatusRequest) (*pb.GetInstanceStatusResponse, error) {
	k8sClient, err := client.New(req.Kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("creating k8s client: %w", err)
	}

	namespace := utils.ExtractNamespace(req.InstanceUrl)
	status, err := actions.GetInstanceStatus(ctx, k8sClient, namespace)
	if err != nil {
		return nil, fmt.Errorf("getting instance status: %w", err)
	}

	if !status.NamespaceExists {
		return &pb.GetInstanceStatusResponse{NamespaceExists: false}, nil
	}

	var pods []*pb.InstancePodStatus
	for _, pod := range status.Pods {
		var containers []*pb.ContainerStatus
		for _, c := range pod.Containers {
			containers = append(containers, &pb.ContainerStatus{
				Name:              c.Name,
				Tag:               c.Tag,
				ContainerRegistry: c.ContainerRegistry,
				Ready:             c.Ready,
				Started:           c.Started,
				EnvVars:           c.EnvVars,
			})
		}
		pods = append(pods, &pb.InstancePodStatus{
			Name:       pod.Name,
			Service:    pod.Service,
			Node:       pod.Node,
			Containers: containers,
		})
	}

	return &pb.GetInstanceStatusResponse{
		NamespaceExists: true,
		Pods:            pods,
		ServiceCounts:   status.ServiceCounts,
	}, nil
}
