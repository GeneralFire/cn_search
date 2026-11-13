package models

import (
	"errors"
	"slices"
)

// ErrNotFound — запрошенной записи нет.
var ErrNotFound = errors.New("not found")

// SearchOpts — параметры поиска фиксаций.
type SearchOpts struct {
	Plate           string // строка поиска по началу слов номера; пусто — все записи
	Offset          int
	Limit           int
	IncludeTestData bool   // отдавать и строки с is_test_data = true (заголовок include-test-data)
	Market          Market // «рынок» номеров; пусто — без фильтра
	IncludeHidden   bool   // отдавать и скрытые фиксации (plate_sightings.hidden) — только админу
}

// Market — «рынок» номеров для главной: страна по формату номера (plates.meta: type, region).
type Market string

const (
	MarketRU    Market = "ru"    // type "ru"
	MarketUA    Market = "ua"    // type "ua"
	MarketDE    Market = "de"    // type "eu", region "D"
	MarketUS    Market = "us"    // type "us"
	MarketCN    Market = "cn"    // type "cn"
	MarketOther Market = "other" // всё, что не попало ни в один рынок выше
)

// Markets — все рынки, кроме «прочего» (оно — их дополнение).
var Markets = []Market{MarketRU, MarketUA, MarketDE, MarketUS, MarketCN}

// Valid — известный рынок или пусто (без фильтра).
func (m Market) Valid() bool {
	return m == "" || m == MarketOther || slices.Contains(Markets, m)
}

// SamePlateOpts — параметры выборки всех фиксаций номера.
type SamePlateOpts struct {
	ID              string // uuid любой фиксации номера
	IncludeTestData bool   // отдавать и строки с is_test_data = true (заголовок include-test-data)
	IncludeHidden   bool   // отдавать и скрытые фиксации — только админу
}

// SearchPage — страница результатов поиска.
type SearchPage struct {
	Items      []PlateSighting
	Total      int // всего найдено по запросу
	Offset     int
	Limit      int
	NextOffset *int // nil — дальше ничего нет
	// Suggestions — «возможно, вы искали»: только при Total = 0 и непустом запросе.
	Suggestions []PlateSighting
}

// SuggestLimit — сколько подсказок «возможно, вы искали» отдавать.
const SuggestLimit = 5

// SuggestMaxDistance — наибольшее расстояние правки (levenshtein) между ключами запроса и номера.
const SuggestMaxDistance = 2

// SeeAlsoOpts — параметры блока «свежие фиксации» на странице номера.
type SeeAlsoOpts struct {
	ExcludeSightingID string // номер этой фиксации исключить; пусто — без исключения
	Limit             int
	IncludeTestData   bool
	IncludeHidden     bool // отдавать и скрытые фиксации — только админу
}
