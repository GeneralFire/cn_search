package models

import "time"

// ClarificationStatus — статус уточнения (enum clarification_status в БД).
type ClarificationStatus string

const (
	ClarificationNotAccepted ClarificationStatus = "not_accepted"
	ClarificationAccepted    ClarificationStatus = "accepted"
)

// Clarification — уточнение номера к фиксации (таблица clarifications).
type Clarification struct {
	ID         string              `db:"id"`
	SightingID string              `db:"sighting_id"`
	Plate      string              `db:"plate"`
	Status     ClarificationStatus `db:"status"`
	CreatedAt  time.Time           `db:"created_at"`

	PlateType   string `db:"plate_type"`   // "ru" | "eu" | "us" | "" (другой/неизвестен)
	PlateRegion string `db:"plate_region"` // страна (ЕС) или штат (США)
}

// ClarifyOpts — новое уточнение.
type ClarifyOpts struct {
	SightingID  string
	Plate       string
	PlateType   string // "ru" | "eu" | "us" | ""
	PlateRegion string
}

// ValidationError — запрос не прошёл проверку бизнес-правил; Message — понятный пользователю текст
// (implementation отдаёт его как InvalidArgument).
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }
