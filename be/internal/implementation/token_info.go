package implementation

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

// TokenInfo — rpc TokenInfo: остаток API-токена запроса (панель настроек).
func (s *CarSearch) TokenInfo(ctx context.Context, _ *pb.TokenInfoRequest) (*pb.ApiToken, error) {
	if s.tokens == nil {
		return nil, status.Error(codes.Unimplemented, "API-токены не включены")
	}
	tok := apiTokenFromMD(ctx)
	if tok == "" {
		return nil, status.Error(codes.Unauthenticated, "нет API-токена")
	}
	t, err := s.tokens.TokenInfo(ctx, tok)
	if err != nil {
		if e := apiTokenErr(err); e != nil {
			return nil, e
		}
		return nil, internalErr(ctx, err, "не удалось получить токен")
	}
	return apiTokenPB(t), nil
}
