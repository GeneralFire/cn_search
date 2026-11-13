package usecase

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/GeneralFire/cn_search/be/internal/apitoken"
	"github.com/GeneralFire/cn_search/be/internal/db"
	"github.com/GeneralFire/cn_search/be/internal/models"
)

// ApiTokens — сценарии API-токенов: выпуск и учёт в админке, списание запросов при открытии
// страницы номера и журнал открытий (plate_views).
type ApiTokens struct {
	tx     transactor
	repo   apiTokensRepository
	signer *apitoken.Signer
	now    func() time.Time
}

func NewApiTokens(tx transactor, repo apiTokensRepository, signer *apitoken.Signer) *ApiTokens {
	return &ApiTokens{tx: tx, repo: repo, signer: signer, now: time.Now}
}

// CreateApiToken выпускает токен: подписывает {tid, iat} и сохраняет только hash подписи.
// Токен в открытом виде возвращается вызывающему и больше нигде не остаётся.
func (u *ApiTokens) CreateApiToken(ctx context.Context, name string, requests int) (token string, t models.ApiToken, err error) {
	id := uuid.NewString()
	token, hash := u.signer.Issue(id, u.now())
	err = u.tx.InWriteTx(ctx, func(ctx context.Context, tx db.Executor) error {
		t, err = u.repo.CreateApiToken(ctx, tx, models.CreateApiTokenOpts{ID: id, Name: name, Requests: requests, SignatureHash: hash})
		return err
	})
	if err != nil {
		return "", models.ApiToken{}, err
	}
	return token, t, nil
}

func (u *ApiTokens) ListApiTokens(ctx context.Context, o models.ListApiTokensOpts) (page models.ApiTokensPage, err error) {
	err = u.tx.InReadTx(ctx, func(ctx context.Context, tx db.Executor) error {
		page, err = u.repo.ListApiTokens(ctx, tx, o)
		return err
	})
	return page, err
}

func (u *ApiTokens) RevokeApiToken(ctx context.Context, id string) (t models.ApiToken, err error) {
	err = u.tx.InWriteTx(ctx, func(ctx context.Context, tx db.Executor) error {
		t, err = u.repo.RevokeApiToken(ctx, tx, id)
		return err
	})
	return t, err
}

func (u *ApiTokens) ListPlateViews(ctx context.Context, o models.ListPlateViewsOpts) (page models.PlateViewsPage, err error) {
	err = u.tx.InReadTx(ctx, func(ctx context.Context, tx db.Executor) error {
		page, err = u.repo.ListPlateViews(ctx, tx, o)
		return err
	})
	return page, err
}

// findToken проверяет подпись и находит токен в БД; ErrTokenInvalid — подпись неверна или токена нет.
func (u *ApiTokens) findToken(ctx context.Context, tx db.Executor, token string) (models.ApiToken, error) {
	id, hash, err := u.signer.Verify(token)
	if err != nil {
		return models.ApiToken{}, models.ErrTokenInvalid
	}
	t, err := u.repo.FindApiToken(ctx, tx, id, hash)
	if errors.Is(err, models.ErrNotFound) {
		return models.ApiToken{}, models.ErrTokenInvalid
	}
	return t, err
}

// TokenInfo — состояние токена запроса (остаток для панели настроек).
func (u *ApiTokens) TokenInfo(ctx context.Context, token string) (t models.ApiToken, err error) {
	if _, _, err := u.signer.Verify(token); err != nil {
		return models.ApiToken{}, models.ErrTokenInvalid
	}
	err = u.tx.InReadTx(ctx, func(ctx context.Context, tx db.Executor) error {
		t, err = u.findToken(ctx, tx, token)
		return err
	})
	return t, err
}

// OpenPlate — открытие страницы номера (rpc Get): фиксации номера, списание запроса из API-токена
// и запись в журнал — в одной транзакции на запись. Так квота списывается, только если страница
// реально отдана и открытие записано: ошибка чтения или журнала откатывает и списание.
//
// Неверная подпись — ErrTokenInvalid до транзакции (в журнал не пишется: токен не опознан).
// Отозванный или исчерпанный токен — открытие пишется в журнал с consumed = false (транзакция
// фиксируется), затем ErrTokenRevoked / ErrTokenExhausted. Админ (IsAdmin) квоту не тратит, токен
// не учитывается. Без токена — как раньше, consumed = false.
func (u *ApiTokens) OpenPlate(ctx context.Context, o models.OpenPlateOpts) (res models.SamePlate, err error) {
	useToken := o.Token != "" && !o.IsAdmin
	if useToken {
		if _, _, err := u.signer.Verify(o.Token); err != nil {
			return models.SamePlate{}, models.ErrTokenInvalid
		}
	}
	var denied error // отказ по токену — после фиксации журнала
	err = u.tx.InWriteTx(ctx, func(ctx context.Context, tx db.Executor) error {
		denied = nil
		if res.Items, err = u.repo.ListSamePlate(ctx, tx, o.SamePlateOpts); err != nil {
			return err
		}
		if res.Clarifications, err = u.repo.ListPlateClarifications(ctx, tx, o.SamePlateOpts); err != nil {
			return err
		}
		view := models.NewPlateView{SightingID: o.ID, IsAdmin: o.IsAdmin, UserID: o.UserID, UserAgent: o.UserAgent}
		for _, it := range res.Items {
			if it.ID == o.ID {
				view.PlateID = it.PlateID
			}
		}
		if useToken {
			t, err := u.findToken(ctx, tx, o.Token)
			if err != nil {
				return err
			}
			view.TokenID = &t.ID
			_, ok, err := u.repo.ConsumeApiToken(ctx, tx, t.ID)
			switch {
			case err != nil:
				return err
			case ok:
				view.Consumed = true
			case t.Revoked:
				denied = models.ErrTokenRevoked
			default:
				denied = models.ErrTokenExhausted
			}
		}
		return u.repo.CreatePlateView(ctx, tx, view)
	})
	if err != nil {
		return models.SamePlate{}, err
	}
	if denied != nil {
		return models.SamePlate{}, denied
	}
	return res, nil
}
