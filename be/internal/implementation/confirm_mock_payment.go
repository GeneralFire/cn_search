package implementation

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/GeneralFire/cn_search/be/internal/models"
	"github.com/GeneralFire/cn_search/be/internal/usecase"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

// ConfirmMockPayment — rpc ConfirmMockPayment: итог демо-оплаты своего платежа.
func (s *CarSearch) ConfirmMockPayment(ctx context.Context, req *pb.ConfirmMockPaymentRequest) (*pb.Payment, error) {
	u, err := s.currentUser(ctx)
	if err != nil {
		return nil, err
	}
	if uuid.Validate(req.GetPaymentId()) != nil {
		return nil, status.Error(codes.InvalidArgument, "payment_id: нужен uuid платежа")
	}
	p, err := s.accounts.ConfirmMockPayment(ctx, u.ID, req.GetPaymentId(), req.GetSuccess())
	switch {
	case errors.Is(err, models.ErrNotFound):
		return nil, status.Error(codes.NotFound, "платёж не найден")
	case errors.Is(err, usecase.ErrPaymentFinal):
		return nil, status.Error(codes.FailedPrecondition, "платёж уже завершён")
	case err != nil:
		return nil, internalErr(ctx, err, "не удалось провести платёж")
	}
	return paymentPB(p), nil
}
