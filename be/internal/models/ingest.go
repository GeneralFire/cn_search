package models

import (
	"errors"
	"time"
)

// ErrRecognizerUnavailable — сервис распознавания номеров не ответил (недоступен, таймаут, 5xx).
var ErrRecognizerUnavailable = errors.New("recognizer unavailable")

// IngestOpts — новая фиксация по фото (rpc Ingest).
type IngestOpts struct {
	Photo   []byte     // JPEG или PNG
	TakenAt *time.Time // nil — из EXIF, иначе сейчас
	Lat     *float64   // nil — из EXIF GPS; задаются только парой с Lon
	Lon     *float64
	Address string
}

// PlateCandidate — номер, распознанный на фото.
type PlateCandidate struct {
	Plate      string  // после нормализации по формату: "Р466УК 79"
	Raw        string  // как прочитала модель: "P466YK79"
	Confidence float64 // 0..1
	PlateType  string  // "ru", "eu", "us"; "" — формат неизвестен
	Region     string  // страна ЕС ("D", "E"), если понятна по формату
	BBox       []int   // x1, y1, x2, y2
}

// IngestResult — созданная фиксация и все распознанные номера (по убыванию уверенности).
type IngestResult struct {
	Sighting   PlateSighting
	Candidates []PlateCandidate
}

// UpsertPlateOpts — номер для поиска или создания (таблица plates).
type UpsertPlateOpts struct {
	Plate  string
	Type   string // meta.type; "" — не записывается
	Region string // meta.region; "" — не записывается
}

// CreateSightingOpts — новая фиксация (таблица plate_sightings), не тестовая.
type CreateSightingOpts struct {
	PlateID       string
	Address       string
	AddressLocale string
	PhotoKey      string
	ThumbKey      string
	CreatedAt     time.Time
	Lat, Lon      *float64
}
