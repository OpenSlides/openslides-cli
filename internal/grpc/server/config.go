package server

import (
	"context"

	instanceconfig "github.com/OpenSlides/openslides-cli/internal/instance/config"
	pb "github.com/OpenSlides/openslides-cli/proto/osmanage"
)

func (s *OsmanageServiceServer) ConfigInstance(ctx context.Context, req *pb.InstanceConfigRequest) (*pb.InstanceConfigResponse, error) {
	err := instanceconfig.Run(
		req.InstanceDir,
		req.Force,
		req.StackTemplatePath,
		nil,
		req.Configs,
	)
	if err != nil {
		return &pb.InstanceConfigResponse{Success: false, Error: err.Error()}, nil
	}
	return &pb.InstanceConfigResponse{Success: true}, nil
}
