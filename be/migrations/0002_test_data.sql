-- Тестовые данные: 600 фиксаций номеров из пула (~ треть от числа фиксаций), у номера обычно
-- несколько фиксаций в разных местах его города. Все строки помечены is_test_data = true:
-- Down удаляет только их, а в бою их можно отфильтровать или удалить тем же условием.
-- Фото car-N.jpg и миниатюры car-N_thumb.jpg (N = 1..10) заливает в S3 сервис s3-init (be/internal/test_utils/s3, docker-compose.yml).

-- +goose Up
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
INSERT INTO plates (plate, meta, is_test_data) VALUES
  ('А123ВС 777', '{"type": "ru", "city": "Москва"}', true),
  ('М456ОР 178', '{"type": "ru", "city": "Санкт-Петербург"}', true),
  ('Т789УХ 116', '{"type": "ru", "city": "Казань"}', true);

INSERT INTO plate_sightings (id, plate_id, address, photo_key, photo_thumb_key, created_at, is_test_data)
SELECT ('00000000-0000-0000-0000-00000000000' || v.n)::uuid, p.id, v.address,
       'car-' || ((v.n * 7) % 10 + 1) || '.jpg', 'car-' || ((v.n * 7) % 10 + 1) || '_thumb.jpg',
       v.created_at, true
FROM (VALUES
  (1, 'А123ВС 777', 'Москва, Пресненская наб., 10',     timestamptz '2026-11-02 09:15:00Z'),
  (2, 'М456ОР 178', 'Санкт-Петербург, Невский пр., 28', timestamptz '2026-10-27 18:40:00Z'),
  (3, 'Т789УХ 116', 'Казань, ул. Баумана, 44',          timestamptz '2026-11-08 12:05:00Z'),
  (4, 'А123ВС 777', 'Москва, ул. Тверская, 7',          timestamptz '2026-11-05 18:30:00Z'),
  (5, 'А123ВС 777', 'Москва, Кутузовский пр., 30',      timestamptz '2026-10-20 08:00:00Z')
) AS v (n, plate, address, created_at)
JOIN plates p ON p.plate = v.plate;

-- Пул случайных номеров: корректные буквы (12 «латинских» кириллицы) и коды регионов города.
INSERT INTO plates (plate, meta, is_test_data)
SELECT DISTINCT ON (plate) plate, jsonb_build_object('type', 'ru', 'city', name), true
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
FROM plates WHERE is_test_data AND plate NOT IN ('А123ВС 777', 'М456ОР 178', 'Т789УХ 116');

INSERT INTO plate_sightings (id, plate_id, address, photo_key, photo_thumb_key, created_at, is_test_data)
SELECT gen_random_uuid(),
       pool.id,
       c.name || ', ' || c.streets[1 + floor(random() * array_length(c.streets, 1))::int]
              || ', ' || (1 + floor(random() * 150))::int,
       'car-' || (((5 + r.g) * 7) % 10 + 1) || '.jpg',
       'car-' || (((5 + r.g) * 7) % 10 + 1) || '_thumb.jpg',
       date_trunc('second', timestamptz '2026-11-09 21:00:00Z' - random() * interval '180 days'),
       true
FROM (SELECT g, 1 + floor(random() * (SELECT count(*) FROM pool))::int AS pi FROM generate_series(1, 595) g) r
JOIN pool ON pool.rn = r.pi
JOIN cities c ON c.name = pool.city;

-- Номера не российского формата: ЕС (Германия) и США — с фиксациями в своих городах,
-- адреса на языке страны (address_locale). Один номер без типа — формат неизвестен (рисуется по умолчанию).
INSERT INTO plates (plate, meta, is_test_data) VALUES
  ('B MW 1234', '{"type": "eu", "region": "D", "city": "Berlin"}', true),
  ('B KL 77',   '{"type": "eu", "region": "D", "city": "Berlin"}', true),
  ('M AB 4021', '{"type": "eu", "region": "D", "city": "München"}', true),
  ('HH CS 908', '{"type": "eu", "region": "D", "city": "Hamburg"}', true),
  ('7XYZ123',   '{"type": "us", "region": "CA", "city": "Los Angeles"}', true),
  ('8LMN472',   '{"type": "us", "region": "CA", "city": "San Francisco"}', true),
  ('KDT-4419',  '{"type": "us", "region": "NY", "city": "New York"}', true),
  ('JPX 5520',  '{"type": "us", "region": "TX", "city": "Austin"}', true),
  ('ТАКСИ 01',  '{"city": "Москва"}', true);

INSERT INTO plate_sightings (plate_id, address, address_locale, photo_key, photo_thumb_key, created_at, is_test_data)
SELECT p.id, v.address, v.locale,
       'car-' || v.photo || '.jpg', 'car-' || v.photo || '_thumb.jpg', v.created_at, true
FROM (VALUES
  ('B MW 1234', 'Berlin, Unter den Linden 12',        'de', 1, timestamptz '2026-11-09 08:10:00Z'),
  ('B MW 1234', 'Berlin, Friedrichstraße 43',         'de', 1, timestamptz '2026-11-03 17:45:00Z'),
  ('B MW 1234', 'Berlin, Kurfürstendamm 101',         'de', 1, timestamptz '2026-10-15 12:20:00Z'),
  ('B KL 77',   'Berlin, Alexanderplatz 3',           'de', 7, timestamptz '2026-11-07 19:05:00Z'),
  ('M AB 4021', 'München, Leopoldstraße 58',          'de', 4, timestamptz '2026-11-06 09:30:00Z'),
  ('M AB 4021', 'München, Maximilianstraße 9',        'de', 4, timestamptz '2026-10-29 14:00:00Z'),
  ('HH CS 908', 'Hamburg, Jungfernstieg 21',          'de', 2, timestamptz '2026-11-01 11:11:00Z'),
  ('7XYZ123',   'Los Angeles, 6801 Hollywood Blvd',   'en', 8, timestamptz '2026-11-08 22:40:00Z'),
  ('7XYZ123',   'Los Angeles, 200 Santa Monica Pier', 'en', 8, timestamptz '2026-11-02 18:15:00Z'),
  ('8LMN472',   'San Francisco, 1 Ferry Building',    'en', 3, timestamptz '2026-11-05 07:55:00Z'),
  ('KDT-4419',  'New York, 350 5th Ave',              'en', 6, timestamptz '2026-11-04 13:25:00Z'),
  ('KDT-4419',  'New York, 89 E 42nd St',             'en', 6, timestamptz '2026-10-25 08:45:00Z'),
  ('JPX 5520',  'Austin, 1100 Congress Ave',          'en', 5, timestamptz '2026-10-30 16:35:00Z'),
  ('ТАКСИ 01',  'Москва, Тверская ул., 1',            'ru', 9, timestamptz '2026-11-09 06:00:00Z')
) AS v (plate, address, locale, photo, created_at)
JOIN plates p ON p.plate = v.plate;

-- Уточнения: к фиксации Т789УХ 116 предложили Т789УХ 716 (принято), к М456ОР 178 — М456ОР 198 (не принято).
INSERT INTO clarifications (sighting_id, plate, status, created_at, is_test_data) VALUES
  ('00000000-0000-0000-0000-000000000003', 'Т789УХ 716', 'accepted',     timestamptz '2026-11-08 13:00:00Z', true),
  ('00000000-0000-0000-0000-000000000002', 'М456ОР 198', 'not_accepted', timestamptz '2026-10-28 10:00:00Z', true);

-- +goose Down
DELETE FROM clarifications WHERE is_test_data;
DELETE FROM plate_sightings WHERE is_test_data;
DELETE FROM plates WHERE is_test_data;
