package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/gojuno/minimock/v3"

	"github.com/GeneralFire/cn_search/be/internal/db"
	dbmocks "github.com/GeneralFire/cn_search/be/internal/db/mocks"
	"github.com/GeneralFire/cn_search/be/internal/models"
	"github.com/GeneralFire/cn_search/be/internal/usecase/mocks"
)

// readTx — InReadTx, который выполняет fn с переданным tx (InWriteTx при этом не ожидается).
func readTx(t *testing.T, tx db.Executor) *mocks.TransactorMock {
	m := mocks.NewTransactorMock(t)
	m.InReadTxMock.Set(func(ctx context.Context, fn db.TxFunc) error { return fn(ctx, tx) })
	return m
}

func TestSearchInReadTx(t *testing.T) {
	tx := dbmocks.NewExecutorMock(t) // маркер: репозиторий должен получить именно его
	opts := models.SearchOpts{Plate: "А123ВС 777", Limit: 35}
	repo := mocks.NewRepositoryMock(t)
	repo.SearchMock.Expect(minimock.AnyContext, tx, opts).Return(models.SearchPage{Total: 3}, nil)

	page, err := New(readTx(t, tx), repo).Search(context.Background(), opts)
	if err != nil || page.Total != 3 {
		t.Fatalf("Search: %+v %v", page, err)
	}
}

func TestListSamePlateInReadTx(t *testing.T) {
	tx := dbmocks.NewExecutorMock(t)
	repo := mocks.NewRepositoryMock(t)
	repo.ListSamePlateMock.Expect(minimock.AnyContext, tx, "id-1").Return([]models.PlateSighting{{ID: "id-4"}, {ID: "id-1"}}, nil)

	items, err := New(readTx(t, tx), repo).ListSamePlate(context.Background(), "id-1")
	if err != nil || len(items) != 2 {
		t.Fatalf("ListSamePlate: %v %v", items, err)
	}
}

func TestErrorPropagates(t *testing.T) {
	tx := dbmocks.NewExecutorMock(t)
	repo := mocks.NewRepositoryMock(t)
	repo.ListSamePlateMock.Return(nil, models.ErrNotFound)

	if _, err := New(readTx(t, tx), repo).ListSamePlate(context.Background(), "id-1"); !errors.Is(err, models.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}
