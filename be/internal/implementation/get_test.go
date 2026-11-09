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

func TestGet(t *testing.T) {
	second := sighting
	second.ID, second.Address = 4, "Москва, ул. Тверская, 7"
	repo := &stubRepo{list: []models.PlateSighting{second, sighting}}

	resp, err := newSvc(repo).Get(context.Background(), &pb.GetRequest{Id: 1})
	if err != nil {
		t.Fatal(err)
	}
	if repo.listFor != 1 || len(resp.GetItems()) != 2 || resp.GetItems()[0].GetId() != 4 || resp.GetItems()[1].GetId() != 1 {
		t.Fatalf("Get(1): %v (порядок из репозитория должен сохраниться)", resp)
	}
}

func TestGetErrors(t *testing.T) {
	ctx := context.Background()
	tests := map[string]struct {
		repo *stubRepo
		id   int32
		want codes.Code
	}{
		"id = 0":     {&stubRepo{}, 0, codes.InvalidArgument},
		"id < 0":     {&stubRepo{}, -1, codes.InvalidArgument},
		"нет такого": {&stubRepo{err: repository.ErrNotFound}, 100500, codes.NotFound},
		"ошибка БД":  {&stubRepo{err: errors.New("db down")}, 1, codes.Internal},
	}
	for name, tt := range tests {
		if _, err := newSvc(tt.repo).Get(ctx, &pb.GetRequest{Id: tt.id}); status.Code(err) != tt.want {
			t.Errorf("%s: code = %v, want %v", name, status.Code(err), tt.want)
		}
	}
}
