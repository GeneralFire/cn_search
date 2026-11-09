package models

import "encoding/json"

// Plate — госномер (таблица plates).
type Plate struct {
	ID    string          `db:"id"`    // uuid
	Plate string          `db:"plate"` // кириллицей: "А123ВС 777"
	Meta  json.RawMessage `db:"meta"`  // произвольные данные о номере, JSON-объект
}
