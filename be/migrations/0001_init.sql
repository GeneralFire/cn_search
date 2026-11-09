-- Номера (plates) и их фиксации (plate_sightings): где и когда замечена машина.

-- +goose Up
CREATE TABLE plates (
    id    uuid  PRIMARY KEY DEFAULT gen_random_uuid(),
    plate text  NOT NULL UNIQUE, -- госномер кириллицей: "А123ВС 777"
    meta  jsonb NOT NULL DEFAULT '{}',
    is_test_data boolean NOT NULL DEFAULT false -- тестовые данные (миграция 0002_test_data)
);

-- Поиск по номеру (rpc Search). Выражение должно совпадать с запросом в internal/repository/postgres.go.
CREATE INDEX plates_plate_gin_idx ON plates USING gin (to_tsvector('simple', plate));

CREATE TABLE plate_sightings (
    id         uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    plate_id   uuid        NOT NULL REFERENCES plates (id),
    address    text        NOT NULL,
    photo_key       text   NOT NULL DEFAULT '', -- ключ полного фото в S3-бакете
    photo_thumb_key text   NOT NULL DEFAULT '', -- ключ миниатюры в S3-бакете
    created_at timestamptz NOT NULL DEFAULT now(), -- когда номер зафиксирован
    is_test_data boolean   NOT NULL DEFAULT false  -- тестовые данные (миграция 0002_test_data)
);

-- Выдача: от новых к старым.
CREATE INDEX plate_sightings_created_at_id_idx ON plate_sightings (created_at DESC, id);

-- Все фиксации номера (rpc Get).
CREATE INDEX plate_sightings_plate_id_idx ON plate_sightings (plate_id, created_at DESC, id);

-- Уточнения номера к фиксации: пользователь предлагает, какой номер на самом деле (кнопка «Уточнить»).
CREATE TYPE clarification_status AS ENUM ('not_accepted', 'accepted');

CREATE TABLE clarifications (
    id           uuid                 PRIMARY KEY DEFAULT gen_random_uuid(),
    sighting_id  uuid                 NOT NULL REFERENCES plate_sightings (id),
    plate        text                 NOT NULL, -- предложенный номер, как ввели
    status       clarification_status NOT NULL DEFAULT 'not_accepted',
    created_at   timestamptz          NOT NULL DEFAULT now(),
    is_test_data boolean              NOT NULL DEFAULT false
);

CREATE INDEX clarifications_sighting_id_idx ON clarifications (sighting_id);
-- Поиск по уточнённым номерам (rpc Search) — то же выражение, что для plates.
CREATE INDEX clarifications_plate_gin_idx ON clarifications USING gin (to_tsvector('simple', plate));

-- +goose Down
DROP TABLE clarifications;
DROP TYPE clarification_status;
DROP TABLE plate_sightings;
DROP TABLE plates;
