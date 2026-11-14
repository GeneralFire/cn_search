package models

import "time"

// Watch — подписка пользователя на номер (таблица watches). Номера в базе может ещё не быть:
// тогда Latest == nil («ждём появления»).
type Watch struct {
	PlateKey     string         // plate_watch_key(номер)
	PlateDisplay string         // номер, как его ввели или видели
	Latest       *PlateSighting // последняя видимая фиксация номера
	NewCount     int            // фиксаций новее last_seen_at
	WatchedAt    time.Time
}

// ListWatchesOpts — подписки пользователя.
type ListWatchesOpts struct {
	UserID          string
	IncludeTestData bool
}
