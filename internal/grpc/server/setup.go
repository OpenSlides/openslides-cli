package server

import (
	"context"

	"github.com/OpenSlides/openslides-cli/internal/instance/setup"
	pb "github.com/OpenSlides/openslides-cli/proto/osmanage"
)

func (s *OsmanageServiceServer) SetupInstance(ctx context.Context, req *pb.InstanceConfigRequest) (*pb.InstanceConfigResponse, error) {
	err := setup.Run(
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
