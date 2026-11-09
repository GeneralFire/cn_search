package db_test

import (
	"context"
	"errors"
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/GeneralFire/cn_search/be/internal/db"
	"github.com/GeneralFire/cn_search/be/internal/db/mocks"
)

func TestExecSql(t *testing.T) {
	q := mocks.NewQuerierMock(t)
	q.ExecMock.Expect(minimock.AnyContext, "DELETE FROM plates WHERE id = $1", "x").
		Return(pgconn.NewCommandTag("DELETE 1"), nil)
	b := mocks.NewSqlizerMock(t)
	b.ToSqlMock.Return("DELETE FROM plates WHERE id = $1", []any{"x"}, nil)

	tag, err := db.NewExecutorForTest(q).ExecSqr(context.Background(), b)
	if err != nil || tag.RowsAffected() != 1 {
		t.Fatalf("ExecSqr: %v %v", tag, err)
	}
}

func TestSqlBuilderError(t *testing.T) {
	buildErr := errors.New("bad builder")
	ex := db.NewExecutorForTest(mocks.NewQuerierMock(t)) // до БД дело дойти не должно
	for name, run := range map[string]func(db.Sqlizer) error{
		"ExecSqr":   func(b db.Sqlizer) error { _, err := ex.ExecSqr(context.Background(), b); return err },
		"SelectSqr": func(b db.Sqlizer) error { var v []int; return ex.SelectSqr(context.Background(), &v, b) },
		"GetSqr":    func(b db.Sqlizer) error { var v int; return ex.GetSqr(context.Background(), &v, b) },
	} {
		b := mocks.NewSqlizerMock(t)
		b.ToSqlMock.Return("", nil, buildErr)
		if err := run(b); !errors.Is(err, buildErr) {
			t.Errorf("%s: ошибка построителя не пробросилась: %v", name, err)
		}
	}
}
