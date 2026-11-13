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

	s, err := NewAdmin(writeTx(t, tx), repo).HideSighting(context.Background(), "s1", true)
	if err != nil || !s.Hidden {
		t.Fatalf("HideSighting: %+v %v", s, err)
	}
}

func TestSetClarificationStatusInWriteTx(t *testing.T) {
	tx := dbmocks.NewExecutorMock(t)
	repo := mocks.NewAdminRepositoryMock(t)
	repo.SetClarificationStatusMock.Expect(minimock.AnyContext, tx, "c1", models.ClarificationAccepted).
		Return(models.Clarification{ID: "c1", Status: models.ClarificationAccepted}, nil)

	c, err := NewAdmin(writeTx(t, tx), repo).SetClarificationStatus(context.Background(), "c1", models.ClarificationAccepted)
	if err != nil || c.Status != models.ClarificationAccepted {
		t.Fatalf("SetClarificationStatus: %+v %v", c, err)
	}
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
