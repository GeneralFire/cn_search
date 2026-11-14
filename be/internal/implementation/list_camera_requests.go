package implementation

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/GeneralFire/cn_search/be/internal/models"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

const (
	defaultCameraRequestsLimit = 100
	maxCameraRequestsLimit     = 500 // все точки на карту одним запросом
)

// ListCameraRequests — rpc ListCameraRequests: заявки от новых к старым. Публичный (без contact);
// с заголовком admin-token — для администратора, с contact (токен неверный — PermissionDenied).
func (s *CarSearch) ListCameraRequests(ctx context.Context, req *pb.ListCameraRequestsRequest) (*pb.ListCameraRequestsResponse, error) {
	admin := false
	if md, _ := metadata.FromIncomingContext(ctx); len(md.Get(adminTokenHeader)) > 0 {
		if err := requireAdmin(ctx, s.adminToken); err != nil {
			return nil, err
		}
		admin = true
	}
	if s.cameraRequests == nil {
		return nil, errCameraRequestsOff
	}
	var st models.CameraRequestStatus
	if req.GetStatus() != pb.CameraRequestStatus_CAMERA_REQUEST_STATUS_UNSPECIFIED {
		var ok bool
		if st, ok = cameraStatusModel[req.GetStatus()]; !ok {
			return nil, status.Error(codes.InvalidArgument, "status: неизвестный статус")
		}
	}
	switch {
	case req.GetOffset() < 0:
		return nil, status.Error(codes.InvalidArgument, "offset: не может быть отрицательным")
	case req.GetLimit() < 0 || req.GetLimit() > maxCameraRequestsLimit:
		return nil, status.Errorf(codes.InvalidArgument, "limit: от 1 до %d (0 — по умолчанию %d)", maxCameraRequestsLimit, defaultCameraRequestsLimit)
	}
	o := models.ListCameraRequestsOpts{
		Status: st, Offset: int(req.GetOffset()), Limit: int(req.GetLimit()), IncludeTestData: includeTestData(ctx),
	}
	if o.Limit == 0 {
		o.Limit = defaultCameraRequestsLimit
	}
	if b := req.GetBbox(); b != nil {
		if !validLatLon(b.GetMinLat(), b.GetMinLon()) || !validLatLon(b.GetMaxLat(), b.GetMaxLon()) ||
			b.GetMinLat() > b.GetMaxLat() || b.GetMinLon() > b.GetMaxLon() {
			return nil, status.Error(codes.InvalidArgument, "bbox: неверная рамка")
		}
		o.BBox = &models.GeoBox{MinLat: b.GetMinLat(), MinLon: b.GetMinLon(), MaxLat: b.GetMaxLat(), MaxLon: b.GetMaxLon()}
	}

	page, err := s.cameraRequests.List(ctx, o)
	if err != nil {
		return nil, internalErr(ctx, err, "не удалось получить заявки")
	}
	resp := &pb.ListCameraRequestsResponse{Items: make([]*pb.CameraRequest, len(page.Items)), Total: int32(page.Total)}
	for i, c := range page.Items {
		resp.Items[i] = cameraRequestPB(c, admin)
	}
	if page.NextOffset != nil {
		next := int32(*page.NextOffset)
		resp.NextOffset = &next
	}
	return resp, nil
}
