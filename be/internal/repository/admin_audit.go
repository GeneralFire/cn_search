package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/GeneralFire/cn_search/be/internal/db"
	"github.com/GeneralFire/cn_search/be/internal/models"
)

const adminAuditTable = "admin_audit"

// WriteAudit пишет запись журнала админа; вызывается в транзакции самого действия.
func (r *repository) WriteAudit(ctx context.Context, tx db.Executor, e models.AuditEntry) error {
	details := e.Details
	if len(details) == 0 {
		details = json.RawMessage(`{}`)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO `+adminAuditTable+` (action, actor, target_type, target_id, details, request_id)
		VALUES ($1, $2, $3, $4, $5, $6)`, e.Action, e.Actor, e.TargetType, e.TargetID, []byte(details), e.RequestID); err != nil {
		return fmt.Errorf("write audit: %w", err)
	}
	return nil
}

// AuditSnapshot — состояние объекта (JSON) до/после действия для поля details; nil — объекта нет.
func (r *repository) AuditSnapshot(ctx context.Context, tx db.Executor, targetType, id string) (json.RawMessage, error) {
	var query string
	switch targetType {
	case models.AuditTargetSighting:
		query = `SELECT jsonb_build_object('hidden', hidden, 'photo_hidden', photo_hidden) FROM ` + plateSightingsTable + ` WHERE id = $1`
	case models.AuditTargetClarification:
		query = `SELECT jsonb_build_object('status', status, 'plate', plate) FROM ` + clarificationsTable + ` WHERE id = $1`
	case models.AuditTargetComplaint:
		query = `SELECT jsonb_build_object('status', c.status, 'photo_hidden', s.photo_hidden) FROM ` + complaintsTable + ` c
			JOIN ` + plateSightingsTable + ` s ON s.id = c.sighting_id WHERE c.id = $1`
	case models.AuditTargetCameraRequest:
		query = `SELECT jsonb_build_object('status', status) FROM ` + cameraRequestsTable + ` WHERE id = $1`
	case models.AuditTargetApiToken:
		query = `SELECT jsonb_build_object('name', name, 'requests_total', requests_total, 'revoked', revoked) FROM api_tokens WHERE id = $1`
	default:
		return nil, fmt.Errorf("audit snapshot: unknown target type %q", targetType)
	}
	var out []byte
	err := tx.Get(ctx, &out, query, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("audit snapshot %s: %w", targetType, err)
	}
	return out, nil
}

// ListAudit — страница журнала, от новых к старым; Total — всего по фильтру. Для токенов подтягивает имя.
func (r *repository) ListAudit(ctx context.Context, tx db.Executor, o models.ListAuditOpts) (models.AuditPage, error) {
	page := models.AuditPage{Items: []models.AuditEntry{}}
	var conds []string
	var args []any
	if o.Action != "" {
		args = append(args, o.Action)
		conds = append(conds, fmt.Sprintf("a.action = $%d", len(args)))
	}
	if o.TargetID != "" {
		args = append(args, o.TargetID)
		conds = append(conds, fmt.Sprintf("a.target_id = $%d", len(args)))
	}
	where := whereClause(conds)
	if err := tx.Get(ctx, &page.Total, `SELECT count(*) FROM `+adminAuditTable+` a`+where, args...); err != nil {
		return models.AuditPage{}, fmt.Errorf("count audit: %w", err)
	}
	if page.Total == 0 || o.Offset >= page.Total {
		return page, nil
	}
	n := len(args)
	query := fmt.Sprintf(`SELECT a.id, a.action, a.actor, a.target_type, a.target_id, a.details, a.request_id, a.created_at,
			coalesce(t.name, '') AS target_name
		FROM %s a LEFT JOIN api_tokens t ON a.target_type = 'api_token' AND t.id::text = a.target_id%s
		ORDER BY a.created_at DESC, a.id LIMIT $%d OFFSET $%d`, adminAuditTable, where, n+1, n+2)
	if err := tx.Select(ctx, &page.Items, query, append(args, o.Limit, o.Offset)...); err != nil {
		return models.AuditPage{}, fmt.Errorf("list audit: %w", err)
	}
	if next := o.Offset + len(page.Items); len(page.Items) > 0 && next < page.Total {
		page.NextOffset = &next
	}
	return page, nil
}
