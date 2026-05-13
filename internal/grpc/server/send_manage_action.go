package server

import (
	"context"

	manageclient "github.com/OpenSlides/openslides-cli/internal/manage/client"
	"github.com/OpenSlides/openslides-cli/internal/utils"
	pb "github.com/OpenSlides/openslides-cli/proto/osmanage"
)

func (s *OsmanageServiceServer) SendManageAction(ctx context.Context, req *pb.SendManageActionRequest) (*pb.SendManageActionResponse, error) {
	password, err := utils.ReadPassword(req.PasswordFilePath)
	if err != nil {
		return &pb.SendManageActionResponse{Success: false, Error: err.Error()}, nil
	}

	cl := manageclient.New(req.AddressBackendmanage, password)
	resp, err := cl.SendAction(req.Action, req.Payload)
	if err != nil {
		return &pb.SendManageActionResponse{Success: false, Error: err.Error()}, nil
	}

	body, err := manageclient.CheckResponse(resp)
	if err != nil {
		return &pb.SendManageActionResponse{
			Success: false,
			Error:   err.Error(),
			Body:    body,
		}, nil
	}

	return &pb.SendManageActionResponse{
		Success: true,
		Body:    body,
	}, nil
}
