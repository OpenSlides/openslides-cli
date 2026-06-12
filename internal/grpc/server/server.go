package server

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/OpenSlides/openslides-cli/internal/constants"
	"github.com/OpenSlides/openslides-cli/internal/logger"
	pb "github.com/OpenSlides/openslides-cli/proto/osmanage"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

// OsmanageServiceServer implements the OsmanageService gRPC interface
type OsmanageServiceServer struct {
	pb.UnimplementedOsmanageServiceServer
}

func NewOsmanageServiceServer() *OsmanageServiceServer {
	return &OsmanageServiceServer{}
}

func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start gRPC server",
		Long:  "Start the osmanage gRPC server for client-server communication",
	}

	host := cmd.Flags().String("host", constants.GRPCHost, "Host to listen on")
	port := cmd.Flags().String("port", constants.GRPCPort, "Port to listen on")
	unsafe := cmd.Flags().Bool("unsafe", constants.Unsafe, "Guardrail to prevent people starting a bare gRPC server, unless they really want to and know what they're doing.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if *host == "" {
			return fmt.Errorf("gRPC host flag cannot be empty")
		}
		if *port == "" {
			return fmt.Errorf("gRPC port flag cannot be empty")
		}
		if !*unsafe {
			return fmt.Errorf("Here be dragons")
		}
		address := fmt.Sprintf("%s:%s", *host, *port)
		return start(address)
	}

	return cmd
}

func start(address string) error {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", address, err)
	}

	grpcSrv := grpc.NewServer()
	pb.RegisterOsmanageServiceServer(grpcSrv, NewOsmanageServiceServer())

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-quit
		logger.Info("shutting down gRPC server...")

		// Force shutdown after timeout if graceful takes too long
		go func() {
			time.Sleep(constants.GRPCGracefulStopTimeout)
			logger.Info("graceful shutdown timed out, forcing stop")
			grpcSrv.Stop()
		}()

		grpcSrv.GracefulStop()
	}()

	logger.Info("gRPC server listening on %s", address)
	if err := grpcSrv.Serve(lis); err != nil {
		return fmt.Errorf("gRPC server error: %w", err)
	}
	return nil
}
