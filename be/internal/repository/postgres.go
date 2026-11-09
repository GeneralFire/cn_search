// Package repository — запросы к Postgres (plain SQL). Транзакциями не управляет:
// каждый метод получает tx от вызывающего (usecase) и выполняет запросы в ней.
package repository

import (
	"context"
	"fmt"

	"github.com/GeneralFire/cn_search/be/internal/db"
	"github.com/GeneralFire/cn_search/be/internal/models"
)

// repository — запросы к номерам (plates) и их фиксациям (plate_sightings). Состояния нет:
// транзакцию (tx) каждый метод получает вторым аргументом от usecase.
type repository struct{}

// New возвращает репозиторий; в usecase он передаётся как интерфейс из usecase/contracts.go.
func New() *repository { return &repository{} }

// plateMatch: строка из запроса как есть уходит в plainto_tsquery. Использует GIN-индекс
// plates_plate_gin_idx — выражение to_tsvector('simple', p.plate) должно совпадать с миграцией.
const plateMatch = `to_tsvector('simple', p.plate) @@ plainto_tsquery('simple', $1)`

// Search находит фиксации номеров по строке поиска, постранично, от новых к старым.
func (r *repository) Search(ctx context.Context, tx db.Executor, q models.SearchOpts) (models.SearchPage, error) {
	page := models.SearchPage{Items: []models.PlateSighting{}, Offset: q.Offset, Limit: q.Limit}

	where, args := "", []any{}
	if q.Plate != "" {
		where, args = " WHERE "+plateMatch, []any{q.Plate}
	}

	if err := tx.Get(ctx, &page.Total, `SELECT count(*) FROM `+plateSightingFrom+where, args...); err != nil {
		return models.SearchPage{}, fmt.Errorf("count: %w", err)
	}
	if page.Total == 0 || q.Offset >= page.Total {
		return page, nil
	}

	n := len(args)
	query := fmt.Sprintf(`SELECT %s FROM %s%s ORDER BY s.created_at DESC, s.id LIMIT $%d OFFSET $%d`,
		plateSightingSelect, plateSightingFrom, where, n+1, n+2)
	if err := tx.Select(ctx, &page.Items, query, append(args, q.Limit, q.Offset)...); err != nil {
		return models.SearchPage{}, fmt.Errorf("search: %w", err)
	}

	if next := q.Offset + len(page.Items); len(page.Items) > 0 && next < page.Total {
		page.NextOffset = &next
	}
	return page, nil
}

// ListSamePlate возвращает все фиксации номера, к которому относится фиксация id, от новых
// к старым; models.ErrNotFound, если такой фиксации нет. Индекс plate_sightings_plate_id_idx.
func (r *repository) ListSamePlate(ctx context.Context, tx db.Executor, id string) ([]models.PlateSighting, error) {
	var items []models.PlateSighting
	query := fmt.Sprintf(`
		SELECT %s FROM %s
		WHERE s.plate_id = (SELECT plate_id FROM %s WHERE id = $1)
		ORDER BY s.created_at DESC, s.id`, plateSightingSelect, plateSightingFrom, plateSightingsTable)
	if err := tx.Select(ctx, &items, query, id); err != nil {
		return nil, fmt.Errorf("list same plate: %w", err)
	}
	if len(items) == 0 {
		return nil, models.ErrNotFound
	}
	return items, nil
}
