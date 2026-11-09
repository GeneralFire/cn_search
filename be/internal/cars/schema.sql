-- Схема применяется идемпотентно командой `be migrate` (и перед `be seed`).
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE IF NOT EXISTS cars (
    id         integer PRIMARY KEY,
    address    text    NOT NULL,
    plate      text    NOT NULL UNIQUE, -- как показываем: "А123ВС 777"
    key        text    NOT NULL UNIQUE, -- как показываем: "KEY-0451"
    date       date    NOT NULL,
    photo_key  text    NOT NULL DEFAULT '', -- ключ объекта в S3-бакете
    plate_norm text    NOT NULL, -- нормализованный номер для поиска: "А123ВС777"
    key_norm   text    NOT NULL  -- нормализованный ключ для поиска: "KEY-0451"
);

-- Порядок выдачи: от новых к старым.
CREATE INDEX IF NOT EXISTS cars_date_id_idx ON cars (date DESC, id);

-- Поиск по подстроке (LIKE '%…%').
CREATE INDEX IF NOT EXISTS cars_plate_norm_trgm_idx ON cars USING gin (plate_norm gin_trgm_ops);
CREATE INDEX IF NOT EXISTS cars_key_norm_trgm_idx ON cars USING gin (key_norm gin_trgm_ops);
