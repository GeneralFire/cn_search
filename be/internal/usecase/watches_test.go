package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/gojuno/minimock/v3"

	dbmocks "github.com/GeneralFire/cn_search/be/internal/db/mocks"
	"github.com/GeneralFire/cn_search/be/internal/models"
	"github.com/GeneralFire/cn_search/be/internal/usecase/mocks"
)

func TestWatchInWriteTx(t *testing.T) {
	tx := dbmocks.NewExecutorMock(t)
	repo := mocks.NewWatchesRepositoryMock(t)
	repo.CreateWatchMock.Expect(minimock.AnyContext, tx, "u1", "А123ВС 777").Return(nil)
	if err := NewWatches(writeTx(t, tx), repo).Watch(context.Background(), "u1", "А123ВС 777"); err != nil {
		t.Fatal(err)
	}
}

func TestMarkSeenNotFound(t *testing.T) {
	tx := dbmocks.NewExecutorMock(t)
	repo := mocks.NewWatchesRepositoryMock(t)
	repo.MarkWatchSeenMock.Expect(minimock.AnyContext, tx, "u1", "А123ВС 777").Return(models.ErrNotFound)
	if err := NewWatches(writeTx(t, tx), repo).MarkSeen(context.Background(), "u1", "А123ВС 777"); !errors.Is(err, models.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestUnwatchInWriteTx(t *testing.T) {
	tx := dbmocks.NewExecutorMock(t)
	repo := mocks.NewWatchesRepositoryMock(t)
	repo.DeleteWatchMock.Expect(minimock.AnyContext, tx, "u1", "А123ВС 777").Return(nil)
	if err := NewWatches(writeTx(t, tx), repo).Unwatch(context.Background(), "u1", "А123ВС 777"); err != nil {
		t.Fatal(err)
	}
}

func TestListWatchesInReadTx(t *testing.T) {
	tx := dbmocks.NewExecutorMock(t)
	repo := mocks.NewWatchesRepositoryMock(t)
	o := models.ListWatchesOpts{UserID: "u1", IncludeTestData: true}
	repo.ListWatchesMock.Expect(minimock.AnyContext, tx, o).Return([]models.Watch{{NewCount: 2}}, nil)
	items, err := NewWatches(readTx(t, tx), repo).List(context.Background(), o)
	if err != nil || len(items) != 1 || items[0].NewCount != 2 {
		t.Fatalf("List: %+v %v", items, err)
	}
}

func TestMarkSeenInWriteTx(t *testing.T) {
	tx := dbmocks.NewExecutorMock(t)
	repo := mocks.NewWatchesRepositoryMock(t)
	repo.MarkWatchSeenMock.Expect(minimock.AnyContext, tx, "u1", "А123ВС 777").Return(nil)
	if err := NewWatches(writeTx(t, tx), repo).MarkSeen(context.Background(), "u1", "А123ВС 777"); err != nil {
		t.Fatal(err)
	}
}
