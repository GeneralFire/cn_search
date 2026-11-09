# Поиск по авто

Поиск автомобиля по госномеру **и** ключу: бэкенд на Go (контракт в proto, ручки генерирует
[clay](https://github.com/utrack/clay)), Postgres, S3-хранилище картинок и фронтенд на
TypeScript + Vite с TS-клиентом, сгенерированным по `swagger.json`.

> **Pet-проект.** Зависимости — только из публичных реестров
> (npmjs.org, proxy.golang.org, Docker Hub). Подробнее — в
> [«Зависимости и окружение»](#зависимости-и-окружение) и `CLAUDE.md`.

## Быстрый старт

Нужны Docker (Docker Desktop / Colima / OrbStack), Docker Compose и `make`.

```bash
make up     # собрать и поднять db, s3, be, fe
make seed   # залить тестовые данные: 600 машин в Postgres, 5 картинок в S3
```

Открыть: **http://localhost:7173** (из локальной сети — `http://<ваш-IP>:7173`).

| Сервис | Что | Адрес |
|---|---|---|
| `fe` | Vite dev-сервер, hot-reload | `0.0.0.0:7173` — единственный порт, нужный браузеру |
| `be` | HTTP-ручки clay + gRPC | `127.0.0.1:8080` (HTTP), `127.0.0.1:8082` (gRPC) |
| `db` | Postgres 17 | `127.0.0.1:5432`, БД `carsearch` |
| `s3` | [RustFS](https://github.com/rustfs/rustfs) — S3-совместимое хранилище | `127.0.0.1:9000` (S3 API), `127.0.0.1:9001` (консоль) |

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
│   ├── api/carsearch/v1/
│   │   ├── carsearch.proto          # КОНТРАКТ (без HTTP-аннотаций)
│   │   └── carsearch.swagger.json   # сгенерирован clay
│   ├── pkg/carsearch/v1/    # сгенерировано: *.pb.go, *_grpc.pb.go, *.pb.goclay.go (ручки)
│   ├── internal/
│   │   ├── cars/            # домен: нормализация, генератор данных, хранилища (Postgres, память)
│   │   ├── service/         # реализация CarSearchService
│   │   └── storage/         # S3-клиент
│   ├── assets/cars/         # 5 PNG, вшиты в бинарник; `be seed` заливает их в S3
│   ├── httpruntime.go       # настройка JSON и формата ошибок ручек clay
│   └── main.go              # be serve | migrate | seed
├── fe/                      # фронтенд, TypeScript + Vite
│   ├── src/api/generated.ts # сгенерированный TS-клиент (не править)
│   ├── src/api/index.ts     # обёртка над клиентом
│   ├── src/main.ts          # UI: поиск, бесконечная прокрутка, модалка
│   └── scripts/gen-car-images.mjs   # генератор PNG-картинок → be/assets/cars
└── mockups/                 # HTML-макеты дизайна
```

## Makefile

| Команда | Что делает |
|---|---|
| `make up` / `down` | поднять / остановить всё (данные остаются в томах) |
| `make reset` | остановить и **удалить** данные БД и S3 |
| `make seed` | схема БД + 600 тестовых машин + картинки в S3. Размер: `make seed SEED_TOTAL=5000` |
| `make migrate` | только схема БД |
| `make psql` | консоль Postgres |
| `make logs` / `ps` | логи / статус |
| `make tools` | поставить генераторы `protoc-gen-go`, `-go-grpc`, `-goclay` в `be/bin` и npm-зависимости |
| `make gen` | всё из proto: `gen-go` + `gen-ts` |
| `make gen-go` | proto → Go + ручки clay + `swagger.json` |
| `make gen-ts` | `swagger.json` → `fe/src/api/generated.ts` |
| `make test` | тесты бэка (`go test -race`) и проверка типов фронта |
| `make check-registries` | в `package-lock.json` только registry.npmjs.org |

С плагином Compose v2: `make DC="docker compose" up`.

## API

Контракт — `be/api/carsearch/v1/carsearch.proto`, **без HTTP-аннотаций**.
clay генерирует ручки с флагом `generate_unbound_methods`: каждый метод доступен как
`POST /<package>.<Service>/<Method>` с JSON-телом; бэкенд монтирует их под `/api`.

| Метод | HTTP |
|---|---|
| `SearchCars` | `POST /api/carsearch.v1.CarSearchService/SearchCars` |
| `GetCar` | `POST /api/carsearch.v1.CarSearchService/GetCar` |
| — | `GET /api/swagger.json` — спецификация |
| — | `GET /api/health` |

Тот же сервис доступен по gRPC на `:8082` (с reflection: `grpcurl -plaintext localhost:8082 list`).

### SearchCars

```jsonc
// запрос — все поля необязательны
{ "plate": "a123bc", "key": "0451", "offset": 0, "limit": 35 }

// ответ
{
  "items": [{ "id": 1, "photoUrl": "/s3/cars/car-3.png", "address": "Москва, …",
              "plate": "А123ВС 777", "key": "KEY-0451", "date": "2026-11-02" }],
  "total": 1,          // всего найдено — есть всегда, в т. ч. 0
  "nextOffset": 35     // смещение следующей страницы; нет поля — дальше пусто
}
```

- Номер **И** ключ: условия объединяются, пустое поле не ограничивает.
- Подстрока, без учёта регистра и пробелов; номер можно вводить латиницей (`A123BC` → `А123ВС`).
- От новых к старым. `limit`: 1–100, `0` — по умолчанию 35 (кратно 5 и 7 — колонкам сетки).

### Ошибки

```json
{ "code": "invalid_argument", "message": "limit: от 1 до 100 (0 — по умолчанию 35)" }
```

| HTTP | `code` | Когда |
|---|---|---|
| 400 | `invalid_argument` | неверные параметры, битый JSON |
| 404 | `not_found` | машины нет |
| 500 | `internal` | ошибка сервера (детали — только в логах) |

## Генерация кода

```
carsearch.proto ──protoc──► be/pkg/…/*.pb.go, *_grpc.pb.go      (protoc-gen-go, -go-grpc)
                └─────────► be/pkg/…/*.pb.goclay.go              (protoc-gen-goclay: HTTP-ручки)
                └─────────► be/api/…/carsearch.swagger.json      (clay)
                                   └──swagger-typescript-api──► fe/src/api/generated.ts
```

Поменяли proto → `make gen` → реализовать метод в `be/internal/service` → использовать в
`fe/src/api/index.ts`. Нужен `protoc` (`brew install protobuf`) и `make tools` один раз.
Сгенерированный код лежит в репозитории — для сборки генераторы не нужны.

Нюансы, зашитые в `be/httpruntime.go`:
- **Нулевые значения в JSON.** По умолчанию proto3 JSON их не передаёт: пустая выдача была бы `{}`.
  Включён `EmitUnpopulated` — `total` и `items` есть всегда; необязательный `nextOffset` пропадает,
  когда страниц больше нет.
- **Ошибки** — `{code, message}` вместо стандартного `{"error": "rpc error: code = … desc = …"}`.

## Фронтенд

- Поля **Номер** и **Ключ**, поиск с задержкой 250 мс; устаревшие запросы отменяются.
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

Таблица `cars` (`be/internal/cars/schema.sql`): отображаемые `plate`/`key` и нормализованные
`plate_norm`/`key_norm` для поиска, trigram-индексы для `LIKE '%…%'`, `photo_key` — ключ объекта в S3.

`make seed` (= `be seed`):
1. создаёт бакет `cars` с анонимным чтением и заливает 5 PNG из `be/assets/cars`;
2. применяет схему и **заменяет** все машины сгенерированным набором.

Генератор детерминированный (фиксированный seed): у всех одинаковые данные. 8 городов,
корректные буквы и коды регионов, уникальные номера и ключи, даты за последние 180 дней.
Первые три записи фиксированные — для ручной проверки: `А123ВС 777` / `KEY-0451`,
`М456ОР 178` / `KEY-1187`, `Т789УХ 116` / `KEY-2093`.

Картинки (седан, хэтчбек, внедорожник, пикап, минивэн; 1280×720) рисует
`node fe/scripts/gen-car-images.mjs` → `be/assets/cars/`. После перегенерации — `make up && make seed`.

### Переменные окружения бэкенда

| Переменная | По умолчанию | |
|---|---|---|
| `PORT` / `GRPC_PORT` | `8080` / `8082` | |
| `DATABASE_URL` | — | `postgres://user:pass@host:5432/carsearch?sslmode=disable` |
| `S3_ENDPOINT` | `localhost:9000` | host:port |
| `S3_ACCESS_KEY` / `S3_SECRET_KEY` | — | |
| `S3_BUCKET` | `cars` | |
| `S3_PUBLIC_URL` | `/s3/cars` | откуда браузер берёт картинки |
| `SEED_TOTAL` | `600` | сколько машин генерирует `be seed` |

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
