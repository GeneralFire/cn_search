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
}

// ClarifyOpts — новое уточнение.
type ClarifyOpts struct {
	SightingID string
	Plate      string
}
