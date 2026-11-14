package implementation

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/GeneralFire/cn_search/be/internal/models"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

const defaultClarificationsLimit = 30

// ListClarifications — rpc ListClarifications (админ): уточнения с фиксациями, от новых к старым, постранично.
func (s *CarSearch) ListClarifications(ctx context.Context, req *pb.ListClarificationsRequest) (*pb.ListClarificationsResponse, error) {
	if err := requireAdmin(ctx, s.adminToken); err != nil {
		return nil, err
	}
	if s.admin == nil {
		return nil, status.Error(codes.Unimplemented, "админские методы не включены")
	}
	var st models.ClarificationStatus
	if req.GetStatus() != pb.ClarificationStatus_CLARIFICATION_STATUS_UNSPECIFIED {
		var ok bool
		if st, ok = clarificationStatusModel[req.GetStatus()]; !ok {
			return nil, status.Error(codes.InvalidArgument, "status: неизвестный статус")
		}
	}
	switch {
	case req.GetSightingId() != "" && uuid.Validate(req.GetSightingId()) != nil:
		return nil, status.Error(codes.InvalidArgument, "sighting_id: нужен uuid фиксации")
	case req.GetOffset() < 0:
		return nil, status.Error(codes.InvalidArgument, "offset: не может быть отрицательным")
	case req.GetLimit() < 0 || req.GetLimit() > MaxLimit:
		return nil, status.Errorf(codes.InvalidArgument, "limit: от 1 до %d (0 — по умолчанию %d)", MaxLimit, defaultClarificationsLimit)
	}
	limit := int(req.GetLimit())
	if limit == 0 {
		limit = defaultClarificationsLimit
	}

	page, err := s.admin.ListClarifications(ctx, models.ListClarificationsOpts{
		Status: st, Offset: int(req.GetOffset()), Limit: limit, IncludeTestData: includeTestData(ctx),
		SightingID: req.GetSightingId(),
	})
	if err != nil {
		return nil, internalErr(ctx, err, "не удалось получить уточнения")
	}
	resp := &pb.ListClarificationsResponse{Items: make([]*pb.Clarification, len(page.Items)), Total: int32(page.Total)}
	for i, it := range page.Items {
		resp.Items[i] = clarificationPB(it.Clarification)
		resp.Items[i].Sighting = s.toPB(it.Sighting, true)
	}
	if page.NextOffset != nil {
		next := int32(*page.NextOffset)
		resp.NextOffset = &next
	}
	return resp, nil
}
