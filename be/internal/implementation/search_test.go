package implementation

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/GeneralFire/cn_search/be/internal/models"
	"github.com/GeneralFire/cn_search/be/internal/repository"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

func TestSearch(t *testing.T) {
	next := 35
	repo := &stubRepo{page: repository.Page{Items: []models.PlateSighting{sighting}, Total: 600, NextOffset: &next}}
	resp, err := newSvc(repo).Search(context.Background(), &pb.SearchRequest{Plate: "А123ВС 777", Offset: 0})
	if err != nil {
		t.Fatal(err)
	}
	if repo.query != (repository.Query{Plate: "А123ВС 777", Offset: 0, Limit: DefaultLimit}) {
		t.Errorf("запрос в репозиторий: %+v (строка как есть, limit по умолчанию)", repo.query)
	}
	if resp.GetTotal() != 600 || resp.NextOffset == nil || resp.GetNextOffset() != 35 || len(resp.GetItems()) != 1 {
		t.Fatalf("ответ: %v", resp)
	}
	it := resp.GetItems()[0]
	if it.GetPhotoUrl() != "/s3/cars/car-1.png" || it.GetPlate() != "А123ВС 777" || !it.GetCreatedAt().AsTime().Equal(sighting.CreatedAt) {
		t.Errorf("маппинг: %v", it)
	}

	// Последняя страница — без next_offset.
	repo.page.NextOffset = nil
	resp, _ = newSvc(repo).Search(context.Background(), &pb.SearchRequest{Limit: 10})
	if resp.NextOffset != nil || repo.query.Limit != 10 {
		t.Errorf("последняя страница: next=%v limit=%d", resp.NextOffset, repo.query.Limit)
	}
}

func TestSearchErrors(t *testing.T) {
	for name, req := range map[string]*pb.SearchRequest{
		"limit > max":   {Limit: MaxLimit + 1},
		"limit < 0":     {Limit: -1},
		"offset < 0":    {Offset: -1},
		"длинный номер": {Plate: "АААААААААААААААААААААААААААААААААААААА"},
	} {
		if _, err := newSvc(&stubRepo{}).Search(context.Background(), req); status.Code(err) != codes.InvalidArgument {
			t.Errorf("%s: code = %v, want InvalidArgument", name, status.Code(err))
		}
	}
	_, err := newSvc(&stubRepo{err: errors.New("db down")}).Search(context.Background(), &pb.SearchRequest{})
	if status.Code(err) != codes.Internal {
		t.Errorf("ошибка БД: %v", status.Code(err))
	}
}
