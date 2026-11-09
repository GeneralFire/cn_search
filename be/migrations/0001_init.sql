-- Появления номеров: где и когда замечена машина.

-- +goose Up
CREATE TABLE plate_sightings (
    id         integer     PRIMARY KEY,
    address    text        NOT NULL,
    plate      text        NOT NULL, -- госномер кириллицей: "А123ВС 777"
    photo_key  text        NOT NULL DEFAULT '', -- ключ объекта в S3-бакете
    created_at timestamptz NOT NULL DEFAULT now() -- когда номер появился
);

-- Порядок выдачи: от новых к старым.
CREATE INDEX plate_sightings_created_at_id_idx ON plate_sightings (created_at DESC, id);

-- Поиск по частям номера (серия "А123ВС", регион "777"). Выражение должно совпадать
-- с запросом в internal/repository/postgres.go.
CREATE INDEX plate_sightings_plate_gin_idx ON plate_sightings USING gin (to_tsvector('simple', plate));

-- +goose Down
DROP TABLE plate_sightings;
