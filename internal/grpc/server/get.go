package server

import (
	"context"

	"github.com/OpenSlides/openslides-cli/internal/manage/actions/get"
	pb "github.com/OpenSlides/openslides-cli/proto/osmanage"
)

func (s *OsmanageServiceServer) GetCollection(
	ctx context.Context,
	req *pb.GetCollectionRequest,
) (*pb.GetCollectionResponse, error) {
	if req.DbConfig == nil {
		return &pb.GetCollectionResponse{
			Success: false,
			Error:   "db_config is required",
		}, nil
	}
	if req.QueryParams == nil {
		return &pb.GetCollectionResponse{
			Success: false,
			Error:   "query_params is required",
		}, nil
	}

	result, err := get.ExecuteGetCollection(ctx, req.DbConfig, req.QueryParams)
	if err != nil {
		return &pb.GetCollectionResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return result, nil
}
