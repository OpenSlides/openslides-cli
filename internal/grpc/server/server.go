package server

import (
	"fmt"
	"net"

	pb "github.com/OpenSlides/openslides-cli/proto/cluster"
	"google.golang.org/grpc"
)

func Start(port int) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterClusterServiceServer(grpcServer, NewClusterServer())

	fmt.Printf("gRPC server listening on :%d\n", port)
	return grpcServer.Serve(lis)
}
