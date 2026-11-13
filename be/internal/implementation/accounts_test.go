package implementation

import (
	"context"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/GeneralFire/cn_search/be/internal/implementation/mocks"
	"github.com/GeneralFire/cn_search/be/internal/models"
	"github.com/GeneralFire/cn_search/be/internal/usecase"
	pb "github.com/GeneralFire/cn_search/be/pkg/carsearch"
)

var proUntil = time.Date(2026, 12, 13, 0, 0, 0, 0, time.UTC)
var demoUser = models.User{ID: "u1", Email: "demo@example.com", Plan: models.PlanPro, ProUntil: &proUntil}

func newAccountsSvc(t *testing.T) (*CarSearch, *mocks.AccountsMock) {
	acc := mocks.NewAccountsMock(t)
	return NewCarSearch(mocks.NewPlateSightingsMock(t), "/s3", WithAccounts(acc), WithComplaints(nil, "adm")), acc
}

func withMD(kv ...string) context.Context {
	return metadata.NewIncomingContext(context.Background(), metadata.Pairs(kv...))
}

func TestUserToken(t *testing.T) {
	tests := map[string]struct {
		ctx  context.Context
		want string
	}{
		"нет metadata":    {context.Background(), ""},
		"cookie":          {withMD("cookie", "theme=dark; cs_token=cs_abc"), "cs_abc"},
		"cookie важнее":   {withMD("cookie", "cs_token=c1", "authorization", "Bearer b1"), "c1"},
		"bearer":          {withMD("authorization", "Bearer b1"), "b1"},
		"bearer регистр":  {withMD("authorization", "bearer  b1 "), "b1"},
		"basic — не наше": {withMD("authorization", "Basic xxx"), ""},
		"другая cookie":   {withMD("cookie", "cs_token2=x"), ""},
	}
	for name, tt := range tests {
		if got := userToken(tt.ctx); got != tt.want {
			t.Errorf("%s: %q, want %q", name, got, tt.want)
		}
	}
}

func TestMeAccount(t *testing.T) {
	svc, acc := newAccountsSvc(t)
	acc.UserByTokenMock.Set(func(_ context.Context, token string, includeTest bool) (models.User, error) {
		if token == "demo-token" && includeTest {
			return demoUser, nil
		}
		return models.User{}, models.ErrNotFound
	})

	r, err := svc.Me(context.Background(), &pb.MeRequest{})
	if err != nil || r.Authenticated || r.IsAdmin || r.User != nil {
		t.Errorf("аноним: %+v %v", r, err)
	}
	r, err = svc.Me(withMD("cookie", "cs_token=demo-token", "include-test-data", "true", "admin-token", "adm"), &pb.MeRequest{})
	if err != nil || !r.Authenticated || !r.IsAdmin || r.User.GetId() != "u1" || r.User.GetPlan() != pb.UserPlan_USER_PLAN_PRO ||
		!r.User.GetProUntil().AsTime().Equal(proUntil) || r.User.GetEmail() != "demo@example.com" {
		t.Errorf("по cookie: %+v %v", r, err)
	}
	r, err = svc.Me(withMD("authorization", "Bearer wrong"), &pb.MeRequest{})
	if err != nil || r.Authenticated {
		t.Errorf("неверный токен: %+v %v", r, err)
	}
}

func TestRegister(t *testing.T) {
	svc, acc := newAccountsSvc(t)
	if _, err := svc.Register(context.Background(), &pb.RegisterRequest{Email: "не почта"}); status.Code(err) != codes.InvalidArgument {
		t.Errorf("плохой email: %v", err)
	}
	acc.RegisterMock.Expect(minimock.AnyContext, "a@b.ru").Return("cs_tok", models.User{ID: "u2", Plan: models.PlanFree}, nil)
	r, err := svc.Register(context.Background(), &pb.RegisterRequest{Email: " a@b.ru "})
	if err != nil || r.Token != "cs_tok" || r.User.GetPlan() != pb.UserPlan_USER_PLAN_FREE {
		t.Errorf("Register: %+v %v", r, err)
	}
}

func TestRegisterEmailTaken(t *testing.T) {
	svc, acc := newAccountsSvc(t)
	acc.RegisterMock.Return("", models.User{}, models.ErrEmailTaken)
	if _, err := svc.Register(context.Background(), &pb.RegisterRequest{Email: "a@b.ru"}); status.Code(err) != codes.AlreadyExists {
		t.Errorf("занятый email: %v", err)
	}
}

func TestCreatePayment(t *testing.T) {
	svc, acc := newAccountsSvc(t)
	if _, err := svc.CreatePayment(context.Background(), &pb.CreatePaymentRequest{Plan: pb.UserPlan_USER_PLAN_PRO}); status.Code(err) != codes.Unauthenticated {
		t.Errorf("без токена: %v", err)
	}
	acc.UserByTokenMock.Return(demoUser, nil)
	ctx := withMD("authorization", "Bearer demo-token")
	if _, err := svc.CreatePayment(ctx, &pb.CreatePaymentRequest{Plan: pb.UserPlan_USER_PLAN_FREE}); status.Code(err) != codes.InvalidArgument {
		t.Errorf("free: %v", err)
	}
	acc.CreatePaymentMock.Expect(minimock.AnyContext, "u1", models.PlanPro).
		Return(models.Payment{ID: "p1", Plan: models.PlanPro, AmountCents: 29900, Currency: "RUB", Status: models.PaymentPending}, nil)
	r, err := svc.CreatePayment(ctx, &pb.CreatePaymentRequest{Plan: pb.UserPlan_USER_PLAN_PRO})
	if err != nil || r.PaymentId != "p1" || r.CheckoutUrl != "/pay/mock/p1" || r.Payment.GetStatus() != pb.PaymentStatus_PAYMENT_STATUS_PENDING {
		t.Errorf("CreatePayment: %+v %v", r, err)
	}
}

func TestCreatePaymentWrongToken(t *testing.T) {
	svc, acc := newAccountsSvc(t)
	acc.UserByTokenMock.Return(models.User{}, models.ErrNotFound)
	if _, err := svc.CreatePayment(withMD("cookie", "cs_token=bad"), &pb.CreatePaymentRequest{Plan: pb.UserPlan_USER_PLAN_PRO}); status.Code(err) != codes.Unauthenticated {
		t.Errorf("неверный токен: %v", err)
	}
}

func TestConfirmMockPayment(t *testing.T) {
	const pid = "00000000-0000-0000-0000-0000000b0009"
	svc, acc := newAccountsSvc(t)
	acc.UserByTokenMock.Return(demoUser, nil)
	ctx := withMD("cookie", "cs_token=demo-token")
	if _, err := svc.ConfirmMockPayment(ctx, &pb.ConfirmMockPaymentRequest{PaymentId: "x"}); status.Code(err) != codes.InvalidArgument {
		t.Errorf("не uuid: %v", err)
	}
	acc.ConfirmMockPaymentMock.Set(func(_ context.Context, userID, paymentID string, success bool) (models.Payment, error) {
		switch {
		case userID != "u1" || paymentID != pid:
			return models.Payment{}, models.ErrNotFound
		case !success:
			return models.Payment{}, usecase.ErrPaymentFinal
		}
		return models.Payment{ID: pid, Status: models.PaymentSucceeded}, nil
	})
	if p, err := svc.ConfirmMockPayment(ctx, &pb.ConfirmMockPaymentRequest{PaymentId: pid, Success: true}); err != nil || p.Status != pb.PaymentStatus_PAYMENT_STATUS_SUCCEEDED {
		t.Errorf("оплата: %+v %v", p, err)
	}
	if _, err := svc.ConfirmMockPayment(ctx, &pb.ConfirmMockPaymentRequest{PaymentId: pid}); status.Code(err) != codes.FailedPrecondition {
		t.Errorf("завершённый: %v", err)
	}
	if _, err := svc.ConfirmMockPayment(ctx, &pb.ConfirmMockPaymentRequest{PaymentId: "00000000-0000-0000-0000-0000000b0001", Success: true}); status.Code(err) != codes.NotFound {
		t.Errorf("чужой: %v", err)
	}
}

func TestPlansAndPayments(t *testing.T) {
	svc, acc := newAccountsSvc(t)
	acc.PlansMock.Return([]models.PlanInfo{{Plan: models.PlanPro, AmountCents: 29900, PeriodDays: 30, Features: []string{"pro_badge"}}})
	r, err := svc.Plans(context.Background(), &pb.PlansRequest{})
	if err != nil || len(r.Items) != 1 || r.Items[0].Plan != pb.UserPlan_USER_PLAN_PRO || r.Items[0].AmountCents != 29900 {
		t.Errorf("Plans: %+v %v", r, err)
	}
	if _, err := svc.ListMyPayments(context.Background(), &pb.ListMyPaymentsRequest{}); status.Code(err) != codes.Unauthenticated {
		t.Errorf("история без токена: %v", err)
	}
	acc.UserByTokenMock.Return(demoUser, nil)
	acc.ListPaymentsMock.Expect(minimock.AnyContext, "u1").Return([]models.Payment{{ID: "p1", Status: models.PaymentCanceled}}, nil)
	h, err := svc.ListMyPayments(withMD("cookie", "cs_token=t"), &pb.ListMyPaymentsRequest{})
	if err != nil || len(h.Items) != 1 || h.Items[0].Status != pb.PaymentStatus_PAYMENT_STATUS_CANCELED {
		t.Errorf("ListMyPayments: %+v %v", h, err)
	}
}
