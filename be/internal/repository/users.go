package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/GeneralFire/cn_search/be/internal/db"
	"github.com/GeneralFire/cn_search/be/internal/models"
)

const (
	usersTable    = "users"
	paymentsTable = "payments"
)

// userColumns — колонки users (алиас u); совпадают с тегами db models.User. plan — действующий тариф:
// Pro с истёкшим pro_until отдаётся как free.
const userColumns = `u.id, coalesce(u.email, '') AS email,
	CASE WHEN u.plan = 'pro' AND (u.pro_until IS NULL OR u.pro_until > now()) THEN 'pro' ELSE 'free' END AS plan,
	u.pro_until, u.created_at`

// paymentColumns — колонки payments (алиас pm); совпадают с тегами db models.Payment.
const paymentColumns = `pm.id, pm.user_id, pm.plan, pm.amount_cents, pm.currency, pm.status, pm.provider,
	pm.created_at, pm.confirmed_at`

// CreateUser — новый пользователь Free с хешем токена; email "" — не указан. models.ErrEmailTaken — e-mail занят.
func (r *repository) CreateUser(ctx context.Context, tx db.Executor, tokenHash, email string) (models.User, error) {
	var u models.User
	err := tx.Get(ctx, &u, `INSERT INTO `+usersTable+` AS u (token_hash, email) VALUES ($1, nullif($2, ''))
		ON CONFLICT DO NOTHING RETURNING `+userColumns, tokenHash, email)
	if errors.Is(err, pgx.ErrNoRows) {
		// Хеш случайного 32-байтового токена не совпадёт — конфликт может быть только по e-mail.
		return models.User{}, models.ErrEmailTaken
	}
	if err != nil {
		return models.User{}, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

// UserByTokenHash — пользователь по sha256 токена; models.ErrNotFound — нет такого (или он тестовый,
// а includeTestData = false).
func (r *repository) UserByTokenHash(ctx context.Context, tx db.Executor, tokenHash string, includeTestData bool) (models.User, error) {
	var u models.User
	err := tx.Get(ctx, &u, `SELECT `+userColumns+` FROM `+usersTable+` u
		WHERE u.token_hash = $1 AND (NOT u.is_test_data OR $2)`, tokenHash, includeTestData)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.User{}, models.ErrNotFound
	}
	if err != nil {
		return models.User{}, fmt.Errorf("user by token: %w", err)
	}
	return u, nil
}

// CreatePayment — платёж pending; is_test_data — как у пользователя.
func (r *repository) CreatePayment(ctx context.Context, tx db.Executor, o models.NewPaymentOpts) (models.Payment, error) {
	var p models.Payment
	err := tx.Get(ctx, &p, `INSERT INTO `+paymentsTable+` AS pm (user_id, plan, amount_cents, currency, is_test_data)
		SELECT u.id, $2, $3, $4, u.is_test_data FROM `+usersTable+` u WHERE u.id = $1
		RETURNING `+paymentColumns, o.UserID, o.Plan, o.AmountCents, o.Currency)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Payment{}, models.ErrNotFound
	}
	if err != nil {
		return models.Payment{}, fmt.Errorf("create payment: %w", err)
	}
	return p, nil
}

// PaymentForUpdate — платёж пользователя с блокировкой строки (FOR UPDATE) до конца транзакции;
// models.ErrNotFound — нет платежа или он чужой.
func (r *repository) PaymentForUpdate(ctx context.Context, tx db.Executor, userID, paymentID string) (models.Payment, error) {
	var p models.Payment
	err := tx.Get(ctx, &p, `SELECT `+paymentColumns+` FROM `+paymentsTable+` pm
		WHERE pm.id = $1 AND pm.user_id = $2 FOR UPDATE`, paymentID, userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Payment{}, models.ErrNotFound
	}
	if err != nil {
		return models.Payment{}, fmt.Errorf("payment for update: %w", err)
	}
	return p, nil
}

// SetPaymentStatus — итог платежа (succeeded / canceled), confirmed_at = now().
func (r *repository) SetPaymentStatus(ctx context.Context, tx db.Executor, paymentID string, st models.PaymentStatus) (models.Payment, error) {
	var p models.Payment
	err := tx.Get(ctx, &p, `UPDATE `+paymentsTable+` AS pm SET status = $2, confirmed_at = now()
		WHERE pm.id = $1 RETURNING `+paymentColumns, paymentID, st)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Payment{}, models.ErrNotFound
	}
	if err != nil {
		return models.Payment{}, fmt.Errorf("set payment status: %w", err)
	}
	return p, nil
}

// ExtendPro — тариф Pro: pro_until = max(now, pro_until) + period.
func (r *repository) ExtendPro(ctx context.Context, tx db.Executor, userID string, period time.Duration) (models.User, error) {
	var u models.User
	err := tx.Get(ctx, &u, `UPDATE `+usersTable+` AS u SET plan = 'pro',
		pro_until = greatest(now(), coalesce(u.pro_until, now())) + $2 * interval '1 second'
		WHERE u.id = $1 RETURNING `+userColumns, userID, period.Seconds())
	if errors.Is(err, pgx.ErrNoRows) {
		return models.User{}, models.ErrNotFound
	}
	if err != nil {
		return models.User{}, fmt.Errorf("extend pro: %w", err)
	}
	return u, nil
}

// ListPayments — платежи пользователя, от новых к старым (не больше limit).
func (r *repository) ListPayments(ctx context.Context, tx db.Executor, userID string, limit int) ([]models.Payment, error) {
	items := []models.Payment{}
	if err := tx.Select(ctx, &items, `SELECT `+paymentColumns+` FROM `+paymentsTable+` pm
		WHERE pm.user_id = $1 ORDER BY pm.created_at DESC, pm.id LIMIT $2`, userID, limit); err != nil {
		return nil, fmt.Errorf("list payments: %w", err)
	}
	return items, nil
}
