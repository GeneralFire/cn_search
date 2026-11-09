package implementation

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/gojuno/minimock/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/GeneralFire/cn_search/be/internal/implementation/mocks"
	"github.com/GeneralFire/cn_search/be/internal/models"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

const testAdminToken = "s3cret"

func newComplaintsSvc(t *testing.T, adminToken string) (*CarSearch, *mocks.ComplaintsMock) {
	c := mocks.NewComplaintsMock(t)
	return NewCarSearch(mocks.NewPlateSightingsMock(t), "/s3/cars/", WithComplaints(c, adminToken)), c
}

func adminCtx(token string) context.Context {
	return metadata.NewIncomingContext(context.Background(), metadata.Pairs("admin-token", token))
}

var complaint = models.Complaint{
	ID: "00000000-0000-0000-0000-0000000c0001", SightingID: sighting.ID, Reason: models.ComplaintPersonalData,
	Comment: "лицо в кадре", Status: models.ComplaintNew, CreatedAt: sighting.CreatedAt,
}

func TestComplain(t *testing.T) {
	svc, c := newComplaintsSvc(t, "")
	c.ComplainMock.Expect(minimock.AnyContext, models.ComplainOpts{SightingID: sighting.ID, Reason: models.ComplaintPersonalData, Comment: "лицо в кадре"}).
		Return(complaint, nil)

	resp, err := svc.Complain(context.Background(), &pb.ComplainRequest{
		SightingId: sighting.ID, Reason: pb.ComplaintReason_COMPLAINT_REASON_PERSONAL_DATA, Comment: "  лицо в кадре ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetStatus() != pb.ComplaintStatus_COMPLAINT_STATUS_NEW || resp.GetReason() != pb.ComplaintReason_COMPLAINT_REASON_PERSONAL_DATA ||
		resp.GetComment() != "лицо в кадре" || resp.GetCreatedAt() == nil || resp.GetSighting() != nil {
		t.Errorf("ответ: %v", resp)
	}
}

func TestComplainErrors(t *testing.T) {
	for name, req := range map[string]*pb.ComplainRequest{
		"не uuid":         {SightingId: "x", Reason: pb.ComplaintReason_COMPLAINT_REASON_OTHER},
		"без причины":     {SightingId: sighting.ID},
		"плохая причина":  {SightingId: sighting.ID, Reason: 42},
		"длинный коммент": {SightingId: sighting.ID, Reason: pb.ComplaintReason_COMPLAINT_REASON_OTHER, Comment: strings.Repeat("ж", maxComplaintComment+1)},
	} {
		svc, _ := newComplaintsSvc(t, "")
		if _, err := svc.Complain(context.Background(), req); status.Code(err) != codes.InvalidArgument {
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
		svc, c := newComplaintsSvc(t, "")
		c.ComplainMock.Return(models.Complaint{}, tt.err)
		req := &pb.ComplainRequest{SightingId: sighting.ID, Reason: pb.ComplaintReason_COMPLAINT_REASON_BAD_PHOTO, Comment: strings.Repeat("ж", maxComplaintComment)}
		if _, err := svc.Complain(context.Background(), req); status.Code(err) != tt.want {
			t.Errorf("%s: %v, want %v", name, status.Code(err), tt.want)
		}
	}
	// Без WithComplaints — Unimplemented, а не паника.
	svc := NewCarSearch(mocks.NewPlateSightingsMock(t), "")
	if _, err := svc.Complain(context.Background(), &pb.ComplainRequest{}); status.Code(err) != codes.Unimplemented {
		t.Errorf("без жалоб: %v", status.Code(err))
	}
}

func TestListComplaints(t *testing.T) {
	svc, c := newComplaintsSvc(t, testAdminToken)
	next := 1
	c.ListComplaintsMock.Expect(minimock.AnyContext, models.ListComplaintsOpts{Status: models.ComplaintNew, Limit: defaultComplaintsLimit}).
		Return(models.ComplaintsPage{Items: []models.ComplaintItem{{Complaint: complaint, Sighting: sighting}}, Total: 2, NextOffset: &next}, nil)

	resp, err := svc.ListComplaints(adminCtx(testAdminToken), &pb.ListComplaintsRequest{Status: pb.ComplaintStatus_COMPLAINT_STATUS_NEW})
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetTotal() != 2 || resp.GetNextOffset() != 1 || len(resp.GetItems()) != 1 {
		t.Fatalf("ответ: %v", resp)
	}
	s := resp.GetItems()[0].GetSighting()
	if s.GetPlate() != "А123ВС 777" || s.GetThumbUrl() != "/s3/cars/car-1_thumb.jpg" || s.GetAddress() == "" {
		t.Errorf("фиксация жалобы: %v", s)
	}
}

func TestListComplaintsErrors(t *testing.T) {
	for name, req := range map[string]*pb.ListComplaintsRequest{
		"offset < 0":    {Offset: -1},
		"limit > max":   {Limit: MaxLimit + 1},
		"плохой фильтр": {Status: 7},
	} {
		svc, _ := newComplaintsSvc(t, testAdminToken)
		if _, err := svc.ListComplaints(adminCtx(testAdminToken), req); status.Code(err) != codes.InvalidArgument {
			t.Errorf("%s: %v", name, status.Code(err))
		}
	}
	svc, c := newComplaintsSvc(t, testAdminToken)
	c.ListComplaintsMock.Return(models.ComplaintsPage{}, errors.New("db down"))
	if _, err := svc.ListComplaints(adminCtx(testAdminToken), &pb.ListComplaintsRequest{}); status.Code(err) != codes.Internal {
		t.Errorf("ошибка БД: %v", status.Code(err))
	}
}

func TestResolveComplaint(t *testing.T) {
	svc, c := newComplaintsSvc(t, testAdminToken)
	resolved := complaint
	resolved.Status = models.ComplaintResolved
	c.ResolveComplaintMock.Expect(minimock.AnyContext, complaint.ID).Return(resolved, nil)

	resp, err := svc.ResolveComplaint(adminCtx(testAdminToken), &pb.ResolveComplaintRequest{Id: complaint.ID})
	if err != nil || resp.GetStatus() != pb.ComplaintStatus_COMPLAINT_STATUS_RESOLVED {
		t.Fatalf("Resolve: %v %v", resp, err)
	}

	for name, tt := range map[string]struct {
		id   string
		err  error
		want codes.Code
	}{
		"не uuid":    {"1", nil, codes.InvalidArgument},
		"нет жалобы": {complaint.ID, models.ErrNotFound, codes.NotFound},
		"ошибка БД":  {complaint.ID, errors.New("db down"), codes.Internal},
	} {
		svc, c := newComplaintsSvc(t, testAdminToken)
		if tt.err != nil {
			c.ResolveComplaintMock.Return(models.Complaint{}, tt.err)
		}
		if _, err := svc.ResolveComplaint(adminCtx(testAdminToken), &pb.ResolveComplaintRequest{Id: tt.id}); status.Code(err) != tt.want {
			t.Errorf("%s: %v, want %v", name, status.Code(err), tt.want)
		}
	}
}

// Админские методы без верного admin-token — PermissionDenied, usecase не вызывается.
func TestAdminMethodsRequireToken(t *testing.T) {
	for name, tt := range map[string]struct {
		configured string
		ctx        context.Context
	}{
		"нет заголовка":         {testAdminToken, context.Background()},
		"неверный":              {testAdminToken, adminCtx("wrong")},
		"префикс":               {testAdminToken, adminCtx("s3cre")},
		"пустой":                {testAdminToken, adminCtx("")},
		"два значения":          {testAdminToken, metadata.NewIncomingContext(context.Background(), metadata.Pairs("admin-token", testAdminToken, "admin-token", "x"))},
		"ADMIN_TOKEN не задан":  {"", adminCtx("")},
		"не задан, любой токен": {"", adminCtx("anything")},
	} {
		svc, _ := newComplaintsSvc(t, tt.configured)
		if _, err := svc.ListComplaints(tt.ctx, &pb.ListComplaintsRequest{}); status.Code(err) != codes.PermissionDenied {
			t.Errorf("List, %s: %v", name, status.Code(err))
		}
		if _, err := svc.ResolveComplaint(tt.ctx, &pb.ResolveComplaintRequest{Id: complaint.ID}); status.Code(err) != codes.PermissionDenied {
			t.Errorf("Resolve, %s: %v", name, status.Code(err))
		}
	}
}
