package server

import (
	"github.com/OpenSlides/openslides-cli/internal/logger"
	pb "github.com/OpenSlides/openslides-cli/proto/osmanage"
)

// StreamLogs streams log entries to the client at the requested log level.
func (s *OsmanageServiceServer) StreamLogs(req *pb.LogStreamRequest, stream pb.OsmanageService_StreamLogsServer) error {
	minLevel, err := logger.ParseLevel(req.Level)
	if err != nil {
		minLevel = logger.LevelWarn
	}

	sub := logger.Subscribe()
	defer logger.Unsubscribe(sub)

	for {
		select {
		case entry, ok := <-sub:
			if !ok {
				return nil
			}
			if entry.LevelValue < minLevel {
				continue
			}
			if err := stream.Send(&pb.LogEntry{
				Level:   entry.Level,
				Message: entry.Message,
			}); err != nil {
				return err
			}
		case <-stream.Context().Done():
			return nil
		}
	}
}
