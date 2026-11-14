package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/GeneralFire/cn_search/be/internal/db"
	"github.com/GeneralFire/cn_search/be/internal/models"
)

// lastVisibleAt — время последней не скрытой фиксации номера с ключом plate_watch_key($2) (нет — now()).
const lastVisibleAt = `coalesce((SELECT max(s.created_at) FROM ` + plateSightingFrom + `
	WHERE plate_watch_key(p.plate) = plate_watch_key($2) AND NOT s.hidden), now())`

// CreateWatch подписывает пользователя на номер plate (строкой: в базе его может ещё не быть); last_seen_at —
// время последней видимой фиксации номера (всё, что есть сейчас, пользователь уже видел). Повторная подписка
// ничего не меняет.
func (r *repository) CreateWatch(ctx context.Context, tx db.Executor, userID, plate string) error {
	if _, err := tx.Exec(ctx, `INSERT INTO `+watchesTable+` (user_id, plate_key, plate_display, last_seen_at)
		VALUES ($1, plate_watch_key($2), $2, `+lastVisibleAt+`) ON CONFLICT DO NOTHING`, userID, plate); err != nil {
		return fmt.Errorf("create watch: %w", err)
	}
	return nil
}

// DeleteWatch снимает подписку на номер; без подписки — без ошибки.
func (r *repository) DeleteWatch(ctx context.Context, tx db.Executor, userID, plate string) error {
	if _, err := tx.Exec(ctx, `DELETE FROM `+watchesTable+` WHERE user_id = $1 AND plate_key = plate_watch_key($2)`, userID, plate); err != nil {
		return fmt.Errorf("delete watch: %w", err)
	}
	return nil
}

// ListWatches — подписки пользователя (новые подписки первыми): последняя видимая фиксация номера, если он уже
// есть, и число новых после last_seen_at. Тестовые номера и фиксации — только с IncludeTestData.
func (r *repository) ListWatches(ctx context.Context, tx db.Executor, o models.ListWatchesOpts) ([]models.Watch, error) {
	visible := `NOT s.hidden AND (NOT s.is_test_data AND NOT p.is_test_data OR $2)`
	var rows []struct {
		PlateKey     string    `db:"plate_key"`
		PlateDisplay string    `db:"plate_display"`
		NewCount     int       `db:"new_count"`
		WatchedAt    time.Time `db:"watched_at"`
	}
	err := tx.Select(ctx, &rows, `SELECT w.plate_key, w.plate_display, w.created_at AS watched_at,
		(SELECT count(*) FROM `+plateSightingFrom+`
			WHERE plate_watch_key(p.plate) = w.plate_key AND `+visible+` AND s.created_at > w.last_seen_at)::int AS new_count
		FROM `+watchesTable+` w WHERE w.user_id = $1 ORDER BY w.created_at DESC, w.plate_key`, o.UserID, o.IncludeTestData)
	if err != nil {
		return nil, fmt.Errorf("list watches: %w", err)
	}
	items := make([]models.Watch, len(rows))
	keys := make([]string, len(rows))
	for i, w := range rows {
		keys[i] = w.PlateKey
		items[i] = models.Watch{PlateKey: w.PlateKey, PlateDisplay: w.PlateDisplay, NewCount: w.NewCount, WatchedAt: w.WatchedAt}
	}
	if len(keys) == 0 {
		return items, nil
	}

	// Последняя видимая фиксация каждого номера одним запросом.
	var latest []struct {
		models.PlateSighting
		WatchKey string `db:"watch_key"`
	}
	err = tx.Select(ctx, &latest, `SELECT DISTINCT ON (plate_watch_key(p.plate)) `+plateSightingSelect+`,
		plate_watch_key(p.plate) AS watch_key
		FROM `+plateSightingFrom+` WHERE plate_watch_key(p.plate) = ANY($1) AND `+visible+`
		ORDER BY plate_watch_key(p.plate), s.created_at DESC, s.id`, keys, o.IncludeTestData)
	if err != nil {
		return nil, fmt.Errorf("list watches: latest: %w", err)
	}
	byKey := make(map[string]*models.PlateSighting, len(latest))
	for i := range latest {
		byKey[latest[i].WatchKey] = &latest[i].PlateSighting
	}
	for i := range items {
		if c := byKey[items[i].PlateKey]; c != nil {
			if c.PhotoHidden {
				c.PhotoKey, c.ThumbKey = "", ""
			}
			items[i].Latest = c
		}
	}
	return items, nil
}

// MarkWatchSeen — last_seen_at = время последней видимой фиксации номера. models.ErrNotFound — нет подписки.
func (r *repository) MarkWatchSeen(ctx context.Context, tx db.Executor, userID, plate string) error {
	var keys []string
	err := tx.Select(ctx, &keys, `UPDATE `+watchesTable+` w SET last_seen_at = `+lastVisibleAt+`
		WHERE w.user_id = $1 AND w.plate_key = plate_watch_key($2) RETURNING w.plate_key`, userID, plate)
	if err != nil {
		return fmt.Errorf("mark watch seen: %w", err)
	}
	if len(keys) == 0 {
		return models.ErrNotFound
	}
	return nil
}
