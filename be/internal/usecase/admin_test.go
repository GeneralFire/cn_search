package usecase

import (
	"context"
	"testing"

	"github.com/gojuno/minimock/v3"

	dbmocks "github.com/GeneralFire/cn_search/be/internal/db/mocks"
	"github.com/GeneralFire/cn_search/be/internal/models"
	"github.com/GeneralFire/cn_search/be/internal/usecase/mocks"
)

func TestHideSightingInWriteTx(t *testing.T) {
	tx := dbmocks.NewExecutorMock(t)
	repo := mocks.NewAdminRepositoryMock(t)
	repo.HideSightingMock.Expect(minimock.AnyContext, tx, "s1", true).Return(models.PlateSighting{ID: "s1", Hidden: true}, nil)
	rec := &auditRec{snaps: []string{`{"hidden":false}`, `{"hidden":true}`}}
	repo.AuditSnapshotMock.Set(rec.snapshot)
	repo.WriteAuditMock.Set(rec.write)

	s, err := NewAdmin(writeTx(t, tx), repo).HideSighting(ctxReq(), "s1", true)
	if err != nil || !s.Hidden {
		t.Fatalf("HideSighting: %+v %v", s, err)
	}
	rec.check(t, models.AuditHideSighting, models.AuditTargetSighting, "s1", `{"hidden":false}`, `{"hidden":true}`)
}

func TestShowSightingWritesAudit(t *testing.T) {
	tx := dbmocks.NewExecutorMock(t)
	repo := mocks.NewAdminRepositoryMock(t)
	repo.HideSightingMock.Expect(minimock.AnyContext, tx, "s1", false).Return(models.PlateSighting{ID: "s1"}, nil)
	rec := &auditRec{snaps: []string{`{"hidden":true}`, `{"hidden":false}`}}
	repo.AuditSnapshotMock.Set(rec.snapshot)
	repo.WriteAuditMock.Set(rec.write)

	if _, err := NewAdmin(writeTx(t, tx), repo).HideSighting(ctxReq(), "s1", false); err != nil {
		t.Fatal(err)
	}
	rec.check(t, models.AuditShowSighting, models.AuditTargetSighting, "s1", `{"hidden":true}`, `{"hidden":false}`)
}

func TestListAuditInReadTx(t *testing.T) {
	tx := dbmocks.NewExecutorMock(t)
	repo := mocks.NewAdminRepositoryMock(t)
	o := models.ListAuditOpts{Action: models.AuditHideSighting, Limit: 30}
	repo.ListAuditMock.Expect(minimock.AnyContext, tx, o).Return(models.AuditPage{Total: 4}, nil)

	page, err := NewAdmin(readTx(t, tx), repo).ListAudit(context.Background(), o)
	if err != nil || page.Total != 4 {
		t.Fatalf("ListAudit: %+v %v", page, err)
	}
}

func TestSetClarificationStatusInWriteTx(t *testing.T) {
	tx := dbmocks.NewExecutorMock(t)
	repo := mocks.NewAdminRepositoryMock(t)
	repo.SetClarificationStatusMock.Expect(minimock.AnyContext, tx, "c1", models.ClarificationAccepted).
		Return(models.Clarification{ID: "c1", Status: models.ClarificationAccepted}, nil)
	rec := &auditRec{snaps: []string{`{"status":"not_accepted"}`, `{"status":"accepted"}`}}
	repo.AuditSnapshotMock.Set(rec.snapshot)
	repo.WriteAuditMock.Set(rec.write)

	c, err := NewAdmin(writeTx(t, tx), repo).SetClarificationStatus(ctxReq(), "c1", models.ClarificationAccepted)
	if err != nil || c.Status != models.ClarificationAccepted {
		t.Fatalf("SetClarificationStatus: %+v %v", c, err)
	}
	rec.check(t, models.AuditSetClarificationState, models.AuditTargetClarification, "c1", `{"status":"not_accepted"}`, `{"status":"accepted"}`)
}

func TestListClarificationsInReadTx(t *testing.T) {
	tx := dbmocks.NewExecutorMock(t)
	repo := mocks.NewAdminRepositoryMock(t)
	o := models.ListClarificationsOpts{Status: models.ClarificationNotAccepted, Limit: 30}
	repo.ListClarificationsMock.Expect(minimock.AnyContext, tx, o).Return(models.ClarificationsPage{Total: 3}, nil)

	page, err := NewAdmin(readTx(t, tx), repo).ListClarifications(context.Background(), o)
	if err != nil || page.Total != 3 {
		t.Fatalf("ListClarifications: %+v %v", page, err)
	}
}
