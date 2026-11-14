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

func TestSubmitCameraRequestInWriteTx(t *testing.T) {
	tx := dbmocks.NewExecutorMock(t)
	repo := mocks.NewCameraRequestsRepositoryMock(t)
	o := models.SubmitCameraRequestOpts{Lat: 55.75, Lon: 37.61, Reason: "перекрёсток"}
	repo.CreateCameraRequestMock.Expect(minimock.AnyContext, tx, o).Return(models.CameraRequest{ID: "r1", Status: models.CameraRequestNew}, nil)

	c, err := NewCameraRequests(writeTx(t, tx), repo).Submit(context.Background(), o)
	if err != nil || c.ID != "r1" {
		t.Fatalf("Submit: %+v %v", c, err)
	}
}

func TestListCameraRequestsInReadTx(t *testing.T) {
	tx := dbmocks.NewExecutorMock(t)
	repo := mocks.NewCameraRequestsRepositoryMock(t)
	o := models.ListCameraRequestsOpts{Status: models.CameraRequestNew, Limit: 100}
	repo.ListCameraRequestsMock.Expect(minimock.AnyContext, tx, o).Return(models.CameraRequestsPage{Total: 2}, nil)

	page, err := NewCameraRequests(readTx(t, tx), repo).List(context.Background(), o)
	if err != nil || page.Total != 2 {
		t.Fatalf("List: %+v %v", page, err)
	}
}

func TestVoteCameraRequestInWriteTx(t *testing.T) {
	tx := dbmocks.NewExecutorMock(t)
	repo := mocks.NewCameraRequestsRepositoryMock(t)
	o := models.VoteCameraRequestOpts{ID: "r1"}
	repo.VoteCameraRequestMock.Expect(minimock.AnyContext, tx, o).Return(models.CameraRequest{ID: "r1", Votes: 3}, nil)

	c, err := NewCameraRequests(writeTx(t, tx), repo).Vote(context.Background(), o)
	if err != nil || c.Votes != 3 {
		t.Fatalf("Vote: %+v %v", c, err)
	}
}

func TestSetCameraRequestStatusInWriteTx(t *testing.T) {
	tx := dbmocks.NewExecutorMock(t)
	repo := mocks.NewCameraRequestsRepositoryMock(t)
	repo.SetCameraRequestStatusMock.Expect(minimock.AnyContext, tx, "r1", models.CameraRequestApproved).Return(models.CameraRequest{}, models.ErrNotFound)
	repo.AuditSnapshotMock.Return(nil, nil) // журнал не пишется: WriteAudit не ожидается

	if _, err := NewCameraRequests(writeTx(t, tx), repo).SetStatus(context.Background(), "r1", models.CameraRequestApproved); !errors.Is(err, models.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestSetCameraRequestStatusWritesAudit(t *testing.T) {
	tx := dbmocks.NewExecutorMock(t)
	repo := mocks.NewCameraRequestsRepositoryMock(t)
	repo.SetCameraRequestStatusMock.Expect(minimock.AnyContext, tx, "r1", models.CameraRequestApproved).Return(models.CameraRequest{ID: "r1"}, nil)
	rec := &auditRec{snaps: []string{`{"status":"new"}`, `{"status":"approved"}`}}
	repo.AuditSnapshotMock.Set(rec.snapshot)
	repo.WriteAuditMock.Set(rec.write)

	if _, err := NewCameraRequests(writeTx(t, tx), repo).SetStatus(ctxReq(), "r1", models.CameraRequestApproved); err != nil {
		t.Fatal(err)
	}
	rec.check(t, models.AuditSetCameraStatus, models.AuditTargetCameraRequest, "r1", `{"status":"new"}`, `{"status":"approved"}`)
}
