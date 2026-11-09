package implementation

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/GeneralFire/cn_search/be/internal/repository"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

// Get — rpc Get: все фиксации номера, к которому относится фиксация id.
func (s *CarSearch) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	if req.GetId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "id должен быть положительным")
	}
	items, err := s.repo.ListSamePlate(ctx, int(req.GetId()))
	switch {
	case errors.Is(err, repository.ErrNotFound):
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
