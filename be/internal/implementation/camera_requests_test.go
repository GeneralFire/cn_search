package implementation

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/gojuno/minimock/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/GeneralFire/cn_search/be/internal/implementation/mocks"
	"github.com/GeneralFire/cn_search/be/internal/models"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

const cameraID = "00000000-0000-0000-0000-0000000ca001"

func newCameraSvc(t *testing.T, adminToken string) (*CarSearch, *mocks.CameraRequestsMock) {
	c := mocks.NewCameraRequestsMock(t)
	return NewCarSearch(mocks.NewPlateSightingsMock(t), "/s3/cars/",
		WithComplaints(mocks.NewComplaintsMock(t), adminToken), WithCameraRequests(c)), c
}

var cameraReq = models.CameraRequest{
	ID: cameraID, Lat: 55.76, Lon: 37.61, Address: "Тверская, 7", Reason: "перекрёсток",
	Contact: "a@b.c", Status: models.CameraRequestNew, Votes: 2, CreatedAt: sighting.CreatedAt,
}

func TestSubmitCameraRequest(t *testing.T) {
	svc, c := newCameraSvc(t, "")
	c.SubmitMock.Expect(minimock.AnyContext, models.SubmitCameraRequestOpts{Lat: 55.76, Lon: 37.61, Address: "Тверская, 7", Reason: "перекрёсток", Contact: "a@b.c"}).
		Return(cameraReq, nil)

	resp, err := svc.SubmitCameraRequest(context.Background(), &pb.SubmitCameraRequestRequest{
		Lat: 55.76, Lon: 37.61, Address: " Тверская, 7 ", Reason: " перекрёсток\n", Contact: " a@b.c ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetStatus() != pb.CameraRequestStatus_CAMERA_REQUEST_STATUS_NEW || resp.GetContact() != "" || resp.GetVotes() != 2 || resp.GetCreatedAt() == nil {
		t.Errorf("ответ (contact не отдаётся): %v", resp)
	}
}

func TestSubmitCameraRequestErrors(t *testing.T) {
	for name, req := range map[string]*pb.SubmitCameraRequestRequest{
		"без причины":     {Lat: 1, Lon: 1, Reason: "  "},
		"широта":          {Lat: 91, Lon: 1, Reason: "x"},
		"долгота":         {Lat: 1, Lon: -181, Reason: "x"},
		"длинная причина": {Lat: 1, Lon: 1, Reason: strings.Repeat("ж", maxCameraReason+1)},
		"длинный контакт": {Lat: 1, Lon: 1, Reason: "x", Contact: strings.Repeat("ж", maxCameraContact+1)},
		"длинный адрес":   {Lat: 1, Lon: 1, Reason: "x", Address: strings.Repeat("ж", maxCameraAddress+1)},
	} {
		svc, _ := newCameraSvc(t, "")
		if _, err := svc.SubmitCameraRequest(context.Background(), req); status.Code(err) != codes.InvalidArgument {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestListCameraRequestsPublicHidesContact(t *testing.T) {
	svc, c := newCameraSvc(t, testAdminToken)
	next := 1
	c.ListMock.Expect(minimock.AnyContext, models.ListCameraRequestsOpts{
		BBox: &models.GeoBox{MinLat: 55, MinLon: 37, MaxLat: 56, MaxLon: 38}, Status: models.CameraRequestNew, Limit: 1,
	}).Return(models.CameraRequestsPage{Items: []models.CameraRequest{cameraReq}, Total: 2, NextOffset: &next}, nil)

	resp, err := svc.ListCameraRequests(context.Background(), &pb.ListCameraRequestsRequest{
		Bbox: &pb.GeoBox{MinLat: 55, MinLon: 37, MaxLat: 56, MaxLon: 38}, Status: pb.CameraRequestStatus_CAMERA_REQUEST_STATUS_NEW, Limit: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.GetItems()) != 1 || resp.GetItems()[0].GetContact() != "" || resp.GetTotal() != 2 || resp.GetNextOffset() != 1 {
		t.Errorf("ответ: %v", resp)
	}
}

func TestListCameraRequestsAdminSeesContact(t *testing.T) {
	svc, c := newCameraSvc(t, testAdminToken)
	c.ListMock.Expect(minimock.AnyContext, models.ListCameraRequestsOpts{Limit: defaultCameraRequestsLimit}).
		Return(models.CameraRequestsPage{Items: []models.CameraRequest{cameraReq}, Total: 1}, nil)

	resp, err := svc.ListCameraRequests(adminCtx(testAdminToken), &pb.ListCameraRequestsRequest{})
	if err != nil || resp.GetItems()[0].GetContact() != "a@b.c" || resp.NextOffset != nil {
		t.Fatalf("ответ: %v %v", resp, err)
	}
	// Неверный токен — не тихий откат к публичному списку, а отказ.
	if _, err := svc.ListCameraRequests(adminCtx("wrong"), &pb.ListCameraRequestsRequest{}); status.Code(err) != codes.PermissionDenied {
		t.Errorf("неверный токен: %v", err)
	}
}

func TestListCameraRequestsErrors(t *testing.T) {
	for name, req := range map[string]*pb.ListCameraRequestsRequest{
		"offset":         {Offset: -1},
		"limit":          {Limit: maxCameraRequestsLimit + 1},
		"статус":         {Status: 42},
		"рамка наоборот": {Bbox: &pb.GeoBox{MinLat: 56, MaxLat: 55, MaxLon: 1}},
		"рамка вне":      {Bbox: &pb.GeoBox{MinLat: -100, MaxLat: 55, MaxLon: 1}},
	} {
		svc, _ := newCameraSvc(t, "")
		if _, err := svc.ListCameraRequests(context.Background(), req); status.Code(err) != codes.InvalidArgument {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestVoteCameraRequest(t *testing.T) {
	svc, c := newCameraSvc(t, "")
	c.VoteMock.Expect(minimock.AnyContext, models.VoteCameraRequestOpts{ID: cameraID}).Return(cameraReq, nil)
	resp, err := svc.VoteCameraRequest(context.Background(), &pb.VoteCameraRequestRequest{Id: cameraID})
	if err != nil || resp.GetVotes() != 2 || resp.GetContact() != "" {
		t.Fatalf("ответ: %v %v", resp, err)
	}

	if _, err := svc.VoteCameraRequest(context.Background(), &pb.VoteCameraRequestRequest{Id: "x"}); status.Code(err) != codes.InvalidArgument {
		t.Errorf("не uuid: %v", err)
	}
	svc, c = newCameraSvc(t, "")
	c.VoteMock.Return(models.CameraRequest{}, models.ErrNotFound)
	if _, err := svc.VoteCameraRequest(context.Background(), &pb.VoteCameraRequestRequest{Id: cameraID}); status.Code(err) != codes.NotFound {
		t.Errorf("нет заявки: %v", err)
	}
}

func TestSetCameraRequestStatus(t *testing.T) {
	svc, c := newCameraSvc(t, testAdminToken)
	approved := cameraReq
	approved.Status = models.CameraRequestApproved
	c.SetStatusMock.Expect(minimock.AnyContext, cameraID, models.CameraRequestApproved).Return(approved, nil)

	resp, err := svc.SetCameraRequestStatus(adminCtx(testAdminToken), &pb.SetCameraRequestStatusRequest{
		Id: cameraID, Status: pb.CameraRequestStatus_CAMERA_REQUEST_STATUS_APPROVED,
	})
	if err != nil || resp.GetStatus() != pb.CameraRequestStatus_CAMERA_REQUEST_STATUS_APPROVED {
		t.Fatalf("ответ: %v %v", resp, err)
	}
}

func TestSetCameraRequestStatusErrors(t *testing.T) {
	svc, _ := newCameraSvc(t, testAdminToken)
	ok := &pb.SetCameraRequestStatusRequest{Id: cameraID, Status: pb.CameraRequestStatus_CAMERA_REQUEST_STATUS_REJECTED}
	if _, err := svc.SetCameraRequestStatus(context.Background(), ok); status.Code(err) != codes.PermissionDenied {
		t.Errorf("без токена: %v", err)
	}
	if _, err := svc.SetCameraRequestStatus(adminCtx(testAdminToken), &pb.SetCameraRequestStatusRequest{Id: cameraID}); status.Code(err) != codes.InvalidArgument {
		t.Errorf("без статуса: %v", err)
	}
	if _, err := svc.SetCameraRequestStatus(adminCtx(testAdminToken), &pb.SetCameraRequestStatusRequest{Id: "x", Status: ok.Status}); status.Code(err) != codes.InvalidArgument {
		t.Errorf("не uuid: %v", err)
	}
	svc, c := newCameraSvc(t, testAdminToken)
	c.SetStatusMock.Return(models.CameraRequest{}, errors.New("db"))
	if _, err := svc.SetCameraRequestStatus(adminCtx(testAdminToken), ok); status.Code(err) != codes.Internal {
		t.Errorf("ошибка БД: %v", err)
	}
}
