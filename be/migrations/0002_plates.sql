-- Номера — в отдельную таблицу plates; фиксации ссылаются на неё через plate_id.

-- +goose Up
CREATE TABLE plates (
    id    uuid  PRIMARY KEY DEFAULT gen_random_uuid(),
    plate text  NOT NULL UNIQUE, -- госномер кириллицей: "А123ВС 777"
    meta  jsonb NOT NULL DEFAULT '{}'
);

-- Поиск по номеру (rpc Search). Выражение должно совпадать с запросом в internal/repository/postgres.go.
CREATE INDEX plates_plate_gin_idx ON plates USING gin (to_tsvector('simple', plate));

INSERT INTO plates (plate) SELECT DISTINCT plate FROM plate_sightings;

ALTER TABLE plate_sightings ADD COLUMN plate_id uuid REFERENCES plates (id);
UPDATE plate_sightings s SET plate_id = p.id FROM plates p WHERE p.plate = s.plate;
ALTER TABLE plate_sightings ALTER COLUMN plate_id SET NOT NULL;

-- Все фиксации номера (rpc Get).
CREATE INDEX plate_sightings_plate_id_idx ON plate_sightings (plate_id, created_at DESC, id);

DROP INDEX plate_sightings_plate_gin_idx;
ALTER TABLE plate_sightings DROP COLUMN plate;

-- +goose Down
ALTER TABLE plate_sightings ADD COLUMN plate text;
UPDATE plate_sightings s SET plate = p.plate FROM plates p WHERE p.id = s.plate_id;
ALTER TABLE plate_sightings ALTER COLUMN plate SET NOT NULL;
CREATE INDEX plate_sightings_plate_gin_idx ON plate_sightings USING gin (to_tsvector('simple', plate));

ALTER TABLE plate_sightings DROP COLUMN plate_id;
DROP TABLE plates;
