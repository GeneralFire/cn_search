package implementation

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/GeneralFire/cn_search/be/internal/models"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

// Get — rpc Get: все фиксации номера, к которому относится фиксация id.
func (s *CarSearch) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	if err := uuid.Validate(req.GetId()); err != nil {
		return nil, status.Error(codes.InvalidArgument, "id: нужен uuid фиксации")
	}
	items, err := s.uc.ListSamePlate(ctx, models.SamePlateOpts{ID: req.GetId(), IncludeTestData: includeTestData(ctx)})
	switch {
	case errors.Is(err, models.ErrNotFound):
		return nil, status.Error(codes.NotFound, "фиксация не найдена")
	case err != nil:
		return nil, status.Error(codes.Internal, "не удалось получить фиксации")
	}
	resp := &pb.GetResponse{Items: make([]*pb.PlateSighting, len(items))}
	for i, c := range items {
		resp.Items[i] = s.toPB(c)
	}
	return resp, nil
}
