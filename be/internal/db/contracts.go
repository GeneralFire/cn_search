package db

//go:generate minimock -i Executor -o ./mocks/executor_mock.go -n ExecutorMock -p mocks
//go:generate minimock -i querier -o ./mocks/querier_mock.go -n QuerierMock -p mocks
//go:generate minimock -i Sqlizer -o ./mocks/sqlizer_mock.go -n SqlizerMock -p mocks

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// querier — то, на чём выполняются запросы: pgx.Tx (или *pgxpool.Pool).
type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// Sqlizer — построитель запроса: тот же контракт, что squirrel.Sqlizer
// (для Postgres: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)).
type Sqlizer interface {
	ToSql() (string, []any, error)
}

// Executor выполняет plain SQL в транзакции и сканирует результат в структуры по тегам db.
//
//	var items []models.PlateSighting
//	err := tx.Select(ctx, &items, `SELECT … FROM plate_sightings WHERE …`, args...)
type Executor interface {
	// Select читает все строки в dest — указатель на слайс структур (или скаляров).
	Select(ctx context.Context, dest any, sql string, args ...any) error
	// Get читает ровно одну строку в dest — указатель на структуру или скаляр; pgx.ErrNoRows, если строк нет.
	Get(ctx context.Context, dest any, sql string, args ...any) error
	// Exec выполняет запрос без результата.
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)

	// SelectSqr, GetSqr, ExecSqr — то же для запроса из построителя (Sqlizer).
	SelectSqr(ctx context.Context, dest any, q Sqlizer) error
	GetSqr(ctx context.Context, dest any, q Sqlizer) error
	ExecSqr(ctx context.Context, q Sqlizer) (pgconn.CommandTag, error)
}
