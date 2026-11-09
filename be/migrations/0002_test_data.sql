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

-- Центры городов (WGS 84): точка случайной фиксации — центр + разброс в несколько км.
-- Адрес улицы с точкой может не совпадать — для тестовых данных это допустимо.
CREATE TEMP TABLE city_centers (name text PRIMARY KEY, lat double precision, lon double precision) ON COMMIT DROP;
INSERT INTO city_centers
SELECT c.name, v.lat, v.lon
FROM (VALUES (1, 55.7558, 37.6173), (2, 59.9343, 30.3351), (3, 55.7963, 49.1088), (4, 56.8389, 60.6057),
             (5, 55.0302, 82.9204), (6, 56.3269, 44.0059), (7, 45.0355, 38.9753), (8, 53.1959, 50.1002)
) AS v (n, lat, lon)
JOIN cities c ON c.n = v.n;

-- Фиксированные записи для ручной проверки: у А123ВС 777 три фиксации
-- (id …0001, …0004, …0005). id фиксированных — предсказуемые uuid, остальных — gen_random_uuid().
INSERT INTO plates (plate, meta, is_test_data) VALUES
  ('А123ВС 777', '{"type": "ru", "city": "Москва"}', true),
  ('М456ОР 178', '{"type": "ru", "city": "Санкт-Петербург"}', true),
  ('Т789УХ 116', '{"type": "ru", "city": "Казань"}', true);

INSERT INTO plate_sightings (id, plate_id, address, photo_key, photo_thumb_key, created_at, lat, lon, is_test_data)
SELECT ('00000000-0000-0000-0000-00000000000' || v.n)::uuid, p.id, v.address,
       'car-' || ((v.n * 7) % 10 + 1) || '.jpg', 'car-' || ((v.n * 7) % 10 + 1) || '_thumb.jpg',
       v.created_at, v.lat, v.lon, true
FROM (VALUES
  (1, 'А123ВС 777', 'Москва, Пресненская наб., 10',     timestamptz '2026-11-02 09:15:00Z', 55.7490, 37.5395),
  (2, 'М456ОР 178', 'Санкт-Петербург, Невский пр., 28', timestamptz '2026-10-27 18:40:00Z', 59.9357, 30.3260),
  (3, 'Т789УХ 116', 'Казань, ул. Баумана, 44',          timestamptz '2026-11-08 12:05:00Z', 55.7887, 49.1170),
  (4, 'А123ВС 777', 'Москва, ул. Тверская, 7',          timestamptz '2026-11-05 18:30:00Z', 55.7579, 37.6115),
  (5, 'А123ВС 777', 'Москва, Кутузовский пр., 30',      timestamptz '2026-10-20 08:00:00Z', 55.7410, 37.5370)
) AS v (n, plate, address, created_at, lat, lon)
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

-- Координаты случайных фиксаций: центр города ± ~0.05° широты и ~0.08° долготы (несколько км).
-- Отдельным UPDATE, чтобы не сдвигать последовательность random() для номеров, адресов и времени выше.
UPDATE plate_sightings s
SET lat = round((cc.lat + (random() - 0.5) * 0.1)::numeric, 5),
    lon = round((cc.lon + (random() - 0.5) * 0.16)::numeric, 5)
FROM plates p
JOIN city_centers cc ON cc.name = p.meta ->> 'city'
WHERE s.plate_id = p.id AND s.is_test_data AND s.lat IS NULL;

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

INSERT INTO plate_sightings (plate_id, address, address_locale, photo_key, photo_thumb_key, created_at, lat, lon, is_test_data)
SELECT p.id, v.address, v.locale,
       'car-' || v.photo || '.jpg', 'car-' || v.photo || '_thumb.jpg', v.created_at, v.lat, v.lon, true
FROM (VALUES
  ('B MW 1234', 'Berlin, Unter den Linden 12',        'de', 1, timestamptz '2026-11-09 08:10:00Z', 52.5170, 13.3889),
  ('B MW 1234', 'Berlin, Friedrichstraße 43',         'de', 1, timestamptz '2026-11-03 17:45:00Z', 52.5070, 13.3903),
  ('B MW 1234', 'Berlin, Kurfürstendamm 101',         'de', 1, timestamptz '2026-10-15 12:20:00Z', 52.4985, 13.2930),
  ('B KL 77',   'Berlin, Alexanderplatz 3',           'de', 7, timestamptz '2026-11-07 19:05:00Z', 52.5219, 13.4132),
  ('M AB 4021', 'München, Leopoldstraße 58',          'de', 4, timestamptz '2026-11-06 09:30:00Z', 48.1610, 11.5860),
  ('M AB 4021', 'München, Maximilianstraße 9',        'de', 4, timestamptz '2026-10-29 14:00:00Z', 48.1393, 11.5806),
  ('HH CS 908', 'Hamburg, Jungfernstieg 21',          'de', 2, timestamptz '2026-11-01 11:11:00Z', 53.5525, 9.9925),
  ('7XYZ123',   'Los Angeles, 6801 Hollywood Blvd',   'en', 8, timestamptz '2026-11-08 22:40:00Z', 34.1022, -118.3404),
  ('7XYZ123',   'Los Angeles, 200 Santa Monica Pier', 'en', 8, timestamptz '2026-11-02 18:15:00Z', 34.0094, -118.4973),
  ('8LMN472',   'San Francisco, 1 Ferry Building',    'en', 3, timestamptz '2026-11-05 07:55:00Z', 37.7955, -122.3937),
  ('KDT-4419',  'New York, 350 5th Ave',              'en', 6, timestamptz '2026-11-04 13:25:00Z', 40.7484, -73.9857),
  ('KDT-4419',  'New York, 89 E 42nd St',             'en', 6, timestamptz '2026-10-25 08:45:00Z', 40.7527, -73.9772),
  ('JPX 5520',  'Austin, 1100 Congress Ave',          'en', 5, timestamptz '2026-10-30 16:35:00Z', 30.2747, -97.7404),
  ('ТАКСИ 01',  'Москва, Тверская ул., 1',            'ru', 9, timestamptz '2026-11-09 06:00:00Z', 55.7570, 37.6145)
) AS v (plate, address, locale, photo, created_at, lat, lon)
JOIN plates p ON p.plate = v.plate;

-- Номера Украины и Китая (рынки главной для языков uk и zh) с адресами на языке страны и пара номеров
-- других стран ЕС — для вкладки «Прочее» (вместе с ТАКСИ 01 без формата).
INSERT INTO plates (plate, meta, is_test_data) VALUES
  ('АА 1234 ВС', '{"type": "ua", "city": "Київ"}', true),
  ('КА 7781 АМ', '{"type": "ua", "city": "Київ"}', true),
  ('ВС 4502 ІК', '{"type": "ua", "city": "Львів"}', true),
  ('ВН 3388 ЕК', '{"type": "ua", "city": "Одеса"}', true),
  ('АХ 9061 НІ', '{"type": "ua", "city": "Харків"}', true),
  ('京A·12345',  '{"type": "cn", "city": "北京"}', true),
  ('沪B·67890',  '{"type": "cn", "city": "上海"}', true),
  ('粤B·D1234',  '{"type": "cn", "city": "深圳"}', true),
  ('AB-123-CD',  '{"type": "eu", "region": "F", "city": "Paris"}', true),
  ('WA 12345',   '{"type": "eu", "region": "PL", "city": "Warszawa"}', true);

INSERT INTO plate_sightings (plate_id, address, address_locale, photo_key, photo_thumb_key, created_at, lat, lon, is_test_data)
SELECT p.id, v.address, v.locale,
       'car-' || v.photo || '.jpg', 'car-' || v.photo || '_thumb.jpg', v.created_at, v.lat, v.lon, true
FROM (VALUES
  ('АА 1234 ВС', 'Київ, вул. Хрещатик, 22',          'uk', 3, timestamptz '2026-11-09 10:20:00Z', 50.4474, 30.5219),
  ('АА 1234 ВС', 'Київ, Майдан Незалежності, 1',     'uk', 3, timestamptz '2026-11-04 15:05:00Z', 50.4501, 30.5234),
  ('КА 7781 АМ', 'Київ, просп. Перемоги, 37',        'uk', 6, timestamptz '2026-11-07 08:40:00Z', 50.4489, 30.4569),
  ('ВС 4502 ІК', 'Львів, пл. Ринок, 1',              'uk', 2, timestamptz '2026-11-08 17:30:00Z', 49.8419, 24.0316),
  ('ВС 4502 ІК', 'Львів, просп. Свободи, 28',        'uk', 2, timestamptz '2026-10-31 12:10:00Z', 49.8429, 24.0293),
  ('ВН 3388 ЕК', 'Одеса, вул. Дерибасівська, 10',    'uk', 9, timestamptz '2026-11-05 19:45:00Z', 46.4846, 30.7385),
  ('АХ 9061 НІ', 'Харків, вул. Сумська, 46',         'uk', 5, timestamptz '2026-11-02 07:15:00Z', 50.0035, 36.231),
  ('京A·12345',  '北京, 东城区东长安街1号',           'zh', 4, timestamptz '2026-11-09 03:30:00Z', 39.9085, 116.4108),
  ('京A·12345',  '北京, 朝阳区建国路87号',            'zh', 4, timestamptz '2026-11-01 09:00:00Z', 39.9087, 116.474),
  ('沪B·67890',  '上海, 黄浦区南京东路100号',         'zh', 7, timestamptz '2026-11-06 11:50:00Z', 31.24, 121.488),
  ('粤B·D1234',  '深圳, 福田区深南大道6008号',        'zh', 1, timestamptz '2026-11-03 06:25:00Z', 22.542, 114.057),
  ('AB-123-CD',  'Paris, 8 Rue de Rivoli',           'fr', 10, timestamptz '2026-11-06 16:00:00Z', 48.8559, 2.3593),
  ('WA 12345',   'Warszawa, ul. Marszałkowska 104',  'pl', 8, timestamptz '2026-10-28 13:35:00Z', 52.2319, 21.0103)
) AS v (plate, address, locale, photo, created_at, lat, lon)
JOIN plates p ON p.plate = v.plate;

-- Страховка: у каждой тестовой фиксации должны быть координаты — иначе на странице номера нет карты.
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM plate_sightings WHERE is_test_data AND lat IS NULL) THEN
    RAISE EXCEPTION 'тестовые фиксации без координат: добавьте lat/lon в 0002_test_data.sql';
  END IF;
END $$;
-- +goose StatementEnd

-- Уточнения: к фиксации Т789УХ 116 предложили Т789УХ 716 (принято), к М456ОР 178 — М456ОР 198 (не принято).
INSERT INTO clarifications (sighting_id, plate, status, created_at, is_test_data) VALUES
  ('00000000-0000-0000-0000-000000000003', 'Т789УХ 716', 'accepted',     timestamptz '2026-11-08 13:00:00Z', true),
  ('00000000-0000-0000-0000-000000000002', 'М456ОР 198', 'not_accepted', timestamptz '2026-10-28 10:00:00Z', true);

-- Жалобы на фото: к А123ВС 777 (…0001) — персональные данные в кадре (новая), к Казани (…0003) — плохое фото (решена).
INSERT INTO complaints (id, sighting_id, reason, comment, status, created_at, is_test_data) VALUES
  ('00000000-0000-0000-0000-0000000c0001', '00000000-0000-0000-0000-000000000001', 'personal_data',
   'В кадре видно лицо пешехода', 'new', timestamptz '2026-11-09 10:00:00Z', true),
  ('00000000-0000-0000-0000-0000000c0002', '00000000-0000-0000-0000-000000000003', 'bad_photo',
   '', 'resolved', timestamptz '2026-11-08 15:30:00Z', true),
  ('00000000-0000-0000-0000-0000000c0003', '00000000-0000-0000-0000-000000000004', 'wrong_plate',
   'На фото другая машина', 'new', timestamptz '2026-11-06 09:00:00Z', true);

-- +goose Down
DELETE FROM complaints WHERE is_test_data;
DELETE FROM clarifications WHERE is_test_data;
DELETE FROM plate_sightings WHERE is_test_data;
DELETE FROM plates WHERE is_test_data;
