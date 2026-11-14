package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/GeneralFire/cn_search/be/internal/db"
	"github.com/GeneralFire/cn_search/be/internal/models"
)

// Админские запросы: скрытие фиксаций и ручной разбор уточнений.

// HideSighting скрывает фиксацию (hidden = true) или возвращает её; models.ErrNotFound, если её нет.
// Возвращает фиксацию с номером — как в выдаче.
func (r *repository) HideSighting(ctx context.Context, tx db.Executor, id string, hidden bool) (models.PlateSighting, error) {
	var s models.PlateSighting
	err := tx.Get(ctx, &s, `WITH s AS (
			UPDATE `+plateSightingsTable+` SET hidden = $2 WHERE id = $1 RETURNING *
		) SELECT `+plateSightingSelect+` FROM s JOIN `+platesTable+` p ON p.id = s.plate_id`, id, hidden)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.PlateSighting{}, models.ErrNotFound
	}
	if err != nil {
		return models.PlateSighting{}, fmt.Errorf("hide sighting: %w", err)
	}
	return s, nil
}

// clarificationColumnsC — колонки clarifications с алиасом c; совпадают с тегами db models.Clarification.
const clarificationColumnsC = `c.id, c.sighting_id, c.plate, c.status, c.created_at, c.plate_type, c.plate_region`

// SetClarificationStatus меняет статус уточнения; models.ErrNotFound, если его нет.
func (r *repository) SetClarificationStatus(ctx context.Context, tx db.Executor, id string, st models.ClarificationStatus) (models.Clarification, error) {
	var c models.Clarification
	err := tx.Get(ctx, &c, `UPDATE `+clarificationsTable+` AS c SET status = $2 WHERE c.id = $1 RETURNING `+clarificationColumnsC, id, st)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Clarification{}, models.ErrNotFound
	}
	if err != nil {
		return models.Clarification{}, fmt.Errorf("set clarification status: %w", err)
	}
	return c, nil
}

// ListClarifications — страница уточнений вместе с фиксациями, от новых к старым; Total — всего по фильтру.
// Индексы clarifications_status_created_at_idx (с фильтром) и clarifications_created_at_idx (все).
func (r *repository) ListClarifications(ctx context.Context, tx db.Executor, o models.ListClarificationsOpts) (models.ClarificationsPage, error) {
	page := models.ClarificationsPage{Items: []models.ClarificationItem{}}
	var conds []string
	var args []any
	if o.Status != "" {
		args = append(args, o.Status)
		conds = append(conds, fmt.Sprintf("c.status = $%d", len(args)))
	}
	if o.SightingID != "" {
		args = append(args, o.SightingID)
		conds = append(conds, fmt.Sprintf("c.sighting_id = $%d", len(args)))
	}
	if !o.IncludeTestData {
		conds = append(conds, "NOT c.is_test_data")
	}
	where := whereClause(conds)

	if err := tx.Get(ctx, &page.Total, `SELECT count(*) FROM `+clarificationsTable+` c`+where, args...); err != nil {
		return models.ClarificationsPage{}, fmt.Errorf("count clarifications: %w", err)
	}
	if page.Total == 0 || o.Offset >= page.Total {
		return page, nil
	}

	n := len(args)
	query := fmt.Sprintf(`SELECT %s, %s FROM %s c
		JOIN %s ON s.id = c.sighting_id%s
		ORDER BY c.created_at DESC, c.id LIMIT $%d OFFSET $%d`,
		clarificationColumnsC, complaintSightingSelect, clarificationsTable, plateSightingFrom, where, n+1, n+2)
	if err := tx.Select(ctx, &page.Items, query, append(args, o.Limit, o.Offset)...); err != nil {
		return models.ClarificationsPage{}, fmt.Errorf("list clarifications: %w", err)
	}
	if next := o.Offset + len(page.Items); len(page.Items) > 0 && next < page.Total {
		page.NextOffset = &next
	}
	return page, nil
}

// RestoreSightingPhoto возвращает фото, скрытое по жалобам (photo_hidden = false), и отмечает решёнными
// новые жалобы на фото этой фиксации (иначе следующая жалоба сразу скрыла бы его снова).
// models.ErrNotFound — фиксации нет.
func (r *repository) RestoreSightingPhoto(ctx context.Context, tx db.Executor, id string) (models.PlateSighting, error) {
	if _, err := tx.Exec(ctx, `UPDATE `+complaintsTable+` c SET status = 'resolved' WHERE `+photoComplaintsNew, id); err != nil {
		return models.PlateSighting{}, fmt.Errorf("resolve photo complaints: %w", err)
	}
	var s models.PlateSighting
	err := tx.Get(ctx, &s, `WITH s AS (
			UPDATE `+plateSightingsTable+` SET photo_hidden = false WHERE id = $1 RETURNING *
		) SELECT `+plateSightingSelect+` FROM s JOIN `+platesTable+` p ON p.id = s.plate_id`, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.PlateSighting{}, models.ErrNotFound
	}
	if err != nil {
		return models.PlateSighting{}, fmt.Errorf("restore photo: %w", err)
	}
	return s, nil
}
