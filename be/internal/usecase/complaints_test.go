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

func TestComplainInWriteTx(t *testing.T) {
	tx := dbmocks.NewExecutorMock(t)
	repo := mocks.NewComplaintsRepositoryMock(t)
	o := models.ComplainOpts{SightingID: "s1", Reason: models.ComplaintBadPhoto, Comment: "темно"}
	repo.CreateComplaintMock.Expect(minimock.AnyContext, tx, o).Return(models.Complaint{ID: "c1", Status: models.ComplaintNew}, nil)

	c, err := NewComplaints(writeTx(t, tx), repo).Complain(context.Background(), o)
	if err != nil || c.ID != "c1" {
		t.Fatalf("Complain: %+v %v", c, err)
	}
}

func TestListComplaintsInReadTx(t *testing.T) {
	tx := dbmocks.NewExecutorMock(t)
	repo := mocks.NewComplaintsRepositoryMock(t)
	o := models.ListComplaintsOpts{Status: models.ComplaintNew, Limit: 30}
	repo.ListComplaintsMock.Expect(minimock.AnyContext, tx, o).Return(models.ComplaintsPage{Total: 2}, nil)

	page, err := NewComplaints(readTx(t, tx), repo).ListComplaints(context.Background(), o)
	if err != nil || page.Total != 2 {
		t.Fatalf("ListComplaints: %+v %v", page, err)
	}
}

func TestResolveComplaintInWriteTx(t *testing.T) {
	tx := dbmocks.NewExecutorMock(t)
	repo := mocks.NewComplaintsRepositoryMock(t)
	repo.ResolveComplaintMock.Expect(minimock.AnyContext, tx, "c1").Return(models.Complaint{}, models.ErrNotFound)
	repo.AuditSnapshotMock.Return(nil, nil) // журнал не пишется: WriteAudit не ожидается

	if _, err := NewComplaints(writeTx(t, tx), repo).ResolveComplaint(context.Background(), "c1"); !errors.Is(err, models.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestComplainPhotoReasonHidesPhoto(t *testing.T) {
	tx := dbmocks.NewExecutorMock(t)
	repo := mocks.NewComplaintsRepositoryMock(t)
	o := models.ComplainOpts{SightingID: "s1", Reason: models.ComplaintPersonalData}
	repo.CreateComplaintMock.Expect(minimock.AnyContext, tx, o).
		Return(models.Complaint{ID: "c1", SightingID: "s1", Reason: models.ComplaintPersonalData}, nil)
	repo.HidePhotoIfComplainedMock.Expect(minimock.AnyContext, tx, "s1", models.PhotoHideThreshold).Return(true, nil)

	if _, err := NewComplaints(writeTx(t, tx), repo).Complain(context.Background(), o); err != nil {
		t.Fatal(err)
	}
}

func TestComplainOtherReasonKeepsPhoto(t *testing.T) {
	tx := dbmocks.NewExecutorMock(t)
	repo := mocks.NewComplaintsRepositoryMock(t)
	o := models.ComplainOpts{SightingID: "s1", Reason: models.ComplaintWrongPlate}
	repo.CreateComplaintMock.Return(models.Complaint{ID: "c1", SightingID: "s1", Reason: models.ComplaintWrongPlate}, nil)

	// HidePhotoIfComplained не ожидается: минимок упадёт при неожиданном вызове.
	if _, err := NewComplaints(writeTx(t, tx), repo).Complain(context.Background(), o); err != nil {
		t.Fatal(err)
	}
}

func TestResolveComplaintShowsPhoto(t *testing.T) {
	tx := dbmocks.NewExecutorMock(t)
	repo := mocks.NewComplaintsRepositoryMock(t)
	repo.ResolveComplaintMock.Expect(minimock.AnyContext, tx, "c1").Return(models.Complaint{ID: "c1", Status: models.ComplaintResolved}, nil)
	repo.ShowComplaintPhotoMock.Expect(minimock.AnyContext, tx, "c1").Return(nil)
	rec := &auditRec{snaps: []string{`{"status":"new","photo_hidden":true}`, `{"status":"resolved","photo_hidden":false}`}}
	repo.AuditSnapshotMock.Set(rec.snapshot)
	repo.WriteAuditMock.Set(rec.write)

	if c, err := NewComplaints(writeTx(t, tx), repo).ResolveComplaint(ctxReq(), "c1"); err != nil || c.Status != models.ComplaintResolved {
		t.Fatalf("ResolveComplaint: %+v %v", c, err)
	}
	rec.check(t, models.AuditResolveComplaint, models.AuditTargetComplaint, "c1",
		`{"status":"new","photo_hidden":true}`, `{"status":"resolved","photo_hidden":false}`)
}

func TestRestoreSightingPhotoInWriteTx(t *testing.T) {
	tx := dbmocks.NewExecutorMock(t)
	repo := mocks.NewAdminRepositoryMock(t)
	repo.RestoreSightingPhotoMock.Expect(minimock.AnyContext, tx, "s1").Return(models.PlateSighting{ID: "s1"}, nil)
	rec := &auditRec{snaps: []string{`{"photo_hidden":true}`, `{"photo_hidden":false}`}}
	repo.AuditSnapshotMock.Set(rec.snapshot)
	repo.WriteAuditMock.Set(rec.write)

	if s, err := NewAdmin(writeTx(t, tx), repo).RestoreSightingPhoto(ctxReq(), "s1"); err != nil || s.ID != "s1" {
		t.Fatalf("RestoreSightingPhoto: %+v %v", s, err)
	}
	rec.check(t, models.AuditRestorePhoto, models.AuditTargetSighting, "s1", `{"photo_hidden":true}`, `{"photo_hidden":false}`)
}
