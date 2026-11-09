package implementation

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/GeneralFire/cn_search/be/internal/models"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

// Clarify — rpc Clarify: уточнение номера к фиксации, статус «не принят».
func (s *CarSearch) Clarify(ctx context.Context, req *pb.ClarifyRequest) (*pb.ClarifyResponse, error) {
	plate := strings.TrimSpace(req.GetPlate())
	switch {
	case uuid.Validate(req.GetSightingId()) != nil:
		return nil, status.Error(codes.InvalidArgument, "sighting_id: нужен uuid фиксации")
	case plate == "":
		return nil, status.Error(codes.InvalidArgument, "plate: укажите номер")
	case utf8.RuneCountInString(plate) > maxQueryLength:
		return nil, status.Errorf(codes.InvalidArgument, "plate: не длиннее %d символов", maxQueryLength)
	}
	c, err := s.uc.Clarify(ctx, models.ClarifyOpts{SightingID: req.GetSightingId(), Plate: plate})
	switch {
	case errors.Is(err, models.ErrNotFound):
		return nil, status.Error(codes.NotFound, "фиксация не найдена")
	case err != nil:
		return nil, status.Error(codes.Internal, "не удалось сохранить уточнение")
	}
	return &pb.ClarifyResponse{Clarification: &pb.Clarification{
		Id:         c.ID,
		SightingId: c.SightingID,
		Plate:      c.Plate,
		Status:     clarificationStatusPB[c.Status],
		CreatedAt:  timestamppb.New(c.CreatedAt),
	}}, nil
}

var clarificationStatusPB = map[models.ClarificationStatus]pb.ClarificationStatus{
	models.ClarificationNotAccepted: pb.ClarificationStatus_CLARIFICATION_STATUS_NOT_ACCEPTED,
	models.ClarificationAccepted:    pb.ClarificationStatus_CLARIFICATION_STATUS_ACCEPTED,
}
