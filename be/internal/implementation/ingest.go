package implementation

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/GeneralFire/cn_search/be/internal/models"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

const (
	// MaxPhotoBytes — предел размера фото в rpc Ingest.
	MaxPhotoBytes = 8 << 20
	// maxAddressLength — адрес фиксации, символов.
	maxAddressLength = 300
)

// Ingest — rpc Ingest: фото → OCR → номер → фиксация. Здесь — вход и проверки запроса; сценарий
// и квота загрузок — в usecase. Анонимно загружать нельзя: квота считается на пользователя.
func (s *CarSearch) Ingest(ctx context.Context, req *pb.IngestRequest) (*pb.IngestResponse, error) {
	if s.ingest == nil {
		return nil, status.Error(codes.Unimplemented, "загрузка фото не включена (нет OCR_URL или S3)")
	}
	user, err := s.currentUser(ctx)
	if status.Code(err) == codes.Unauthenticated {
		return nil, status.Error(codes.Unauthenticated, "войдите, чтобы загружать фото")
	}
	if err != nil {
		return nil, err
	}
	address := strings.TrimSpace(req.GetAddress())
	switch {
	case len(req.GetPhoto()) == 0:
		return nil, status.Error(codes.InvalidArgument, "photo: приложите фото")
	case len(req.GetPhoto()) > MaxPhotoBytes:
		return nil, status.Errorf(codes.InvalidArgument, "photo: не больше %d МБ", MaxPhotoBytes>>20)
	case (req.Lat == nil) != (req.Lon == nil):
		return nil, status.Error(codes.InvalidArgument, "lat, lon: укажите обе координаты или ни одной")
	case req.Lat != nil && (req.GetLat() < -90 || req.GetLat() > 90 || req.GetLon() < -180 || req.GetLon() > 180):
		return nil, status.Error(codes.InvalidArgument, "lat, lon: координаты вне диапазона")
	case utf8.RuneCountInString(address) > maxAddressLength:
		return nil, status.Errorf(codes.InvalidArgument, "address: не длиннее %d символов", maxAddressLength)
	}
	o := models.IngestOpts{Photo: req.GetPhoto(), Lat: req.Lat, Lon: req.Lon, Address: address, User: user}
	if req.GetTakenAt() != nil {
		t := req.GetTakenAt().AsTime()
		o.TakenAt = &t
	}
	res, err := s.ingest.Ingest(ctx, o)
	var invalid *models.ValidationError
	var quota *models.UploadQuotaError
	switch {
	case errors.As(err, &invalid):
		return nil, status.Error(codes.InvalidArgument, invalid.Message)
	case errors.As(err, &quota):
		return nil, status.Errorf(codes.ResourceExhausted,
			"лимит тарифа Free — %d фото за сутки; оформите Pro, чтобы загружать без ограничений", quota.Limit)
	case errors.Is(err, models.ErrRecognizerUnavailable):
		return nil, status.Error(codes.Unavailable, "сервис распознавания недоступен, попробуйте позже")
	case err != nil:
		return nil, internalErr(ctx, err, "не удалось сохранить фиксацию")
	}
	out := &pb.IngestResponse{Sighting: s.toPB(res.Sighting)}
	if res.UploadsLeft != nil {
		out.UploadsLeft = proto.Int32(int32(*res.UploadsLeft))
	}
	for _, c := range res.Candidates {
		bbox := make([]int32, len(c.BBox))
		for i, v := range c.BBox {
			bbox[i] = int32(v)
		}
		out.Candidates = append(out.Candidates, &pb.PlateCandidate{
			Plate: c.Plate, Raw: c.Raw, Confidence: c.Confidence, PlateType: plateTypePB[c.PlateType], Bbox: bbox,
		})
	}
	return out, nil
}
