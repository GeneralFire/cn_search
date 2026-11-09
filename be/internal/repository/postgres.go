// Package repository — запросы к Postgres (plain SQL). Транзакциями не управляет:
// каждый метод получает tx от вызывающего (usecase) и выполняет запросы в ней.
package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/GeneralFire/cn_search/be/internal/db"
	"github.com/GeneralFire/cn_search/be/internal/models"
)

// repository — запросы к номерам (plates) и их фиксациям (plate_sightings). Состояния нет:
// транзакцию (tx) каждый метод получает вторым аргументом от usecase.
type repository struct{}

// New возвращает репозиторий; в usecase он передаётся как интерфейс из usecase/contracts.go.
func New() *repository { return &repository{} }

// tsQueryFrom — tsquery «по началу слов» из сырой строки ($1): ключ plate_search_key (миграция 0001:
// похожие символы сведены к одному — О/0, кириллица/латиница, дефис/пробел), слова по пробелам, в каждом
// только буквы и цифры (остальное ломает синтаксис tsquery), к каждому ":*", слова через "&".
// "о12 77" → to_tsquery('simple', '012:* & 77:*').
const tsQueryFrom = `(
	SELECT to_tsquery('simple', string_agg(w || ':*', ' & '))
	FROM regexp_split_to_table(regexp_replace(plate_search_key($1), '[^[:alnum:][:space:]]', '', 'g'), '\s+') AS w
	WHERE w <> ''
)`

// plateTSVector — tsvector номера для поиска. Индексы plates_plate_gin_idx и clarifications_plate_gin_idx
// построены по этому же выражению (migrations/0001_init.sql) — иначе GIN не используется.
func plateTSVector(col string) string {
	return `to_tsvector('simple', plate_search_key(` + col + `))`
}

// plateMatch — фиксация подходит, если с запросом совпал её номер или номер в любом уточнении к ней.
var plateMatch = `(` + plateTSVector("p.plate") + ` @@ ` + tsQueryFrom + `
	OR EXISTS (SELECT 1 FROM ` + clarificationsTable + ` c
		WHERE c.sighting_id = s.id AND ` + plateTSVector("c.plate") + ` @@ ` + tsQueryFrom + `))`

// notTestData — без include-test-data строки с is_test_data не отдаются (ни фиксации, ни номера).
const notTestData = `NOT s.is_test_data AND NOT p.is_test_data`

// plateTypeExpr, plateRegionExpr — формат и регион номера; coalesce — чтобы условия рынков были
// true/false и без ключа в meta (иначе NULL, и NOT (…) для «прочего» терял бы такие номера).
const (
	plateTypeExpr   = `coalesce(p.meta->>'type', '')`
	plateRegionExpr = `coalesce(p.meta->>'region', '')`
)

// marketConds — условие на номер для каждого рынка (кроме «прочего» — это отрицание их всех).
var marketConds = map[models.Market]string{
	models.MarketRU: plateTypeExpr + ` = 'ru'`,
	models.MarketUA: plateTypeExpr + ` = 'ua'`,
	models.MarketDE: `(` + plateTypeExpr + ` = 'eu' AND ` + plateRegionExpr + ` = 'D')`,
	models.MarketUS: plateTypeExpr + ` = 'us'`,
	models.MarketCN: plateTypeExpr + ` = 'cn'`,
}

// marketCond — условие WHERE для рынка m; "" — без фильтра (пустой или неизвестный рынок).
func marketCond(m models.Market) string {
	if m != models.MarketOther {
		return marketConds[m]
	}
	known := make([]string, len(models.Markets))
	for i, k := range models.Markets {
		known[i] = marketConds[k]
	}
	return `NOT (` + strings.Join(known, " OR ") + `)`
}

// latestPerPlate — по одной фиксации на номер (последней из подходящих), от новых к старым.
func latestPerPlate(where string) string {
	return fmt.Sprintf(`SELECT * FROM (
		SELECT DISTINCT ON (s.plate_id) %s FROM %s%s
		ORDER BY s.plate_id, s.created_at DESC, s.id
	) latest ORDER BY created_at DESC, id`, plateSightingSelect, plateSightingFrom, where)
}

func whereClause(conds []string) string {
	if len(conds) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(conds, " AND ")
}

// Search находит номера по строке поиска (сам номер или уточнения к его фиксациям): один номер —
// одна строка, последняя подходящая фиксация; постранично, от новых к старым. Total — число номеров.
func (r *repository) Search(ctx context.Context, tx db.Executor, q models.SearchOpts) (models.SearchPage, error) {
	page := models.SearchPage{Items: []models.PlateSighting{}, Offset: q.Offset, Limit: q.Limit}

	var conds []string
	var args []any
	if q.Plate != "" {
		args = append(args, q.Plate)
		conds = append(conds, plateMatch) // $1
	}
	if c := marketCond(q.Market); c != "" {
		conds = append(conds, c)
	}
	if !q.IncludeTestData {
		conds = append(conds, notTestData)
	}
	where := whereClause(conds)

	if err := tx.Get(ctx, &page.Total, `SELECT count(DISTINCT s.plate_id) FROM `+plateSightingFrom+where, args...); err != nil {
		return models.SearchPage{}, fmt.Errorf("count: %w", err)
	}
	if page.Total == 0 || q.Offset >= page.Total {
		return page, nil
	}

	n := len(args)
	query := latestPerPlate(where) + fmt.Sprintf(` LIMIT $%d OFFSET $%d`, n+1, n+2)
	if err := tx.Select(ctx, &page.Items, query, append(args, q.Limit, q.Offset)...); err != nil {
		return models.SearchPage{}, fmt.Errorf("search: %w", err)
	}

	if next := q.Offset + len(page.Items); len(page.Items) > 0 && next < page.Total {
		page.NextOffset = &next
	}
	return page, nil
}

// SeeAlso — свежие фиксации других номеров: по одной на номер, номер фиксации o.ExcludeSightingID исключён.
func (r *repository) SeeAlso(ctx context.Context, tx db.Executor, o models.SeeAlsoOpts) ([]models.PlateSighting, error) {
	var conds []string
	args := []any{o.Limit}
	if o.ExcludeSightingID != "" {
		args = append(args, o.ExcludeSightingID)
		conds = append(conds, `s.plate_id <> (SELECT plate_id FROM `+plateSightingsTable+` WHERE id = $2)`)
	}
	if !o.IncludeTestData {
		conds = append(conds, notTestData)
	}
	items := []models.PlateSighting{}
	if err := tx.Select(ctx, &items, latestPerPlate(whereClause(conds))+` LIMIT $1`, args...); err != nil {
		return nil, fmt.Errorf("see also: %w", err)
	}
	return items, nil
}

// foreignKeyViolation — SQLSTATE 23503: ссылка на несуществующую строку (здесь — фиксацию).
const foreignKeyViolation = "23503"

// CreateClarification сохраняет уточнение (статус — «не принят»); models.ErrNotFound, если фиксации нет.
func (r *repository) CreateClarification(ctx context.Context, tx db.Executor, o models.ClarifyOpts) (models.Clarification, error) {
	var c models.Clarification
	err := tx.Get(ctx, &c, `INSERT INTO `+clarificationsTable+` (sighting_id, plate, plate_type, plate_region)
		VALUES ($1, $2, $3, $4) RETURNING `+clarificationColumns, o.SightingID, o.Plate, o.PlateType, o.PlateRegion)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == foreignKeyViolation {
		return models.Clarification{}, models.ErrNotFound
	}
	if err != nil {
		return models.Clarification{}, fmt.Errorf("create clarification: %w", err)
	}
	return c, nil
}

// MatchesSightingPlate — совпадает ли plate с номером фиксации sightingID (по plate_search_key).
// Блокирует строку фиксации (FOR UPDATE) до конца транзакции — уточнения к одной фиксации
// записываются по очереди. models.ErrNotFound, если фиксации нет.
func (r *repository) MatchesSightingPlate(ctx context.Context, tx db.Executor, sightingID, plate string) (bool, error) {
	var same []bool
	err := tx.Select(ctx, &same, `SELECT plate_search_key(p.plate) = plate_search_key($2)
		FROM `+plateSightingFrom+` WHERE s.id = $1 FOR UPDATE OF s`, sightingID, plate)
	if err != nil {
		return false, fmt.Errorf("matches sighting plate: %w", err)
	}
	if len(same) == 0 {
		return false, models.ErrNotFound
	}
	return same[0], nil
}

// clarificationsMatching — уточнения фиксации $1 с тем же номером, что $2 (по plate_search_key).
// Индекс clarifications_sighting_plate_key_idx — выражение должно совпадать с миграцией.
const clarificationsMatching = `sighting_id = $1 AND plate_search_key(plate) = plate_search_key($2)`

// CountMatchingClarifications — сколько уточнений фиксации совпадают с plate (по plate_search_key).
func (r *repository) CountMatchingClarifications(ctx context.Context, tx db.Executor, sightingID, plate string) (int, error) {
	var n int
	if err := tx.Get(ctx, &n, `SELECT count(*) FROM `+clarificationsTable+` WHERE `+clarificationsMatching, sightingID, plate); err != nil {
		return 0, fmt.Errorf("count matching clarifications: %w", err)
	}
	return n, nil
}

// AcceptMatchingClarifications принимает все уточнения фиксации, совпадающие с plate (по plate_search_key).
func (r *repository) AcceptMatchingClarifications(ctx context.Context, tx db.Executor, sightingID, plate string) error {
	if _, err := tx.Exec(ctx, `UPDATE `+clarificationsTable+` SET status = 'accepted'
		WHERE `+clarificationsMatching+` AND status <> 'accepted'`, sightingID, plate); err != nil {
		return fmt.Errorf("accept matching clarifications: %w", err)
	}
	return nil
}

// ListSamePlate возвращает все фиксации номера, к которому относится фиксация o.ID, от новых
// к старым; models.ErrNotFound, если такой фиксации нет (или она тестовая, а тестовые не нужны).
// Индекс plate_sightings_plate_id_idx.
func (r *repository) ListSamePlate(ctx context.Context, tx db.Executor, o models.SamePlateOpts) ([]models.PlateSighting, error) {
	conds := []string{`s.plate_id = (SELECT plate_id FROM ` + plateSightingsTable + ` WHERE id = $1)`}
	if !o.IncludeTestData {
		conds = append(conds, notTestData)
	}
	where := strings.Join(conds, " AND ")
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
