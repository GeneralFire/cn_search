package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/GeneralFire/cn_search/be/internal/db"
	"github.com/GeneralFire/cn_search/be/internal/models"
)

func TestUsersSQL(t *testing.T) {
	const demoID = "00000000-0000-0000-0000-0000000a0001" // тестовый пользователь demo-token (Pro до 2030)
	r := New()
	inRollbackTx(t, func(ctx context.Context, tx db.Executor) {
		demoHash := models.TokenHash("demo-token")
		if _, err := r.UserByTokenHash(ctx, tx, demoHash, false); !errors.Is(err, models.ErrNotFound) {
			t.Errorf("тестовый без флага: %v", err)
		}
		demo, err := r.UserByTokenHash(ctx, tx, demoHash, true)
		if err != nil || demo.ID != demoID || demo.Plan != models.PlanPro || demo.Email != "demo@example.com" || demo.ProUntil == nil {
			t.Fatalf("demo: %+v %v", demo, err)
		}
		if items, err := r.ListPayments(ctx, tx, demoID, 10); err != nil || len(items) != 1 || items[0].Status != models.PaymentSucceeded || items[0].ConfirmedAt == nil {
			t.Errorf("платежи demo: %+v %v", items, err)
		}

		u, err := r.CreateUser(ctx, tx, models.TokenHash("t1"), "")
		if err != nil || u.Plan != models.PlanFree || u.Email != "" || u.ProUntil != nil || u.CreatedAt.IsZero() {
			t.Fatalf("CreateUser: %+v %v", u, err)
		}
		if _, err := r.CreateUser(ctx, tx, models.TokenHash("t2"), "Demo@Example.com"); !errors.Is(err, models.ErrEmailTaken) {
			t.Errorf("занятый email (без учёта регистра): %v", err)
		}
		if _, err := r.CreateUser(ctx, tx, models.TokenHash("t3"), ""); err != nil {
			t.Errorf("второй без email: %v", err)
		}

		p, err := r.CreatePayment(ctx, tx, models.NewPaymentOpts{UserID: u.ID, Plan: models.PlanPro, AmountCents: 29900, Currency: "RUB"})
		if err != nil || p.Status != models.PaymentPending || p.Provider != "mock" || p.ConfirmedAt != nil {
			t.Fatalf("CreatePayment: %+v %v", p, err)
		}
		if _, err := r.PaymentForUpdate(ctx, tx, demoID, p.ID); !errors.Is(err, models.ErrNotFound) {
			t.Errorf("чужой платёж: %v", err)
		}
		if got, err := r.PaymentForUpdate(ctx, tx, u.ID, p.ID); err != nil || got.ID != p.ID {
			t.Errorf("PaymentForUpdate: %+v %v", got, err)
		}
		if got, err := r.SetPaymentStatus(ctx, tx, p.ID, models.PaymentSucceeded); err != nil || got.Status != models.PaymentSucceeded || got.ConfirmedAt == nil {
			t.Errorf("SetPaymentStatus: %+v %v", got, err)
		}

		// Pro с нуля: now + 30 дней; повторно — прибавляется к текущему сроку.
		const month = 30 * 24 * time.Hour
		pro, err := r.ExtendPro(ctx, tx, u.ID, month)
		if err != nil || pro.Plan != models.PlanPro || pro.ProUntil == nil || time.Until(*pro.ProUntil) < month-time.Hour {
			t.Fatalf("ExtendPro: %+v %v", pro, err)
		}
		pro2, err := r.ExtendPro(ctx, tx, u.ID, month)
		if err != nil || pro2.ProUntil.Sub(*pro.ProUntil) != month {
			t.Errorf("продление: %v → %v %v", pro.ProUntil, pro2.ProUntil, err)
		}

		// Истёкший Pro — действующий тариф free.
		if _, err := tx.Exec(ctx, `UPDATE users SET pro_until = now() - interval '1 day' WHERE id = $1`, u.ID); err != nil {
			t.Fatal(err)
		}
		if got, err := r.UserByTokenHash(ctx, tx, models.TokenHash("t1"), false); err != nil || got.Plan != models.PlanFree {
			t.Errorf("истёкший Pro: %+v %v", got, err)
		}
		if got, err := r.ExtendPro(ctx, tx, u.ID, month); err != nil || time.Until(*got.ProUntil) < month-time.Hour {
			t.Errorf("продление истёкшего — от now: %+v %v", got, err)
		}
	})
}
