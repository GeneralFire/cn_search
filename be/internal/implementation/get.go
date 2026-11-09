package implementation

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/GeneralFire/cn_search/be/internal/repository"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch/v1"
)

// Get — rpc Get: появление номера по id.
func (s *CarSearch) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	if req.GetId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "id должен быть положительным")
	}
	c, err := s.repo.Get(ctx, int(req.GetId()))
	switch {
	case errors.Is(err, repository.ErrNotFound):
		return nil, status.Error(codes.NotFound, "запись не найдена")
	case err != nil:
		return nil, status.Error(codes.Internal, "не удалось получить запись")
	}
	return &pb.GetResponse{Car: s.toPB(c)}, nil
}
