package implementation

import (
	"context"
	"errors"
	"testing"

	"github.com/gojuno/minimock/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/GeneralFire/cn_search/be/internal/implementation/mocks"
	"github.com/GeneralFire/cn_search/be/internal/models"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

func newAdminSvc(t *testing.T) (*CarSearch, *mocks.AdminMock, *mocks.PlateSightingsMock) {
	a, uc := mocks.NewAdminMock(t), mocks.NewPlateSightingsMock(t)
	return NewCarSearch(uc, "/s3/cars/", WithAdmin(a, testAdminToken)), a, uc
}

func TestMe(t *testing.T) {
	svc, _, _ := newAdminSvc(t)
	for name, tt := range map[string]struct {
		ctx  context.Context
		want bool
	}{
		"без токена":     {context.Background(), false},
		"неверный токен": {adminCtx("nope"), false},
		"верный токен":   {adminCtx(testAdminToken), true},
	} {
		resp, err := svc.Me(tt.ctx, &pb.MeRequest{})
		if err != nil || resp.GetIsAdmin() != tt.want {
			t.Errorf("%s: %v %v", name, resp, err)
		}
	}
	// ADMIN_TOKEN не задан — никто не админ, даже с пустым заголовком.
	off := NewCarSearch(mocks.NewPlateSightingsMock(t), "")
	if resp, _ := off.Me(adminCtx(""), &pb.MeRequest{}); resp.GetIsAdmin() {
		t.Error("без ADMIN_TOKEN is_admin = true")
	}
}

func TestAdminMethodsDenied(t *testing.T) {
	svc, _, _ := newAdminSvc(t)
	id := sighting.ID
	for name, call := range map[string]func(ctx context.Context) error{
		"HideSighting": func(ctx context.Context) error {
			_, err := svc.HideSighting(ctx, &pb.HideSightingRequest{Id: id, Hidden: true})
			return err
		},
		"RestoreSightingPhoto": func(ctx context.Context) error {
			_, err := svc.RestoreSightingPhoto(ctx, &pb.RestoreSightingPhotoRequest{Id: id})
			return err
		},
		"SetClarificationStatus": func(ctx context.Context) error {
			_, err := svc.SetClarificationStatus(ctx, &pb.SetClarificationStatusRequest{Id: id, Status: pb.ClarificationStatus_CLARIFICATION_STATUS_ACCEPTED})
			return err
		},
		"ListAdminAudit": func(ctx context.Context) error {
			_, err := svc.ListAdminAudit(ctx, &pb.ListAdminAuditRequest{})
			return err
		},
		"ListClarifications": func(ctx context.Context) error {
			_, err := svc.ListClarifications(ctx, &pb.ListClarificationsRequest{})
			return err
		},
	} {
		for _, ctx := range []context.Context{context.Background(), adminCtx("nope")} {
			if code := status.Code(call(ctx)); code != codes.PermissionDenied {
				t.Errorf("%s: %v", name, code)
			}
		}
	}
}

func TestHideSighting(t *testing.T) {
	svc, a, _ := newAdminSvc(t)
	hidden := sighting
	hidden.Hidden = true
	a.HideSightingMock.Expect(minimock.AnyContext, sighting.ID, true).Return(hidden, nil)
	resp, err := svc.HideSighting(adminCtx(testAdminToken), &pb.HideSightingRequest{Id: sighting.ID, Hidden: true})
	if err != nil || !resp.GetHidden() || resp.GetId() != sighting.ID {
		t.Fatalf("HideSighting: %v %v", resp, err)
	}

	if _, err := svc.HideSighting(adminCtx(testAdminToken), &pb.HideSightingRequest{Id: "x"}); status.Code(err) != codes.InvalidArgument {
		t.Errorf("не uuid: %v", err)
	}
	svc, a, _ = newAdminSvc(t)
	a.HideSightingMock.Return(models.PlateSighting{}, models.ErrNotFound)
	if _, err := svc.HideSighting(adminCtx(testAdminToken), &pb.HideSightingRequest{Id: sighting.ID}); status.Code(err) != codes.NotFound {
		t.Errorf("нет фиксации: %v", err)
	}
}

func TestRestoreSightingPhoto(t *testing.T) {
	svc, a, _ := newAdminSvc(t)
	a.RestoreSightingPhotoMock.Expect(minimock.AnyContext, sighting.ID).Return(sighting, nil)
	resp, err := svc.RestoreSightingPhoto(adminCtx(testAdminToken), &pb.RestoreSightingPhotoRequest{Id: sighting.ID})
	if err != nil || resp.GetPhotoHidden() || resp.GetId() != sighting.ID {
		t.Fatalf("RestoreSightingPhoto: %v %v", resp, err)
	}
	if _, err := svc.RestoreSightingPhoto(adminCtx(testAdminToken), &pb.RestoreSightingPhotoRequest{Id: "x"}); status.Code(err) != codes.InvalidArgument {
		t.Errorf("не uuid: %v", err)
	}
	svc, a, _ = newAdminSvc(t)
	a.RestoreSightingPhotoMock.Return(models.PlateSighting{}, models.ErrNotFound)
	if _, err := svc.RestoreSightingPhoto(adminCtx(testAdminToken), &pb.RestoreSightingPhotoRequest{Id: sighting.ID}); status.Code(err) != codes.NotFound {
		t.Errorf("нет фиксации: %v", err)
	}
}

func TestSetClarificationStatus(t *testing.T) {
	const id = "00000000-0000-0000-0000-0000000d0001"
	svc, a, _ := newAdminSvc(t)
	a.SetClarificationStatusMock.Expect(minimock.AnyContext, id, models.ClarificationAccepted).
		Return(models.Clarification{ID: id, SightingID: sighting.ID, Plate: "А124ВС 777", Status: models.ClarificationAccepted, PlateType: "ru"}, nil)
	resp, err := svc.SetClarificationStatus(adminCtx(testAdminToken), &pb.SetClarificationStatusRequest{Id: id, Status: pb.ClarificationStatus_CLARIFICATION_STATUS_ACCEPTED})
	if err != nil || resp.GetStatus() != pb.ClarificationStatus_CLARIFICATION_STATUS_ACCEPTED || resp.GetPlateType() != pb.PlateType_PLATE_TYPE_RU {
		t.Fatalf("SetClarificationStatus: %v %v", resp, err)
	}

	for name, req := range map[string]*pb.SetClarificationStatusRequest{
		"не uuid":     {Id: "x", Status: pb.ClarificationStatus_CLARIFICATION_STATUS_ACCEPTED},
		"без статуса": {Id: id},
	} {
		if _, err := svc.SetClarificationStatus(adminCtx(testAdminToken), req); status.Code(err) != codes.InvalidArgument {
			t.Errorf("%s: %v", name, err)
		}
	}
	svc, a, _ = newAdminSvc(t)
	a.SetClarificationStatusMock.Return(models.Clarification{}, errors.New("db down"))
	if _, err := svc.SetClarificationStatus(adminCtx(testAdminToken), &pb.SetClarificationStatusRequest{Id: id, Status: pb.ClarificationStatus_CLARIFICATION_STATUS_NOT_ACCEPTED}); status.Code(err) != codes.Internal {
		t.Errorf("ошибка БД: %v", err)
	}
}

func TestListClarifications(t *testing.T) {
	svc, a, _ := newAdminSvc(t)
	next := 1
	a.ListClarificationsMock.Expect(minimock.AnyContext, models.ListClarificationsOpts{Status: models.ClarificationNotAccepted, Limit: 1}).
		Return(models.ClarificationsPage{Items: []models.ClarificationItem{{
			Clarification: models.Clarification{ID: "c1", SightingID: sighting.ID, Plate: "А124ВС 777", Status: models.ClarificationNotAccepted},
			Sighting:      sighting,
		}}, Total: 2, NextOffset: &next}, nil)
	resp, err := svc.ListClarifications(adminCtx(testAdminToken), &pb.ListClarificationsRequest{
		Status: pb.ClarificationStatus_CLARIFICATION_STATUS_NOT_ACCEPTED, Limit: 1,
	})
	if err != nil || resp.GetTotal() != 2 || resp.GetNextOffset() != 1 || len(resp.GetItems()) != 1 ||
		resp.GetItems()[0].GetSighting().GetPhotoUrl() != "/s3/cars/car-1.jpg" {
		t.Fatalf("ListClarifications: %v %v", resp, err)
	}
	for name, req := range map[string]*pb.ListClarificationsRequest{
		"offset < 0":    {Offset: -1},
		"limit > max":   {Limit: MaxLimit + 1},
		"плохой статус": {Status: 42},
	} {
		if _, err := svc.ListClarifications(adminCtx(testAdminToken), req); status.Code(err) != codes.InvalidArgument {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// Скрытые фиксации: Search/Get/SeeAlso просят их у usecase только для админа.
func TestIncludeHiddenOnlyForAdmin(t *testing.T) {
	for _, tt := range []struct {
		ctx  context.Context
		want bool
	}{{context.Background(), false}, {adminCtx("nope"), false}, {adminCtx(testAdminToken), true}} {
		svc, _, uc := newAdminSvc(t)
		uc.SearchMock.Expect(minimock.AnyContext, models.SearchOpts{Limit: DefaultLimit, IncludeHidden: tt.want}).Return(models.SearchPage{}, nil)
		uc.ListSamePlateMock.Expect(minimock.AnyContext, models.SamePlateOpts{ID: sighting.ID, IncludeHidden: tt.want}).Return(models.SamePlate{Items: []models.PlateSighting{sighting}}, nil)
		uc.SeeAlsoMock.Expect(minimock.AnyContext, models.SeeAlsoOpts{Limit: MaxSeeAlso, IncludeHidden: tt.want}).Return(nil, nil)
		if _, err := svc.Search(tt.ctx, &pb.SearchRequest{}); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.Get(tt.ctx, &pb.GetRequest{Id: sighting.ID}); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.SeeAlso(tt.ctx, &pb.SeeAlsoRequest{}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestListAdminAudit(t *testing.T) {
	svc, a, _ := newAdminSvc(t)
	next := 1
	a.ListAuditMock.Expect(minimock.AnyContext, models.ListAuditOpts{Action: models.AuditHideSighting, Offset: 0, Limit: defaultComplaintsLimit}).
		Return(models.AuditPage{Total: 2, NextOffset: &next, Items: []models.AuditEntry{{
			ID: "a1", Action: models.AuditHideSighting, Actor: "admin-token", TargetType: models.AuditTargetSighting,
			TargetID: sighting.ID, Details: []byte(`{"from":{"hidden":false},"to":{"hidden":true}}`), RequestID: "req-1",
		}}}, nil)

	resp, err := svc.ListAdminAudit(adminCtx(testAdminToken), &pb.ListAdminAuditRequest{Action: models.AuditHideSighting})
	if err != nil || resp.GetTotal() != 2 || resp.GetNextOffset() != 1 || len(resp.GetItems()) != 1 {
		t.Fatalf("ListAdminAudit: %v %v", resp, err)
	}
	if it := resp.Items[0]; it.RequestId != "req-1" || it.Actor != "admin-token" || it.Details != `{"from":{"hidden":false},"to":{"hidden":true}}` {
		t.Errorf("item: %v", it)
	}
	if _, err := svc.ListAdminAudit(adminCtx(testAdminToken), &pb.ListAdminAuditRequest{TargetId: "не-uuid"}); status.Code(err) != codes.InvalidArgument {
		t.Errorf("target_id: %v", err)
	}
}
