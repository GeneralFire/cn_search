package models

import "errors"

// ErrNotFound — запрошенной записи нет.
var ErrNotFound = errors.New("not found")

// SearchOpts — параметры поиска фиксаций.
type SearchOpts struct {
	Plate  string // как пришло в запросе → plainto_tsquery; пусто — все записи
	Offset int
	Limit  int
}

// SearchPage — страница результатов поиска.
type SearchPage struct {
	Items      []PlateSighting
	Total      int // всего найдено по запросу
	Offset     int
	Limit      int
	NextOffset *int // nil — дальше ничего нет
}
