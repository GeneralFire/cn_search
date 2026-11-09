// Package repository — работа с БД: поиск и чтение появлений номеров.
// PostgresStore — рабочее хранилище, MemoryStore — для тестов, Generate — тестовые данные.
package repository

import (
	"context"
	"errors"

	"github.com/GeneralFire/cn_search/be/internal/models"
)

// ErrNotFound — записи с таким id нет.
var ErrNotFound = errors.New("car not found")

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

// Repository — хранилище: PostgresStore в работе, MemoryStore — в тестах.
type Repository interface {
	Search(ctx context.Context, q Query) (Page, error)
	Get(ctx context.Context, id int) (models.PlateSighting, error)
}
