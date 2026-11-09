// Package usecase — бизнес-сценарии между RPC (implementation) и БД (repository).
// Usecase управляет транзакциями: открывает InReadTx / InWriteTx и передаёт tx в репозиторий.
package usecase

import (
	"context"

	"github.com/GeneralFire/cn_search/be/internal/db"
	"github.com/GeneralFire/cn_search/be/internal/models"
)

// PlateSightings — сценарии над фиксациями номеров.
type PlateSightings struct {
	tx   transactor
	repo repository
}

func New(tx transactor, repo repository) *PlateSightings {
	return &PlateSightings{tx: tx, repo: repo}
}

// Search — поиск фиксаций; count и страница читаются в одной транзакции (один снимок).
func (u *PlateSightings) Search(ctx context.Context, q models.SearchOpts) (page models.SearchPage, err error) {
	err = u.tx.InReadTx(ctx, func(ctx context.Context, tx db.Executor) error {
		page, err = u.repo.Search(ctx, tx, q)
		return err
	})
	return page, err
}

// ListSamePlate — все фиксации номера, к которому относится фиксация id.
func (u *PlateSightings) ListSamePlate(ctx context.Context, id string) (items []models.PlateSighting, err error) {
	err = u.tx.InReadTx(ctx, func(ctx context.Context, tx db.Executor) error {
		items, err = u.repo.ListSamePlate(ctx, tx, id)
		return err
	})
	return items, err
}
