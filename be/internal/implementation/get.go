package implementation

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/GeneralFire/cn_search/be/internal/models"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

// Get — rpc Get: все фиксации номера, к которому относится фиксация id.
func (s *CarSearch) Get(ctx context.Context, req *pb.GetRequest) (*pb.GetResponse, error) {
	if err := uuid.Validate(req.GetId()); err != nil {
		return nil, status.Error(codes.InvalidArgument, "id: нужен uuid фиксации")
	}
	res, err := s.uc.ListSamePlate(ctx, models.SamePlateOpts{ID: req.GetId(), IncludeTestData: includeTestData(ctx), IncludeHidden: s.isAdmin(ctx)})
	switch {
	case errors.Is(err, models.ErrNotFound):
		return nil, status.Error(codes.NotFound, "фиксация не найдена")
	case err != nil:
		return nil, status.Error(codes.Internal, "не удалось получить фиксации")
	}
	resp := &pb.GetResponse{
		Items:          make([]*pb.PlateSighting, len(res.Items)),
		Clarifications: make([]*pb.PlateClarification, len(res.Clarifications)),
	}
	for i, c := range res.Items {
		resp.Items[i] = s.toPB(c)
	}
	for i, c := range res.Clarifications {
		resp.Clarifications[i] = &pb.PlateClarification{
			Plate:       c.Plate,
			PlateType:   plateTypePB[c.PlateType],
			PlateRegion: c.PlateRegion,
			Count:       int32(c.Count),
			Status:      clarificationStatusPB[c.Status],
			SightingIds: c.SightingIDs,
			Items:       make([]*pb.Clarification, len(c.Items)),
		}
		for j, it := range c.Items {
			resp.Clarifications[i].Items[j] = clarificationPB(it)
		}
	}
	return resp, nil
}
