package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/GeneralFire/cn_search/be/internal/models"
)

// PostgresStore — хранилище номеров (plates) и их фиксаций (plate_sightings) в Postgres.
// Запросы — plain SQL через Select/Get/Exec (executor).
type PostgresStore struct {
	executor
	pool *pgxpool.Pool
}

var _ Repository = (*PostgresStore)(nil)

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{executor: executor{q: pool}, pool: pool}
}

// inTx выполняет fn в транзакции; ex — executor поверх неё.
func (s *PostgresStore) inTx(ctx context.Context, fn func(tx pgx.Tx, ex executor) error) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error { return fn(tx, executor{q: tx}) })
}

// plateMatch: строка из запроса как есть уходит в plainto_tsquery. Использует GIN-индекс
// plates_plate_gin_idx — выражение to_tsvector('simple', p.plate) должно совпадать с миграцией.
const plateMatch = `to_tsvector('simple', p.plate) @@ plainto_tsquery('simple', $1)`

// Search находит фиксации номеров по строке поиска.
func (s *PostgresStore) Search(ctx context.Context, q Query) (Page, error) {
	page := Page{Items: []models.PlateSighting{}, Offset: q.Offset, Limit: q.Limit}

	where, args := "", []any{}
	if q.Plate != "" {
		where, args = " WHERE "+plateMatch, []any{q.Plate}
	}

	if err := s.Get(ctx, &page.Total, `SELECT count(*) FROM `+plateSightingFrom+where, args...); err != nil {
		return Page{}, fmt.Errorf("count: %w", err)
	}
	if page.Total == 0 || q.Offset >= page.Total {
		return page, nil
	}

	n := len(args)
	query := fmt.Sprintf(`SELECT %s FROM %s%s ORDER BY s.created_at DESC, s.id LIMIT $%d OFFSET $%d`,
		plateSightingSelect, plateSightingFrom, where, n+1, n+2)
	if err := s.Select(ctx, &page.Items, query, append(args, q.Limit, q.Offset)...); err != nil {
		return Page{}, fmt.Errorf("search: %w", err)
	}

	if next := q.Offset + len(page.Items); len(page.Items) > 0 && next < page.Total {
		page.NextOffset = &next
	}
	return page, nil
}

// ListSamePlate возвращает все фиксации номера, к которому относится фиксация id.
// Индекс plate_sightings_plate_id_idx.
func (s *PostgresStore) ListSamePlate(ctx context.Context, id int) ([]models.PlateSighting, error) {
	var items []models.PlateSighting
	query := fmt.Sprintf(`
		SELECT %s FROM %s
		WHERE s.plate_id = (SELECT plate_id FROM %s WHERE id = $1)
		ORDER BY s.created_at DESC, s.id`, plateSightingSelect, plateSightingFrom, plateSightingsTable)
	if err := s.Select(ctx, &items, query, id); err != nil {
		return nil, fmt.Errorf("list same plate: %w", err)
	}
	if len(items) == 0 {
		return nil, ErrNotFound
	}
	return items, nil
}

// ReplaceAll атомарно заменяет все номера и фиксации (для заливки тестовых данных).
// Номера берутся из фиксаций (PlateSighting.Plate), их id генерирует БД.
func (s *PostgresStore) ReplaceAll(ctx context.Context, all []models.PlateSighting) error {
	return s.inTx(ctx, func(tx pgx.Tx, ex executor) error {
		if _, err := ex.Exec(ctx, `TRUNCATE `+plateSightingsTable+`, `+platesTable); err != nil {
			return fmt.Errorf("truncate: %w", err)
		}

		var names []string
		seen := map[string]bool{}
		for _, c := range all {
			if !seen[c.Plate] {
				seen[c.Plate] = true
				names = append(names, c.Plate)
			}
		}
		var plates []models.Plate
		if err := ex.Select(ctx, &plates,
			`INSERT INTO `+platesTable+` (plate) SELECT unnest($1::text[]) RETURNING id, plate, meta`, names); err != nil {
			return fmt.Errorf("insert plates: %w", err)
		}
		idOf := make(map[string]string, len(plates))
		for _, p := range plates {
			idOf[p.Plate] = p.ID
		}

		// COPY — самый быстрый способ залить много строк; через Select/Exec его не сделать.
		_, err := tx.CopyFrom(ctx,
			pgx.Identifier{plateSightingsTable},
			plateSightingCopyColumns,
			pgx.CopyFromSlice(len(all), func(i int) ([]any, error) {
				c := all[i]
				return []any{int32(c.ID), idOf[c.Plate], c.Address, c.PhotoKey, c.CreatedAt}, nil
			}),
		)
		if err != nil {
			return fmt.Errorf("copy sightings: %w", err)
		}
		return nil
	})
}
