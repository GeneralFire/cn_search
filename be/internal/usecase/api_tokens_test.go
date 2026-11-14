package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/GeneralFire/cn_search/be/internal/apitoken"
	"github.com/GeneralFire/cn_search/be/internal/db"
	dbmocks "github.com/GeneralFire/cn_search/be/internal/db/mocks"
	"github.com/GeneralFire/cn_search/be/internal/models"
	"github.com/GeneralFire/cn_search/be/internal/usecase/mocks"
)

const (
	tokSighting = "00000000-0000-0000-0000-000000000001"
	tokPlate    = "10000000-0000-0000-0000-000000000001"
	tokID       = "20000000-0000-0000-0000-000000000001"
)

var tokSigner = apitoken.NewSigner("test-key")

func TestCreateApiTokenStoresOnlyHash(t *testing.T) {
	tx := dbmocks.NewExecutorMock(t)
	repo := mocks.NewApiTokensRepositoryMock(t)
	var saved models.CreateApiTokenOpts
	repo.CreateApiTokenMock.Set(func(_ context.Context, _ db.Executor, o models.CreateApiTokenOpts) (models.ApiToken, error) {
		saved = o
		return models.ApiToken{ID: o.ID, Name: o.Name, RequestsTotal: o.Requests, RequestsLeft: o.Requests}, nil
	})
	rec := &auditRec{snaps: []string{`null`, `{"name":"партнёр","requests_total":3,"revoked":false}`}}
	repo.AuditSnapshotMock.Set(func(ctx context.Context, tx db.Executor, tt, id string) (json.RawMessage, error) {
		if tt != models.AuditTargetApiToken || id == "" {
			t.Errorf("snapshot(%q, %q)", tt, id)
		}
		return rec.snapshot(ctx, tx, tt, id)
	})
	repo.WriteAuditMock.Set(rec.write)
	token, info, err := NewApiTokens(writeTx(t, tx), repo, tokSigner).CreateApiToken(ctxReq(), "партнёр", 3)
	if err != nil || info.RequestsLeft != 3 {
		t.Fatalf("CreateApiToken: %+v %v", info, err)
	}
	rec.check(t, models.AuditCreateApiToken, models.AuditTargetApiToken, saved.ID, `null`, `{"name":"партнёр","requests_total":3,"revoked":false}`)
	id, hash, err := tokSigner.Verify(token)
	if err != nil || id != saved.ID || hash != saved.SignatureHash || saved.Name != "партнёр" {
		t.Fatalf("сохранено %+v, токен %q → %q %q %v", saved, token, id, hash, err)
	}
}

// openPlate готовит OpenPlate: ListSamePlate отдаёт фиксацию, CreatePlateView запоминает запись.
func openPlate(t *testing.T) (*ApiTokens, *mocks.ApiTokensRepositoryMock, *models.NewPlateView) {
	tx := dbmocks.NewExecutorMock(t)
	repo := mocks.NewApiTokensRepositoryMock(t)
	repo.ListSamePlateMock.Return([]models.PlateSighting{{ID: tokSighting, PlateID: tokPlate}}, nil)
	repo.ListPlateClarificationsMock.Optional().Return(nil, nil)
	var view models.NewPlateView
	repo.CreatePlateViewMock.Set(func(_ context.Context, _ db.Executor, v models.NewPlateView) error {
		view = v
		return nil
	})
	return NewApiTokens(writeTx(t, tx), repo, tokSigner), repo, &view
}

func TestOpenPlate(t *testing.T) {
	token, hash := tokSigner.Issue(tokID, time.Now())
	opts := func(tok string, admin bool) models.OpenPlateOpts {
		return models.OpenPlateOpts{SamePlateOpts: models.SamePlateOpts{ID: tokSighting}, Token: tok, IsAdmin: admin, UserAgent: "UA"}
	}

	t.Run("без токена", func(t *testing.T) {
		u, _, view := openPlate(t)
		res, err := u.OpenPlate(context.Background(), opts("", false))
		if err != nil || len(res.Items) != 1 || view.TokenID != nil || view.Consumed || view.PlateID != tokPlate || view.UserAgent != "UA" {
			t.Fatalf("%v %v %+v", res, err, view)
		}
	})
	t.Run("списание", func(t *testing.T) {
		u, repo, view := openPlate(t)
		repo.FindApiTokenMock.Set(func(_ context.Context, _ db.Executor, id, h string) (models.ApiToken, error) {
			if id != tokID || h != hash {
				t.Errorf("FindApiToken(%q, %q)", id, h)
			}
			return models.ApiToken{ID: tokID, RequestsLeft: 1}, nil
		})
		repo.ConsumeApiTokenMock.Return(models.ApiToken{ID: tokID}, true, nil)
		if _, err := u.OpenPlate(context.Background(), opts(token, false)); err != nil || !view.Consumed || view.TokenID == nil || *view.TokenID != tokID {
			t.Fatalf("%v %+v", err, view)
		}
	})
	t.Run("исчерпан — журнал пишется, ошибка после", func(t *testing.T) {
		u, repo, view := openPlate(t)
		repo.FindApiTokenMock.Return(models.ApiToken{ID: tokID}, nil)
		repo.ConsumeApiTokenMock.Return(models.ApiToken{}, false, nil)
		if _, err := u.OpenPlate(context.Background(), opts(token, false)); !errors.Is(err, models.ErrTokenExhausted) || view.Consumed || view.TokenID == nil {
			t.Fatalf("%v %+v", err, view)
		}
	})
	t.Run("отозван", func(t *testing.T) {
		u, repo, _ := openPlate(t)
		repo.FindApiTokenMock.Return(models.ApiToken{ID: tokID, Revoked: true, RequestsLeft: 5}, nil)
		repo.ConsumeApiTokenMock.Return(models.ApiToken{}, false, nil)
		if _, err := u.OpenPlate(context.Background(), opts(token, false)); !errors.Is(err, models.ErrTokenRevoked) {
			t.Fatal(err)
		}
	})
	t.Run("админ квоту не тратит", func(t *testing.T) {
		u, _, view := openPlate(t)
		if _, err := u.OpenPlate(context.Background(), opts(token, true)); err != nil || !view.IsAdmin || view.TokenID != nil || view.Consumed {
			t.Fatalf("%v %+v", err, view)
		}
	})
	t.Run("подпись неверна — до транзакции", func(t *testing.T) {
		repo := mocks.NewApiTokensRepositoryMock(t)
		u := NewApiTokens(nil, repo, tokSigner)
		bad, _ := apitoken.NewSigner("other").Issue(tokID, time.Now())
		if _, err := u.OpenPlate(context.Background(), opts(bad, false)); !errors.Is(err, models.ErrTokenInvalid) {
			t.Fatal(err)
		}
	})
}

func TestRevokeApiTokenWritesAudit(t *testing.T) {
	tx := dbmocks.NewExecutorMock(t)
	repo := mocks.NewApiTokensRepositoryMock(t)
	repo.RevokeApiTokenMock.Return(models.ApiToken{ID: tokID, Revoked: true}, nil)
	rec := &auditRec{snaps: []string{`{"name":"n","requests_total":3,"revoked":false}`, `{"name":"n","requests_total":3,"revoked":true}`}}
	repo.AuditSnapshotMock.Set(rec.snapshot)
	repo.WriteAuditMock.Set(rec.write)

	if _, err := NewApiTokens(writeTx(t, tx), repo, tokSigner).RevokeApiToken(ctxReq(), tokID); err != nil {
		t.Fatal(err)
	}
	rec.check(t, models.AuditRevokeApiToken, models.AuditTargetApiToken, tokID,
		`{"name":"n","requests_total":3,"revoked":false}`, `{"name":"n","requests_total":3,"revoked":true}`)
}
