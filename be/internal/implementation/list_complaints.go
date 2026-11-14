package implementation

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/GeneralFire/cn_search/be/internal/models"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

const defaultComplaintsLimit = 30

// ListComplaints — rpc ListComplaints (админ): жалобы с фиксациями, от новых к старым, постранично.
func (s *CarSearch) ListComplaints(ctx context.Context, req *pb.ListComplaintsRequest) (*pb.ListComplaintsResponse, error) {
	if err := requireAdmin(ctx, s.adminToken); err != nil {
		return nil, err
	}
	if s.complaints == nil {
		return nil, status.Error(codes.Unimplemented, "жалобы не включены")
	}
	var st models.ComplaintStatus
	switch req.GetStatus() {
	case pb.ComplaintStatus_COMPLAINT_STATUS_UNSPECIFIED:
	case pb.ComplaintStatus_COMPLAINT_STATUS_NEW:
		st = models.ComplaintNew
	case pb.ComplaintStatus_COMPLAINT_STATUS_RESOLVED:
		st = models.ComplaintResolved
	default:
		return nil, status.Error(codes.InvalidArgument, "status: неизвестный статус")
	}
	switch {
	case req.GetSightingId() != "" && uuid.Validate(req.GetSightingId()) != nil:
		return nil, status.Error(codes.InvalidArgument, "sighting_id: нужен uuid фиксации")
	case req.GetOffset() < 0:
		return nil, status.Error(codes.InvalidArgument, "offset: не может быть отрицательным")
	case req.GetLimit() < 0 || req.GetLimit() > MaxLimit:
		return nil, status.Errorf(codes.InvalidArgument, "limit: от 1 до %d (0 — по умолчанию %d)", MaxLimit, defaultComplaintsLimit)
	}
	limit := int(req.GetLimit())
	if limit == 0 {
		limit = defaultComplaintsLimit
	}

	page, err := s.complaints.ListComplaints(ctx, models.ListComplaintsOpts{
		Status: st, Offset: int(req.GetOffset()), Limit: limit, IncludeTestData: includeTestData(ctx),
		SightingID: req.GetSightingId(),
	})
	if err != nil {
		return nil, internalErr(ctx, err, "не удалось получить жалобы")
	}
	resp := &pb.ListComplaintsResponse{Items: make([]*pb.Complaint, len(page.Items)), Total: int32(page.Total)}
	for i, it := range page.Items {
		resp.Items[i] = complaintPB(it.Complaint)
		resp.Items[i].Sighting = s.toPB(it.Sighting)
	}
	if page.NextOffset != nil {
		next := int32(*page.NextOffset)
		resp.NextOffset = &next
	}
	return resp, nil
}
