package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// fakeQuerier запоминает запрос; Query не нужен — проверяем только связку *Sql → ToSql → Exec.
type fakeQuerier struct {
	sql  string
	args []any
}

func (f *fakeQuerier) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeQuerier) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	f.sql, f.args = sql, args
	return pgconn.NewCommandTag("DELETE 1"), nil
}

type rawSql struct {
	sql  string
	args []any
	err  error
}

func (r rawSql) ToSql() (string, []any, error) { return r.sql, r.args, r.err }

func TestExecSql(t *testing.T) {
	q := &fakeQuerier{}
	ex := executor{q: q}
	tag, err := ex.ExecSql(context.Background(), rawSql{sql: "DELETE FROM plates WHERE id = $1", args: []any{"x"}})
	if err != nil || tag.RowsAffected() != 1 || q.sql != "DELETE FROM plates WHERE id = $1" || len(q.args) != 1 {
		t.Fatalf("ExecSql: tag=%v err=%v sql=%q args=%v", tag, err, q.sql, q.args)
	}

	buildErr := errors.New("bad builder")
	for name, run := range map[string]func() error{
		"ExecSql":   func() error { _, err := ex.ExecSql(context.Background(), rawSql{err: buildErr}); return err },
		"SelectSql": func() error { var v []int; return ex.SelectSql(context.Background(), &v, rawSql{err: buildErr}) },
		"GetSql":    func() error { var v int; return ex.GetSql(context.Background(), &v, rawSql{err: buildErr}) },
	} {
		if err := run(); !errors.Is(err, buildErr) {
			t.Errorf("%s: ошибка построителя не пробросилась: %v", name, err)
		}
	}
}
