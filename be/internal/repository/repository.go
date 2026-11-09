// Package repository — работа с БД (Postgres): поиск и чтение фиксаций номеров.
// Generate — тестовые данные для `be seed`.
package repository

import (
	"context"
	"errors"

	"github.com/GeneralFire/cn_search/be/internal/models"
)

// ErrNotFound — фиксации с таким id нет.
var ErrNotFound = errors.New("plate sighting not found")

// Query — параметры поиска.
type Query struct {
	Plate  string // как пришло в запросе → plainto_tsquery; пусто — все записи
	Offset int
	Limit  int
}

// Page — страница результатов поиска.
type Page struct {
	Items      []models.PlateSighting
	Total      int // всего найдено по запросу
	Offset     int
	Limit      int
	NextOffset *int // nil — дальше ничего нет
}

// Repository — хранилище фиксаций. Реализация — PostgresStore; интерфейс нужен,
// чтобы подменять БД заглушкой в тестах сервиса.
type Repository interface {
	// Search — фиксации по строке поиска, постранично, от новых к старым.
	Search(ctx context.Context, q Query) (Page, error)
	// ListSamePlate — все фиксации номера, к которому относится фиксация id,
	// от новых к старым; ErrNotFound, если такой фиксации нет.
	ListSamePlate(ctx context.Context, id int) ([]models.PlateSighting, error)
}
