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

// ListSamePlate — все фиксации номера, к которому относится фиксация o.ID.
func (u *PlateSightings) ListSamePlate(ctx context.Context, o models.SamePlateOpts) (items []models.PlateSighting, err error) {
	err = u.tx.InReadTx(ctx, func(ctx context.Context, tx db.Executor) error {
		items, err = u.repo.ListSamePlate(ctx, tx, o)
		return err
	})
	return items, err
}

// SeeAlso — свежие фиксации других номеров.
func (u *PlateSightings) SeeAlso(ctx context.Context, o models.SeeAlsoOpts) (items []models.PlateSighting, err error) {
	err = u.tx.InReadTx(ctx, func(ctx context.Context, tx db.Executor) error {
		items, err = u.repo.SeeAlso(ctx, tx, o)
		return err
	})
	return items, err
}

// Clarify — новое уточнение номера к фиксации (транзакция на запись). Номер нормализуется и
// проверяется по формату (normalizeClarifyPlate); совпадающее с текущим номером фиксации уточнение
// отклоняется. Если после вставки ClarificationConsensus и больше уточнений фиксации совпали
// (по plate_search_key) — все они принимаются; в ответе — актуальный статус.
func (u *PlateSightings) Clarify(ctx context.Context, o models.ClarifyOpts) (c models.Clarification, err error) {
	if o.Plate, err = normalizeClarifyPlate(o.Plate, o.PlateType); err != nil {
		return models.Clarification{}, err
	}
	err = u.tx.InWriteTx(ctx, func(ctx context.Context, tx db.Executor) error {
		// Заодно блокирует фиксацию: уточнения к ней идут по очереди, и подсчёт ниже не теряет
		// параллельно вставленные строки.
		same, err := u.repo.MatchesSightingPlate(ctx, tx, o.SightingID, o.Plate)
		if err != nil {
			return err
		}
		if same {
			return &models.ValidationError{Message: errSameAsCurrent}
		}
		if c, err = u.repo.CreateClarification(ctx, tx, o); err != nil {
			return err
		}
		n, err := u.repo.CountMatchingClarifications(ctx, tx, o.SightingID, o.Plate)
		if err != nil {
			return err
		}
		if n < ClarificationConsensus {
			return nil
		}
		if err := u.repo.AcceptMatchingClarifications(ctx, tx, o.SightingID, o.Plate); err != nil {
			return err
		}
		c.Status = models.ClarificationAccepted
		return nil
	})
	if err != nil {
		return models.Clarification{}, err
	}
	return c, nil
}
