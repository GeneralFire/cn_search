package implementation

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/GeneralFire/cn_search/be/internal/implementation/mocks"
	"github.com/GeneralFire/cn_search/be/internal/models"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

var uploader = models.User{ID: "u1", Plan: models.PlanFree}

// newIngestSvc — сервис с загрузкой фото; запросы с userCtx() — от пользователя uploader.
func newIngestSvc(t *testing.T) (*CarSearch, *mocks.IngestMock) {
	i, acc := mocks.NewIngestMock(t), mocks.NewAccountsMock(t)
	acc.UserByTokenMock.Optional().Expect(minimock.AnyContext, "cs_t1", false).Return(uploader, nil)
	return NewCarSearch(mocks.NewPlateSightingsMock(t), "/s3/cars/", WithIngest(i), WithAccounts(acc)), i
}

func userCtx() context.Context { return withMD("authorization", "Bearer cs_t1") }

func TestIngest(t *testing.T) {
	svc, i := newIngestSvc(t)
	taken := time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)
	lat, lon, left := 55.7, 37.6, 3
	i.IngestMock.Expect(minimock.AnyContext, models.IngestOpts{Photo: []byte("jpeg"), TakenAt: &taken, Lat: &lat, Lon: &lon, Address: "Москва", User: uploader}).
		Return(models.IngestResult{
			Sighting:    models.PlateSighting{ID: sighting.ID, Plate: "Н311УО 797", PlateType: "ru", PhotoKey: "u/a.jpg", ThumbKey: "u/a_thumb.jpg"},
			Candidates:  []models.PlateCandidate{{Plate: "Н311УО 797", Raw: "H311YO797", Confidence: 0.98, PlateType: "ru", BBox: []int{1, 2, 3, 4}}},
			UploadsLeft: &left,
		}, nil)

	resp, err := svc.Ingest(userCtx(), &pb.IngestRequest{
		Photo: []byte("jpeg"), TakenAt: timestamppb.New(taken), Lat: proto.Float64(lat), Lon: proto.Float64(lon), Address: " Москва ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if s := resp.GetSighting(); s.GetPhotoUrl() != "/s3/cars/u/a.jpg" || s.GetPlateType() != pb.PlateType_PLATE_TYPE_RU {
		t.Errorf("фиксация: %v", s)
	}
	if c := resp.GetCandidates(); len(c) != 1 || c[0].GetRaw() != "H311YO797" || len(c[0].GetBbox()) != 4 || c[0].GetConfidence() != 0.98 {
		t.Errorf("кандидаты: %v", c)
	}
	if resp.UploadsLeft == nil || resp.GetUploadsLeft() != 3 {
		t.Errorf("uploads_left: %v", resp.UploadsLeft)
	}
}

func TestIngestErrors(t *testing.T) {
	for name, req := range map[string]*pb.IngestRequest{
		"без фото":        {},
		"большое фото":    {Photo: make([]byte, MaxPhotoBytes+1)},
		"одна координата": {Photo: []byte("x"), Lat: proto.Float64(1)},
		"вне диапазона":   {Photo: []byte("x"), Lat: proto.Float64(91), Lon: proto.Float64(0)},
	} {
		svc, _ := newIngestSvc(t)
		if _, err := svc.Ingest(userCtx(), req); status.Code(err) != codes.InvalidArgument {
			t.Errorf("%s: %v", name, err)
		}
	}
	for err, want := range map[error]codes.Code{
		&models.ValidationError{Message: "номер не распознан"}: codes.InvalidArgument,
		models.ErrRecognizerUnavailable:                        codes.Unavailable,
		&models.UploadQuotaError{Limit: 5}:                     codes.ResourceExhausted,
		context.DeadlineExceeded:                               codes.Internal,
	} {
		svc, i := newIngestSvc(t)
		i.IngestMock.Return(models.IngestResult{}, err)
		if _, got := svc.Ingest(userCtx(), &pb.IngestRequest{Photo: []byte("x")}); status.Code(got) != want {
			t.Errorf("%v: %v", err, got)
		}
	}
	// Без входа — Unauthenticated с понятным текстом.
	anon, _ := newIngestSvc(t)
	if _, err := anon.Ingest(context.Background(), &pb.IngestRequest{Photo: []byte("x")}); status.Code(err) != codes.Unauthenticated ||
		!strings.Contains(status.Convert(err).Message(), "войдите") {
		t.Errorf("аноним: %v", err)
	}
	svc := NewCarSearch(mocks.NewPlateSightingsMock(t), "")
	if _, err := svc.Ingest(context.Background(), &pb.IngestRequest{Photo: []byte("x")}); status.Code(err) != codes.Unimplemented {
		t.Errorf("без WithIngest: %v", err)
	}
}

// Me отдаёт остаток загрузок вошедшему пользователю.
func TestMeUploadsLeft(t *testing.T) {
	svc, i := newIngestSvc(t)
	left := 2
	i.UploadsLeftMock.Expect(minimock.AnyContext, uploader).Return(&left, nil)
	resp, err := svc.Me(userCtx(), &pb.MeRequest{})
	if err != nil || !resp.GetAuthenticated() || resp.UploadsLeft == nil || resp.GetUploadsLeft() != 2 {
		t.Errorf("Me: %v %v", resp, err)
	}
}
