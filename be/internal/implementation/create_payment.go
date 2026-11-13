package implementation

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/GeneralFire/cn_search/be/internal/models"
	"github.com/GeneralFire/cn_search/be/internal/usecase"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

// mockCheckoutPath — демо-касса фронта: /pay/mock/<id платежа>.
const mockCheckoutPath = "/pay/mock/"

// CreatePayment — rpc CreatePayment: платёж pending за тариф и адрес демо-кассы.
func (s *CarSearch) CreatePayment(ctx context.Context, req *pb.CreatePaymentRequest) (*pb.CreatePaymentResponse, error) {
	u, err := s.currentUser(ctx)
	if err != nil {
		return nil, err
	}
	if req.GetPlan() != pb.UserPlan_USER_PLAN_PRO {
		return nil, status.Error(codes.InvalidArgument, "plan: купить можно только PRO")
	}
	p, err := s.accounts.CreatePayment(ctx, u.ID, models.PlanPro)
	switch {
	case errors.Is(err, usecase.ErrUnknownPlan):
		return nil, status.Error(codes.InvalidArgument, "plan: купить можно только PRO")
	case err != nil:
		return nil, status.Error(codes.Internal, "не удалось создать платёж")
	}
	return &pb.CreatePaymentResponse{PaymentId: p.ID, CheckoutUrl: mockCheckoutPath + p.ID, Payment: paymentPB(p)}, nil
}

func paymentPB(p models.Payment) *pb.Payment {
	out := &pb.Payment{
		Id: p.ID, Plan: planPB[p.Plan], AmountCents: int32(p.AmountCents), Currency: p.Currency,
		Status: paymentStatusPB[p.Status], Provider: p.Provider, CreatedAt: timestamppb.New(p.CreatedAt),
	}
	if p.ConfirmedAt != nil {
		out.ConfirmedAt = timestamppb.New(*p.ConfirmedAt)
	}
	return out
}

var paymentStatusPB = map[models.PaymentStatus]pb.PaymentStatus{
	models.PaymentPending:   pb.PaymentStatus_PAYMENT_STATUS_PENDING,
	models.PaymentSucceeded: pb.PaymentStatus_PAYMENT_STATUS_SUCCEEDED,
	models.PaymentCanceled:  pb.PaymentStatus_PAYMENT_STATUS_CANCELED,
}
