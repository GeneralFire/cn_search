package implementation

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/GeneralFire/cn_search/be/internal/models"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

// clarificationStatusModel — статус уточнения из proto; UNSPECIFIED в карте нет.
var clarificationStatusModel = map[pb.ClarificationStatus]models.ClarificationStatus{
	pb.ClarificationStatus_CLARIFICATION_STATUS_NOT_ACCEPTED: models.ClarificationNotAccepted,
	pb.ClarificationStatus_CLARIFICATION_STATUS_ACCEPTED:     models.ClarificationAccepted,
}

// SetClarificationStatus — rpc SetClarificationStatus (админ): принять или отклонить уточнение.
func (s *CarSearch) SetClarificationStatus(ctx context.Context, req *pb.SetClarificationStatusRequest) (*pb.Clarification, error) {
	if err := requireAdmin(ctx, s.adminToken); err != nil {
		return nil, err
	}
	if s.admin == nil {
		return nil, status.Error(codes.Unimplemented, "админские методы не включены")
	}
	st, ok := clarificationStatusModel[req.GetStatus()]
	switch {
	case uuid.Validate(req.GetId()) != nil:
		return nil, status.Error(codes.InvalidArgument, "id: нужен uuid уточнения")
	case !ok:
		return nil, status.Error(codes.InvalidArgument, "status: ACCEPTED или NOT_ACCEPTED")
	}
	c, err := s.admin.SetClarificationStatus(ctx, req.GetId(), st)
	switch {
	case errors.Is(err, models.ErrNotFound):
		return nil, status.Error(codes.NotFound, "уточнение не найдено")
	case err != nil:
		return nil, status.Error(codes.Internal, "не удалось обновить уточнение")
	}
	return clarificationPB(c), nil
}

func clarificationPB(c models.Clarification) *pb.Clarification {
	return &pb.Clarification{
		Id:          c.ID,
		SightingId:  c.SightingID,
		Plate:       c.Plate,
		Status:      clarificationStatusPB[c.Status],
		CreatedAt:   timestamppb.New(c.CreatedAt),
		PlateType:   plateTypePB[c.PlateType],
		PlateRegion: c.PlateRegion,
	}
}
