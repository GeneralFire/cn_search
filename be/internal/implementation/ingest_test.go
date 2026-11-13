package implementation

import (
	"context"
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

func newIngestSvc(t *testing.T) (*CarSearch, *mocks.IngestMock) {
	i := mocks.NewIngestMock(t)
	return NewCarSearch(mocks.NewPlateSightingsMock(t), "/s3/cars/", WithIngest(i)), i
}

func TestIngest(t *testing.T) {
	svc, i := newIngestSvc(t)
	taken := time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)
	lat, lon := 55.7, 37.6
	i.IngestMock.Expect(minimock.AnyContext, models.IngestOpts{Photo: []byte("jpeg"), TakenAt: &taken, Lat: &lat, Lon: &lon, Address: "Москва"}).
		Return(models.IngestResult{
			Sighting:   models.PlateSighting{ID: sighting.ID, Plate: "Н311УО 797", PlateType: "ru", PhotoKey: "u/a.jpg", ThumbKey: "u/a_thumb.jpg"},
			Candidates: []models.PlateCandidate{{Plate: "Н311УО 797", Raw: "H311YO797", Confidence: 0.98, PlateType: "ru", BBox: []int{1, 2, 3, 4}}},
		}, nil)

	resp, err := svc.Ingest(context.Background(), &pb.IngestRequest{
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
}

func TestIngestErrors(t *testing.T) {
	for name, req := range map[string]*pb.IngestRequest{
		"без фото":        {},
		"большое фото":    {Photo: make([]byte, MaxPhotoBytes+1)},
		"одна координата": {Photo: []byte("x"), Lat: proto.Float64(1)},
		"вне диапазона":   {Photo: []byte("x"), Lat: proto.Float64(91), Lon: proto.Float64(0)},
	} {
		svc, _ := newIngestSvc(t)
		if _, err := svc.Ingest(context.Background(), req); status.Code(err) != codes.InvalidArgument {
			t.Errorf("%s: %v", name, err)
		}
	}
	for err, want := range map[error]codes.Code{
		&models.ValidationError{Message: "номер не распознан"}: codes.InvalidArgument,
		models.ErrRecognizerUnavailable:                        codes.Unavailable,
		context.DeadlineExceeded:                               codes.Internal,
	} {
		svc, i := newIngestSvc(t)
		i.IngestMock.Return(models.IngestResult{}, err)
		if _, got := svc.Ingest(context.Background(), &pb.IngestRequest{Photo: []byte("x")}); status.Code(got) != want {
			t.Errorf("%v: %v", err, got)
		}
	}
	svc := NewCarSearch(mocks.NewPlateSightingsMock(t), "")
	if _, err := svc.Ingest(context.Background(), &pb.IngestRequest{Photo: []byte("x")}); status.Code(err) != codes.Unimplemented {
		t.Errorf("без WithIngest: %v", err)
	}
}
