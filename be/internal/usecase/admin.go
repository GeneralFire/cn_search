package usecase

import (
	"context"

	"github.com/GeneralFire/cn_search/be/internal/db"
	"github.com/GeneralFire/cn_search/be/internal/models"
)

// Admin — админские сценарии: скрыть/показать фиксацию, принять/отклонить уточнение, список уточнений.
type Admin struct {
	tx   transactor
	repo adminRepository
}

func NewAdmin(tx transactor, repo adminRepository) *Admin {
	return &Admin{tx: tx, repo: repo}
}

// HideSighting — скрыть фиксацию или вернуть её (транзакция на запись).
func (u *Admin) HideSighting(ctx context.Context, id string, hidden bool) (s models.PlateSighting, err error) {
	err = u.tx.InWriteTx(ctx, func(ctx context.Context, tx db.Executor) error {
		action := models.AuditShowSighting
		if hidden {
			action = models.AuditHideSighting
		}
		return audited(ctx, u.repo, tx, action, models.AuditTargetSighting, id, func() (err error) {
			s, err = u.repo.HideSighting(ctx, tx, id, hidden)
			return err
		})
	})
	return s, err
}

// RestoreSightingPhoto — вернуть фото, скрытое по жалобам (транзакция на запись).
func (u *Admin) RestoreSightingPhoto(ctx context.Context, id string) (s models.PlateSighting, err error) {
	err = u.tx.InWriteTx(ctx, func(ctx context.Context, tx db.Executor) error {
		return audited(ctx, u.repo, tx, models.AuditRestorePhoto, models.AuditTargetSighting, id, func() (err error) {
			s, err = u.repo.RestoreSightingPhoto(ctx, tx, id)
			return err
		})
	})
	return s, err
}

// SetClarificationStatus — принять или отклонить уточнение вручную (транзакция на запись).
func (u *Admin) SetClarificationStatus(ctx context.Context, id string, st models.ClarificationStatus) (c models.Clarification, err error) {
	err = u.tx.InWriteTx(ctx, func(ctx context.Context, tx db.Executor) error {
		return audited(ctx, u.repo, tx, models.AuditSetClarificationState, models.AuditTargetClarification, id, func() (err error) {
			c, err = u.repo.SetClarificationStatus(ctx, tx, id, st)
			return err
		})
	})
	return c, err
}

// ListClarifications — страница уточнений; count и страница читаются в одном снимке.
func (u *Admin) ListClarifications(ctx context.Context, o models.ListClarificationsOpts) (page models.ClarificationsPage, err error) {
	err = u.tx.InReadTx(ctx, func(ctx context.Context, tx db.Executor) error {
		page, err = u.repo.ListClarifications(ctx, tx, o)
		return err
	})
	return page, err
}
