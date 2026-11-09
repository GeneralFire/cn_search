// Package models — доменные модели.
package models

import "time"

// PlateSighting — появление номера: где и когда замечена машина.
type PlateSighting struct {
	ID        int
	PhotoKey  string // ключ объекта в S3-бакете с картинками: "car-3.png"
	Address   string
	Plate     string // госномер кириллицей: "А123ВС 777"
	CreatedAt time.Time
}
