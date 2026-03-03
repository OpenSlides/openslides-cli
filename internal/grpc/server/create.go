package server

import (
	"context"

	"github.com/OpenSlides/openslides-cli/internal/instance/create"
	pb "github.com/OpenSlides/openslides-cli/proto/osmanage"
)

func (s *OsmanageServiceServer) CreateInstance(ctx context.Context, req *pb.CreateInstanceRequest) (*pb.CreateInstanceResponse, error) {
	if err := create.CreateInstance(req.InstanceDir, req.DbPassword, req.SuperadminPassword); err != nil {
		return &pb.CreateInstanceResponse{Success: false, Error: err.Error()}, nil
	}
	return &pb.CreateInstanceResponse{Success: true}, nil
}
