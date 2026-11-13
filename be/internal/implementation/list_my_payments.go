package implementation

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

// ListMyPayments — rpc ListMyPayments: история платежей пользователя.
func (s *CarSearch) ListMyPayments(ctx context.Context, _ *pb.ListMyPaymentsRequest) (*pb.ListMyPaymentsResponse, error) {
	u, err := s.currentUser(ctx)
	if err != nil {
		return nil, err
	}
	items, err := s.accounts.ListPayments(ctx, u.ID)
	if err != nil {
		return nil, status.Error(codes.Internal, "не удалось получить платежи")
	}
	resp := &pb.ListMyPaymentsResponse{Items: make([]*pb.Payment, len(items))}
	for i, p := range items {
		resp.Items[i] = paymentPB(p)
	}
	return resp, nil
}
