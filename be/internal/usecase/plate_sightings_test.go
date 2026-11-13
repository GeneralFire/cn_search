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

// Пустая выдача по непустому запросу — подсказки из той же транзакции.
func TestSearchSuggestsWhenEmpty(t *testing.T) {
	tx := dbmocks.NewExecutorMock(t)
	opts := models.SearchOpts{Plate: "А123ВС 778", Limit: 35, IncludeTestData: true}
	repo := mocks.NewRepositoryMock(t)
	repo.SearchMock.Expect(minimock.AnyContext, tx, opts).Return(models.SearchPage{}, nil)
	repo.SuggestMock.Expect(minimock.AnyContext, tx, opts).Return([]models.PlateSighting{{Plate: "А123ВС 777"}}, nil)

	page, err := New(readTx(t, tx), repo).Search(context.Background(), opts)
	if err != nil || len(page.Suggestions) != 1 || page.Suggestions[0].Plate != "А123ВС 777" {
		t.Fatalf("Search: %+v %v", page, err)
	}
}

// Подсказки не нужны, если что-то нашлось или запрос пустой (Suggest не вызывается — мок упадёт).
func TestSearchNoSuggestions(t *testing.T) {
	for name, c := range map[string]struct {
		plate string
		total int
	}{"нашлось": {"А123", 3}, "пустой запрос": {" ", 0}} {
		t.Run(name, func(t *testing.T) {
			tx := dbmocks.NewExecutorMock(t)
			repo := mocks.NewRepositoryMock(t)
			repo.SearchMock.Return(models.SearchPage{Total: c.total}, nil)
			page, err := New(readTx(t, tx), repo).Search(context.Background(), models.SearchOpts{Plate: c.plate})
			if err != nil || page.Suggestions != nil {
				t.Fatalf("Search: %+v %v", page, err)
			}
		})
	}
}

func TestListSamePlateInReadTx(t *testing.T) {
	tx := dbmocks.NewExecutorMock(t)
	repo := mocks.NewRepositoryMock(t)
	repo.ListSamePlateMock.Expect(minimock.AnyContext, tx, models.SamePlateOpts{ID: "id-1", IncludeTestData: true}).Return([]models.PlateSighting{{ID: "id-4"}, {ID: "id-1"}}, nil)

	items, err := New(readTx(t, tx), repo).ListSamePlate(context.Background(), models.SamePlateOpts{ID: "id-1", IncludeTestData: true})
	if err != nil || len(items) != 2 {
		t.Fatalf("ListSamePlate: %v %v", items, err)
	}
}

func TestErrorPropagates(t *testing.T) {
	tx := dbmocks.NewExecutorMock(t)
	repo := mocks.NewRepositoryMock(t)
	repo.ListSamePlateMock.Return(nil, models.ErrNotFound)

	if _, err := New(readTx(t, tx), repo).ListSamePlate(context.Background(), models.SamePlateOpts{ID: "id-1"}); !errors.Is(err, models.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestSeeAlsoInReadTx(t *testing.T) {
	tx := dbmocks.NewExecutorMock(t)
	repo := mocks.NewRepositoryMock(t)
	o := models.SeeAlsoOpts{Limit: 21}
	repo.SeeAlsoMock.Expect(minimock.AnyContext, tx, o).Return([]models.PlateSighting{{ID: "x"}}, nil)

	items, err := New(readTx(t, tx), repo).SeeAlso(context.Background(), o)
	if err != nil || len(items) != 1 {
		t.Fatalf("SeeAlso: %v %v", items, err)
	}
}
