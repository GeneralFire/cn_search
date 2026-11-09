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

	if _, err := NewComplaints(writeTx(t, tx), repo).ResolveComplaint(context.Background(), "c1"); !errors.Is(err, models.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}
