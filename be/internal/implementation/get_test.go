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

func TestGet(t *testing.T) {
	svc, uc := newSvc(t)
	second := sighting
	second.ID, second.Address = "00000000-0000-0000-0000-000000000004", "Москва, ул. Тверская, 7"
	uc.ListSamePlateMock.Expect(minimock.AnyContext, "00000000-0000-0000-0000-000000000001").Return([]models.PlateSighting{second, sighting}, nil)

	resp, err := svc.Get(context.Background(), &pb.GetRequest{Id: "00000000-0000-0000-0000-000000000001"})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.GetItems()) != 2 || resp.GetItems()[0].GetId() != "00000000-0000-0000-0000-000000000004" || resp.GetItems()[1].GetId() != "00000000-0000-0000-0000-000000000001" {
		t.Fatalf("Get(1): %v (порядок из usecase должен сохраниться)", resp)
	}
}

func TestGetErrors(t *testing.T) {
	ctx := context.Background()
	for name, id := range map[string]string{"пусто": "", "не uuid": "1", "мусор": "a'b"} {
		svc, _ := newSvc(t) // usecase не должен вызываться
		if _, err := svc.Get(ctx, &pb.GetRequest{Id: id}); status.Code(err) != codes.InvalidArgument {
			t.Errorf("%s: code = %v", name, status.Code(err))
		}
	}
	for name, tt := range map[string]struct {
		err  error
		want codes.Code
	}{
		"нет такого": {models.ErrNotFound, codes.NotFound},
		"ошибка БД":  {errors.New("db down"), codes.Internal},
	} {
		svc, uc := newSvc(t)
		uc.ListSamePlateMock.Return(nil, tt.err)
		if _, err := svc.Get(ctx, &pb.GetRequest{Id: "00000000-0000-0000-0000-0000000a0500"}); status.Code(err) != tt.want {
			t.Errorf("%s: code = %v, want %v", name, status.Code(err), tt.want)
		}
	}
}
