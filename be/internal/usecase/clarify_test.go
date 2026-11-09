package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/gojuno/minimock/v3"

	"github.com/GeneralFire/cn_search/be/internal/db"
	dbmocks "github.com/GeneralFire/cn_search/be/internal/db/mocks"
	"github.com/GeneralFire/cn_search/be/internal/models"
	"github.com/GeneralFire/cn_search/be/internal/usecase/mocks"
)

func TestNormalizeClarifyPlate(t *testing.T) {
	for _, tt := range []struct{ in, typ, want string }{
		{"  а123вс 77 ", "ru", "А123ВС 77"},
		{"A123BC777", "ru", "А123ВС 777"},      // латиница → кириллица, пробел перед регионом
		{"0123ВС 77", "ru", "О123ВС 77"},       // «0» в буквенной позиции → «О»
		{"0100о0 750", "ru", "О100ОО 750"},     // в цифровых позициях 0 остаётся цифрой
		{"a 123   bc   77", "ru", "А123ВС 77"}, // лишние пробелы
		{" b  mw-1234 ", "eu", "B MW-1234"},
		{"7xyz123", "us", "7XYZ123"},
		{"  что   угодно ", "", "ЧТО УГОДНО"}, // тип не задан — без проверки формата
	} {
		got, err := normalizeClarifyPlate(tt.in, tt.typ)
		if err != nil || got != tt.want {
			t.Errorf("%q (%s) = %q, %v; want %q", tt.in, tt.typ, got, err, tt.want)
		}
	}
}

func TestNormalizeClarifyPlateInvalid(t *testing.T) {
	for _, tt := range []struct{ in, typ, want string }{
		{"А123ВС", "ru", errRUPlateFormat},           // без региона
		{"А123ВС 7", "ru", errRUPlateFormat},         // регион из одной цифры
		{"А123ВС 7777", "ru", errRUPlateFormat},      // регион из 4 цифр
		{"Б123ВС 77", "ru", errRUPlateFormat},        // буквы нет на номерах РФ
		{"А12ЗВС 77", "ru", errRUPlateFormat},        // буква в цифровой позиции
		{"1123ВС 77", "ru", errRUPlateFormat},        // цифра (не 0) в буквенной позиции
		{"B", "eu", errLatinPlateFormat},             // короче 2
		{"ABCDEFGHIJKLM", "eu", errLatinPlateFormat}, // длиннее 12
		{"АВС 123", "us", errLatinPlateFormat},       // кириллица
		{"AB_12", "eu", errLatinPlateFormat},
	} {
		_, err := normalizeClarifyPlate(tt.in, tt.typ)
		var v *models.ValidationError
		if !errors.As(err, &v) || v.Message != tt.want {
			t.Errorf("%q (%s): %v; want %q", tt.in, tt.typ, err, tt.want)
		}
	}
}

// writeTx — InWriteTx, который выполняет fn с переданным tx.
func writeTx(t *testing.T, tx db.Executor) *mocks.TransactorMock {
	m := mocks.NewTransactorMock(t)
	m.InWriteTxMock.Set(func(ctx context.Context, fn db.TxFunc) error { return fn(ctx, tx) })
	return m
}

func TestClarifyNotAcceptedBelowConsensus(t *testing.T) {
	tx := dbmocks.NewExecutorMock(t)
	repo := mocks.NewRepositoryMock(t)
	norm := models.ClarifyOpts{SightingID: "s1", Plate: "А123ВС 77", PlateType: "ru"}
	repo.MatchesSightingPlateMock.Expect(minimock.AnyContext, tx, "s1", "А123ВС 77").Return(false, nil)
	repo.CreateClarificationMock.Expect(minimock.AnyContext, tx, norm).
		Return(models.Clarification{ID: "c1", Plate: norm.Plate, Status: models.ClarificationNotAccepted}, nil)
	repo.CountMatchingClarificationsMock.Expect(minimock.AnyContext, tx, "s1", "А123ВС 77").Return(ClarificationConsensus-1, nil)

	c, err := New(writeTx(t, tx), repo).Clarify(context.Background(),
		models.ClarifyOpts{SightingID: "s1", Plate: " a123bc77 ", PlateType: "ru"})
	if err != nil || c.ID != "c1" || c.Status != models.ClarificationNotAccepted {
		t.Fatalf("Clarify: %+v %v", c, err)
	}
}

func TestClarifyAcceptedOnConsensus(t *testing.T) {
	tx := dbmocks.NewExecutorMock(t)
	repo := mocks.NewRepositoryMock(t)
	repo.MatchesSightingPlateMock.Return(false, nil)
	repo.CreateClarificationMock.Return(models.Clarification{ID: "c3", Status: models.ClarificationNotAccepted}, nil)
	repo.CountMatchingClarificationsMock.Return(ClarificationConsensus, nil)
	repo.AcceptMatchingClarificationsMock.Expect(minimock.AnyContext, tx, "s1", "B MW 1234").Return(nil)

	c, err := New(writeTx(t, tx), repo).Clarify(context.Background(),
		models.ClarifyOpts{SightingID: "s1", Plate: "b mw 1234", PlateType: "eu", PlateRegion: "D"})
	if err != nil || c.Status != models.ClarificationAccepted {
		t.Fatalf("Clarify: %+v %v", c, err)
	}
}

func TestClarifySameAsCurrent(t *testing.T) {
	tx := dbmocks.NewExecutorMock(t)
	repo := mocks.NewRepositoryMock(t)
	repo.MatchesSightingPlateMock.Return(true, nil) // CreateClarification не ожидается

	_, err := New(writeTx(t, tx), repo).Clarify(context.Background(), models.ClarifyOpts{SightingID: "s1", Plate: "А123ВС 777", PlateType: "ru"})
	var v *models.ValidationError
	if !errors.As(err, &v) || v.Message != errSameAsCurrent {
		t.Fatalf("err = %v, want %q", err, errSameAsCurrent)
	}
}

func TestClarifyInvalidFormatNoTx(t *testing.T) {
	// Неверный формат отклоняется до транзакции: ни InWriteTx, ни репозиторий не вызываются.
	_, err := New(mocks.NewTransactorMock(t), mocks.NewRepositoryMock(t)).Clarify(context.Background(),
		models.ClarifyOpts{SightingID: "s1", Plate: "А1", PlateType: "ru"})
	var v *models.ValidationError
	if !errors.As(err, &v) {
		t.Fatalf("err = %v, want ValidationError", err)
	}
}

func TestClarifyErrorsPropagate(t *testing.T) {
	tx := dbmocks.NewExecutorMock(t)
	repo := mocks.NewRepositoryMock(t)
	repo.MatchesSightingPlateMock.Return(false, models.ErrNotFound)
	if _, err := New(writeTx(t, tx), repo).Clarify(context.Background(), models.ClarifyOpts{SightingID: "s1", Plate: "x"}); !errors.Is(err, models.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}

	repo = mocks.NewRepositoryMock(t)
	boom := errors.New("db down")
	repo.MatchesSightingPlateMock.Return(false, nil)
	repo.CreateClarificationMock.Return(models.Clarification{}, nil)
	repo.CountMatchingClarificationsMock.Return(0, boom)
	if _, err := New(writeTx(t, tx), repo).Clarify(context.Background(), models.ClarifyOpts{SightingID: "s1", Plate: "x"}); !errors.Is(err, boom) {
		t.Errorf("err = %v, want %v", err, boom)
	}
}
