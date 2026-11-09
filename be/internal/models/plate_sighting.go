// Package models — доменные модели. Теги db — имена колонок для сканирования из Postgres.
package models

import "time"

// PlateSighting — фиксация номера: где и когда замечена машина (таблица plate_sightings).
type PlateSighting struct {
	ID        int       `db:"id"`
	PlateID   string    `db:"plate_id"` // uuid номера в plates
	Plate     string    `db:"plate"`    // госномер кириллицей: "А123ВС 777" (plates.plate, через JOIN)
	PhotoKey  string    `db:"photo_key"`
	Address   string    `db:"address"`
	CreatedAt time.Time `db:"created_at"`
}
