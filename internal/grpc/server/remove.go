package server

import (
	"context"

	"github.com/OpenSlides/openslides-cli/internal/instance/remove"
	pb "github.com/OpenSlides/openslides-cli/proto/osmanage"
)

func (s *OsmanageServiceServer) RemoveInstance(ctx context.Context, req *pb.RemoveInstanceRequest) (*pb.RemoveInstanceResponse, error) {
	if err := remove.RemoveInstance(req.InstanceDir, req.Force); err != nil {
		return &pb.RemoveInstanceResponse{Success: false, Error: err.Error()}, nil
	}
	return &pb.RemoveInstanceResponse{Success: true}, nil
}
