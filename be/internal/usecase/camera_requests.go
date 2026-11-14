package usecase

import (
	"context"

	"github.com/GeneralFire/cn_search/be/internal/db"
	"github.com/GeneralFire/cn_search/be/internal/models"
)

// CameraRequests — сценарии над заявками на расположение камер.
type CameraRequests struct {
	tx   transactor
	repo cameraRequestsRepository
}

func NewCameraRequests(tx transactor, repo cameraRequestsRepository) *CameraRequests {
	return &CameraRequests{tx: tx, repo: repo}
}

// Submit — новая заявка (транзакция на запись).
func (u *CameraRequests) Submit(ctx context.Context, o models.SubmitCameraRequestOpts) (c models.CameraRequest, err error) {
	err = u.tx.InWriteTx(ctx, func(ctx context.Context, tx db.Executor) error {
		c, err = u.repo.CreateCameraRequest(ctx, tx, o)
		return err
	})
	return c, err
}

// List — страница заявок; count и страница читаются в одном снимке.
func (u *CameraRequests) List(ctx context.Context, o models.ListCameraRequestsOpts) (page models.CameraRequestsPage, err error) {
	err = u.tx.InReadTx(ctx, func(ctx context.Context, tx db.Executor) error {
		page, err = u.repo.ListCameraRequests(ctx, tx, o)
		return err
	})
	return page, err
}

// Vote — голос +1 (транзакция на запись).
func (u *CameraRequests) Vote(ctx context.Context, o models.VoteCameraRequestOpts) (c models.CameraRequest, err error) {
	err = u.tx.InWriteTx(ctx, func(ctx context.Context, tx db.Executor) error {
		c, err = u.repo.VoteCameraRequest(ctx, tx, o)
		return err
	})
	return c, err
}

// SetStatus — сменить статус заявки (транзакция на запись).
func (u *CameraRequests) SetStatus(ctx context.Context, id string, st models.CameraRequestStatus) (c models.CameraRequest, err error) {
	err = u.tx.InWriteTx(ctx, func(ctx context.Context, tx db.Executor) error {
		return audited(ctx, u.repo, tx, models.AuditSetCameraStatus, models.AuditTargetCameraRequest, id, func() (err error) {
			c, err = u.repo.SetCameraRequestStatus(ctx, tx, id, st)
			return err
		})
	})
	return c, err
}
