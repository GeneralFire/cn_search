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

// VoteCameraRequest — rpc VoteCameraRequest: голос «+1» за заявку (повторы с браузера отсекает фронт).
func (s *CarSearch) VoteCameraRequest(ctx context.Context, req *pb.VoteCameraRequestRequest) (*pb.CameraRequest, error) {
	if s.cameraRequests == nil {
		return nil, errCameraRequestsOff
	}
	if uuid.Validate(req.GetId()) != nil {
		return nil, status.Error(codes.InvalidArgument, "id: нужен uuid заявки")
	}
	c, err := s.cameraRequests.Vote(ctx, models.VoteCameraRequestOpts{ID: req.GetId(), IncludeTestData: includeTestData(ctx)})
	switch {
	case errors.Is(err, models.ErrNotFound):
		return nil, status.Error(codes.NotFound, "заявка не найдена")
	case err != nil:
		return nil, status.Error(codes.Internal, "не удалось учесть голос")
	}
	return cameraRequestPB(c, false), nil
}
