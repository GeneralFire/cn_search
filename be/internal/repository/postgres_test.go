package repository

import (
	"os"
	"strings"
	"testing"
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
	if !strings.Contains(migration, "CREATE FUNCTION plate_search_key(s text)") {
		t.Error("migration has no plate_search_key")
	}
}
