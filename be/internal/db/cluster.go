package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TxFunc — работа внутри транзакции; tx — Executor поверх неё.
type TxFunc func(ctx context.Context, tx Executor) error

// Cluster открывает транзакции к Postgres. Ошибка или паника в fn — rollback, иначе commit.
type Cluster struct {
	pool *pgxpool.Pool
}

func NewCluster(pool *pgxpool.Pool) *Cluster {
	return &Cluster{pool: pool}
}

// InReadTx — транзакция только на чтение, REPEATABLE READ: все запросы видят один снимок
// (например, count и страница выдачи в Search согласованы).
func (c *Cluster) InReadTx(ctx context.Context, fn TxFunc) error {
	return c.inTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly, IsoLevel: pgx.RepeatableRead}, fn)
}

// InWriteTx — транзакция на запись (READ COMMITTED).
func (c *Cluster) InWriteTx(ctx context.Context, fn TxFunc) error {
	return c.inTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadWrite}, fn)
}

func (c *Cluster) inTx(ctx context.Context, opts pgx.TxOptions, fn TxFunc) error {
	return pgx.BeginTxFunc(ctx, c.pool, opts, func(tx pgx.Tx) error {
		return fn(ctx, executor{q: tx})
	})
}
