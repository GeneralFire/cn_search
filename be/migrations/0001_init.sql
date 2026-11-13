-- Номера (plates) и их фиксации (plate_sightings): где и когда замечена машина.

-- +goose Up
-- levenshtein() для подсказок «возможно, вы искали» (rpc Search при пустой выдаче); contrib Postgres.
CREATE EXTENSION IF NOT EXISTS fuzzystrmatch;

-- Ключ поиска по номеру: похожие символы сводятся к одному. Кириллические буквы номера
-- (А В Е К М Н О Р С Т У Х, украинская І) → латинские двойники, буква О (кир. и лат.) → цифра 0,
-- дефис и точка китайских номеров "·" → пробел (иначе парсер tsvector читает "KDT-4419" как 'kdt'
-- и '-4419', а "京A·12345" — как одно слово с точкой); регистр снимает
-- to_tsvector/to_tsquery ('simple' приводит к нижнему). "О515НХ 52" → "0515HX 52", запросы
-- "0515", "O515hx", "о515нх" дают тот же ключ. Применяется и в индексах, и к строке запроса
-- (internal/repository/postgres.go) — выражения должны совпадать, иначе GIN не используется.
-- +goose StatementBegin
CREATE FUNCTION plate_search_key(s text) RETURNS text
LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE
RETURN translate(s, 'АВЕКМНОРСТУХавекмнорстухOo-Іі·', 'ABEKMH0PCTYXabekmh0pctyx00 Ii ');
-- +goose StatementEnd

CREATE TABLE plates (
    id    uuid  PRIMARY KEY DEFAULT gen_random_uuid(),
    plate text  NOT NULL UNIQUE, -- госномер кириллицей: "А123ВС 777"
    meta  jsonb NOT NULL DEFAULT '{}', -- {"type": "ru"|"eu"|"us"|"ua"|"cn", "region": "D"|"CA"…, …}; нет type — формат неизвестен
    is_test_data boolean NOT NULL DEFAULT false -- тестовые данные (миграция 0002_test_data)
);

-- Поиск по номеру (rpc Search). Выражение должно совпадать с запросом в internal/repository/postgres.go.
CREATE INDEX plates_plate_gin_idx ON plates USING gin (to_tsvector('simple', plate_search_key(plate)));

CREATE TABLE plate_sightings (
    id         uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    plate_id   uuid        NOT NULL REFERENCES plates (id),
    address    text        NOT NULL,
    address_locale text    NOT NULL DEFAULT 'ru', -- язык, на котором записан адрес: ru, en, de, …
    photo_key       text   NOT NULL DEFAULT '', -- ключ полного фото в S3-бакете
    photo_thumb_key text   NOT NULL DEFAULT '', -- ключ миниатюры в S3-бакете
    created_at timestamptz NOT NULL DEFAULT now(), -- когда номер зафиксирован
    lat        double precision, -- координаты места фиксации (WGS 84, градусы); NULL — неизвестны
    lon        double precision,
    is_test_data boolean   NOT NULL DEFAULT false, -- тестовые данные (миграция 0002_test_data)
    hidden     boolean     NOT NULL DEFAULT false, -- скрыта админом (rpc HideSighting): обычным пользователям не отдаётся
    CHECK ((lat IS NULL) = (lon IS NULL) AND lat BETWEEN -90 AND 90 AND lon BETWEEN -180 AND 180)
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
    plate_type   text                 NOT NULL DEFAULT '' CHECK (plate_type IN ('', 'ru', 'eu', 'us', 'ua', 'cn')), -- формат; '' — другой/неизвестен
    plate_region text                 NOT NULL DEFAULT '', -- страна (ЕС: "D") или штат (США: "CA")
    status       clarification_status NOT NULL DEFAULT 'not_accepted',
    created_at   timestamptz          NOT NULL DEFAULT now(),
    is_test_data boolean              NOT NULL DEFAULT false
);

-- Уточнения фиксации (rpc Search — EXISTS по sighting_id) и совпадающие с номером (rpc Clarify —
-- консенсус). Выражение должно совпадать с clarificationsMatching в internal/repository/postgres.go.
CREATE INDEX clarifications_sighting_plate_key_idx ON clarifications (sighting_id, plate_search_key(plate));
-- Поиск по уточнённым номерам (rpc Search) — то же выражение, что для plates.
CREATE INDEX clarifications_plate_gin_idx ON clarifications USING gin (to_tsvector('simple', plate_search_key(plate)));
-- Админ-панель (rpc ListClarifications): от новых к старым, с фильтром по статусу и без него.
CREATE INDEX clarifications_status_created_at_idx ON clarifications (status, created_at DESC, id);
CREATE INDEX clarifications_created_at_idx ON clarifications (created_at DESC, id);

-- Жалобы на фото фиксации (кнопка «Пожаловаться»); разбирает администратор (админ-панель /admin).
CREATE TYPE complaint_reason AS ENUM ('wrong_plate', 'bad_photo', 'personal_data', 'other');
CREATE TYPE complaint_status AS ENUM ('new', 'resolved');

CREATE TABLE complaints (
    id           uuid             PRIMARY KEY DEFAULT gen_random_uuid(),
    sighting_id  uuid             NOT NULL REFERENCES plate_sightings (id),
    reason       complaint_reason NOT NULL,
    comment      text             NOT NULL DEFAULT '' CHECK (char_length(comment) <= 500),
    status       complaint_status NOT NULL DEFAULT 'new',
    created_at   timestamptz      NOT NULL DEFAULT now(),
    is_test_data boolean          NOT NULL DEFAULT false -- как у фиксации, на которую жалоба
);

CREATE INDEX complaints_sighting_id_idx ON complaints (sighting_id);
-- Админ-панель (rpc ListComplaints): от новых к старым, с фильтром по статусу и без него.
CREATE INDEX complaints_status_created_at_idx ON complaints (status, created_at DESC, id);
CREATE INDEX complaints_created_at_idx ON complaints (created_at DESC, id);

-- Пользователи: вход по токену (cookie cs_token или Authorization: Bearer). Сам токен не храним —
-- только sha256 (hex). Логина и пароля нет; e-mail необязателен.
CREATE TABLE users (
    id           uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    token_hash   text        NOT NULL UNIQUE,
    email        text,
    plan         text        NOT NULL DEFAULT 'free' CHECK (plan IN ('free', 'pro')),
    pro_until    timestamptz, -- до какого момента действует Pro; NULL — не покупал
    created_at   timestamptz NOT NULL DEFAULT now(),
    is_test_data boolean     NOT NULL DEFAULT false
);

CREATE UNIQUE INDEX users_email_idx ON users (lower(email)) WHERE email IS NOT NULL;

-- Платежи за тариф. Провайдер пока один — 'mock' (демо-касса /pay/mock/<id>, деньги не списываются).
CREATE TABLE payments (
    id           uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      uuid        NOT NULL REFERENCES users (id),
    plan         text        NOT NULL CHECK (plan IN ('pro')),
    amount_cents int         NOT NULL CHECK (amount_cents >= 0),
    currency     text        NOT NULL,
    status       text        NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'succeeded', 'canceled')),
    provider     text        NOT NULL DEFAULT 'mock',
    created_at   timestamptz NOT NULL DEFAULT now(),
    confirmed_at timestamptz,
    is_test_data boolean     NOT NULL DEFAULT false -- как у пользователя
);

-- История платежей пользователя (rpc ListMyPayments).
CREATE INDEX payments_user_created_at_idx ON payments (user_id, created_at DESC, id);

-- +goose Down
DROP TABLE payments;
DROP TABLE users;
DROP TABLE complaints;
DROP TYPE complaint_status;
DROP TYPE complaint_reason;
DROP TABLE clarifications;
DROP TYPE clarification_status;
DROP TABLE plate_sightings;
DROP TABLE plates;
DROP FUNCTION plate_search_key(text);
DROP EXTENSION IF EXISTS fuzzystrmatch;
