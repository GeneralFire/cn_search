package repository

// Таблицы и SQL-фрагменты (схема — миграции be/migrations).
// Строки сканируются прямо в модели (models.Plate, models.PlateSighting) по тегам db.

const (
	platesTable         = "plates"
	plateSightingsTable = "plate_sightings"
)

// plateSightingSelect — колонки фиксации вместе с номером; совпадают с тегами db models.PlateSighting.
const plateSightingSelect = `s.id, s.plate_id, p.plate, s.address, s.photo_key, s.photo_thumb_key, s.created_at`

// plateSightingFrom — фиксации с JOIN номера.
const plateSightingFrom = plateSightingsTable + ` s JOIN ` + platesTable + ` p ON p.id = s.plate_id`
