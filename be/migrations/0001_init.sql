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

-- Заявки на расположение камеры (страница /cameras): пользователь предлагает точку, другие голосуют «+1»,
-- администратор меняет статус. contact (e-mail/телеграм) видит только администратор.
CREATE TYPE camera_request_status AS ENUM ('new', 'approved', 'rejected', 'installed');

CREATE TABLE camera_requests (
    id           uuid                  PRIMARY KEY DEFAULT gen_random_uuid(),
    lat          double precision      NOT NULL CHECK (lat BETWEEN -90 AND 90), -- точка (WGS 84, градусы)
    lon          double precision      NOT NULL CHECK (lon BETWEEN -180 AND 180),
    address      text                  NOT NULL DEFAULT '' CHECK (char_length(address) <= 300),
    reason       text                  NOT NULL CHECK (char_length(reason) BETWEEN 1 AND 1000), -- зачем камера здесь
    contact      text                  NOT NULL DEFAULT '' CHECK (char_length(contact) <= 200), -- e-mail/телеграм, необязательно
    status       camera_request_status NOT NULL DEFAULT 'new',
    votes        int                   NOT NULL DEFAULT 0 CHECK (votes >= 0),
    created_at   timestamptz           NOT NULL DEFAULT now(),
    is_test_data boolean               NOT NULL DEFAULT false
);

-- Список (rpc ListCameraRequests): от новых к старым; рамка карты — фильтр по lat/lon.
CREATE INDEX camera_requests_created_at_idx ON camera_requests (created_at DESC, id);
CREATE INDEX camera_requests_lat_lon_idx ON camera_requests (lat, lon);

-- API-токены (админка, вкладка «Токены»). Токен = "cst_" + base64url(payload) + "." + base64url(HMAC-SHA256(payload,
-- TOKEN_SIGNING_KEY)), payload = {"tid": id, "iat": unix}. Ни токен, ни подпись в открытом виде не храним:
-- signature_hash = hex(sha256(подпись)). Остаток запросов — requests_left (списывает rpc Get).
CREATE TABLE api_tokens (
    id             uuid        PRIMARY KEY,
    name           text        NOT NULL CHECK (char_length(name) BETWEEN 1 AND 100),
    signature_hash text        NOT NULL UNIQUE,
    requests_total integer     NOT NULL CHECK (requests_total > 0),
    requests_left  integer     NOT NULL CHECK (requests_left >= 0),
    revoked        boolean     NOT NULL DEFAULT false,
    created_at     timestamptz NOT NULL DEFAULT now(),
    last_used_at   timestamptz, -- последний списанный запрос; NULL — ещё не использовался
    is_test_data   boolean     NOT NULL DEFAULT false
);

-- Админка (rpc ListApiTokens): от новых к старым.
CREATE INDEX api_tokens_created_at_idx ON api_tokens (created_at DESC, id);

-- Журнал открытий страницы номера (rpc Get): кто открыл и списан ли запрос из API-токена.
CREATE TABLE plate_views (
    id           uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    sighting_id  uuid        NOT NULL REFERENCES plate_sightings (id), -- открытая фиксация
    plate_id     uuid        NOT NULL REFERENCES plates (id),
    token_id     uuid        REFERENCES api_tokens (id), -- API-токен запроса; NULL — без токена
    consumed     boolean     NOT NULL DEFAULT false, -- списан ли запрос из токена
    user_id      uuid        REFERENCES users (id), -- пользователь (токен "cs_…"); NULL — не вошёл
    is_admin     boolean     NOT NULL DEFAULT false, -- открыл администратор (admin-token): квота не списывается
    user_agent   text        NOT NULL DEFAULT '' CHECK (char_length(user_agent) <= 300),
    created_at   timestamptz NOT NULL DEFAULT now()
);

-- Журнал токена (rpc ListPlateViews с token_id, views_count в ListApiTokens) и журнал номера.
CREATE INDEX plate_views_token_created_at_idx ON plate_views (token_id, created_at DESC);
CREATE INDEX plate_views_plate_created_at_idx ON plate_views (plate_id, created_at DESC);
-- Общий журнал без фильтра.
CREATE INDEX plate_views_created_at_idx ON plate_views (created_at DESC, id);

-- Кто загрузил фиксацию (rpc Ingest) и когда. created_at — время съёмки (может быть в прошлом, из EXIF),
-- поэтому суточная квота загрузок считается по uploaded_at. NULL — не загрузка пользователя (тестовые данные).
ALTER TABLE plate_sightings
    ADD COLUMN uploaded_by uuid REFERENCES users (id),
    ADD COLUMN uploaded_at timestamptz;
-- Квота загрузок: число загрузок пользователя за последние сутки.
CREATE INDEX plate_sightings_uploaded_by_idx ON plate_sightings (uploaded_by, uploaded_at) WHERE uploaded_by IS NOT NULL;

-- +goose Down
DROP TABLE plate_views;
DROP TABLE api_tokens;
DROP TABLE camera_requests;
DROP TYPE camera_request_status;
ALTER TABLE plate_sightings DROP COLUMN uploaded_by, DROP COLUMN uploaded_at;
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
