package implementation

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

// Plans — rpc Plans: тарифы (константы usecase).
func (s *CarSearch) Plans(_ context.Context, _ *pb.PlansRequest) (*pb.PlansResponse, error) {
	if s.accounts == nil {
		return nil, status.Error(codes.Unimplemented, "аккаунты не включены")
	}
	resp := &pb.PlansResponse{}
	for _, p := range s.accounts.Plans() {
		resp.Items = append(resp.Items, &pb.PlanInfo{
			Plan: planPB[p.Plan], Title: p.Title, AmountCents: int32(p.AmountCents), Currency: p.Currency,
			PeriodDays: int32(p.PeriodDays), Features: p.Features, UploadsPerDay: int32(p.UploadsPerDay),
		})
	}
	return resp, nil
}
