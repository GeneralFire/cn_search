package implementation

import (
	"context"

	"github.com/GeneralFire/cn_search/be/internal/models"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

// ListApiTokens — rpc ListApiTokens (админ): токены от новых к старым.
func (s *CarSearch) ListApiTokens(ctx context.Context, req *pb.ListApiTokensRequest) (*pb.ListApiTokensResponse, error) {
	if err := s.requireTokensAdmin(ctx); err != nil {
		return nil, err
	}
	offset, limit, err := pageArgs(req.GetOffset(), req.GetLimit())
	if err != nil {
		return nil, err
	}
	page, err := s.tokens.ListApiTokens(ctx, models.ListApiTokensOpts{Offset: offset, Limit: limit, IncludeTestData: includeTestData(ctx)})
	if err != nil {
		return nil, internalErr(ctx, err, "не удалось получить токены")
	}
	resp := &pb.ListApiTokensResponse{Items: make([]*pb.ApiToken, len(page.Items)), Total: int32(page.Total), NextOffset: nextOffsetPB(page.NextOffset)}
	for i, t := range page.Items {
		resp.Items[i] = apiTokenPB(t)
	}
	return resp, nil
}
