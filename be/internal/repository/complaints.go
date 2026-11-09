package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/GeneralFire/cn_search/be/internal/db"
	"github.com/GeneralFire/cn_search/be/internal/models"
)

// complaintColumns — колонки complaints (с алиасом c); совпадают с тегами db models.Complaint.
const complaintColumns = `c.id, c.sighting_id, c.reason, c.comment, c.status, c.created_at`

// complaintSightingSelect — фиксация жалобы колонками "sighting.<тег db models.PlateSighting>"
// (scany раскладывает их во вложенную структуру models.ComplaintItem.Sighting).
const complaintSightingSelect = `s.id AS "sighting.id", s.plate_id AS "sighting.plate_id", p.plate AS "sighting.plate",
	s.address AS "sighting.address", s.photo_key AS "sighting.photo_key", s.photo_thumb_key AS "sighting.photo_thumb_key",
	s.created_at AS "sighting.created_at", coalesce(p.meta->>'type', '') AS "sighting.plate_type",
	coalesce(p.meta->>'region', '') AS "sighting.plate_region", s.address_locale AS "sighting.address_locale"`

// CreateComplaint сохраняет жалобу на фото фиксации (статус — «новая»); is_test_data берётся у фиксации.
// models.ErrNotFound — фиксации нет (или она тестовая, а тестовые не нужны).
func (r *repository) CreateComplaint(ctx context.Context, tx db.Executor, o models.ComplainOpts) (models.Complaint, error) {
	var c models.Complaint
	err := tx.Get(ctx, &c, `INSERT INTO `+complaintsTable+` AS c (sighting_id, reason, comment, is_test_data)
		SELECT s.id, $2, $3, s.is_test_data FROM `+plateSightingsTable+` s WHERE s.id = $1 AND (NOT s.is_test_data OR $4)
		RETURNING `+complaintColumns, o.SightingID, o.Reason, o.Comment, o.IncludeTestData)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Complaint{}, models.ErrNotFound
	}
	if err != nil {
		return models.Complaint{}, fmt.Errorf("create complaint: %w", err)
	}
	return c, nil
}

// ListComplaints — страница жалоб вместе с фиксациями, от новых к старым; Total — всего по фильтру.
// Индексы complaints_status_created_at_idx (с фильтром) и complaints_created_at_idx (все).
func (r *repository) ListComplaints(ctx context.Context, tx db.Executor, o models.ListComplaintsOpts) (models.ComplaintsPage, error) {
	page := models.ComplaintsPage{Items: []models.ComplaintItem{}}
	var conds []string
	var args []any
	if o.Status != "" {
		args = append(args, o.Status)
		conds = append(conds, fmt.Sprintf("c.status = $%d", len(args)))
	}
	if !o.IncludeTestData {
		conds = append(conds, "NOT c.is_test_data")
	}
	where := whereClause(conds)

	if err := tx.Get(ctx, &page.Total, `SELECT count(*) FROM `+complaintsTable+` c`+where, args...); err != nil {
		return models.ComplaintsPage{}, fmt.Errorf("count complaints: %w", err)
	}
	if page.Total == 0 || o.Offset >= page.Total {
		return page, nil
	}

	n := len(args)
	query := fmt.Sprintf(`SELECT %s, %s FROM %s c
		JOIN %s ON s.id = c.sighting_id%s
		ORDER BY c.created_at DESC, c.id LIMIT $%d OFFSET $%d`,
		complaintColumns, complaintSightingSelect, complaintsTable, plateSightingFrom, where, n+1, n+2)
	if err := tx.Select(ctx, &page.Items, query, append(args, o.Limit, o.Offset)...); err != nil {
		return models.ComplaintsPage{}, fmt.Errorf("list complaints: %w", err)
	}
	if next := o.Offset + len(page.Items); len(page.Items) > 0 && next < page.Total {
		page.NextOffset = &next
	}
	return page, nil
}

// ResolveComplaint отмечает жалобу решённой (повторно — без ошибки); models.ErrNotFound, если её нет.
func (r *repository) ResolveComplaint(ctx context.Context, tx db.Executor, id string) (models.Complaint, error) {
	var c models.Complaint
	err := tx.Get(ctx, &c, `UPDATE `+complaintsTable+` AS c SET status = 'resolved' WHERE c.id = $1 RETURNING `+complaintColumns, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Complaint{}, models.ErrNotFound
	}
	if err != nil {
		return models.Complaint{}, fmt.Errorf("resolve complaint: %w", err)
	}
	return c, nil
}
