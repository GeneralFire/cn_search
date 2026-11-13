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

// Уточнения на странице номера (rpc Get): группы по предложенному номеру ко всем фиксациям номера.
// Нужна временная БД с миграциями 0001–0002 (тестовые данные) — см. TestClarifyConsensusDB.
func TestPlateClarificationsDB(t *testing.T) {
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
	uc := usecase.New(cluster, New())

	// Новый номер с двумя фиксациями: Х001ХХ 99 в разном написании предложен к обеим (3 раза),
	// Х002ХХ 99 — один раз к первой. Группы — по plate_search_key поверх всех фиксаций номера.
	var ids []string
	err = cluster.InWriteTx(ctx, func(ctx context.Context, tx db.Executor) error {
		return tx.Select(ctx, &ids, `WITH p AS (
			INSERT INTO plates (plate, meta, is_test_data) VALUES ('Е' || substr(md5(random()::text), 1, 8), '{}', true) RETURNING id)
			INSERT INTO plate_sightings (plate_id, address, created_at, is_test_data)
			SELECT id, 'тест', now() - g * interval '1 hour', true FROM p, generate_series(1, 2) g RETURNING id::text`)
	})
	if err != nil || len(ids) != 2 {
		t.Fatalf("фиксации: %v %v", ids, err)
	}
	for _, c := range []struct{ id, plate string }{
		{ids[0], "Х001ХХ 99"}, {ids[0], "Х002ХХ 99"}, {ids[1], "x001xx99"}, {ids[0], "x001xx 99"},
	} {
		if _, err := uc.Clarify(ctx, models.ClarifyOpts{SightingID: c.id, Plate: c.plate, PlateType: "ru"}); err != nil {
			t.Fatalf("%s %s: %v", c.id, c.plate, err)
		}
	}
	res, err := uc.ListSamePlate(ctx, models.SamePlateOpts{ID: ids[1], IncludeTestData: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Clarifications) != 2 {
		t.Fatalf("группы: %+v", res.Clarifications)
	}
	// Х001ХХ 99 — 3 уточнения к обеим фиксациям; принятых нет (у первой фиксации их только 2).
	if g := res.Clarifications[0]; g.Plate != "Х001ХХ 99" || g.PlateType != "ru" || g.Count != 3 ||
		g.Status != models.ClarificationNotAccepted || len(g.SightingIDs) != 2 || len(g.Items) != 3 ||
		g.Items[0].ID == "" || g.Items[0].SightingID != ids[0] || g.Items[0].Status != models.ClarificationNotAccepted ||
		g.Items[0].CreatedAt.IsZero() || g.Items[0].PlateType != "ru" || g.Items[0].CreatedAt.Before(g.Items[2].CreatedAt) {
		t.Errorf("группа 1: %+v", g)
	}
	if g := res.Clarifications[1]; g.Plate != "Х002ХХ 99" || g.Count != 1 || len(g.SightingIDs) != 1 || g.SightingIDs[0] != ids[0] {
		t.Errorf("группа 2: %+v", g)
	}

	// Тестовые данные (0002): к Т789УХ 116 (…0003) — Т789УХ 716, принято; к М456ОР 178 (…0002) — не принято.
	for id, want := range map[string]models.PlateClarification{
		"00000000-0000-0000-0000-000000000003": {Plate: "Т789УХ 716", Count: 1, Status: models.ClarificationAccepted},
		"00000000-0000-0000-0000-000000000002": {Plate: "М456ОР 198", Count: 1, Status: models.ClarificationNotAccepted},
	} {
		res, err := uc.ListSamePlate(ctx, models.SamePlateOpts{ID: id, IncludeTestData: true})
		if err != nil || len(res.Clarifications) == 0 {
			t.Errorf("%s: %+v %v", id, res, err)
			continue
		}
		if g := res.Clarifications[0]; g.Plate != want.Plate || g.Status != want.Status || g.Count < want.Count {
			t.Errorf("%s: %+v, want %+v", id, g, want)
		}
	}
	// Без include-test-data тестовые уточнения не отдаются (и сама фиксация — ErrNotFound).
	if _, err := uc.ListSamePlate(ctx, models.SamePlateOpts{ID: ids[1]}); !errors.Is(err, models.ErrNotFound) {
		t.Errorf("без тестовых: %v", err)
	}
}
