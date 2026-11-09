package implementation

import (
	"context"
	"time"

	"github.com/GeneralFire/cn_search/be/internal/models"
	"github.com/GeneralFire/cn_search/be/internal/repository"
)

// stubRepo — заглушка репозитория: отдаёт заранее заданные ответы и запоминает запрос.
// Поиск в БД здесь не проверяется — только логика сервиса.
type stubRepo struct {
	page    repository.Page
	list    []models.PlateSighting
	err     error
	query   repository.Query
	listFor int
}

func (r *stubRepo) Search(_ context.Context, q repository.Query) (repository.Page, error) {
	r.query = q
	return r.page, r.err
}

func (r *stubRepo) ListSamePlate(_ context.Context, id int) ([]models.PlateSighting, error) {
	r.listFor = id
	return r.list, r.err
}

var sighting = models.PlateSighting{
	ID: 1, PhotoKey: "car-1.png", Address: "Москва, Пресненская наб., 10", Plate: "А123ВС 777",
	CreatedAt: time.Date(2026, 11, 2, 9, 15, 0, 0, time.UTC),
}

func newSvc(r *stubRepo) *CarSearch { return NewCarSearch(r, "/s3/cars/") }
