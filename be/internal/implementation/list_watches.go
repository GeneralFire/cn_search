package implementation

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/GeneralFire/cn_search/be/internal/models"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

// ListWatches — rpc ListWatches: подписки пользователя с последней фиксацией и числом новых.
func (s *CarSearch) ListWatches(ctx context.Context, _ *pb.ListWatchesRequest) (*pb.ListWatchesResponse, error) {
	u, err := s.watchUser(ctx)
	if err != nil {
		return nil, err
	}
	items, err := s.watches.List(ctx, models.ListWatchesOpts{UserID: u.ID, IncludeTestData: includeTestData(ctx)})
	if err != nil {
		return nil, status.Error(codes.Internal, "не удалось получить подписки")
	}
	resp := &pb.ListWatchesResponse{Items: make([]*pb.WatchedPlate, len(items))}
	for i, w := range items {
		resp.Items[i] = &pb.WatchedPlate{
			PlateDisplay: w.PlateDisplay, NewCount: int32(w.NewCount), WatchedAt: timestamppb.New(w.WatchedAt),
		}
		if w.Latest != nil {
			resp.Items[i].Plate = s.toPB(*w.Latest)
		}
	}
	return resp, nil
}
