package implementation

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/GeneralFire/cn_search/be/internal/models"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

func TestSearch(t *testing.T) {
	svc, uc := newSvc(t)
	next := 35
	// Строка уходит как есть, limit 0 → DefaultLimit.
	uc.SearchMock.Expect(minimock.AnyContext, models.SearchOpts{Plate: "А123ВС 777", Limit: DefaultLimit}).
		Return(models.SearchPage{Items: []models.PlateSighting{sighting}, Total: 600, NextOffset: &next}, nil)

	resp, err := svc.Search(context.Background(), &pb.SearchRequest{Plate: "А123ВС 777"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetTotal() != 600 || resp.NextOffset == nil || resp.GetNextOffset() != 35 || len(resp.GetItems()) != 1 {
		t.Fatalf("ответ: %v", resp)
	}
	it := resp.GetItems()[0]
	if it.GetPhotoUrl() != "/s3/cars/car-1.jpg" || it.GetThumbUrl() != "/s3/cars/car-1_thumb.jpg" || it.GetPlate() != "А123ВС 777" || !it.GetCreatedAt().AsTime().Equal(sighting.CreatedAt) {
		t.Errorf("маппинг: %v", it)
	}
}

func TestSearchSuggestions(t *testing.T) {
	svc, uc := newSvc(t)
	uc.SearchMock.Return(models.SearchPage{Suggestions: []models.PlateSighting{sighting}}, nil)

	resp, err := svc.Search(context.Background(), &pb.SearchRequest{Plate: "А123ВС 778"})
	if err != nil || resp.GetTotal() != 0 || len(resp.GetItems()) != 0 || len(resp.GetSuggestions()) != 1 {
		t.Fatalf("ответ: %v %v", resp, err)
	}
	if s := resp.GetSuggestions()[0]; s.GetPlate() != "А123ВС 777" || s.GetId() != sighting.ID || s.GetThumbUrl() != "/s3/cars/car-1_thumb.jpg" {
		t.Errorf("маппинг подсказки: %v", s)
	}
}

func TestSearchPassesIncludeTestData(t *testing.T) {
	svc, uc := newSvc(t)
	uc.SearchMock.Expect(minimock.AnyContext, models.SearchOpts{Limit: DefaultLimit, IncludeTestData: true}).
		Return(models.SearchPage{}, nil)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("include-test-data", "1"))
	if _, err := svc.Search(ctx, &pb.SearchRequest{}); err != nil {
		t.Fatal(err)
	}
}

func TestSearchPassesMarket(t *testing.T) {
	for _, m := range []string{"ru", "ua", "de", "us", "cn", "other"} {
		svc, uc := newSvc(t)
		uc.SearchMock.Expect(minimock.AnyContext, models.SearchOpts{Limit: 14, Market: models.Market(m)}).
			Return(models.SearchPage{}, nil)
		if _, err := svc.Search(context.Background(), &pb.SearchRequest{Market: m, Limit: 14}); err != nil {
			t.Errorf("%s: %v", m, err)
		}
	}
}

func TestSearchLastPage(t *testing.T) {
	svc, uc := newSvc(t)
	uc.SearchMock.Expect(minimock.AnyContext, models.SearchOpts{Offset: 595, Limit: 10}).
		Return(models.SearchPage{Items: []models.PlateSighting{sighting}, Total: 596}, nil)

	resp, err := svc.Search(context.Background(), &pb.SearchRequest{Offset: 595, Limit: 10})
	if err != nil || resp.NextOffset != nil {
		t.Errorf("последняя страница: next=%v err=%v", resp.NextOffset, err)
	}
}

func TestSearchErrors(t *testing.T) {
	for name, req := range map[string]*pb.SearchRequest{
		"limit > max":   {Limit: MaxLimit + 1},
		"limit < 0":     {Limit: -1},
		"offset < 0":    {Offset: -1},
		"длинный номер": {Plate: "АААААААААААААААААААААААААААААААААААААА"},
		"чужой рынок":   {Market: "fr"},
		"рынок в caps":  {Market: "RU"},
	} {
		svc, _ := newSvc(t) // usecase не должен вызываться
		if _, err := svc.Search(context.Background(), req); status.Code(err) != codes.InvalidArgument {
			t.Errorf("%s: code = %v, want InvalidArgument", name, status.Code(err))
		}
	}

	svc, uc := newSvc(t)
	uc.SearchMock.Return(models.SearchPage{}, errors.New("db down"))
	if _, err := svc.Search(context.Background(), &pb.SearchRequest{}); status.Code(err) != codes.Internal {
		t.Errorf("ошибка БД: %v", status.Code(err))
	}
}

func TestPlateTypeMapping(t *testing.T) {
	svc, _ := newSvc(t)
	for typ, want := range map[string]pb.PlateType{
		"ru": pb.PlateType_PLATE_TYPE_RU, "eu": pb.PlateType_PLATE_TYPE_EU, "us": pb.PlateType_PLATE_TYPE_US,
		"ua": pb.PlateType_PLATE_TYPE_UA, "cn": pb.PlateType_PLATE_TYPE_CN, "": pb.PlateType_PLATE_TYPE_DEFAULT,
	} {
		c := sighting
		c.PlateType = typ
		if got := svc.toPB(c).GetPlateType(); got != want {
			t.Errorf("%q → %v, want %v", typ, got, want)
		}
		if typ != "" && plateTypeModel[want] != typ {
			t.Errorf("clarify: %v → %q, want %q", want, plateTypeModel[want], typ)
		}
	}
}

func TestSearchPassesFilters(t *testing.T) {
	svc, uc := newSvc(t)
	since := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	until := since.Add(24 * time.Hour)
	uc.SearchMock.Expect(minimock.AnyContext, models.SearchOpts{Limit: DefaultLimit, Since: since, Until: until, HasCoords: true}).
		Return(models.SearchPage{}, nil)
	if _, err := svc.Search(context.Background(), &pb.SearchRequest{
		Since: timestamppb.New(since), Until: timestamppb.New(until), HasCoords: true,
	}); err != nil {
		t.Fatal(err)
	}
}

func TestSearchBadPeriod(t *testing.T) {
	svc, _ := newSvc(t)
	at := timestamppb.New(time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC))
	for _, req := range []*pb.SearchRequest{
		{Since: at, Until: at},
		{Since: &timestamppb.Timestamp{Seconds: 1, Nanos: -1}},
	} {
		if _, err := svc.Search(context.Background(), req); status.Code(err) != codes.InvalidArgument {
			t.Errorf("%v: %v", req, err)
		}
	}
}
