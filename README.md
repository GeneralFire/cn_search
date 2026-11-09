# Поиск по авто

Поиск фиксаций автомобиля (где и когда замечен) по госномеру: бэкенд на Go (контракт в proto, ручки генерирует
[clay](https://github.com/utrack/clay)), Postgres, S3-хранилище картинок и фронтенд на
TypeScript + Vite с TS-клиентом, сгенерированным по `swagger.json`.

> **Pet-проект.** Зависимости — только из публичных реестров
> (npmjs.org, proxy.golang.org, Docker Hub). Подробнее — в
> [«Зависимости и окружение»](#зависимости-и-окружение) и `CLAUDE.md`.

## Быстрый старт

Нужны Docker (Docker Desktop / Colima / OrbStack), Docker Compose и `make`.

```bash
make run    # поднять всё: db + s3 (+ миграции), be, fe
make sample-data   # тестовые данные: 600 фиксаций в Postgres, 5 картинок в S3
```

По отдельности:

```bash
make etc      # только БД и S3 + миграции goose
make run-be   # только бэк (пересборка); требует make etc
make run-fe   # только фронт; без бэка страница откроется, но поиск вернёт ошибку
make stop-fe  # / stop-be / stop-etc — остановить часть; make down — всё
```

Открыть: **http://localhost:7173** (из локальной сети — `http://<ваш-IP>:7173`).

| Сервис | Что | Адрес |
|---|---|---|
| `fe` | Vite dev-сервер, hot-reload | `0.0.0.0:7173` — единственный порт, нужный браузеру |
| `be` | HTTP-ручки clay + gRPC | `127.0.0.1:8080` (HTTP), `127.0.0.1:8082` (gRPC) |
| `db` | Postgres 17 | `127.0.0.1:5432`, БД `carsearch` |
| `s3` | [RustFS](https://github.com/rustfs/rustfs) — S3-совместимое хранилище | `127.0.0.1:9000` (S3 API), `127.0.0.1:9001` (консоль) |
| `migrate` | [goose](https://github.com/pressly/goose) (`kukymbr/goose-docker`): `goose up` при `make etc` / `make run`, затем завершается | — |

Данные БД и S3 лежат на хосте в `./.data` (bind mount, в `.gitignore`): переживают перезапуск,
`docker-compose down -v`, очистку томов и пересоздание VM Docker. Удаляет их только `make reset`.

Браузер ходит только на фронт: Vite проксирует `/api/*` → `be`, `/s3/*` → `s3`.
Поэтому коллегам в сети достаточно одного открытого порта.

> MinIO не используется: образы `minio/minio` больше не публикуются на Docker Hub.
> RustFS совместим с ним по API — бэкенд работает через `minio-go`.

## Структура

```
car-search/
├── Makefile                 # все команды: make help
├── docker-compose.yml       # db + s3 + be + fe
├── be/                      # бэкенд, Go
│   ├── api/carsearch/carsearch.proto      # КОНТРАКТ (без HTTP-аннотаций)
│   ├── schema/carsearch.swagger.json      # сгенерирован clay
│   ├── pkg/carsearch/       # сгенерировано: *.pb.go, *_grpc.pb.go, *.pb.goclay.go (ручки)
│   ├── cmd/
│   │   └── main.go          # be: HTTP (ручки clay) + gRPC, сборка слоёв
│   ├── internal/
│   │   ├── implementation/  # RPC CarSearchService, файл на метод (руками, clay её не генерирует)
│   │   ├── usecase/         # сценарии; управляет транзакциями (InReadTx / InWriteTx)
│   │   ├── repository/      # plain SQL к Postgres; tx — вторым аргументом каждого метода
│   │   ├── db/              # Executor (Select/Get/Exec, *Sqr) и Cluster (транзакции)
│   │   ├── models/          # доменные модели (теги db)
│   │   ├── test_utils/      # тестовые данные: migrations/ (goose) и s3/ (картинки + upload.sh)
│   │   └── s3/              # S3-клиент
│   ├── migrations/          # SQL-миграции схемы (goose): 0001_init.sql (в бинарник не входят)
├── fe/                      # фронтенд, TypeScript + Vite
│   ├── src/api/generated.ts # сгенерированный TS-клиент (не править)
│   ├── src/api/index.ts     # обёртка над клиентом
│   ├── src/main.ts          # UI: поиск, бесконечная прокрутка, модалка
└── mockups/                 # HTML-макеты дизайна
```

## Makefile

| Команда | Что делает |
|---|---|
| `make run` | поднять всё: `etc` → `run-be` → `run-fe` |
| `make etc` | БД и S3 + миграции goose |
| `make run-be` / `run-fe` | собрать и поднять только бэк / только фронт (`--no-deps`) |
| `make stop-be` / `stop-fe` / `stop-etc` | остановить часть стека |
| `make down` | остановить всё; данные остаются в `./.data` |
| `make reset` | остановить и **удалить** данные БД и S3 (`./.data`) |
| `make sample-data` | тестовые данные из `be/internal/test_utils`: миграция goose в БД + картинки в S3; `CMD=redo` — залить заново |
| `make migrate` | миграции goose вручную; `CMD=status\|down\|redo\|version`, `CMD=up-to ARG=2` |
| `make migration NAME=add_vin` | создать следующую миграцию `be/migrations/000N_add_vin.sql` |
| `make psql` | консоль Postgres |
| `make logs` / `ps` | логи / статус |
| `make tools` | поставить `protoc-gen-go`, `-go-grpc`, `-goclay`, `minimock` в `be/bin` и npm-зависимости |
| `make gen` | всё: `gen-go` + `gen-ts` + `gen-mocks` |
| `make gen-go` | proto → Go + ручки clay + `swagger.json` |
| `make gen-ts` | `swagger.json` → `fe/src/api/generated.ts` |
| `make gen-mocks` | моки minimock по `//go:generate` в `contracts.go` → `<пакет>/mocks/*_mock.go` |
| `make test` | тесты бэка (`go test -race`) и проверка типов фронта |
| `make check-registries` | в `package-lock.json` только registry.npmjs.org |

С плагином Compose v2: `make DC="docker compose" up`.

## API

Контракт — `be/api/carsearch/carsearch.proto` (`package carsearch`), **без HTTP-аннотаций**.
clay генерирует ручки с флагом `generate_unbound_methods`: каждый метод доступен как
`POST /<package>.<Service>/<Method>` с JSON-телом; бэкенд монтирует их под `/api`.

| Метод | HTTP |
|---|---|
| `Search` | `POST /api/carsearch.CarSearchService/Search` |
| `Get` | `POST /api/carsearch.CarSearchService/Get` |
| — | `GET /api/swagger.json` — спецификация |
| — | `GET /api/health` |

Тот же сервис доступен по gRPC на `:8082` (с reflection: `grpcurl -plaintext localhost:8082 list`).

### Search

```jsonc
// запрос — все поля необязательны
{ "plate": "А123ВС 777", "offset": 0, "limit": 35 }

// ответ
{
  "items": [{ "id": "00000000-0000-0000-0000-000000000004", "photoUrl": "/s3/cars/car-3.png", "address": "Москва, …",
              "plate": "А123ВС 777", "createdAt": "2026-11-02T09:15:00Z" }],
  "total": 1,          // всего найдено — есть всегда, в т. ч. 0
  "nextOffset": 35     // смещение следующей страницы; нет поля — дальше пусто
}
```

- Строка `plate` без обработки уходит в `plainto_tsquery('simple', $1)` и сравнивается с
  `to_tsvector('simple', plate)`: каждое слово запроса должно совпасть со словом номера целиком —
  серией (`А123ВС`) или регионом (`777`). Регистр не важен. `А12` и латиница не находят ничего.
  Пусто — все записи. `plainto_tsquery`, а не `to_tsquery`: последний падает на обычном вводе с пробелом.
- От новых к старым (`createdAt`). `limit`: 1–100, `0` — по умолчанию 35 (кратно 5 и 7 — колонкам сетки).
- Элемент `items` — `PlateSighting`, фиксация номера.

### Get

По id любой фиксации — **все фиксации того же номера**, от новых к старым:

```jsonc
// запрос
{ "id": "00000000-0000-0000-0000-000000000001" }

// ответ
{ "items": [
  { "id": "…0004", "address": "Москва, ул. Тверская, 7",     "plate": "А123ВС 777", "createdAt": "2026-11-05T18:30:00Z", … },
  { "id": "…0001", "address": "Москва, Пресненская наб., 10", "plate": "А123ВС 777", "createdAt": "2026-11-02T09:15:00Z", … },
  { "id": "…0005", "address": "Москва, Кутузовский пр., 30",  "plate": "А123ВС 777", "createdAt": "2026-10-20T08:00:00Z", … }
] }
```

### Ошибки

```json
{ "error": "rpc error: code = InvalidArgument desc = limit: от 1 до 100 (0 — по умолчанию 35)" }
```

| HTTP | gRPC-код | Когда |
|---|---|---|
| 400 | `InvalidArgument` | неверные параметры (`limit`, `offset`, не uuid в `Get`) |
| 404 | `NotFound` | фиксации нет |
| 500 | `Internal` | ошибка сервера |
| 500 | — | битый JSON или неизвестное поле в запросе (ошибка разбора clay) |

Маршалер и формат ошибок — стандартные clay (`runtime.JSONPb{}` и `DefaultSetError`), без настройки:
нулевые поля в ответе опускаются (пустой поиск — `{}`), фронт подставляет значения по умолчанию.

## Генерация кода

```
carsearch.proto ──protoc──► be/pkg/…/*.pb.go, *_grpc.pb.go      (protoc-gen-go, -go-grpc)
                └─────────► be/pkg/…/*.pb.goclay.go              (protoc-gen-goclay: HTTP-ручки)
                └─────────► be/schema/carsearch.swagger.json     (clay, swagger_path)
                                   └──swagger-typescript-api──► fe/src/api/generated.ts
```

Поменяли proto → `make gen` → метод в `internal/implementation` → сценарий в `internal/usecase` →
запрос в `internal/repository` → использовать в `fe/src/api/index.ts`. Нужен `protoc` (`brew install protobuf`) и `make tools` один раз.
Сгенерированный код лежит в репозитории — для сборки генераторы не нужны.

clay кладёт отдельный swagger в `<goclay_out>/<swagger_path>/<имя proto>.json`. Чтобы он попал в
`be/schema`, `gen-go` передаёт `swagger_path=../schema` и флаг protoc `--unsafe_allow_out_dir_escape`
(иначе protoc запрещает выход за каталог вывода), затем переименовывает файл в `carsearch.swagger.json`.

## Фронтенд

- Поле **Номер**, поиск с задержкой 250 мс; устаревшие запросы отменяются. На карточке — дата и время фиксации.
- **Бесконечная прокрутка**: страницы по 35 подгружаются, когда низ списка подходит к экрану
  (`IntersectionObserver`). Внизу — кнопка **«Загрузить ещё»** и «Показано N из total»;
  кнопка работает и вручную, а после ошибки превращается в «Повторить».
- Сетка: карточка не уже 400px, не больше 7 колонок (5 колонок — от ~2150px ширины окна).
- На карточке — кнопка **«Уточнить»**: открывает модалку-заглушку (`<dialog>`, закрывается по Esc,
  клику на подложку и кнопкам).
- Hot-reload в Docker: `fe/` смонтирована в контейнер, Vite следит за файлами опросом
  (`VITE_USE_POLLING`) — на macOS события ФС из bind-mount ненадёжны. `node_modules` — из образа;
  после изменения `package.json`: `docker-compose up -d --build fe`.

## Данные

Схема БД — только миграции [goose](https://github.com/pressly/goose) в `be/migrations`: файлы
`NNNN_название.sql` (номера подряд с `0001`, секции `-- +goose Up` / `-- +goose Down`).
Применяет их сервис `migrate` в docker-compose (`goose up` при `make etc` / `make run`),
вручную — `make migrate [CMD=…]`; версии — в таблице `goose_db_version`. Код `be` миграции не знает.
Применённую миграцию не меняем — добавляем следующую: `make migration NAME=…`.

Таблицы:
- `plates` — номера: `id uuid` (`gen_random_uuid()`), `plate text UNIQUE` (кириллицей, `А123ВС 777`),
  `meta jsonb` (по умолчанию `{}`). GIN по `to_tsvector('simple', plate)` — поиск
  `plainto_tsquery('simple', <строка из rpc>)`, расширения Postgres не нужны.
- `plate_sightings` — фиксации: `id uuid` (`gen_random_uuid()`), `plate_id uuid → plates(id)`, `address`, `photo_key` (ключ объекта
  в S3), `created_at` (`timestamptz`). Индексы: `(created_at DESC, id)` для выдачи,
  `(plate_id, created_at DESC, id)` для `Get`.

Схема целиком — в `0001_init.sql` (до первого выката её правим на месте).

### Слои бэкенда

```
implementation (RPC, валидация, маппинг в proto)
   └─► usecase (сценарии, транзакции: InReadTx / InWriteTx)
          └─► repository (plain SQL; методы принимают tx db.Executor вторым аргументом)
                 └─► db (Executor поверх pgx.Tx: Select/Get/Exec + SelectSqr/GetSqr/ExecSqr; Cluster)
```

- Интерфейсы зависимостей объявляет потребитель в своём `contracts.go`: `implementation` — usecase,
  `usecase` — `transactor` и `repository`, `db` — `Executor`, `querier`, `Sqlizer`.
- Там же `//go:generate minimock …` → моки в `<пакет>/mocks/*_mock.go` (`make gen-mocks`); тесты — на моках.
- `InReadTx` — read only + REPEATABLE READ: `count` и страница в `Search` из одного снимка.
- `models` — `Plate`, `PlateSighting`, `SearchOpts`, `SearchPage`, `ErrNotFound`; теги `db` — для scany.
- `*Sqr`-методы принимают построитель с `ToSql()` (контракт `squirrel.Sqlizer`); squirrel пока не подключён.

### Тестовые данные — `make sample-data`

Всё лежит в `be/internal/test_utils` и заливается снаружи, бинарник `be` о нём не знает:

1. `migrations/0001_test_data.sql` — миграция goose (сервис `sample-data-db`, отдельная таблица версий
   `goose_test_data_version`). Очищает таблицы и генерирует набор прямо в SQL (`setseed` — одинаково
   при каждой заливке): 8 городов, корректные буквы и коды регионов, ~200 номеров, 600 фиксаций —
   у номера обычно несколько в разных местах его города, время — за 180 дней. Фиксированные записи
   для ручной проверки: `А123ВС 777` — 3 фиксации (id `…0001`, `…0004`, `…0005`), `М456ОР 178`, `Т789УХ 116`.
2. `s3/upload.sh` — сервис `sample-data-s3` (`amazon/aws-cli`): бакет `cars` с анонимным чтением
   (`s3/policy.json`) и картинки `s3/cars/car-1..5.png`.

Залить заново: `make sample-data CMD=redo`.

### Переменные окружения бэкенда

| Переменная | По умолчанию | |
|---|---|---|
| `PORT` / `GRPC_PORT` | `8080` / `8082` | |
| `DATABASE_URL` | — | `postgres://user:pass@host:5432/carsearch?sslmode=disable` |
| `S3_PUBLIC_URL` | `/s3/cars` | откуда браузер берёт картинки (бэку S3-клиент не нужен) |

Пароли БД и S3 в `docker-compose.yml` — только для локальных контейнеров (их порты открыты на
`127.0.0.1`); переопределяются через `.env`: `POSTGRES_PASSWORD`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`.

## Зависимости и окружение

Если на машине глобально настроены корпоративные/приватные реестры (`~/.npmrc`,
`go env -w GOPROXY=...`), они легко просачиваются в lock-файлы. Защита:

| Где | Как защищено |
|---|---|
| npm | `fe/.npmrc` → `registry=https://registry.npmjs.org/`; lock-файл собран из npmjs |
| Go | `Makefile` и `be/Dockerfile` задают `GOPROXY=https://proxy.golang.org,direct`, `GOSUMDB=sum.golang.org` |
| Docker-образы | Docker Hub (`postgres`, `rustfs/rustfs`, `golang`, `node`, `alpine`) |
| Локальный шелл | `.env` (ниже) перекрывает глобальные переменные |

### `.env`

Секретов в нём нет. Создать в корне проекта:

```bash
cat > .env <<'EOF'
# Pet-проект: перекрываем глобальные настройки — только публичные реестры.
# npm: игнорировать ~/.npmrc (глобальный registry и скоупы)
NPM_CONFIG_USERCONFIG=/dev/null
NPM_CONFIG_REGISTRY=https://registry.npmjs.org/
# Go: публичный прокси и проверка контрольных сумм
GOPROXY=https://proxy.golang.org,direct
GOSUMDB=sum.golang.org
GOPRIVATE=
GONOPROXY=
GONOSUMDB=
GOFLAGS=
EOF
```

Подключать перед `npm install` / `go get` в текущем шелле: `set -a; source .env; set +a`.
Команды `make` задают эти переменные сами. `.env` в `.gitignore`.

## Доступ из локальной сети

Vite слушает все интерфейсы (`server.host: true`), Docker публикует `7173` на `0.0.0.0`.
Ссылка для коллег: `http://<ваш-IP>:7173`.

Порт `7173` выбран вместо стандартного `5173` под ограничения файрвола конкретной машины.
Если у вас файрвол пропускает входящие только на определённые порты — смените порт в
`fe/vite.config.ts` (`server.port`) и в `docker-compose.yml` (`ports`).
`strictPort: true` не даёт Vite молча перескочить на соседний — возможно, закрытый — порт.
Порты `7000` и `5000` на macOS занимает приёмник AirPlay.

Типичная картина при закрытом порте: ping проходит, а страница у коллег не открывается.

### Если коллеги не могут открыть страницу

1. **Сервер слушает все интерфейсы:** `lsof -nP -iTCP:7173 -sTCP:LISTEN` → должно быть `*:7173`.
2. **Отвечает по сетевому адресу:** `curl -I http://<ваш-IP>:7173/` с вашей машины.
3. **Пакеты доходят:** `sudo tcpdump -i en0 -nn 'tcp port 7173'`, попросите коллегу открыть ссылку.

   | В tcpdump | Значит |
   |---|---|
   | пусто | запрос не доходит: VPN/прокси у коллеги, изоляция клиентов в Wi-Fi, не тот IP |
   | только `[S]`, без ответа `[S.]` | ваш Mac отбрасывает: файрвол (`sudo pfctl -s rules`), EDR, сетевой фильтр |
   | `[S]` → `[R.]` | на порту никто не слушает |
   | полный обмен, `[P.]` | сеть в порядке — смотрите браузер коллеги |

4. **Проверьте также:** коллега открывает именно `http://` (не `https://`); его IP из той же
   подсети; у терминала/IDE, из которого запущен сервер без Docker, включено
   «Системные настройки → Конфиденциальность и безопасность → Локальная сеть».

Dev-сервер виден всем в сети — в публичных Wi-Fi так не делайте.
Чужие (корпоративные) правила файрвола не отключайте — выберите разрешённый порт.
