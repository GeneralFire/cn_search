package repository

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/GeneralFire/cn_search/be/internal/db"
	"github.com/GeneralFire/cn_search/be/internal/models"
)

// «Возможно, вы искали» на живой БД (TEST_DATABASE_URL): levenshtein по plate_search_key без пробелов.
func TestSuggestSQL(t *testing.T) {
	r := New()
	inRollbackTx(t, func(ctx context.Context, tx db.Executor) {
		plates := func(q models.SearchOpts) []string {
			t.Helper()
			items, err := r.Suggest(ctx, tx, q)
			if err != nil {
				t.Fatalf("Suggest %+v: %v", q, err)
			}
			out := make([]string, len(items))
			for i, it := range items {
				if it.ID == "" || it.Plate == "" {
					t.Errorf("пустая фиксация: %+v", it)
				}
				out[i] = it.Plate
			}
			return out
		}

		// Одна правка, похожие символы (латиница, О/0) и пробелы не в счёт; ближайший — первым.
		for _, q := range []string{"А123ВС 778", "a123bc778", "А123ВС77", "A 123 BC 777"} {
			got := plates(models.SearchOpts{Plate: q, IncludeTestData: true})
			if len(got) == 0 || got[0] != "А123ВС 777" || len(got) > models.SuggestLimit {
				t.Errorf("%q: %v", q, got)
			}
		}
		// Тестовые номера без include-test-data не подсказываются.
		if got := plates(models.SearchOpts{Plate: "А123ВС 778"}); slices.Contains(got, "А123ВС 777") {
			t.Errorf("тестовый номер без флага: %v", got)
		}
		// Дальше 2 правок — ничего; чужой рынок — тоже.
		if got := plates(models.SearchOpts{Plate: "А999ХХ 111", IncludeTestData: true}); slices.Contains(got, "А123ВС 777") {
			t.Errorf("3+ правок: %v", got)
		}
		if got := plates(models.SearchOpts{Plate: "А123ВС 778", Market: models.MarketUA, IncludeTestData: true}); slices.Contains(got, "А123ВС 777") {
			t.Errorf("рынок ua: %v", got)
		}
		// Фильтры выдачи действуют и на подсказки: период в будущем — ничего.
		if got := plates(models.SearchOpts{Plate: "А123ВС 778", Since: time.Now().Add(time.Hour), IncludeTestData: true}); len(got) != 0 {
			t.Errorf("since в будущем: %v", got)
		}
		if got := plates(models.SearchOpts{Plate: "А123ВС 778", HasCoords: true, IncludeTestData: true}); !slices.Contains(got, "А123ВС 777") {
			t.Errorf("has_coords: %v", got)
		}
	})
}
