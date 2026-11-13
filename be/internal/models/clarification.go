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

	PlateType   string `db:"plate_type"`   // "ru" | "eu" | "us" | "ua" | "cn" | "" (другой/неизвестен)
	PlateRegion string `db:"plate_region"` // страна (ЕС) или штат (США)
}

// ClarifyOpts — новое уточнение.
type ClarifyOpts struct {
	SightingID  string
	Plate       string
	PlateType   string // "ru" | "eu" | "us" | "ua" | "cn" | ""
	PlateRegion string
}

// ValidationError — запрос не прошёл проверку бизнес-правил; Message — понятный пользователю текст
// (implementation отдаёт его как InvalidArgument).
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

// ClarificationItem — уточнение вместе с фиксацией (админ-панель). Колонки фиксации — "sighting.<колонка>".
type ClarificationItem struct {
	Clarification
	Sighting PlateSighting `db:"sighting"`
}

// ListClarificationsOpts — страница уточнений для админ-панели.
type ListClarificationsOpts struct {
	Status          ClarificationStatus // пусто — все
	SightingID      string              // только уточнения к этой фиксации; пусто — все
	Offset          int
	Limit           int
	IncludeTestData bool
}

// ClarificationsPage — страница уточнений.
type ClarificationsPage struct {
	Items      []ClarificationItem
	Total      int  // всего по фильтру
	NextOffset *int // nil — дальше ничего нет
}

// PlateClarification — уточнения к фиксациям номера с одним предложенным номером
// (сгруппированы по plate_search_key).
type PlateClarification struct {
	Plate       string              `db:"plate"`        // написание из самого свежего уточнения группы
	PlateType   string              `db:"plate_type"`   // как в Clarification
	PlateRegion string              `db:"plate_region"` // как в Clarification
	Count       int                 `db:"count"`        // сколько уточнений
	Status      ClarificationStatus `db:"status"`       // accepted, если есть принятые
	SightingIDs []string            `db:"sighting_ids"` // фиксации, к которым его предлагали
	Items       []Clarification     `db:"items"`        // уточнения группы, от новых к старым
}

// SamePlate — фиксации номера и уточнения к ним (rpc Get).
type SamePlate struct {
	Items          []PlateSighting
	Clarifications []PlateClarification
}
