package implementation

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/GeneralFire/cn_search/be/internal/models"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

// Clarify — rpc Clarify: уточнение номера к фиксации. Здесь — проверки запроса (uuid, длина, регион);
// нормализация номера, формат по типу и консенсус — в usecase. В ответе — номер в нормализованном виде
// и актуальный статус (уточнение принимается, когда совпали несколько уточнений фиксации).
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
	plateType := plateTypeModel[req.GetPlateType()]
	region := strings.ToUpper(strings.TrimSpace(req.GetPlateRegion()))
	if plateType != "eu" && plateType != "us" {
		region = "" // регион нужен только ЕС (страна) и США (штат)
	} else if !regionRe.MatchString(region) {
		return nil, status.Error(codes.InvalidArgument, "plate_region: до 3 латинских букв (страна ЕС, штат США)")
	}
	c, err := s.uc.Clarify(ctx, models.ClarifyOpts{
		SightingID: req.GetSightingId(), Plate: plate, PlateType: plateType, PlateRegion: region,
	})
	var invalid *models.ValidationError
	switch {
	case errors.As(err, &invalid):
		return nil, status.Error(codes.InvalidArgument, invalid.Message)
	case errors.Is(err, models.ErrNotFound):
		return nil, status.Error(codes.NotFound, "фиксация не найдена")
	case err != nil:
		return nil, status.Error(codes.Internal, "не удалось сохранить уточнение")
	}
	return &pb.ClarifyResponse{Clarification: &pb.Clarification{
		Id:          c.ID,
		SightingId:  c.SightingID,
		Plate:       c.Plate,
		Status:      clarificationStatusPB[c.Status],
		CreatedAt:   timestamppb.New(c.CreatedAt),
		PlateType:   plateTypePB[c.PlateType],
		PlateRegion: c.PlateRegion,
	}}, nil
}

// regionRe — регион номера: пусто или 1–3 латинские буквы ("D", "CA", "NL").
var regionRe = regexp.MustCompile(`^[A-Z]{0,3}$`)

var plateTypeModel = map[pb.PlateType]string{
	pb.PlateType_PLATE_TYPE_RU: "ru",
	pb.PlateType_PLATE_TYPE_EU: "eu",
	pb.PlateType_PLATE_TYPE_US: "us",
	pb.PlateType_PLATE_TYPE_UA: "ua",
	pb.PlateType_PLATE_TYPE_CN: "cn",
}

var clarificationStatusPB = map[models.ClarificationStatus]pb.ClarificationStatus{
	models.ClarificationNotAccepted: pb.ClarificationStatus_CLARIFICATION_STATUS_NOT_ACCEPTED,
	models.ClarificationAccepted:    pb.ClarificationStatus_CLARIFICATION_STATUS_ACCEPTED,
}
