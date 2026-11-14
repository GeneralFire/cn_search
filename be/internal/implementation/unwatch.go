package implementation

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

// Unwatch — rpc Unwatch: отписка от номера.
func (s *CarSearch) Unwatch(ctx context.Context, req *pb.UnwatchRequest) (*pb.UnwatchResponse, error) {
	u, err := s.watchUser(ctx)
	if err != nil {
		return nil, err
	}
	plate, err := watchPlate("plate", req.GetPlate())
	if err != nil {
		return nil, err
	}
	if err := s.watches.Unwatch(ctx, u.ID, plate); err != nil {
		return nil, status.Error(codes.Internal, "не удалось отписаться")
	}
	return &pb.UnwatchResponse{}, nil
}
