package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/GeneralFire/cn_search/be/internal/db"
	"github.com/GeneralFire/cn_search/be/internal/models"
)

const (
	apiTokensTable  = "api_tokens"
	plateViewsTable = "plate_views"
)

// apiTokenColumns — колонки api_tokens (алиас t); совпадают с тегами db models.ApiToken (без views_count).
const apiTokenColumns = `t.id, t.name, t.requests_total, t.requests_left, t.revoked, t.created_at, t.last_used_at`

// CreateApiToken сохраняет токен: остаток = всего.
func (r *repository) CreateApiToken(ctx context.Context, tx db.Executor, o models.CreateApiTokenOpts) (models.ApiToken, error) {
	var t models.ApiToken
	err := tx.Get(ctx, &t, `INSERT INTO `+apiTokensTable+` AS t (id, name, signature_hash, requests_total, requests_left)
		VALUES ($1, $2, $3, $4, $4) RETURNING `+apiTokenColumns, o.ID, o.Name, o.SignatureHash, o.Requests)
	if err != nil {
		return models.ApiToken{}, fmt.Errorf("create api token: %w", err)
	}
	return t, nil
}

// ListApiTokens — страница токенов от новых к старым, с числом открытий страниц (views_count).
func (r *repository) ListApiTokens(ctx context.Context, tx db.Executor, o models.ListApiTokensOpts) (models.ApiTokensPage, error) {
	page := models.ApiTokensPage{Items: []models.ApiToken{}}
	where := ""
	if !o.IncludeTestData {
		where = " WHERE NOT t.is_test_data"
	}
	if err := tx.Get(ctx, &page.Total, `SELECT count(*) FROM `+apiTokensTable+` t`+where); err != nil {
		return models.ApiTokensPage{}, fmt.Errorf("count api tokens: %w", err)
	}
	if page.Total == 0 || o.Offset >= page.Total {
		return page, nil
	}
	query := `SELECT ` + apiTokenColumns + `,
		(SELECT count(*) FROM ` + plateViewsTable + ` v WHERE v.token_id = t.id) AS views_count
		FROM ` + apiTokensTable + ` t` + where + ` ORDER BY t.created_at DESC, t.id LIMIT $1 OFFSET $2`
	if err := tx.Select(ctx, &page.Items, query, o.Limit, o.Offset); err != nil {
		return models.ApiTokensPage{}, fmt.Errorf("list api tokens: %w", err)
	}
	if next := o.Offset + len(page.Items); len(page.Items) > 0 && next < page.Total {
		page.NextOffset = &next
	}
	return page, nil
}

// RevokeApiToken отзывает токен (повторно — без ошибки); models.ErrNotFound, если его нет.
func (r *repository) RevokeApiToken(ctx context.Context, tx db.Executor, id string) (models.ApiToken, error) {
	var t models.ApiToken
	err := tx.Get(ctx, &t, `UPDATE `+apiTokensTable+` AS t SET revoked = true WHERE t.id = $1 RETURNING `+apiTokenColumns, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.ApiToken{}, models.ErrNotFound
	}
	if err != nil {
		return models.ApiToken{}, fmt.Errorf("revoke api token: %w", err)
	}
	return t, nil
}

// FindApiToken — токен по id и hash подписи (оба из проверенного токена); models.ErrNotFound — нет такого.
func (r *repository) FindApiToken(ctx context.Context, tx db.Executor, id, sigHash string) (models.ApiToken, error) {
	var t models.ApiToken
	err := tx.Get(ctx, &t, `SELECT `+apiTokenColumns+` FROM `+apiTokensTable+` t WHERE t.signature_hash = $1 AND t.id = $2`, sigHash, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.ApiToken{}, models.ErrNotFound
	}
	if err != nil {
		return models.ApiToken{}, fmt.Errorf("find api token: %w", err)
	}
	return t, nil
}

// ConsumeApiToken атомарно списывает один запрос: ok = false — токен отозван или запросы кончились
// (в том числе параллельным запросом). Возвращает актуальное состояние токена.
func (r *repository) ConsumeApiToken(ctx context.Context, tx db.Executor, id string) (t models.ApiToken, ok bool, err error) {
	err = tx.Get(ctx, &t, `UPDATE `+apiTokensTable+` AS t SET requests_left = requests_left - 1, last_used_at = now()
		WHERE t.id = $1 AND NOT t.revoked AND t.requests_left > 0 RETURNING `+apiTokenColumns, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.ApiToken{}, false, nil
	}
	if err != nil {
		return models.ApiToken{}, false, fmt.Errorf("consume api token: %w", err)
	}
	return t, true, nil
}

// CreatePlateView пишет открытие страницы номера в журнал; user-agent обрезается до MaxUserAgent символов.
func (r *repository) CreatePlateView(ctx context.Context, tx db.Executor, v models.NewPlateView) error {
	ua := []rune(v.UserAgent)
	if len(ua) > models.MaxUserAgent {
		ua = ua[:models.MaxUserAgent]
	}
	if _, err := tx.Exec(ctx, `INSERT INTO `+plateViewsTable+` (sighting_id, plate_id, token_id, consumed, is_admin, user_id, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`, v.SightingID, v.PlateID, v.TokenID, v.Consumed, v.IsAdmin, v.UserID, string(ua)); err != nil {
		return fmt.Errorf("create plate view: %w", err)
	}
	return nil
}

// ListPlateViews — страница журнала от новых к старым, с фиксацией и именем токена.
func (r *repository) ListPlateViews(ctx context.Context, tx db.Executor, o models.ListPlateViewsOpts) (models.PlateViewsPage, error) {
	page := models.PlateViewsPage{Items: []models.PlateView{}}
	var conds []string
	var args []any
	if o.TokenID != "" {
		args = append(args, o.TokenID)
		conds = append(conds, fmt.Sprintf("v.token_id = $%d", len(args)))
	}
	if o.PlateID != "" {
		args = append(args, o.PlateID)
		conds = append(conds, fmt.Sprintf("v.plate_id = $%d", len(args)))
	}
	where := whereClause(conds)
	if err := tx.Get(ctx, &page.Total, `SELECT count(*) FROM `+plateViewsTable+` v`+where, args...); err != nil {
		return models.PlateViewsPage{}, fmt.Errorf("count plate views: %w", err)
	}
	if page.Total == 0 || o.Offset >= page.Total {
		return page, nil
	}
	n := len(args)
	query := fmt.Sprintf(`SELECT v.id, v.created_at, v.token_id, t.name AS token_name, v.consumed, v.is_admin, v.user_agent, %s
		FROM %s v
		JOIN %s ON s.id = v.sighting_id
		LEFT JOIN %s t ON t.id = v.token_id%s
		ORDER BY v.created_at DESC, v.id LIMIT $%d OFFSET $%d`,
		complaintSightingSelect, plateViewsTable, plateSightingFrom, apiTokensTable, where, n+1, n+2)
	if err := tx.Select(ctx, &page.Items, query, append(args, o.Limit, o.Offset)...); err != nil {
		return models.PlateViewsPage{}, fmt.Errorf("list plate views: %w", err)
	}
	if next := o.Offset + len(page.Items); len(page.Items) > 0 && next < page.Total {
		page.NextOffset = &next
	}
	return page, nil
}
