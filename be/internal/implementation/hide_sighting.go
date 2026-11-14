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

// HideSighting — rpc HideSighting (админ): скрыть фиксацию из выдачи или вернуть её.
func (s *CarSearch) HideSighting(ctx context.Context, req *pb.HideSightingRequest) (*pb.PlateSighting, error) {
	if err := requireAdmin(ctx, s.adminToken); err != nil {
		return nil, err
	}
	if s.admin == nil {
		return nil, status.Error(codes.Unimplemented, "админские методы не включены")
	}
	if uuid.Validate(req.GetId()) != nil {
		return nil, status.Error(codes.InvalidArgument, "id: нужен uuid фиксации")
	}
	c, err := s.admin.HideSighting(ctx, req.GetId(), req.GetHidden())
	switch {
	case errors.Is(err, models.ErrNotFound):
		return nil, status.Error(codes.NotFound, "фиксация не найдена")
	case err != nil:
		return nil, internalErr(ctx, err, "не удалось обновить фиксацию")
	}
	return s.toPB(c, true), nil
}
