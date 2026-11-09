package repository

// Таблицы и SQL-фрагменты (схема — миграции be/migrations).
// Строки сканируются прямо в модели (models.Plate, models.PlateSighting) по тегам db.

const (
	platesTable         = "plates"
	plateSightingsTable = "plate_sightings"
	clarificationsTable = "clarifications"
)

// clarificationColumns — колонки clarifications; совпадают с тегами db models.Clarification.
const clarificationColumns = `id, sighting_id, plate, status, created_at`

// plateSightingSelect — колонки фиксации вместе с номером; совпадают с тегами db models.PlateSighting.
const plateSightingSelect = `s.id, s.plate_id, p.plate, s.address, s.photo_key, s.photo_thumb_key, s.created_at,
	coalesce(p.meta->>'type', '') AS plate_type, coalesce(p.meta->>'region', '') AS plate_region, s.address_locale`

// plateSightingFrom — фиксации с JOIN номера.
const plateSightingFrom = plateSightingsTable + ` s JOIN ` + platesTable + ` p ON p.id = s.plate_id`
