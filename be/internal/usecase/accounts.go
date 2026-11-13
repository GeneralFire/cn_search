package usecase

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/GeneralFire/cn_search/be/internal/db"
	"github.com/GeneralFire/cn_search/be/internal/models"
)

// Тарифы — константы (цены и состав). Реальное ограничение одно: Free — не больше FreeUploadsPerDay
// загрузок фото в сутки (см. UploadQuotaExceeded).
const (
	FreeUploadsPerDay = 5
	ProPriceCents     = 29900
	ProCurrency       = "RUB"
	ProPeriod         = 30 * 24 * time.Hour
	tokenPrefix       = "cs_"
	maxPaymentsList   = 50
)

var plans = []models.PlanInfo{
	{Plan: models.PlanFree, Title: "Free", Currency: ProCurrency,
		Features: []string{"search", "uploads_limited"}, UploadsPerDay: FreeUploadsPerDay},
	{Plan: models.PlanPro, Title: "Pro", AmountCents: ProPriceCents, Currency: ProCurrency, PeriodDays: int(ProPeriod / (24 * time.Hour)),
		Features: []string{"search", "pro_badge", "unlimited_uploads", "notifications"}},
}

// ErrPaymentFinal — платёж уже оплачен или отменён, повторно подтвердить нельзя.
var ErrPaymentFinal = errors.New("payment already finished")

// ErrUnknownPlan — тариф нельзя купить (Free или неизвестный).
var ErrUnknownPlan = errors.New("unknown plan")

// Accounts — пользователи (вход по токену), тарифы и мок-оплата.
type Accounts struct {
	tx   transactor
	repo accountsRepository
}

func NewAccounts(tx transactor, repo accountsRepository) *Accounts {
	return &Accounts{tx: tx, repo: repo}
}

// Plans — тарифы для страницы «Тарифы».
func (u *Accounts) Plans() []models.PlanInfo { return plans }

// newToken — 32 случайных байта в base64url с префиксом "cs_".
func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("token: %w", err)
	}
	return tokenPrefix + base64.RawURLEncoding.EncodeToString(b), nil
}

// Register — новый пользователь Free; токен возвращается один раз (в базе — только его sha256).
func (u *Accounts) Register(ctx context.Context, email string) (token string, user models.User, err error) {
	if token, err = newToken(); err != nil {
		return "", models.User{}, err
	}
	err = u.tx.InWriteTx(ctx, func(ctx context.Context, tx db.Executor) error {
		user, err = u.repo.CreateUser(ctx, tx, models.TokenHash(token), email)
		return err
	})
	if err != nil {
		return "", models.User{}, err
	}
	return token, user, nil
}

// UserByToken — пользователь по токену; models.ErrNotFound — токен неверный.
func (u *Accounts) UserByToken(ctx context.Context, token string, includeTestData bool) (user models.User, err error) {
	err = u.tx.InReadTx(ctx, func(ctx context.Context, tx db.Executor) error {
		user, err = u.repo.UserByTokenHash(ctx, tx, models.TokenHash(token), includeTestData)
		return err
	})
	return user, err
}

// CreatePayment — платёж pending за тариф по цене из констант; ErrUnknownPlan — тариф не продаётся.
func (u *Accounts) CreatePayment(ctx context.Context, userID string, plan models.Plan) (p models.Payment, err error) {
	if plan != models.PlanPro {
		return models.Payment{}, ErrUnknownPlan
	}
	err = u.tx.InWriteTx(ctx, func(ctx context.Context, tx db.Executor) error {
		p, err = u.repo.CreatePayment(ctx, tx, models.NewPaymentOpts{
			UserID: userID, Plan: plan, AmountCents: ProPriceCents, Currency: ProCurrency,
		})
		return err
	})
	return p, err
}

// ConfirmMockPayment — итог демо-оплаты в одной транзакции: success — платёж «оплачен» и Pro продлевается
// на ProPeriod от max(now, pro_until); иначе «отменён». Чужой платёж — models.ErrNotFound,
// уже завершённый — ErrPaymentFinal.
func (u *Accounts) ConfirmMockPayment(ctx context.Context, userID, paymentID string, success bool) (p models.Payment, err error) {
	err = u.tx.InWriteTx(ctx, func(ctx context.Context, tx db.Executor) error {
		cur, err := u.repo.PaymentForUpdate(ctx, tx, userID, paymentID)
		if err != nil {
			return err
		}
		if cur.Status != models.PaymentPending {
			return ErrPaymentFinal
		}
		st := models.PaymentCanceled
		if success {
			st = models.PaymentSucceeded
		}
		if p, err = u.repo.SetPaymentStatus(ctx, tx, paymentID, st); err != nil {
			return err
		}
		if success {
			_, err = u.repo.ExtendPro(ctx, tx, userID, ProPeriod)
		}
		return err
	})
	return p, err
}

// ListPayments — история платежей пользователя (последние 50).
func (u *Accounts) ListPayments(ctx context.Context, userID string) (items []models.Payment, err error) {
	err = u.tx.InReadTx(ctx, func(ctx context.Context, tx db.Executor) error {
		items, err = u.repo.ListPayments(ctx, tx, userID, maxPaymentsList)
		return err
	})
	return items, err
}

// UploadQuotaExceeded — исчерпан ли суточный лимит загрузок фото: Free — не больше FreeUploadsPerDay,
// Pro — без ограничений. uploadsToday — загрузки за UploadQuotaWindow (вызывается из Ingest).
func UploadQuotaExceeded(user models.User, uploadsToday int) bool {
	return user.Plan != models.PlanPro && uploadsToday >= FreeUploadsPerDay
}
