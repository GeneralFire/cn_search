package repository

import (
	"time"

	"github.com/GeneralFire/cn_search/be/internal/models"
)

// plateSightingTable — таблица появлений номеров (миграция 0001_init.sql).
const plateSightingTable = "plate_sightings"

// plateSightingColumns — порядок колонок для SELECT и COPY.
var plateSightingColumns = []string{"id", "address", "plate", "photo_key", "created_at"}

// plateSightingEntity — строка таблицы plate_sightings.
type plateSightingEntity struct {
	ID        int32     `db:"id"`
	Address   string    `db:"address"`
	Plate     string    `db:"plate"`
	PhotoKey  string    `db:"photo_key"`
	CreatedAt time.Time `db:"created_at"`
}

func (e plateSightingEntity) toModel() models.PlateSighting {
	return models.PlateSighting{
		ID:        int(e.ID),
		Address:   e.Address,
		Plate:     e.Plate,
		PhotoKey:  e.PhotoKey,
		CreatedAt: e.CreatedAt,
	}
}

// values — значения в порядке plateSightingColumns (для COPY).
func plateSightingValues(m models.PlateSighting) []any {
	return []any{int32(m.ID), m.Address, m.Plate, m.PhotoKey, m.CreatedAt}
}
