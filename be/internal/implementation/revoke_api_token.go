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

// RevokeApiToken — rpc RevokeApiToken (админ): отозвать токен.
func (s *CarSearch) RevokeApiToken(ctx context.Context, req *pb.RevokeApiTokenRequest) (*pb.ApiToken, error) {
	if err := s.requireTokensAdmin(ctx); err != nil {
		return nil, err
	}
	if uuid.Validate(req.GetId()) != nil {
		return nil, status.Error(codes.InvalidArgument, "id: нужен uuid токена")
	}
	t, err := s.tokens.RevokeApiToken(ctx, req.GetId())
	switch {
	case errors.Is(err, models.ErrNotFound):
		return nil, status.Error(codes.NotFound, "токен не найден")
	case err != nil:
		return nil, internalErr(ctx, err, "не удалось отозвать токен")
	}
	return apiTokenPB(t), nil
}
