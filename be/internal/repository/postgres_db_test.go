package repository

import (
	"context"
	"os"
	"testing"

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
