package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/GeneralFire/cn_search/be/internal/models"
)

// PostgresStore — хранилище появлений номеров в Postgres (таблица plate_sightings).
type PostgresStore struct {
	pool *pgxpool.Pool
}

var _ Repository = (*PostgresStore)(nil)

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

// plateMatch: строка из запроса как есть уходит в plainto_tsquery. Использует GIN-индекс
// plate_sightings_plate_gin_idx — выражение to_tsvector('simple', plate) должно совпадать с миграцией.
const plateMatch = `to_tsvector('simple', plate) @@ plainto_tsquery('simple', $1)`

var selectColumns = strings.Join(plateSightingColumns, ", ")

// Search находит записи по частям номера.
func (s *PostgresStore) Search(ctx context.Context, q Query) (Page, error) {
	page := Page{Items: []models.PlateSighting{}, Offset: q.Offset, Limit: q.Limit}

	where, args := "", []any{}
	if q.Plate != "" {
		where, args = " WHERE "+plateMatch, []any{q.Plate}
	}

	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM `+plateSightingTable+where, args...).Scan(&page.Total); err != nil {
		return Page{}, fmt.Errorf("count: %w", err)
	}
	if page.Total == 0 || q.Offset >= page.Total {
		return page, nil
	}

	n := len(args)
	rows, err := s.pool.Query(ctx,
		fmt.Sprintf(`SELECT %s FROM %s%s ORDER BY created_at DESC, id LIMIT $%d OFFSET $%d`,
			selectColumns, plateSightingTable, where, n+1, n+2),
		append(args, q.Limit, q.Offset)...)
	if err != nil {
		return Page{}, fmt.Errorf("search: %w", err)
	}
	page.Items, err = collect(rows)
	if err != nil {
		return Page{}, fmt.Errorf("search: %w", err)
	}

	if next := q.Offset + len(page.Items); len(page.Items) > 0 && next < page.Total {
		page.NextOffset = &next
	}
	return page, nil
}

// Get возвращает запись по id.
func (s *PostgresStore) Get(ctx context.Context, id int) (models.PlateSighting, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+selectColumns+` FROM `+plateSightingTable+` WHERE id = $1`, id)
	if err != nil {
		return models.PlateSighting{}, fmt.Errorf("get: %w", err)
	}
	e, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[plateSightingEntity])
	if errors.Is(err, pgx.ErrNoRows) {
		return models.PlateSighting{}, ErrNotFound
	}
	if err != nil {
		return models.PlateSighting{}, fmt.Errorf("get: %w", err)
	}
	return e.toModel(), nil
}

// ReplaceAll атомарно заменяет все записи (для заливки тестовых данных).
func (s *PostgresStore) ReplaceAll(ctx context.Context, all []models.PlateSighting) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `TRUNCATE `+plateSightingTable); err != nil {
			return fmt.Errorf("truncate: %w", err)
		}
		_, err := tx.CopyFrom(ctx,
			pgx.Identifier{plateSightingTable},
			plateSightingColumns,
			pgx.CopyFromSlice(len(all), func(i int) ([]any, error) { return plateSightingValues(all[i]), nil }),
		)
		if err != nil {
			return fmt.Errorf("copy: %w", err)
		}
		return nil
	})
}

// collect читает строки plate_sightings в модели.
func collect(rows pgx.Rows) ([]models.PlateSighting, error) {
	es, err := pgx.CollectRows(rows, pgx.RowToStructByName[plateSightingEntity])
	if err != nil {
		return nil, err
	}
	out := make([]models.PlateSighting, len(es))
	for i, e := range es {
		out[i] = e.toModel()
	}
	return out, nil
}
