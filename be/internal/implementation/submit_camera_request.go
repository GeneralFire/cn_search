package implementation

import (
	"context"
	"math"
	"strings"
	"unicode/utf8"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/GeneralFire/cn_search/be/internal/models"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

// Ограничения полей заявки, символов (CHECK в таблице camera_requests).
const (
	maxCameraAddress = 300
	maxCameraReason  = 1000
	maxCameraContact = 200
)

var errCameraRequestsOff = status.Error(codes.Unimplemented, "заявки на камеры не включены")

// SubmitCameraRequest — rpc SubmitCameraRequest: заявка на расположение камеры, статус «новая».
func (s *CarSearch) SubmitCameraRequest(ctx context.Context, req *pb.SubmitCameraRequestRequest) (*pb.CameraRequest, error) {
	if s.cameraRequests == nil {
		return nil, errCameraRequestsOff
	}
	o := models.SubmitCameraRequestOpts{
		Lat: req.GetLat(), Lon: req.GetLon(),
		Address: strings.TrimSpace(req.GetAddress()),
		Reason:  strings.TrimSpace(req.GetReason()),
		Contact: strings.TrimSpace(req.GetContact()),
	}
	switch {
	case !validLatLon(o.Lat, o.Lon):
		return nil, status.Error(codes.InvalidArgument, "lat/lon: укажите точку на карте")
	case o.Reason == "":
		return nil, status.Error(codes.InvalidArgument, "reason: укажите причину")
	case utf8.RuneCountInString(o.Reason) > maxCameraReason:
		return nil, status.Errorf(codes.InvalidArgument, "reason: не длиннее %d символов", maxCameraReason)
	case utf8.RuneCountInString(o.Address) > maxCameraAddress:
		return nil, status.Errorf(codes.InvalidArgument, "address: не длиннее %d символов", maxCameraAddress)
	case utf8.RuneCountInString(o.Contact) > maxCameraContact:
		return nil, status.Errorf(codes.InvalidArgument, "contact: не длиннее %d символов", maxCameraContact)
	}
	c, err := s.cameraRequests.Submit(ctx, o)
	if err != nil {
		return nil, status.Error(codes.Internal, "не удалось сохранить заявку")
	}
	return cameraRequestPB(c, false), nil
}

// validLatLon — точка в допустимых границах (NaN/Inf не проходят сравнения).
func validLatLon(lat, lon float64) bool {
	return lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180 && !math.IsNaN(lat) && !math.IsNaN(lon)
}

// cameraRequestPB — заявка в ответ; contact — только администратору (withContact).
func cameraRequestPB(c models.CameraRequest, withContact bool) *pb.CameraRequest {
	out := &pb.CameraRequest{
		Id: c.ID, Lat: c.Lat, Lon: c.Lon, Address: c.Address, Reason: c.Reason,
		Status: cameraStatusPB[c.Status], Votes: int32(c.Votes), CreatedAt: timestamppb.New(c.CreatedAt),
	}
	if withContact {
		out.Contact = c.Contact
	}
	return out
}

var cameraStatusPB = map[models.CameraRequestStatus]pb.CameraRequestStatus{
	models.CameraRequestNew:       pb.CameraRequestStatus_CAMERA_REQUEST_STATUS_NEW,
	models.CameraRequestApproved:  pb.CameraRequestStatus_CAMERA_REQUEST_STATUS_APPROVED,
	models.CameraRequestRejected:  pb.CameraRequestStatus_CAMERA_REQUEST_STATUS_REJECTED,
	models.CameraRequestInstalled: pb.CameraRequestStatus_CAMERA_REQUEST_STATUS_INSTALLED,
}

var cameraStatusModel = map[pb.CameraRequestStatus]models.CameraRequestStatus{
	pb.CameraRequestStatus_CAMERA_REQUEST_STATUS_NEW:       models.CameraRequestNew,
	pb.CameraRequestStatus_CAMERA_REQUEST_STATUS_APPROVED:  models.CameraRequestApproved,
	pb.CameraRequestStatus_CAMERA_REQUEST_STATUS_REJECTED:  models.CameraRequestRejected,
	pb.CameraRequestStatus_CAMERA_REQUEST_STATUS_INSTALLED: models.CameraRequestInstalled,
}
