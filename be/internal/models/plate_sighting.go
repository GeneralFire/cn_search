// Package models — доменные модели. Теги db — имена колонок для сканирования из Postgres.
package models

import "time"

// PlateSighting — фиксация номера: где и когда замечена машина (таблица plate_sightings).
type PlateSighting struct {
	ID        string    `db:"id"`              // uuid
	PlateID   string    `db:"plate_id"`        // uuid номера в plates
	Plate     string    `db:"plate"`           // госномер кириллицей: "А123ВС 777" (plates.plate, через JOIN)
	PhotoKey  string    `db:"photo_key"`       // полное фото в S3: "car-3.jpg"
	ThumbKey  string    `db:"photo_thumb_key"` // миниатюра в S3: "car-3_thumb.jpg"
	Address   string    `db:"address"`
	CreatedAt time.Time `db:"created_at"`
}
