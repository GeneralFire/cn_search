package implementation

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/GeneralFire/cn_search/be/internal/models"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

// ListPlateViews — rpc ListPlateViews (админ): журнал открытий страниц номеров.
func (s *CarSearch) ListPlateViews(ctx context.Context, req *pb.ListPlateViewsRequest) (*pb.ListPlateViewsResponse, error) {
	if err := s.requireTokensAdmin(ctx); err != nil {
		return nil, err
	}
	if id := req.GetTokenId(); id != "" && uuid.Validate(id) != nil {
		return nil, status.Error(codes.InvalidArgument, "token_id: нужен uuid токена")
	}
	if id := req.GetPlateId(); id != "" && uuid.Validate(id) != nil {
		return nil, status.Error(codes.InvalidArgument, "plate_id: нужен uuid номера")
	}
	offset, limit, err := pageArgs(req.GetOffset(), req.GetLimit())
	if err != nil {
		return nil, err
	}
	page, err := s.tokens.ListPlateViews(ctx, models.ListPlateViewsOpts{
		TokenID: req.GetTokenId(), PlateID: req.GetPlateId(), Offset: offset, Limit: limit,
	})
	if err != nil {
		return nil, internalErr(ctx, err, "не удалось получить журнал")
	}
	resp := &pb.ListPlateViewsResponse{Items: make([]*pb.PlateView, len(page.Items)), Total: int32(page.Total), NextOffset: nextOffsetPB(page.NextOffset)}
	for i, v := range page.Items {
		out := &pb.PlateView{
			Id: v.ID, CreatedAt: timestamppb.New(v.CreatedAt), Sighting: s.toPB(v.Sighting, true),
			Consumed: v.Consumed, IsAdmin: v.IsAdmin, UserAgent: v.UserAgent,
		}
		if v.TokenID != nil {
			out.TokenId = *v.TokenID
		}
		if v.TokenName != nil {
			out.TokenName = *v.TokenName
		}
		resp.Items[i] = out
	}
	return resp, nil
}
