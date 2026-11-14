package implementation

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/GeneralFire/cn_search/be/internal/models"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

// MaxSeeAlso — не больше стольких карточек в блоке «свежие фиксации».
const MaxSeeAlso = 21

// SeeAlso — rpc SeeAlso: свежие фиксации других номеров.
func (s *CarSearch) SeeAlso(ctx context.Context, req *pb.SeeAlsoRequest) (*pb.SeeAlsoResponse, error) {
	if req.GetId() != "" {
		if err := uuid.Validate(req.GetId()); err != nil {
			return nil, status.Error(codes.InvalidArgument, "id: нужен uuid фиксации")
		}
	}
	limit := int(req.GetLimit())
	switch {
	case limit < 0 || limit > MaxSeeAlso:
		return nil, status.Errorf(codes.InvalidArgument, "limit: от 1 до %d (0 — %d)", MaxSeeAlso, MaxSeeAlso)
	case limit == 0:
		limit = MaxSeeAlso
	}
	items, err := s.uc.SeeAlso(ctx, models.SeeAlsoOpts{
		ExcludeSightingID: req.GetId(),
		Limit:             limit,
		IncludeTestData:   includeTestData(ctx),
		IncludeHidden:     s.isAdmin(ctx),
	})
	if err != nil {
		return nil, internalErr(ctx, err, "не удалось получить свежие фиксации")
	}
	coords := s.canSeeCoords(ctx)
	resp := &pb.SeeAlsoResponse{Items: make([]*pb.PlateSighting, len(items))}
	for i, c := range items {
		resp.Items[i] = s.toPB(c, coords)
	}
	return resp, nil
}
