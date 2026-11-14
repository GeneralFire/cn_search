package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/GeneralFire/cn_search/be/internal/db"
	"github.com/GeneralFire/cn_search/be/internal/models"
)

// Проверки SQL на живой БД с миграциями 0001–0002 (тестовые данные). Без DATABASE_URL пропускаются:
// DATABASE_URL='postgres://carsearch:…@127.0.0.1:5432/carsearch?sslmode=disable' go test ./internal/repository/
func searchDB(t *testing.T) func(q models.SearchOpts) models.SearchPage {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL не задан")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	cluster, r := db.NewCluster(pool), New()
	return func(q models.SearchOpts) models.SearchPage {
		t.Helper()
		q.IncludeTestData = true
		var page models.SearchPage
		if err := cluster.InReadTx(ctx, func(ctx context.Context, tx db.Executor) (err error) {
			page, err = r.Search(ctx, tx, q)
			return err
		}); err != nil {
			t.Fatalf("search %+v: %v", q, err)
		}
		return page
	}
}

// Рынки не пересекаются и вместе с «прочим» покрывают все номера; у каждого — свой формат.
func TestSearchMarketsDB(t *testing.T) {
	search := searchDB(t)
	all := search(models.SearchOpts{Limit: 1}).Total

	sum := 0
	for _, m := range append(models.Markets, models.MarketOther) {
		page := search(models.SearchOpts{Market: m, Limit: 100})
		if page.Total == 0 {
			t.Errorf("%s: пусто", m)
		}
		sum += page.Total
		for _, it := range page.Items {
			var ok bool
			switch m {
			case models.MarketDE:
				ok = it.PlateType == "eu" && it.PlateRegion == "D"
			case models.MarketOther:
				ok = !(it.PlateType == "eu" && it.PlateRegion == "D") && it.PlateType != "ru" &&
					it.PlateType != "ua" && it.PlateType != "us" && it.PlateType != "cn"
			default:
				ok = it.PlateType == string(m)
			}
			if !ok {
				t.Errorf("%s: чужой номер %s (%s/%s)", m, it.Plate, it.PlateType, it.PlateRegion)
			}
		}
	}
	if sum != all {
		t.Errorf("сумма по рынкам %d, всего номеров %d", sum, all)
	}
}

// Ключ поиска: точка китайских номеров — разделитель слов, украинская І = латинская I.
func TestSearchLookAlikeDB(t *testing.T) {
	search := searchDB(t)
	for q, want := range map[string]string{
		"京A·12345":   "京A·12345",
		"京a 123":     "京A·12345",
		"BC 4502 IK": "ВС 4502 ІК",
		"ах 9061 ні": "АХ 9061 НІ",
	} {
		page := search(models.SearchOpts{Plate: q, Limit: 10})
		if len(page.Items) != 1 || page.Items[0].Plate != want {
			t.Errorf("%q: %d шт., want %s", q, len(page.Items), want)
		}
	}
	if page := search(models.SearchOpts{Plate: "4502", Market: models.MarketRU, Limit: 10}); page.Total != 0 {
		t.Errorf("украинский номер в рынке ru: %+v", page.Items)
	}
}

// Фильтры периода и «есть на карте»: выдача — подмножество, время фиксаций в границах.
func TestSearchFiltersDB(t *testing.T) {
	search := searchDB(t)
	all := search(models.SearchOpts{Limit: 1}).Total
	// У тестовых фиксаций координаты обязательны (0002_test_data.sql), но другие тесты могут
	// оставить фиксации без них — проверяем, что в выдаче только фиксации с координатами.
	geo := search(models.SearchOpts{Limit: 100, HasCoords: true})
	if geo.Total == 0 || geo.Total > all {
		t.Errorf("has_coords: %d из %d", geo.Total, all)
	}
	for _, it := range geo.Items {
		if it.Lat == nil {
			t.Errorf("has_coords: %s без координат", it.Plate)
		}
	}
	if got := search(models.SearchOpts{Limit: 1, Since: time.Now().Add(time.Hour)}).Total; got != 0 {
		t.Errorf("since в будущем: %d", got)
	}
	if got := search(models.SearchOpts{Limit: 1, Until: time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)}).Total; got != 0 {
		t.Errorf("until в прошлом: %d", got)
	}
	// Делим период по времени средней фиксации: обе половины непусты и не больше всего.
	items := search(models.SearchOpts{Limit: 100}).Items
	mid := items[len(items)/2].CreatedAt
	newer := search(models.SearchOpts{Limit: 100, Since: mid})
	older := search(models.SearchOpts{Limit: 100, Until: mid})
	if newer.Total == 0 || older.Total == 0 || newer.Total > all || older.Total > all {
		t.Errorf("половины: %d + %d из %d", newer.Total, older.Total, all)
	}
	for _, it := range newer.Items {
		if it.CreatedAt.Before(mid) {
			t.Errorf("since: %s раньше %s", it.CreatedAt, mid)
		}
	}
	for _, it := range older.Items {
		if !it.CreatedAt.Before(mid) {
			t.Errorf("until: %s не раньше %s", it.CreatedAt, mid)
		}
	}
}

// Поиск по маске: «*» — один любой символ, регистр и похожие символы не важны, % и _ — не подстановки.
func TestSearchMaskDB(t *testing.T) {
	search := searchDB(t)
	plates := func(q string) []string {
		var out []string
		for _, it := range search(models.SearchOpts{Plate: q, Mask: true, Limit: 100}).Items {
			out = append(out, it.Plate)
		}
		return out
	}
	has := func(q, want string) {
		t.Helper()
		for _, p := range plates(q) {
			if p == want {
				return
			}
		}
		t.Errorf("маска %q: нет %q в %q", q, want, plates(q))
	}
	has("А*23ВС 77*", "А123ВС 777")
	has("а*23вс 77*", "А123ВС 777") // нижний регистр, латиница A/B/C
	has("A*23BC77*", "А123ВС 777")  // без пробела
	has("А123ВС 77*", "А123ВС 777") // «*» на конце
	has("*123*C*77", "А123ВС 777")  // несколько «*»

	// «*» — ровно один символ: на 2 символа не растягивается, на 0 — тоже.
	if got := plates("А*3ВС 777"); len(got) != 0 {
		t.Errorf("«*» как 0 символов: %q", got)
	}
	if got := plates("А**23ВС 777"); len(got) != 0 {
		t.Errorf("«*» как 2 символа: %q", got)
	}
	// % и _ — обычные символы, не подстановки.
	for _, q := range []string{"%123%", "_123BC777", "A123BC77_", "A%23BC777", `\123BC777`} {
		if got := plates(q); len(got) != 0 {
			t.Errorf("%q: нашлось %q", q, got)
		}
	}
	// Фильтры и total — как в обычном поиске.
	all := search(models.SearchOpts{Plate: "*******", Mask: true, Limit: 1})
	if all.Total < 2 {
		t.Fatalf("******* : %d", all.Total)
	}
	if got := search(models.SearchOpts{Plate: "*******", Mask: true, Limit: 1, Since: time.Now().Add(time.Hour)}).Total; got != 0 {
		t.Errorf("since в будущем: %d", got)
	}
	if got := search(models.SearchOpts{Plate: "*******", Mask: true, Limit: 1, Market: models.MarketCN}).Total; got == 0 || got >= all.Total {
		t.Errorf("market cn: %d из %d", got, all.Total)
	}
	if page := search(models.SearchOpts{Plate: "*******", Mask: true, Limit: 1}); page.NextOffset == nil || len(page.Items) != 1 {
		t.Errorf("пагинация: %+v", page)
	}
}
