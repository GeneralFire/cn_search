package models

import (
	"errors"
	"fmt"
	"time"
)

// ErrRecognizerUnavailable — сервис распознавания номеров не ответил (недоступен, таймаут, 5xx).
var ErrRecognizerUnavailable = errors.New("recognizer unavailable")

// UploadQuotaError — суточный лимит загрузок фото исчерпан (тариф Free); Limit — лимит за сутки.
type UploadQuotaError struct {
	Limit int
}

func (e *UploadQuotaError) Error() string {
	return fmt.Sprintf("upload quota exceeded: %d per day", e.Limit)
}

// IngestOpts — новая фиксация по фото (rpc Ingest).
type IngestOpts struct {
	Photo   []byte     // JPEG или PNG
	TakenAt *time.Time // nil — из EXIF, иначе сейчас
	Lat     *float64   // nil — из EXIF GPS; задаются только парой с Lon
	Lon     *float64
	Address string
	User    User // кто загружает: квота по тарифу, plate_sightings.uploaded_by
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
	// UploadsLeft — сколько загрузок ещё можно за сутки после этой; nil — без ограничений (Pro).
	UploadsLeft *int
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
	UploadedBy    string    // users.id
	UploadedAt    time.Time // момент загрузки — для суточной квоты
}
