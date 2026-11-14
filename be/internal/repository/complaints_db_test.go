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

// Автоскрытие фото: 3 новые жалобы на фото → скрыто для обычных (ключи пустые), админ видит;
// решение жалобы и RestoreSightingPhoto возвращают фото.
func TestPhotoAutoHideSQL(t *testing.T) {
	const sightingID = "00000000-0000-0000-0000-000000000001" // уже есть 1 новая personal_data (0002_test_data)
	r := New()
	same := func(ctx context.Context, tx db.Executor, admin bool) models.PlateSighting {
		t.Helper()
		items, err := r.ListSamePlate(ctx, tx, models.SamePlateOpts{ID: sightingID, IncludeTestData: true, IncludeHidden: admin})
		if err != nil {
			t.Fatal(err)
		}
		for _, it := range items {
			if it.ID == sightingID {
				return it
			}
		}
		t.Fatal("фиксации нет в ListSamePlate")
		return models.PlateSighting{}
	}
	complain := func(ctx context.Context, tx db.Executor, reason models.ComplaintReason) (models.Complaint, bool) {
		t.Helper()
		c, err := r.CreateComplaint(ctx, tx, models.ComplainOpts{SightingID: sightingID, Reason: reason, IncludeTestData: true})
		if err != nil {
			t.Fatal(err)
		}
		hidden, err := r.HidePhotoIfComplained(ctx, tx, sightingID, models.PhotoHideThreshold)
		if err != nil {
			t.Fatal(err)
		}
		return c, hidden
	}
	inRollbackTx(t, func(ctx context.Context, tx db.Executor) {
		if _, hidden := complain(ctx, tx, models.ComplaintBadPhoto); hidden {
			t.Fatal("скрыто после 2 жалоб")
		}
		// wrong_plate в счёт не идёт (repository считает только причины фото).
		if _, err := r.CreateComplaint(ctx, tx, models.ComplainOpts{SightingID: sightingID, Reason: models.ComplaintWrongPlate, IncludeTestData: true}); err != nil {
			t.Fatal(err)
		}
		if hidden, _ := r.HidePhotoIfComplained(ctx, tx, sightingID, models.PhotoHideThreshold); hidden {
			t.Fatal("wrong_plate засчитана как жалоба на фото")
		}
		c3, hidden := complain(ctx, tx, models.ComplaintPersonalData)
		if !hidden {
			t.Fatal("3 жалобы на фото — фото не скрыто")
		}
		if u := same(ctx, tx, false); !u.PhotoHidden || u.PhotoKey != "" || u.ThumbKey != "" {
			t.Errorf("обычный пользователь: %+v", u)
		}
		if a := same(ctx, tx, true); !a.PhotoHidden || a.PhotoKey == "" || a.ThumbKey == "" {
			t.Errorf("админ: %+v", a)
		}
		page, err := r.Search(ctx, tx, models.SearchOpts{Plate: "А123ВС", IncludeTestData: true, Limit: 10})
		if err != nil {
			t.Fatal(err)
		}
		for _, it := range page.Items {
			if it.ID == sightingID && it.PhotoKey != "" {
				t.Errorf("Search отдал скрытое фото: %+v", it)
			}
		}
		if lc, _ := r.ListComplaints(ctx, tx, models.ListComplaintsOpts{SightingID: sightingID, Limit: 1, IncludeTestData: true}); len(lc.Items) != 1 || !lc.Items[0].Sighting.PhotoHidden || lc.Items[0].Sighting.PhotoKey == "" {
			t.Errorf("админка жалоб: %+v", lc.Items)
		}

		// Решение жалобы снимает скрытие.
		if _, err := r.ResolveComplaint(ctx, tx, c3.ID); err != nil {
			t.Fatal(err)
		}
		if err := r.ShowComplaintPhoto(ctx, tx, c3.ID); err != nil {
			t.Fatal(err)
		}
		if u := same(ctx, tx, false); u.PhotoHidden || u.PhotoKey == "" {
			t.Errorf("после resolve: %+v", u)
		}

		// Снова 3 новые → скрыто; «Вернуть фото» показывает его и решает жалобы на фото.
		if _, hidden := complain(ctx, tx, models.ComplaintBadPhoto); !hidden {
			t.Fatal("повторно не скрыто")
		}
		s, err := r.RestoreSightingPhoto(ctx, tx, sightingID)
		if err != nil || s.PhotoHidden || s.PhotoKey == "" || s.ID != sightingID {
			t.Fatalf("RestoreSightingPhoto: %+v %v", s, err)
		}
		var left int
		if err := tx.Get(ctx, &left, `SELECT count(*) FROM complaints c WHERE `+photoComplaintsNew, sightingID); err != nil || left != 0 {
			t.Errorf("новых жалоб на фото осталось %d: %v", left, err)
		}
		if _, hidden := complain(ctx, tx, models.ComplaintBadPhoto); hidden {
			t.Error("после «Вернуть фото» одна жалоба снова скрыла фото")
		}
		if _, err := r.RestoreSightingPhoto(ctx, tx, "00000000-0000-0000-0000-0000000fffff"); !errors.Is(err, models.ErrNotFound) {
			t.Errorf("нет фиксации: %v", err)
		}
	})
}
