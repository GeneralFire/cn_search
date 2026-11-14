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
	lat, lon := 55.7579, 37.6115
	second.Lat, second.Lon = &lat, &lon
	uc.ListSamePlateMock.Expect(minimock.AnyContext, models.SamePlateOpts{ID: "00000000-0000-0000-0000-000000000001"}).Return(models.SamePlate{
		Items: []models.PlateSighting{second, sighting},
		Clarifications: []models.PlateClarification{
			{Plate: "А123ВС 77", PlateType: "ru", Count: 3, Status: models.ClarificationAccepted, SightingIDs: []string{second.ID, sighting.ID},
				Items: []models.Clarification{{ID: "c-1", SightingID: second.ID, Plate: "А123ВС 77", Status: models.ClarificationAccepted, PlateType: "ru"}}},
			{Plate: "AB 123", Count: 1, Status: models.ClarificationNotAccepted, SightingIDs: []string{sighting.ID}},
		},
	}, nil)

	resp, err := svc.Get(context.Background(), &pb.GetRequest{Id: "00000000-0000-0000-0000-000000000001"})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.GetItems()) != 2 || resp.GetItems()[0].GetId() != "00000000-0000-0000-0000-000000000004" || resp.GetItems()[1].GetId() != "00000000-0000-0000-0000-000000000001" {
		t.Fatalf("Get(1): %v (порядок из usecase должен сохраниться)", resp)
	}
	// Запрос без входа: координаты скрыты (с входом — coords_test.go), у всех фиксаций coords_hidden.
	for _, got := range resp.GetItems() {
		if got.Lat != nil || got.Lon != nil || !got.GetCoordsHidden() {
			t.Errorf("аноним: фиксация %s: %v, %v, coords_hidden=%v", got.GetId(), got.Lat, got.Lon, got.GetCoordsHidden())
		}
	}
	// Уточнения — в порядке usecase, со статусом, форматом и фиксациями.
	cl := resp.GetClarifications()
	if len(cl) != 2 {
		t.Fatalf("уточнения: %v", cl)
	}
	if c := cl[0]; c.GetPlate() != "А123ВС 77" || c.GetPlateType() != pb.PlateType_PLATE_TYPE_RU || c.GetCount() != 3 ||
		c.GetStatus() != pb.ClarificationStatus_CLARIFICATION_STATUS_ACCEPTED || len(c.GetSightingIds()) != 2 ||
		len(c.GetItems()) != 1 || c.GetItems()[0].GetId() != "c-1" || c.GetItems()[0].GetStatus() != pb.ClarificationStatus_CLARIFICATION_STATUS_ACCEPTED {
		t.Errorf("уточнение 1: %v", c)
	}
	if c := cl[1]; c.GetPlate() != "AB 123" || c.GetPlateType() != pb.PlateType_PLATE_TYPE_DEFAULT ||
		c.GetStatus() != pb.ClarificationStatus_CLARIFICATION_STATUS_NOT_ACCEPTED {
		t.Errorf("уточнение 2: %v", c)
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
		uc.ListSamePlateMock.Return(models.SamePlate{}, tt.err)
		if _, err := svc.Get(ctx, &pb.GetRequest{Id: "00000000-0000-0000-0000-0000000a0500"}); status.Code(err) != tt.want {
			t.Errorf("%s: code = %v, want %v", name, status.Code(err), tt.want)
		}
	}
}
