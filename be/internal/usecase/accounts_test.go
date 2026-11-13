package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/gojuno/minimock/v3"

	"github.com/GeneralFire/cn_search/be/internal/db"
	dbmocks "github.com/GeneralFire/cn_search/be/internal/db/mocks"
	"github.com/GeneralFire/cn_search/be/internal/models"
	"github.com/GeneralFire/cn_search/be/internal/usecase/mocks"
)

func TestRegisterHashesToken(t *testing.T) {
	tx := dbmocks.NewExecutorMock(t)
	repo := mocks.NewAccountsRepositoryMock(t)
	var gotHash string
	repo.CreateUserMock.Set(func(_ context.Context, e db.Executor, hash, email string) (models.User, error) {
		if e != tx || email != "a@b.ru" {
			t.Errorf("CreateUser: %v %q", e, email)
		}
		gotHash = hash
		return models.User{ID: "u1"}, nil
	})
	token, u, err := NewAccounts(writeTx(t, tx), repo).Register(context.Background(), "a@b.ru")
	if err != nil || u.ID != "u1" || !strings.HasPrefix(token, "cs_") || len(token) != 3+43 {
		t.Fatalf("Register: %q %+v %v", token, u, err)
	}
	if gotHash != models.TokenHash(token) || gotHash == token {
		t.Errorf("в базу — sha256 токена: %q", gotHash)
	}
}

func TestUserByTokenInReadTx(t *testing.T) {
	tx := dbmocks.NewExecutorMock(t)
	repo := mocks.NewAccountsRepositoryMock(t)
	repo.UserByTokenHashMock.Expect(minimock.AnyContext, tx, models.TokenHash("demo-token"), true).Return(models.User{ID: "u1"}, nil)
	if u, err := NewAccounts(readTx(t, tx), repo).UserByToken(context.Background(), "demo-token", true); err != nil || u.ID != "u1" {
		t.Errorf("UserByToken: %+v %v", u, err)
	}
}

func TestCreatePaymentPrice(t *testing.T) {
	tx := dbmocks.NewExecutorMock(t)
	repo := mocks.NewAccountsRepositoryMock(t)
	uc := NewAccounts(writeTx(t, tx), repo)
	if _, err := uc.CreatePayment(context.Background(), "u1", models.PlanFree); !errors.Is(err, ErrUnknownPlan) {
		t.Errorf("free: %v", err)
	}
	repo.CreatePaymentMock.Expect(minimock.AnyContext, tx, models.NewPaymentOpts{UserID: "u1", Plan: models.PlanPro, AmountCents: 29900, Currency: "RUB"}).
		Return(models.Payment{ID: "p1"}, nil)
	if p, err := uc.CreatePayment(context.Background(), "u1", models.PlanPro); err != nil || p.ID != "p1" {
		t.Errorf("pro: %+v %v", p, err)
	}
}

func TestConfirmMockPaymentSuccess(t *testing.T) {
	tx := dbmocks.NewExecutorMock(t)
	repo := mocks.NewAccountsRepositoryMock(t)
	repo.PaymentForUpdateMock.Expect(minimock.AnyContext, tx, "u1", "p1").Return(models.Payment{ID: "p1", Status: models.PaymentPending}, nil)
	repo.SetPaymentStatusMock.Expect(minimock.AnyContext, tx, "p1", models.PaymentSucceeded).Return(models.Payment{ID: "p1", Status: models.PaymentSucceeded}, nil)
	repo.ExtendProMock.Expect(minimock.AnyContext, tx, "u1", 30*24*time.Hour).Return(models.User{Plan: models.PlanPro}, nil)
	if p, err := NewAccounts(writeTx(t, tx), repo).ConfirmMockPayment(context.Background(), "u1", "p1", true); err != nil || p.Status != models.PaymentSucceeded {
		t.Errorf("оплата: %+v %v", p, err)
	}
}

func TestConfirmMockPaymentCancel(t *testing.T) {
	tx := dbmocks.NewExecutorMock(t)
	repo := mocks.NewAccountsRepositoryMock(t)
	repo.PaymentForUpdateMock.Return(models.Payment{ID: "p1", Status: models.PaymentPending}, nil)
	repo.SetPaymentStatusMock.Expect(minimock.AnyContext, tx, "p1", models.PaymentCanceled).Return(models.Payment{ID: "p1", Status: models.PaymentCanceled}, nil)
	// ExtendPro не вызывается — иначе minimock упадёт на неожиданном вызове.
	if p, err := NewAccounts(writeTx(t, tx), repo).ConfirmMockPayment(context.Background(), "u1", "p1", false); err != nil || p.Status != models.PaymentCanceled {
		t.Errorf("отмена: %+v %v", p, err)
	}
}

func TestConfirmMockPaymentFinal(t *testing.T) {
	tx := dbmocks.NewExecutorMock(t)
	repo := mocks.NewAccountsRepositoryMock(t)
	repo.PaymentForUpdateMock.Return(models.Payment{ID: "p1", Status: models.PaymentSucceeded}, nil)
	if _, err := NewAccounts(writeTx(t, tx), repo).ConfirmMockPayment(context.Background(), "u1", "p1", true); !errors.Is(err, ErrPaymentFinal) {
		t.Errorf("повтор: %v", err)
	}
}

func TestUploadQuotaExceeded(t *testing.T) {
	free, pro := models.User{Plan: models.PlanFree}, models.User{Plan: models.PlanPro}
	if UploadQuotaExceeded(free, 4) || !UploadQuotaExceeded(free, 5) || UploadQuotaExceeded(pro, 100) {
		t.Error("квота: free — 5 в сутки, pro — без ограничений")
	}
}
