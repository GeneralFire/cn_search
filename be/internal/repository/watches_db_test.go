package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/GeneralFire/cn_search/be/internal/db"
	"github.com/GeneralFire/cn_search/be/internal/models"
)

func TestWatchesSQL(t *testing.T) {
	r := New()
	inRollbackTx(t, func(ctx context.Context, tx db.Executor) {
		u, err := r.CreateUser(ctx, tx, models.TokenHash("watch-test"), "")
		if err != nil {
			t.Fatal(err)
		}
		list := func(test bool) []models.Watch {
			t.Helper()
			items, err := r.ListWatches(ctx, tx, models.ListWatchesOpts{UserID: u.ID, IncludeTestData: test})
			if err != nil {
				t.Fatal(err)
			}
			return items
		}

		// Тестовый номер (А123ВС 777) и номер, которого нет в базе. Латиница/регистр/пробелы — тот же ключ.
		for _, plate := range []string{"a123bc777", "A123BC777", "Z999ZZ 99"} {
			if err := r.CreateWatch(ctx, tx, u.ID, plate); err != nil { // повторы — без ошибки
				t.Fatal(err)
			}
		}
		got := list(true)
		if len(got) != 2 {
			t.Fatalf("две подписки (повтор схлопнулся): %+v", got)
		}
		var known, waiting models.Watch
		for _, w := range got {
			if w.Latest != nil {
				known = w
			} else {
				waiting = w
			}
		}
		if known.Latest == nil || known.Latest.Plate != "А123ВС 777" || known.NewCount != 0 || known.WatchedAt.IsZero() {
			t.Fatalf("известный номер: %+v", known)
		}
		if waiting.PlateDisplay != "Z999ZZ 99" || waiting.NewCount != 0 {
			t.Fatalf("ожидание: %+v", waiting)
		}
		// Без include-test-data тестовый номер для подписчика — как будто его нет.
		for _, w := range list(false) {
			if w.Latest != nil || w.NewCount != 0 {
				t.Fatalf("без тестовых данных: %+v", w)
			}
		}

		// Новая фиксация известного номера → +1 (скрытая не считается); номер из ожидания появился → +1 и Latest.
		ins := `INSERT INTO plates (plate, is_test_data) VALUES ('Z999ZZ 99', true) ON CONFLICT DO NOTHING`
		if _, err := tx.Exec(ctx, ins); err != nil {
			t.Fatal(err)
		}
		for _, hidden := range []bool{false, true} {
			if _, err := tx.Exec(ctx, `INSERT INTO plate_sightings (plate_id, address, created_at, is_test_data, hidden)
				SELECT id, 'Новый адрес', now() + interval '1 hour', true, $2 FROM plates WHERE plate_watch_key(plate) = plate_watch_key($1)`,
				"A123BC777", hidden); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := tx.Exec(ctx, `INSERT INTO plate_sightings (plate_id, address, created_at, is_test_data)
			SELECT id, 'Ожидание', now() + interval '1 hour', true FROM plates WHERE plate = 'Z999ZZ 99'`); err != nil {
			t.Fatal(err)
		}
		for _, w := range list(true) {
			if w.NewCount != 1 || w.Latest == nil {
				t.Fatalf("после новых фиксаций: %+v", w)
			}
		}
		if err := r.MarkWatchSeen(ctx, tx, u.ID, "Z999ZZ 99"); err != nil {
			t.Fatal(err)
		}
		for _, w := range list(true) {
			if w.PlateDisplay == "Z999ZZ 99" && w.NewCount != 0 {
				t.Fatalf("после просмотра: %+v", w)
			}
		}
		if err := r.DeleteWatch(ctx, tx, u.ID, "z999zz99"); err != nil {
			t.Fatal(err)
		}
		if err := r.DeleteWatch(ctx, tx, u.ID, "A123BC777"); err != nil {
			t.Fatal(err)
		}
		if got := list(true); len(got) != 0 {
			t.Fatalf("после отписки: %+v", got)
		}
		if err := r.MarkWatchSeen(ctx, tx, u.ID, "A123BC777"); !errors.Is(err, models.ErrNotFound) {
			t.Fatalf("просмотр без подписки: %v", err)
		}
	})
}
