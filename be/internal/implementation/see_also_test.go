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

func TestSeeAlso(t *testing.T) {
	svc, uc := newSvc(t)
	uc.SeeAlsoMock.Expect(minimock.AnyContext, models.SeeAlsoOpts{ExcludeSightingID: sighting.ID, Limit: MaxSeeAlso}).
		Return([]models.PlateSighting{sighting}, nil)

	resp, err := svc.SeeAlso(context.Background(), &pb.SeeAlsoRequest{Id: sighting.ID})
	if err != nil || len(resp.GetItems()) != 1 || resp.GetItems()[0].GetThumbUrl() == "" {
		t.Fatalf("SeeAlso: %v %v", resp, err)
	}
}

func TestSeeAlsoErrors(t *testing.T) {
	for name, req := range map[string]*pb.SeeAlsoRequest{
		"не uuid":    {Id: "x"},
		"limit > 21": {Limit: MaxSeeAlso + 1},
		"limit < 0":  {Limit: -1},
	} {
		svc, _ := newSvc(t)
		if _, err := svc.SeeAlso(context.Background(), req); status.Code(err) != codes.InvalidArgument {
			t.Errorf("%s: %v", name, status.Code(err))
		}
	}
	svc, uc := newSvc(t)
	uc.SeeAlsoMock.Return(nil, errors.New("db down"))
	if _, err := svc.SeeAlso(context.Background(), &pb.SeeAlsoRequest{}); status.Code(err) != codes.Internal {
		t.Errorf("ошибка БД: %v", status.Code(err))
	}
}
