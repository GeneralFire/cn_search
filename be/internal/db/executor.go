// Package db — доступ к Postgres: plain SQL через Executor и транзакции через Cluster.
package db

import (
	"context"
	"fmt"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5/pgconn"
)

// executor — реализация Executor поверх querier (pgx.Tx).
type executor struct {
	q querier
}

var _ Executor = executor{}

func (e executor) Select(ctx context.Context, dest any, sql string, args ...any) error {
	return pgxscan.Select(ctx, e.q, dest, sql, args...)
}

func (e executor) Get(ctx context.Context, dest any, sql string, args ...any) error {
	return pgxscan.Get(ctx, e.q, dest, sql, args...)
}

func (e executor) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	return e.q.Exec(ctx, sql, args...)
}

func (e executor) SelectSqr(ctx context.Context, dest any, q Sqlizer) error {
	sql, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("build sql: %w", err)
	}
	return e.Select(ctx, dest, sql, args...)
}

func (e executor) GetSqr(ctx context.Context, dest any, q Sqlizer) error {
	sql, args, err := q.ToSql()
	if err != nil {
		return fmt.Errorf("build sql: %w", err)
	}
	return e.Get(ctx, dest, sql, args...)
}

func (e executor) ExecSqr(ctx context.Context, q Sqlizer) (pgconn.CommandTag, error) {
	sql, args, err := q.ToSql()
	if err != nil {
		return pgconn.CommandTag{}, fmt.Errorf("build sql: %w", err)
	}
	return e.Exec(ctx, sql, args...)
}
