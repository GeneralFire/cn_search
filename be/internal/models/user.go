package models

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"
)

// ErrEmailTaken — e-mail уже занят другим пользователем.
var ErrEmailTaken = errors.New("email taken")

// Plan — тариф пользователя (users.plan, payments.plan).
type Plan string

const (
	PlanFree Plan = "free"
	PlanPro  Plan = "pro"
)

// User — пользователь (таблица users). Plan — действующий тариф: Pro с истёкшим pro_until — уже Free
// (считает repository).
type User struct {
	ID        string     `db:"id"`
	Email     string     `db:"email"` // "" — не указан
	Plan      Plan       `db:"plan"`
	ProUntil  *time.Time `db:"pro_until"`
	CreatedAt time.Time  `db:"created_at"`
}

// PaymentStatus — статус платежа (payments.status).
type PaymentStatus string

const (
	PaymentPending   PaymentStatus = "pending"
	PaymentSucceeded PaymentStatus = "succeeded"
	PaymentCanceled  PaymentStatus = "canceled"
)

// Payment — платёж за тариф (таблица payments).
type Payment struct {
	ID          string        `db:"id"`
	UserID      string        `db:"user_id"`
	Plan        Plan          `db:"plan"`
	AmountCents int           `db:"amount_cents"`
	Currency    string        `db:"currency"`
	Status      PaymentStatus `db:"status"`
	Provider    string        `db:"provider"`
	CreatedAt   time.Time     `db:"created_at"`
	ConfirmedAt *time.Time    `db:"confirmed_at"`
}

// PlanInfo — тариф для страницы «Тарифы»: цена и состав (константы — usecase.Plans).
type PlanInfo struct {
	Plan          Plan
	Title         string
	AmountCents   int
	Currency      string
	PeriodDays    int      // 0 — бессрочно (Free)
	Features      []string // ключи возможностей: "pro_badge", "unlimited_uploads", "notifications", …
	UploadsPerDay int      // лимит загрузок фото в сутки; 0 — без ограничений
}

// NewPaymentOpts — новый платёж (статус pending).
type NewPaymentOpts struct {
	UserID      string
	Plan        Plan
	AmountCents int
	Currency    string
}

// TokenHash — sha256 токена в hex: так токен хранится в users.token_hash.
func TokenHash(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
