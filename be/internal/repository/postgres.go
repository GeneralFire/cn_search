// Package repository — запросы к Postgres (plain SQL). Транзакциями не управляет:
// каждый метод получает tx от вызывающего (usecase) и выполняет запросы в ней.
package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/GeneralFire/cn_search/be/internal/db"
	"github.com/GeneralFire/cn_search/be/internal/models"
)

// repository — запросы к номерам (plates) и их фиксациям (plate_sightings). Состояния нет:
// транзакцию (tx) каждый метод получает вторым аргументом от usecase.
type repository struct{}

// New возвращает репозиторий; в usecase он передаётся как интерфейс из usecase/contracts.go.
func New() *repository { return &repository{} }

// plateMatch — поиск по началу слов номера: "У54" находит "У543РУ 77", "У543РУ 7" — его же.
// Строка из запроса разбирается в SQL: слова по пробелам, в каждом остаются только буквы и цифры
// (остальное ломает синтаксис tsquery), к каждому — префикс ":*", слова через "&".
// "А12 77" → to_tsquery('simple', 'А12:* & 77:*'). Использует GIN-индекс plates_plate_gin_idx —
// выражение to_tsvector('simple', p.plate) должно совпадать с миграцией.
const plateMatch = `to_tsvector('simple', p.plate) @@ (
	SELECT to_tsquery('simple', string_agg(w || ':*', ' & '))
	FROM regexp_split_to_table(regexp_replace($1, '[^[:alnum:][:space:]]', '', 'g'), '\s+') AS w
	WHERE w <> ''
)`

// notTestData — без include-test-data строки с is_test_data не отдаются (ни фиксации, ни номера).
const notTestData = `NOT s.is_test_data AND NOT p.is_test_data`

// Search находит фиксации номеров по строке поиска, постранично, от новых к старым.
func (r *repository) Search(ctx context.Context, tx db.Executor, q models.SearchOpts) (models.SearchPage, error) {
	page := models.SearchPage{Items: []models.PlateSighting{}, Offset: q.Offset, Limit: q.Limit}

	var conds []string
	var args []any
	if q.Plate != "" {
		args = append(args, q.Plate)
		conds = append(conds, plateMatch) // $1
	}
	if !q.IncludeTestData {
		conds = append(conds, notTestData)
	}
	where := ""
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
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

// ListSamePlate возвращает все фиксации номера, к которому относится фиксация o.ID, от новых
// к старым; models.ErrNotFound, если такой фиксации нет (или она тестовая, а тестовые не нужны).
// Индекс plate_sightings_plate_id_idx.
func (r *repository) ListSamePlate(ctx context.Context, tx db.Executor, o models.SamePlateOpts) ([]models.PlateSighting, error) {
	where := `s.plate_id = (SELECT plate_id FROM ` + plateSightingsTable + ` WHERE id = $1)`
	if !o.IncludeTestData {
		where += ` AND ` + notTestData
	}
	var items []models.PlateSighting
	query := fmt.Sprintf(`SELECT %s FROM %s WHERE %s ORDER BY s.created_at DESC, s.id`,
		plateSightingSelect, plateSightingFrom, where)
	if err := tx.Select(ctx, &items, query, o.ID); err != nil {
		return nil, fmt.Errorf("list same plate: %w", err)
	}
	if len(items) == 0 {
		return nil, models.ErrNotFound
	}
	return items, nil
}
