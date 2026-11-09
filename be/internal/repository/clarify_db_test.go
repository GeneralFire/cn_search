package repository

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/GeneralFire/cn_search/be/internal/db"
	"github.com/GeneralFire/cn_search/be/internal/models"
	"github.com/GeneralFire/cn_search/be/internal/usecase"
)

// Проверка SQL уточнений на живой БД. Только на отдельной временной базе с применёнными миграциями
// (тест вставляет строки): TEST_DATABASE_URL=postgres://… go test ./internal/repository/
// Без TEST_DATABASE_URL пропускается.
func TestClarifyConsensusDB(t *testing.T) {
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
	cluster := db.NewCluster(pool)

	// Новая фиксация номера "А123ВС 777" (номер берётся существующий, если есть) — уточнения
	// прошлых запусков к ней не относятся; строки помечены тестовыми.
	var sightingID string
	err = cluster.InWriteTx(ctx, func(ctx context.Context, tx db.Executor) error {
		return tx.Get(ctx, &sightingID, `WITH p AS (
			INSERT INTO plates (plate, meta, is_test_data) VALUES ('А123ВС 777', '{"type":"ru"}', true)
			ON CONFLICT (plate) DO UPDATE SET plate = EXCLUDED.plate RETURNING id)
			INSERT INTO plate_sightings (plate_id, address, is_test_data) SELECT id, 'тест', true FROM p RETURNING id`)
	})
	if err != nil {
		t.Fatal(err)
	}
	uc := usecase.New(cluster, New())
	clarify := func(plate string) (models.Clarification, error) {
		return uc.Clarify(ctx, models.ClarifyOpts{SightingID: sightingID, Plate: plate, PlateType: "ru"})
	}

	// Разное написание одного номера: plate_search_key сводит их к одному ключу.
	for i, plate := range []string{"Х001ХХ 99", "x001xx99", "Х001ХХ 99"} {
		c, err := clarify(plate)
		if err != nil {
			t.Fatalf("#%d: %v", i+1, err)
		}
		want := models.ClarificationNotAccepted
		if i+1 >= usecase.ClarificationConsensus {
			want = models.ClarificationAccepted
		}
		if c.Status != want || c.Plate != "Х001ХХ 99" {
			t.Errorf("#%d %q: %s %q, want %s", i+1, plate, c.Status, c.Plate, want)
		}
	}
	// Другой номер той же фиксации не принят.
	if c, err := clarify("Х002ХХ 99"); err != nil || c.Status != models.ClarificationNotAccepted {
		t.Errorf("другой номер: %+v %v", c, err)
	}

	var accepted int
	err = cluster.InReadTx(ctx, func(ctx context.Context, tx db.Executor) error {
		return tx.Get(ctx, &accepted, `SELECT count(*) FROM clarifications WHERE sighting_id = $1 AND status = 'accepted'`, sightingID)
	})
	if err != nil || accepted != usecase.ClarificationConsensus {
		t.Errorf("принято %d, want %d (%v)", accepted, usecase.ClarificationConsensus, err)
	}

	// Совпадает с текущим номером фиксации (латиница, без пробела) — отклоняется.
	var v *models.ValidationError
	if _, err := clarify("a123bc777"); !errors.As(err, &v) {
		t.Errorf("совпадает с текущим: %v", err)
	}

	// Нет фиксации — ErrNotFound.
	_, err = uc.Clarify(ctx, models.ClarifyOpts{SightingID: "00000000-0000-0000-0000-000000000000", Plate: "Х001ХХ 99", PlateType: "ru"})
	if !errors.Is(err, models.ErrNotFound) {
		t.Errorf("нет фиксации: %v", err)
	}
}
