package cars

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql
var schemaSQL string

// PostgresStore — хранилище машин в Postgres.
type PostgresStore struct {
	pool *pgxpool.Pool
}

var _ Repository = (*PostgresStore)(nil)

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

// Migrate создаёт таблицы и индексы, если их ещё нет.
func (s *PostgresStore) Migrate(ctx context.Context) error {
	if _, err := s.pool.Exec(ctx, schemaSQL); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}

// escapeLike экранирует спецсимволы LIKE, чтобы ввод пользователя искался буквально.
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func likeContains(s string) string { return "%" + likeEscaper.Replace(s) + "%" }

const searchWhere = `plate_norm LIKE $1 AND key_norm LIKE $2`

// Search находит машины, совпадающие и по номеру, и по ключу.
func (s *PostgresStore) Search(ctx context.Context, q Query) (Page, error) {
	plate, key := likeContains(NormalizePlate(q.Plate)), likeContains(NormalizeKey(q.Key))
	page := Page{Items: []Car{}, Offset: q.Offset, Limit: q.Limit}

	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM cars WHERE `+searchWhere, plate, key).Scan(&page.Total); err != nil {
		return Page{}, fmt.Errorf("count cars: %w", err)
	}
	if page.Total == 0 || q.Offset >= page.Total {
		return page, nil
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, address, plate, key, to_char(date, 'YYYY-MM-DD'), photo_key
		FROM cars
		WHERE `+searchWhere+`
		ORDER BY date DESC, id
		LIMIT $3 OFFSET $4`,
		plate, key, q.Limit, q.Offset)
	if err != nil {
		return Page{}, fmt.Errorf("search cars: %w", err)
	}
	page.Items, err = pgx.CollectRows(rows, scanCar)
	if err != nil {
		return Page{}, fmt.Errorf("search cars: %w", err)
	}

	if next := q.Offset + len(page.Items); len(page.Items) > 0 && next < page.Total {
		page.NextOffset = &next
	}
	return page, nil
}

// Get возвращает машину по id.
func (s *PostgresStore) Get(ctx context.Context, id int) (Car, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, address, plate, key, to_char(date, 'YYYY-MM-DD'), photo_key
		FROM cars WHERE id = $1`, id)
	if err != nil {
		return Car{}, fmt.Errorf("get car: %w", err)
	}
	c, err := pgx.CollectExactlyOneRow(rows, scanCar)
	if errors.Is(err, pgx.ErrNoRows) {
		return Car{}, ErrNotFound
	}
	if err != nil {
		return Car{}, fmt.Errorf("get car: %w", err)
	}
	return c, nil
}

// ReplaceAll атомарно заменяет все машины в базе (для заливки тестовых данных).
func (s *PostgresStore) ReplaceAll(ctx context.Context, all []Car) error {
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `TRUNCATE cars`); err != nil {
			return fmt.Errorf("truncate: %w", err)
		}
		_, err := tx.CopyFrom(ctx,
			pgx.Identifier{"cars"},
			[]string{"id", "address", "plate", "key", "date", "photo_key", "plate_norm", "key_norm"},
			pgx.CopyFromSlice(len(all), func(i int) ([]any, error) {
				c := all[i]
				return []any{c.ID, c.Address, c.Plate, c.Key, c.Date, c.PhotoKey, NormalizePlate(c.Plate), NormalizeKey(c.Key)}, nil
			}),
		)
		if err != nil {
			return fmt.Errorf("copy cars: %w", err)
		}
		return nil
	})
}

func scanCar(row pgx.CollectableRow) (Car, error) {
	var c Car
	err := row.Scan(&c.ID, &c.Address, &c.Plate, &c.Key, &c.Date, &c.PhotoKey)
	return c, err
}
