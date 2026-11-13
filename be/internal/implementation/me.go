package implementation

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

// Me — rpc Me: админ ли (заголовок admin-token) и пользователь по токену. Нет токена или он неверный —
// не ошибка, а authenticated = false.
func (s *CarSearch) Me(ctx context.Context, _ *pb.MeRequest) (*pb.MeResponse, error) {
	resp := &pb.MeResponse{IsAdmin: s.isAdmin(ctx)}
	if s.accounts == nil || userToken(ctx) == "" {
		return resp, nil
	}
	u, err := s.currentUser(ctx)
	switch status.Code(err) {
	case codes.OK:
		resp.Authenticated, resp.User = true, userPB(u)
	case codes.Unauthenticated:
	default:
		return nil, err
	}
	return resp, nil
}
