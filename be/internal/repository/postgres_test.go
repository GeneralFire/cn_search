package repository

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/GeneralFire/cn_search/be/internal/models"
)

// Выражения поиска в запросе и в GIN-индексах миграции должны совпадать посимвольно,
// иначе Postgres не использует индекс и идёт полным перебором.
func TestPlateSearchMatchesIndexes(t *testing.T) {
	sql, err := os.ReadFile("../../migrations/0001_init.sql")
	if err != nil {
		t.Fatal(err)
	}
	migration := string(sql)

	for _, idx := range []struct{ name, table string }{
		{"plates_plate_gin_idx", "plates"},
		{"clarifications_plate_gin_idx", "clarifications"},
	} {
		want := "CREATE INDEX " + idx.name + " ON " + idx.table + " USING gin (" + plateTSVector("plate") + ");"
		if !strings.Contains(migration, want) {
			t.Errorf("migration has no %q", want)
		}
	}
	for _, col := range []string{"p.plate", "c.plate"} {
		if !strings.Contains(plateMatch, plateTSVector(col)) {
			t.Errorf("plateMatch does not use %s", plateTSVector(col))
		}
	}
	if !strings.Contains(tsQueryFrom, "plate_search_key($1)") {
		t.Error("query string is not normalized with plate_search_key")
	}
	// Консенсус уточнений (rpc Clarify) ищет по (sighting_id, plate_search_key(plate)).
	if want := "ON clarifications (sighting_id, plate_search_key(plate));"; !strings.Contains(migration, want) ||
		!strings.Contains(clarificationsMatching, "sighting_id = $1 AND plate_search_key(plate) =") {
		t.Errorf("clarificationsMatching and index %q differ", want)
	}
	if !strings.Contains(migration, "CREATE FUNCTION plate_search_key(s text)") {
		t.Error("migration has no plate_search_key")
	}
}

// У каждого рынка есть условие, «прочее» — отрицание всех, пустой рынок — без фильтра.
func TestMarketCond(t *testing.T) {
	for _, m := range models.Markets {
		if marketCond(m) == "" {
			t.Errorf("market %q: нет условия", m)
		}
		if !strings.Contains(marketCond(models.MarketOther), marketCond(m)) {
			t.Errorf("other не исключает %q", m)
		}
	}
	if !strings.HasPrefix(marketCond(models.MarketOther), "NOT (") {
		t.Errorf("other: %s", marketCond(models.MarketOther))
	}
	if marketCond("") != "" {
		t.Error("пустой рынок фильтрует")
	}
	if want := `(coalesce(p.meta->>'type', '') = 'eu' AND coalesce(p.meta->>'region', '') = 'D')`; marketCond(models.MarketDE) != want {
		t.Errorf("de: %s", marketCond(models.MarketDE))
	}
}

// Фильтры выдачи: параметры нумеруются после уже набранных, пустые фильтры условий не добавляют.
func TestFilterConds(t *testing.T) {
	if conds, args := filterConds(models.SearchOpts{}, nil, []any{"q"}); len(conds) != 0 || len(args) != 1 {
		t.Errorf("пустые фильтры: %v %v", conds, args)
	}
	at := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	conds, args := filterConds(models.SearchOpts{Since: at, Until: at.Add(time.Hour), HasCoords: true}, []string{"x"}, []any{"q"})
	want := []string{"x", "s.created_at >= $2", "s.created_at < $3", "s.lat IS NOT NULL"}
	if strings.Join(conds, "|") != strings.Join(want, "|") || len(args) != 3 || args[1] != at {
		t.Errorf("фильтры: %v %v", conds, args)
	}
}
