package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/GeneralFire/cn_search/be/internal/db"
	"github.com/GeneralFire/cn_search/be/internal/models"
)

// cameraRequestColumns — колонки camera_requests; совпадают с тегами db models.CameraRequest.
const cameraRequestColumns = `id, lat, lon, address, reason, contact, status, votes, created_at`

// CreateCameraRequest сохраняет заявку на камеру (статус — «новая», голосов 0).
func (r *repository) CreateCameraRequest(ctx context.Context, tx db.Executor, o models.SubmitCameraRequestOpts) (models.CameraRequest, error) {
	var c models.CameraRequest
	err := tx.Get(ctx, &c, `INSERT INTO `+cameraRequestsTable+` (lat, lon, address, reason, contact)
		VALUES ($1, $2, $3, $4, $5) RETURNING `+cameraRequestColumns, o.Lat, o.Lon, o.Address, o.Reason, o.Contact)
	if err != nil {
		return models.CameraRequest{}, fmt.Errorf("create camera request: %w", err)
	}
	return c, nil
}

// ListCameraRequests — страница заявок, от новых к старым; Total — всего по фильтру.
func (r *repository) ListCameraRequests(ctx context.Context, tx db.Executor, o models.ListCameraRequestsOpts) (models.CameraRequestsPage, error) {
	page := models.CameraRequestsPage{Items: []models.CameraRequest{}}
	var conds []string
	var args []any
	if o.Status != "" {
		args = append(args, o.Status)
		conds = append(conds, fmt.Sprintf("status = $%d", len(args)))
	}
	if b := o.BBox; b != nil {
		args = append(args, b.MinLat, b.MaxLat, b.MinLon, b.MaxLon)
		n := len(args)
		conds = append(conds, fmt.Sprintf("lat BETWEEN $%d AND $%d AND lon BETWEEN $%d AND $%d", n-3, n-2, n-1, n))
	}
	if !o.IncludeTestData {
		conds = append(conds, "NOT is_test_data")
	}
	where := whereClause(conds)

	if err := tx.Get(ctx, &page.Total, `SELECT count(*) FROM `+cameraRequestsTable+where, args...); err != nil {
		return models.CameraRequestsPage{}, fmt.Errorf("count camera requests: %w", err)
	}
	if page.Total == 0 || o.Offset >= page.Total {
		return page, nil
	}
	n := len(args)
	query := fmt.Sprintf(`SELECT %s FROM %s%s ORDER BY created_at DESC, id LIMIT $%d OFFSET $%d`,
		cameraRequestColumns, cameraRequestsTable, where, n+1, n+2)
	if err := tx.Select(ctx, &page.Items, query, append(args, o.Limit, o.Offset)...); err != nil {
		return models.CameraRequestsPage{}, fmt.Errorf("list camera requests: %w", err)
	}
	if next := o.Offset + len(page.Items); len(page.Items) > 0 && next < page.Total {
		page.NextOffset = &next
	}
	return page, nil
}

// VoteCameraRequest — голос +1; models.ErrNotFound — заявки нет (или она тестовая, а тестовые не нужны).
func (r *repository) VoteCameraRequest(ctx context.Context, tx db.Executor, o models.VoteCameraRequestOpts) (models.CameraRequest, error) {
	var c models.CameraRequest
	err := tx.Get(ctx, &c, `UPDATE `+cameraRequestsTable+` SET votes = votes + 1
		WHERE id = $1 AND (NOT is_test_data OR $2) RETURNING `+cameraRequestColumns, o.ID, o.IncludeTestData)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.CameraRequest{}, models.ErrNotFound
	}
	if err != nil {
		return models.CameraRequest{}, fmt.Errorf("vote camera request: %w", err)
	}
	return c, nil
}

// SetCameraRequestStatus меняет статус заявки; models.ErrNotFound, если её нет.
func (r *repository) SetCameraRequestStatus(ctx context.Context, tx db.Executor, id string, st models.CameraRequestStatus) (models.CameraRequest, error) {
	var c models.CameraRequest
	err := tx.Get(ctx, &c, `UPDATE `+cameraRequestsTable+` SET status = $2 WHERE id = $1 RETURNING `+cameraRequestColumns, id, st)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.CameraRequest{}, models.ErrNotFound
	}
	if err != nil {
		return models.CameraRequest{}, fmt.Errorf("set camera request status: %w", err)
	}
	return c, nil
}
