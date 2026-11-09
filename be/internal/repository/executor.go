package repository

import (
	"context"
	"fmt"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// querier — то, на чём выполняются запросы: *pgxpool.Pool или pgx.Tx.
type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// executor выполняет plain SQL и сканирует результат в структуры по тегам db.
//
//	var items []models.PlateSighting
//	err := s.Select(ctx, &items, `SELECT … FROM plate_sightings WHERE …`, args...)
type executor struct {
	q querier
}

// Select читает все строки в dest — указатель на слайс структур (или скаляров).
func (e executor) Select(ctx context.Context, dest any, sql string, args ...any) error {
	return pgxscan.Select(ctx, e.q, dest, sql, args...)
}

// Get читает ровно одну строку в dest — указатель на структуру или скаляр;
// pgx.ErrNoRows, если строк нет.
func (e executor) Get(ctx context.Context, dest any, sql string, args ...any) error {
	return pgxscan.Get(ctx, e.q, dest, sql, args...)
}

// Exec выполняет запрос без результата.
func (e executor) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return e.q.Exec(ctx, sql, args...)
}

// sqlizer — построитель запроса: тот же контракт, что squirrel.Sqlizer
// (для Postgres: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)).
type sqlizer interface {
	ToSql() (string, []any, error)
}

// SelectSql — Select для запроса из построителя.
func (e executor) SelectSql(ctx context.Context, dest any, q sqlizer) error {
	sql, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("build sql: %w", err)
	}
	return e.Select(ctx, dest, sql, args...)
}

// GetSql — Get для запроса из построителя.
func (e executor) GetSql(ctx context.Context, dest any, q sqlizer) error {
	sql, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("build sql: %w", err)
	}
	return e.Get(ctx, dest, sql, args...)
}

// ExecSql — Exec для запроса из построителя.
func (e executor) ExecSql(ctx context.Context, q sqlizer) (pgconn.CommandTag, error) {
	sql, args, err := q.ToSql()
	if err != nil {
		return pgconn.CommandTag{}, fmt.Errorf("build sql: %w", err)
	}
	return e.Exec(ctx, sql, args...)
}
