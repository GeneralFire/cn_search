package implementation

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/GeneralFire/cn_search/be/internal/models"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

// MarkWatchSeen — rpc MarkWatchSeen: пользователь увидел все фиксации номера.
func (s *CarSearch) MarkWatchSeen(ctx context.Context, req *pb.MarkWatchSeenRequest) (*pb.MarkWatchSeenResponse, error) {
	u, err := s.watchUser(ctx)
	if err != nil {
		return nil, err
	}
	plate, err := watchPlate("plate", req.GetPlate())
	if err != nil {
		return nil, err
	}
	switch err := s.watches.MarkSeen(ctx, u.ID, plate); {
	case errors.Is(err, models.ErrNotFound):
		return nil, status.Error(codes.NotFound, "подписки на номер нет")
	case err != nil:
		return nil, status.Error(codes.Internal, "не удалось отметить просмотр")
	}
	return &pb.MarkWatchSeenResponse{}, nil
}
