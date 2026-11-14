package usecase

import (
	"context"

	"github.com/GeneralFire/cn_search/be/internal/db"
	"github.com/GeneralFire/cn_search/be/internal/models"
)

// Complaints — сценарии над жалобами на фото: жалоба пользователя и разбор в админ-панели.
type Complaints struct {
	tx   transactor
	repo complaintsRepository
}

func NewComplaints(tx transactor, repo complaintsRepository) *Complaints {
	return &Complaints{tx: tx, repo: repo}
}

// Complain — новая жалоба (транзакция на запись). Жалоба на фото (bad_photo, personal_data) в той же
// транзакции скрывает фото фиксации, если таких новых жалоб набралось models.PhotoHideThreshold.
func (u *Complaints) Complain(ctx context.Context, o models.ComplainOpts) (c models.Complaint, err error) {
	err = u.tx.InWriteTx(ctx, func(ctx context.Context, tx db.Executor) error {
		if c, err = u.repo.CreateComplaint(ctx, tx, o); err != nil {
			return err
		}
		if !c.Reason.IsPhotoReason() {
			return nil
		}
		_, err = u.repo.HidePhotoIfComplained(ctx, tx, c.SightingID, models.PhotoHideThreshold)
		return err
	})
	return c, err
}

// ListComplaints — страница жалоб; count и страница читаются в одном снимке.
func (u *Complaints) ListComplaints(ctx context.Context, o models.ListComplaintsOpts) (page models.ComplaintsPage, err error) {
	err = u.tx.InReadTx(ctx, func(ctx context.Context, tx db.Executor) error {
		page, err = u.repo.ListComplaints(ctx, tx, o)
		return err
	})
	return page, err
}

// ResolveComplaint — отметить жалобу решённой и снять скрытие фото её фиксации (транзакция на запись).
func (u *Complaints) ResolveComplaint(ctx context.Context, id string) (c models.Complaint, err error) {
	err = u.tx.InWriteTx(ctx, func(ctx context.Context, tx db.Executor) error {
		if c, err = u.repo.ResolveComplaint(ctx, tx, id); err != nil {
			return err
		}
		return u.repo.ShowComplaintPhoto(ctx, tx, id)
	})
	return c, err
}
