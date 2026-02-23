package server

import (
	"context"
	"fmt"

	"github.com/OpenSlides/openslides-cli/internal/constants"
	"github.com/OpenSlides/openslides-cli/internal/manage/actions/migrations"
	"github.com/OpenSlides/openslides-cli/internal/manage/client"
	"github.com/OpenSlides/openslides-cli/internal/utils"
	pb "github.com/OpenSlides/openslides-cli/proto/osmanage"
)

func (s *OsmanageServiceServer) MigrationsMigrate(
	req *pb.MigrationsRequest,
	stream pb.OsmanageService_MigrationsMigrateServer,
) error {
	return executeMigrationStream(req, stream, "migrate")
}

func (s *OsmanageServiceServer) MigrationsFinalize(
	req *pb.MigrationsRequest,
	stream pb.OsmanageService_MigrationsFinalizeServer,
) error {
	return executeMigrationStream(req, stream, "finalize")
}

func (s *OsmanageServiceServer) MigrationsReset(
	ctx context.Context,
	req *pb.MigrationsRequest,
) (*pb.MigrationsResponse, error) {
	return executeMigrationUnary(req, "reset")
}

func (s *OsmanageServiceServer) MigrationsClearCollectionfieldTables(
	ctx context.Context,
	req *pb.MigrationsRequest,
) (*pb.MigrationsResponse, error) {
	return executeMigrationUnary(req, "clear-collectionfield-tables")
}

func (s *OsmanageServiceServer) MigrationsStats(
	ctx context.Context,
	req *pb.MigrationsRequest,
) (*pb.MigrationsResponse, error) {
	return executeMigrationUnary(req, "stats")
}

func (s *OsmanageServiceServer) MigrationsProgress(
	ctx context.Context,
	req *pb.MigrationsRequest,
) (*pb.MigrationsResponse, error) {
	return executeMigrationUnary(req, "progress")
}

// executeMigrationStream handles streaming migration commands (migrate, finalize)
func executeMigrationStream(
	req *pb.MigrationsRequest,
	stream interface {
		Send(*pb.MigrationsProgressResponse) error
		Context() context.Context
	},
	command string,
) error {
	authPassword, err := utils.ReadPassword(req.PasswordFilePath)
	if err != nil {
		return fmt.Errorf("reading password from %s: %w", req.PasswordFilePath, err)
	}

	backendClient := client.New(req.AddressBackendmanage, authPassword)

	response, err := migrations.ExecuteMigrationCommand(backendClient, command)
	if err != nil {
		return fmt.Errorf("starting migration: %w", err)
	}

	if !migrations.Running(response) {
		return stream.Send(&pb.MigrationsProgressResponse{
			Output:    response.Output,
			Running:   false,
			Success:   response.Success,
			Exception: response.Exception,
		})
	}

	streamCallback := func(update *pb.MigrationsProgressResponse) error {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		default:
		}

		return stream.Send(update)
	}

	return migrations.TrackMigrationProgress(
		backendClient,
		constants.DefaultMigrationProgressInterval,
		streamCallback,
	)
}

// executeMigrationUnary handles single-response migration commands (reset, stats, progress, clear-collectionfield-tables)
func executeMigrationUnary(req *pb.MigrationsRequest, command string) (*pb.MigrationsResponse, error) {
	authPassword, err := utils.ReadPassword(req.PasswordFilePath)
	if err != nil {
		return nil, fmt.Errorf("reading password from %s: %w", req.PasswordFilePath, err)
	}

	backendClient := client.New(req.AddressBackendmanage, authPassword)

	response, err := migrations.ExecuteMigrationCommand(backendClient, command)
	if err != nil {
		return nil, fmt.Errorf("executing migration command %q: %w", command, err)
	}

	return response, nil
}
