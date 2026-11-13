package models

import "time"

// ComplaintReason — причина жалобы на фото (enum complaint_reason в БД).
type ComplaintReason string

const (
	ComplaintWrongPlate   ComplaintReason = "wrong_plate"   // номер распознан неверно / не та машина
	ComplaintBadPhoto     ComplaintReason = "bad_photo"     // фото плохое
	ComplaintPersonalData ComplaintReason = "personal_data" // в кадре личные данные
	ComplaintOther        ComplaintReason = "other"
)

// ComplaintStatus — статус жалобы (enum complaint_status в БД).
type ComplaintStatus string

const (
	ComplaintNew      ComplaintStatus = "new"
	ComplaintResolved ComplaintStatus = "resolved"
)

// Complaint — жалоба на фото фиксации (таблица complaints).
type Complaint struct {
	ID         string          `db:"id"`
	SightingID string          `db:"sighting_id"`
	Reason     ComplaintReason `db:"reason"`
	Comment    string          `db:"comment"`
	Status     ComplaintStatus `db:"status"`
	CreatedAt  time.Time       `db:"created_at"`
}

// ComplaintItem — жалоба вместе с фиксацией (админ-панель). Колонки фиксации — "sighting.<колонка>".
type ComplaintItem struct {
	Complaint
	Sighting PlateSighting `db:"sighting"`
}

// ComplainOpts — новая жалоба.
type ComplainOpts struct {
	SightingID      string
	Reason          ComplaintReason
	Comment         string
	IncludeTestData bool // можно жаловаться и на тестовую фиксацию (заголовок include-test-data)
}

// ListComplaintsOpts — страница жалоб для админ-панели.
type ListComplaintsOpts struct {
	Status          ComplaintStatus // пусто — все
	SightingID      string          // только жалобы на эту фиксацию; пусто — все
	Offset          int
	Limit           int
	IncludeTestData bool
}

// ComplaintsPage — страница жалоб.
type ComplaintsPage struct {
	Items      []ComplaintItem
	Total      int  // всего по фильтру
	NextOffset *int // nil — дальше ничего нет
}
