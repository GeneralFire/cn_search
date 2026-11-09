-- Тестовые данные: 600 фиксаций номеров из пула (~ треть от числа фиксаций), у номера обычно
-- несколько фиксаций в разных местах его города. Применяется `make sample-data` поверх схемы
-- (be/migrations); версии — в отдельной таблице goose_test_data_version.
-- Картинки car-1..5.png заливает в S3 сервис sample-data-s3 (см. docker-compose.yml).

-- +goose Up
-- Заменяет всё, что было в таблицах (make sample-data заливает набор заново).
TRUNCATE plate_sightings, plates;

SELECT setseed(0.20261109); -- одинаковый набор при каждой заливке

CREATE TEMP TABLE cities (n int PRIMARY KEY, name text, regions text[], streets text[]) ON COMMIT DROP;
INSERT INTO cities VALUES
  (1, 'Москва', '{77,97,99,177,197,199,777,797,799,977}',
      '{"Пресненская наб.","ул. Тверская","Ленинградский пр.","ул. Арбат","Кутузовский пр.","ул. Покровка","Профсоюзная ул.","ул. Большая Ордынка"}'),
  (2, 'Санкт-Петербург', '{78,98,178,198}',
      '{"Невский пр.","Лиговский пр.","ул. Рубинштейна","Московский пр.","наб. реки Фонтанки","Большой пр. П.С."}'),
  (3, 'Казань', '{16,116,716}', '{"ул. Баумана","ул. Пушкина","пр. Победы","ул. Петербургская"}'),
  (4, 'Екатеринбург', '{66,96,196}', '{"ул. Ленина","ул. Малышева","ул. Вайнера","пр. Космонавтов"}'),
  (5, 'Новосибирск', '{54,154}', '{"Красный пр.","ул. Ленина","ул. Кирова","Вокзальная магистраль"}'),
  (6, 'Нижний Новгород', '{52,152}', '{"ул. Большая Покровская","ул. Рождественская","пр. Гагарина"}'),
  (7, 'Краснодар', '{23,93,123,193}', '{"ул. Красная","ул. Северная","ул. Ставропольская"}'),
  (8, 'Самара', '{63,163}', '{"ул. Куйбышева","Московское ш.","ул. Ленинградская"}');

-- Фиксированные записи для ручной проверки: у А123ВС 777 три фиксации
-- (id …0001, …0004, …0005). id фиксированных — предсказуемые uuid, остальных — gen_random_uuid().
INSERT INTO plates (plate, meta) VALUES
  ('А123ВС 777', '{"city": "Москва"}'),
  ('М456ОР 178', '{"city": "Санкт-Петербург"}'),
  ('Т789УХ 116', '{"city": "Казань"}');

INSERT INTO plate_sightings (id, plate_id, address, photo_key, created_at)
SELECT ('00000000-0000-0000-0000-00000000000' || v.n)::uuid, p.id, v.address,
       'car-' || ((v.n * 7) % 5 + 1) || '.png', v.created_at
FROM (VALUES
  (1, 'А123ВС 777', 'Москва, Пресненская наб., 10',     timestamptz '2026-11-02 09:15:00Z'),
  (2, 'М456ОР 178', 'Санкт-Петербург, Невский пр., 28', timestamptz '2026-10-27 18:40:00Z'),
  (3, 'Т789УХ 116', 'Казань, ул. Баумана, 44',          timestamptz '2026-11-08 12:05:00Z'),
  (4, 'А123ВС 777', 'Москва, ул. Тверская, 7',          timestamptz '2026-11-05 18:30:00Z'),
  (5, 'А123ВС 777', 'Москва, Кутузовский пр., 30',      timestamptz '2026-10-20 08:00:00Z')
) AS v (n, plate, address, created_at)
JOIN plates p ON p.plate = v.plate;

-- Пул случайных номеров: корректные буквы (12 «латинских» кириллицы) и коды регионов города.
INSERT INTO plates (plate, meta)
SELECT DISTINCT ON (plate) plate, jsonb_build_object('city', name)
FROM (
  SELECT c.name,
         substr(l, 1 + floor(random() * 12)::int, 1)
           || lpad((1 + floor(random() * 999))::int::text, 3, '0')
           || substr(l, 1 + floor(random() * 12)::int, 1)
           || substr(l, 1 + floor(random() * 12)::int, 1)
           || ' ' || c.regions[1 + floor(random() * array_length(c.regions, 1))::int] AS plate
  FROM (SELECT g, 1 + floor(random() * 8)::int AS ci FROM generate_series(1, 210) g) r
  JOIN cities c ON c.n = r.ci
  CROSS JOIN (SELECT 'АВЕКМНОРСТУХ' AS l) letters
) candidates
WHERE plate NOT IN (SELECT plate FROM plates)
ORDER BY plate
LIMIT 198;

-- 595 случайных фиксаций: номер из пула, адрес в его городе, время — за 180 дней до 09.11.2026 21:00 UTC.
CREATE TEMP TABLE pool ON COMMIT DROP AS
SELECT row_number() OVER (ORDER BY plate) AS rn, id, meta ->> 'city' AS city
FROM plates WHERE plate NOT IN ('А123ВС 777', 'М456ОР 178', 'Т789УХ 116');

INSERT INTO plate_sightings (id, plate_id, address, photo_key, created_at)
SELECT gen_random_uuid(),
       pool.id,
       c.name || ', ' || c.streets[1 + floor(random() * array_length(c.streets, 1))::int]
              || ', ' || (1 + floor(random() * 150))::int,
       'car-' || (((5 + r.g) * 7) % 5 + 1) || '.png',
       date_trunc('second', timestamptz '2026-11-09 21:00:00Z' - random() * interval '180 days')
FROM (SELECT g, 1 + floor(random() * (SELECT count(*) FROM pool))::int AS pi FROM generate_series(1, 595) g) r
JOIN pool ON pool.rn = r.pi
JOIN cities c ON c.name = pool.city;

-- +goose Down
TRUNCATE plate_sightings, plates;
