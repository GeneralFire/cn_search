package implementation

import (
	"context"
	"errors"
	"testing"

	"github.com/gojuno/minimock/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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
	if it.GetPhotoUrl() != "/s3/cars/car-1.png" || it.GetPlate() != "А123ВС 777" || !it.GetCreatedAt().AsTime().Equal(sighting.CreatedAt) {
		t.Errorf("маппинг: %v", it)
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
