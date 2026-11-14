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

// SetCameraRequestStatus — rpc SetCameraRequestStatus (админ): сменить статус заявки.
func (s *CarSearch) SetCameraRequestStatus(ctx context.Context, req *pb.SetCameraRequestStatusRequest) (*pb.CameraRequest, error) {
	if err := requireAdmin(ctx, s.adminToken); err != nil {
		return nil, err
	}
	if s.cameraRequests == nil {
		return nil, errCameraRequestsOff
	}
	st, ok := cameraStatusModel[req.GetStatus()]
	switch {
	case uuid.Validate(req.GetId()) != nil:
		return nil, status.Error(codes.InvalidArgument, "id: нужен uuid заявки")
	case !ok:
		return nil, status.Error(codes.InvalidArgument, "status: укажите статус")
	}
	c, err := s.cameraRequests.SetStatus(ctx, req.GetId(), st)
	switch {
	case errors.Is(err, models.ErrNotFound):
		return nil, status.Error(codes.NotFound, "заявка не найдена")
	case err != nil:
		return nil, internalErr(ctx, err, "не удалось обновить заявку")
	}
	return cameraRequestPB(c, true), nil
}
