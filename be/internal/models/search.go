package models

import "errors"

// ErrNotFound — запрошенной записи нет.
var ErrNotFound = errors.New("not found")

// SearchOpts — параметры поиска фиксаций.
type SearchOpts struct {
	Plate           string // строка поиска по началу слов номера; пусто — все записи
	Offset          int
	Limit           int
	IncludeTestData bool // отдавать и строки с is_test_data = true (заголовок include-test-data)
}

// SamePlateOpts — параметры выборки всех фиксаций номера.
type SamePlateOpts struct {
	ID              string // uuid любой фиксации номера
	IncludeTestData bool   // отдавать и строки с is_test_data = true (заголовок include-test-data)
}

// SearchPage — страница результатов поиска.
type SearchPage struct {
	Items      []PlateSighting
	Total      int // всего найдено по запросу
	Offset     int
	Limit      int
	NextOffset *int // nil — дальше ничего нет
}

// SeeAlsoOpts — параметры блока «свежие фиксации» на странице номера.
type SeeAlsoOpts struct {
	ExcludeSightingID string // номер этой фиксации исключить; пусто — без исключения
	Limit             int
	IncludeTestData   bool
}
