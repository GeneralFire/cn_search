package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/GeneralFire/cn_search/be/internal/db"
	"github.com/GeneralFire/cn_search/be/internal/models"
)

// UpsertPlate возвращает id номера, создавая его при необходимости (meta: type и region, если заданы).
// Номер уже есть — meta не меняется; тестовый номер становится настоящим (его тестовые фиксации
// по-прежнему скрыты флагом is_test_data у фиксаций).
func (r *repository) UpsertPlate(ctx context.Context, tx db.Executor, o models.UpsertPlateOpts) (string, error) {
	meta := map[string]string{}
	if o.Type != "" {
		meta["type"] = o.Type
	}
	if o.Region != "" {
		meta["region"] = o.Region
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return "", fmt.Errorf("plate meta: %w", err)
	}
	var id string
	err = tx.Get(ctx, &id, `INSERT INTO `+platesTable+` (plate, meta) VALUES ($1, $2)
		ON CONFLICT (plate) DO UPDATE SET is_test_data = false RETURNING id`, o.Plate, string(metaJSON))
	if err != nil {
		return "", fmt.Errorf("upsert plate: %w", err)
	}
	return id, nil
}

// CreateSighting сохраняет фиксацию (не тестовую) и возвращает её вместе с номером.
func (r *repository) CreateSighting(ctx context.Context, tx db.Executor, o models.CreateSightingOpts) (models.PlateSighting, error) {
	var s models.PlateSighting
	err := tx.Get(ctx, &s, `WITH s AS (
			INSERT INTO `+plateSightingsTable+` (plate_id, address, address_locale, photo_key, photo_thumb_key, created_at, lat, lon)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING *
		) SELECT `+plateSightingSelect+` FROM s JOIN `+platesTable+` p ON p.id = s.plate_id`,
		o.PlateID, o.Address, o.AddressLocale, o.PhotoKey, o.ThumbKey, o.CreatedAt, o.Lat, o.Lon)
	if err != nil {
		return models.PlateSighting{}, fmt.Errorf("create sighting: %w", err)
	}
	return s, nil
}
