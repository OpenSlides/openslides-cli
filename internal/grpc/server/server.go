package server

import (
	"fmt"
	"net"

	pb "github.com/OpenSlides/openslides-cli/proto/osmanage"
	"google.golang.org/grpc"
)

type OsmanageServiceServer struct {
	pb.UnimplementedOsmanageServiceServer
}

func NewOsmanageServiceServer() *OsmanageServiceServer {
	return &OsmanageServiceServer{}
}

func Start(port int) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterOsmanageServiceServer(grpcServer, NewOsmanageServiceServer())

	fmt.Printf("gRPC server listening on :%d\n", port)
	return grpcServer.Serve(lis)
}
