package usecase

import (
	"context"

	"github.com/GeneralFire/cn_search/be/internal/db"
	"github.com/GeneralFire/cn_search/be/internal/models"
)

// Watches — подписки пользователя на номера.
type Watches struct {
	tx   transactor
	repo watchesRepository
}

func NewWatches(tx transactor, repo watchesRepository) *Watches {
	return &Watches{tx: tx, repo: repo}
}

// Watch — подписка на номер фиксации (транзакция на запись).
func (u *Watches) Watch(ctx context.Context, userID, plate string) error {
	return u.tx.InWriteTx(ctx, func(ctx context.Context, tx db.Executor) error {
		return u.repo.CreateWatch(ctx, tx, userID, plate)
	})
}

// Unwatch — отписка от номера (транзакция на запись).
func (u *Watches) Unwatch(ctx context.Context, userID, plate string) error {
	return u.tx.InWriteTx(ctx, func(ctx context.Context, tx db.Executor) error {
		return u.repo.DeleteWatch(ctx, tx, userID, plate)
	})
}

// List — подписки пользователя (транзакция на чтение).
func (u *Watches) List(ctx context.Context, o models.ListWatchesOpts) (items []models.Watch, err error) {
	err = u.tx.InReadTx(ctx, func(ctx context.Context, tx db.Executor) error {
		items, err = u.repo.ListWatches(ctx, tx, o)
		return err
	})
	return items, err
}

// MarkSeen — пользователь увидел все фиксации номера (транзакция на запись).
func (u *Watches) MarkSeen(ctx context.Context, userID, plate string) error {
	return u.tx.InWriteTx(ctx, func(ctx context.Context, tx db.Executor) error {
		return u.repo.MarkWatchSeen(ctx, tx, userID, plate)
	})
}
