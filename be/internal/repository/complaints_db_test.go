package repository

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/GeneralFire/cn_search/be/internal/db"
	"github.com/GeneralFire/cn_search/be/internal/models"
)

// errRollback — вернуть из транзакции, чтобы откатить изменения теста.
var errRollback = errors.New("rollback")

// inRollbackTx выполняет fn на живой БД (TEST_DATABASE_URL, схема и тестовые данные — миграции goose)
// и откатывает транзакцию. Без TEST_DATABASE_URL тест пропускается.
func inRollbackTx(t *testing.T, fn func(ctx context.Context, tx db.Executor)) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL не задан")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	err = db.NewCluster(pool).InWriteTx(ctx, func(ctx context.Context, tx db.Executor) error {
		fn(ctx, tx)
		return errRollback
	})
	if !errors.Is(err, errRollback) {
		t.Fatal(err)
	}
}

func TestComplaintsSQL(t *testing.T) {
	const sightingID = "00000000-0000-0000-0000-000000000001" // тестовая фиксация А123ВС 777
	r := New()
	inRollbackTx(t, func(ctx context.Context, tx db.Executor) {
		// Тестовая фиксация без include-test-data — как будто её нет.
		if _, err := r.CreateComplaint(ctx, tx, models.ComplainOpts{SightingID: sightingID, Reason: models.ComplaintOther}); !errors.Is(err, models.ErrNotFound) {
			t.Errorf("тестовая фиксация без флага: %v", err)
		}
		if _, err := r.CreateComplaint(ctx, tx, models.ComplainOpts{SightingID: "00000000-0000-0000-0000-0000000fffff", Reason: models.ComplaintOther, IncludeTestData: true}); !errors.Is(err, models.ErrNotFound) {
			t.Errorf("нет фиксации: %v", err)
		}
		if _, err := r.CreateComplaint(ctx, tx, models.ComplainOpts{SightingID: sightingID, Reason: models.ComplaintOther, Comment: strings.Repeat("ж", 501), IncludeTestData: true}); err == nil {
			t.Error("CHECK на 500 символов не сработал")
		}
	})
	inRollbackTx(t, func(ctx context.Context, tx db.Executor) {
		c, err := r.CreateComplaint(ctx, tx, models.ComplainOpts{SightingID: sightingID, Reason: models.ComplaintBadPhoto, Comment: "темно", IncludeTestData: true})
		if err != nil || c.Status != models.ComplaintNew || c.Reason != models.ComplaintBadPhoto || c.ID == "" || c.CreatedAt.IsZero() {
			t.Fatalf("CreateComplaint: %+v %v", c, err)
		}
		var isTest bool
		if err := tx.Get(ctx, &isTest, `SELECT is_test_data FROM complaints WHERE id = $1`, c.ID); err != nil || !isTest {
			t.Errorf("is_test_data берётся у фиксации: %v %v", isTest, err)
		}

		// Новые: 2 из миграции + только что созданная (она первая — самая свежая по now()).
		page, err := r.ListComplaints(ctx, tx, models.ListComplaintsOpts{Status: models.ComplaintNew, Limit: 2, IncludeTestData: true})
		if err != nil || page.Total != 3 || len(page.Items) != 2 || page.NextOffset == nil || *page.NextOffset != 2 {
			t.Fatalf("ListComplaints(new): %+v %v", page, err)
		}
		first := page.Items[0]
		if first.ID != c.ID || first.Sighting.ID != sightingID || first.Sighting.Plate != "А123ВС 777" ||
			first.Sighting.ThumbKey == "" || first.Sighting.Address == "" || first.Sighting.PlateType != "ru" || first.Sighting.CreatedAt.IsZero() {
			t.Errorf("жалоба с фиксацией: %+v", first)
		}
		if all, err := r.ListComplaints(ctx, tx, models.ListComplaintsOpts{Limit: 10, IncludeTestData: true}); err != nil || all.Total != 4 || all.NextOffset != nil {
			t.Errorf("все: %+v %v", all, err)
		}
		if prod, err := r.ListComplaints(ctx, tx, models.ListComplaintsOpts{Limit: 10}); err != nil || prod.Total != 0 || len(prod.Items) != 0 {
			t.Errorf("без тестовых: %+v %v", prod, err)
		}

		res, err := r.ResolveComplaint(ctx, tx, c.ID)
		if err != nil || res.Status != models.ComplaintResolved || res.Comment != "темно" {
			t.Fatalf("ResolveComplaint: %+v %v", res, err)
		}
		if resolved, _ := r.ListComplaints(ctx, tx, models.ListComplaintsOpts{Status: models.ComplaintResolved, Limit: 10, IncludeTestData: true}); resolved.Total != 2 {
			t.Errorf("решённых: %d, want 2", resolved.Total)
		}
		if _, err := r.ResolveComplaint(ctx, tx, "00000000-0000-0000-0000-0000000fffff"); !errors.Is(err, models.ErrNotFound) {
			t.Errorf("нет жалобы: %v", err)
		}
	})
}
