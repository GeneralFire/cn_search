package implementation

import (
	"context"
	"strings"
	"unicode/utf8"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

// maxApiTokenRequests — верхняя граница квоты токена.
const maxApiTokenRequests = 1_000_000

// CreateApiToken — rpc CreateApiToken (админ): новый токен; сам токен — только в этом ответе.
func (s *CarSearch) CreateApiToken(ctx context.Context, req *pb.CreateApiTokenRequest) (*pb.CreateApiTokenResponse, error) {
	if err := s.requireTokensAdmin(ctx); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.GetName())
	if n := utf8.RuneCountInString(name); n < 1 || n > 100 {
		return nil, status.Error(codes.InvalidArgument, "name: от 1 до 100 символов")
	}
	if r := req.GetRequests(); r < 1 || r > maxApiTokenRequests {
		return nil, status.Errorf(codes.InvalidArgument, "requests: от 1 до %d", maxApiTokenRequests)
	}
	token, t, err := s.tokens.CreateApiToken(ctx, name, int(req.GetRequests()))
	if err != nil {
		return nil, internalErr(ctx, err, "не удалось создать токен")
	}
	return &pb.CreateApiTokenResponse{Token: token, Info: apiTokenPB(t)}, nil
}
