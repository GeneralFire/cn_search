package implementation

import (
	"context"

	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

// Me — rpc Me: is_admin по заголовку admin-token. Нет токена или он неверный — false, не ошибка.
func (s *CarSearch) Me(ctx context.Context, _ *pb.MeRequest) (*pb.MeResponse, error) {
	return &pb.MeResponse{IsAdmin: s.isAdmin(ctx)}, nil
}
