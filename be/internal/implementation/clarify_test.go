package implementation

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/gojuno/minimock/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/GeneralFire/cn_search/be/internal/models"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

func TestClarify(t *testing.T) {
	svc, uc := newSvc(t)
	uc.ClarifyMock.Expect(minimock.AnyContext, models.ClarifyOpts{SightingID: sighting.ID, Plate: "А123ВС 177"}).
		Return(models.Clarification{ID: "c1", SightingID: sighting.ID, Plate: "А123ВС 177",
			Status: models.ClarificationNotAccepted, CreatedAt: sighting.CreatedAt}, nil)

	resp, err := svc.Clarify(context.Background(), &pb.ClarifyRequest{SightingId: sighting.ID, Plate: "  А123ВС 177 "})
	if err != nil {
		t.Fatal(err)
	}
	c := resp.GetClarification()
	if c.GetPlate() != "А123ВС 177" || c.GetStatus() != pb.ClarificationStatus_CLARIFICATION_STATUS_NOT_ACCEPTED || c.GetCreatedAt() == nil {
		t.Errorf("ответ: %v", c)
	}
}

func TestClarifyPlateFormat(t *testing.T) {
	svc, uc := newSvc(t)
	uc.ClarifyMock.Expect(minimock.AnyContext, models.ClarifyOpts{SightingID: sighting.ID, Plate: "B MW 1234", PlateType: "eu", PlateRegion: "D"}).
		Return(models.Clarification{ID: "c1", Plate: "B MW 1234", PlateType: "eu", PlateRegion: "D", CreatedAt: sighting.CreatedAt}, nil)

	resp, err := svc.Clarify(context.Background(), &pb.ClarifyRequest{
		SightingId: sighting.ID, Plate: "B MW 1234", PlateType: pb.PlateType_PLATE_TYPE_EU, PlateRegion: " d ",
	})
	if err != nil || resp.GetClarification().GetPlateType() != pb.PlateType_PLATE_TYPE_EU || resp.GetClarification().GetPlateRegion() != "D" {
		t.Fatalf("EU: %v %v", resp, err)
	}

	// Для RU регион не нужен — отбрасывается.
	svc, uc = newSvc(t)
	uc.ClarifyMock.Expect(minimock.AnyContext, models.ClarifyOpts{SightingID: sighting.ID, Plate: "А1", PlateType: "ru"}).
		Return(models.Clarification{PlateType: "ru"}, nil)
	if _, err := svc.Clarify(context.Background(), &pb.ClarifyRequest{SightingId: sighting.ID, Plate: "А1", PlateType: pb.PlateType_PLATE_TYPE_RU, PlateRegion: "CA"}); err != nil {
		t.Fatal(err)
	}
}

func TestClarifyErrors(t *testing.T) {
	for name, req := range map[string]*pb.ClarifyRequest{
		"не uuid":       {SightingId: "x", Plate: "А1"},
		"пустой номер":  {SightingId: sighting.ID, Plate: "   "},
		"длинный номер": {SightingId: sighting.ID, Plate: strings.Repeat("А", maxQueryLength+1)},
		"плохой регион": {SightingId: sighting.ID, Plate: "7XYZ123", PlateType: pb.PlateType_PLATE_TYPE_US, PlateRegion: "CAL1"},
	} {
		svc, _ := newSvc(t)
		if _, err := svc.Clarify(context.Background(), req); status.Code(err) != codes.InvalidArgument {
			t.Errorf("%s: %v", name, status.Code(err))
		}
	}
	for name, tt := range map[string]struct {
		err  error
		want codes.Code
	}{
		"нет фиксации": {models.ErrNotFound, codes.NotFound},
		"ошибка БД":    {errors.New("db down"), codes.Internal},
	} {
		svc, uc := newSvc(t)
		uc.ClarifyMock.Return(models.Clarification{}, tt.err)
		if _, err := svc.Clarify(context.Background(), &pb.ClarifyRequest{SightingId: sighting.ID, Plate: "А1"}); status.Code(err) != tt.want {
			t.Errorf("%s: %v, want %v", name, status.Code(err), tt.want)
		}
	}
}
