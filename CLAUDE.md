# CLAUDE.md — car-search

## ⚠️ Pet-проект

Личный проект, к работодателю отношения не имеет. Репозиторий публичный.

- **Зависимости — только из публичных реестров:**
  - npm → `https://registry.npmjs.org/` (`fe/.npmrc`); в `fe/package-lock.json` — только `registry.npmjs.org`
  - Go → `https://proxy.golang.org`, `GOSUMDB=sum.golang.org`
  - Docker-образы → Docker Hub
- Машина разработчика может быть глобально настроена на корпоративные реестры (`~/.npmrc`, `go env`).
  Команды `make` переопределяют это сами. Вручную (`npm install` / `go get`) — сначала
  `set -a; source .env; set +a` (содержимое — в README; агенту `.env*` закрыт правилами доступа)
  или `NPM_CONFIG_USERCONFIG=/dev/null` / `GOPROXY=https://proxy.golang.org,direct GOSUMDB=sum.golang.org GOFLAGS=`.
  Проверка: `make check-registries`.
- Корпоративные конвенции и внутренние пакеты не используются.
- **В коммитящиеся файлы не писать** внутренние хосты, детали корпоративной инфраструктуры,
  рабочую почту. Такое — только в `CLAUDE.local.md` (не коммитится).

## Устройство

- `be/` — Go. Контракт: `be/api/carsearch/carsearch.proto` (`package carsearch`, без v1) **без HTTP-аннотаций**.
  HTTP-ручки генерирует clay (`generate_unbound_methods` → `POST /api/carsearch.CarSearchService/<Method>`).
  Сгенерированное — `be/pkg/**` и `carsearch.swagger.json`, руками не править: `make gen-go`.
  Swagger clay кладёт в `be/schema/carsearch.swagger.json`.
  `cmd/main.go` — только `serve` и сборка слоёв; маршалер и ошибки clay — стандартные (без настройки).
- Слои be: `implementation` (RPC, файл на метод) → `usecase` (сценарии, транзакции `InReadTx`/`InWriteTx`)
  → `repository` (plain SQL, `tx db.Executor` — второй аргумент каждого метода) → `db` (`Executor`:
  `Select`/`Get`/`Exec` + `SelectSqr`/`GetSqr`/`ExecSqr` для squirrel-построителей; `Cluster`).
  `models` — доменные модели с тегами `db` (сканирует scany), `SearchOpts`, `SearchPage`, `ErrNotFound`.
- Интерфейсы зависимостей — в `contracts.go` пакета-потребителя, там же `//go:generate minimock …`
  → моки в `<пакет>/mocks/*_mock.go` (`make gen-mocks`). Тесты — на моках minimock, ручных заглушек нет.
- Миграции — только goose: `be/migrations/NNNN_название.sql` (подряд с 0001, `-- +goose Up/Down`).
  Применяет сервис `migrate` в docker-compose (при `make etc` / `make run`), вручную — `make migrate [CMD=status|down|…]`.
  В коде `be` миграций нет. До первого выката схему правим прямо в `0001_init.sql`; после — новая
  миграция (`make migration NAME=…`), применённые не менять.
- Тестовые данные — миграция `be/migrations/0002_test_data.sql`, строки с `is_test_data = true`;
  картинки — `be/internal/test_utils/s3` (сервис `s3-init`, aws-cli). Всё при `make etc`.
  Бэк отдаёт тестовые данные только с заголовком `include-test-data` (metadata; clay кладёт туда
  HTTP-заголовки) → `SearchOpts`/`SamePlateOpts.IncludeTestData` → фильтр в repository.
  `fe` ничего из `be` не импортирует; шлёт заголовок при `VITE_INCLUDE_TEST_DATA=true`.
- БД: `plates` (id uuid, plate, meta jsonb, is_test_data) и `plate_sightings` (id uuid, `plate_id → plates.id`,
  is_test_data). Поиск — по началу слов: `to_tsvector('simple', p.plate) @@ to_tsquery(<слова:* через &>)`,
  запрос собирается в SQL (не в Go), GIN-индекс, без pg_trgm. Похожие символы (кириллица/латиница, О/0, дефис)
  сводит `plate_search_key()` — одинаково в индексе и в запросе (тест `repository` следит за совпадением). `Get(id)` — все фиксации того же номера.
  Пустая выдача по непустому запросу — `SearchResponse.suggestions` («возможно, вы искали»): до 5 номеров
  с `levenshtein` (fuzzystrmatch, contrib) ≤ 2 по `plate_search_key` без пробелов (`repository.Suggest`).
- API-токены (`be/internal/apitoken`): `cst_` + base64url({tid,iat}) + `.` + base64url(HMAC-SHA256, `TOKEN_SIGNING_KEY`);
  в `api_tokens` — только sha256 подписи и остаток `requests_left`. Токен — cookie `cs_token` или `authorization: Bearer`.
  rpc Get = `usecase.ApiTokens.OpenPlate` (одна InWriteTx): списание + запись в `plate_views`; admin-token квоту не тратит.
  Админка: вкладки «Токены»/«Просмотры» — `fe/src/admin-tokens.ts`, остаток в настройках — `fe/src/apitoken-settings.ts`.
- Rate limit пишущих ручек (`be/internal/ratelimit`): token bucket в памяти, лимиты — `ratelimit.Limits` (Clarify/Complain 10/мин,
  SubmitCameraRequest 5/мин, Vote 30/мин, Register 5/ч, Ingest 10/мин). Один unary interceptor на оба транспорта
  (`grpc.UnaryInterceptor` + clay `desc.Apply(transport.WithUnaryInterceptor)`); ключ — `implementation.RateLimitKey`
  (хеш токена пользователя, иначе последний X-Forwarded-For, иначе peer; для HTTP peer кладёт `WithPeerFromRequest`).
  Превышение — ResourceExhausted (HTTP 429 + Retry-After), admin-token не ограничивается.
- `GET /api/health` — liveness, `GET /api/ready` — readiness (`be/internal/health`: ping БД и HeadBucket S3, таймаут 1 с,
  иначе 503 с причиной); healthcheck `be` в compose — на `/api/ready`. Линтер — `make lint` (golangci-lint в docker,
  `be/.golangci.yml`, версия в Makefile и CI), обновления — `.github/dependabot.yml`.
- Данные БД и S3 — bind mount в `./.data` (не коммитится), удаляет `make reset`.
- `fe/` — TypeScript + Vite. TS-клиент `fe/src/api/generated.ts` генерируется из swagger: `make gen-ts`.
  Поиск по номеру, бесконечная прокрутка + «Загрузить ещё», модалка «Уточнить» (`src/clarify.ts`; с карточек главной и со страницы номера),
  главная без запроса — блок «для вас» (2 ряда номеров «рынка» языка) и вкладки стран + «Прочее»
  (`src/home.ts`, `SearchRequest.market` → `models.Market` → условия по `plates.meta` type/region; вкладка — `?tab=`),
  запрос поиска — в адресе `?q=` (F5, история, смена языка), страница номера `/plate/<id фиксации>` (`src/plate.ts`, роутер на History API — `src/router.ts`;
  плашка уточнений — `src/clarifications.ts`, данные — `GetResponse.clarifications`).
  Админ-режим: `src/adminmode.ts` (admin-token в sessionStorage → rpc Me → класс `is-admin` и заголовок во всех запросах),
  скрытые панели — `src/adminpanels.ts`, вкладки `/admin` — реестр `src/admintabs.ts`. Скрытые фиксации (`plate_sightings.hidden`)
  отдаются только с верным admin-token (`IncludeHidden`).
- `rpc Ingest` (фото → номер → фиксация): сервис `ocr/` (Python, fast-alpr, порт только в сети compose) →
  usecase `Ingest` (EXIF, нормализация номера `plate_format.go`, S3 `u/<uuid>.jpg` через minio-go, запись в БД);
  фронт — `/upload` (`src/upload.ts`). Схема и модели — `docs/OCR_PIPELINE.md`.
- `docker-compose.yml`: db (Postgres), s3 (RustFS — minio/minio больше нет на Docker Hub), ocr, be, fe.
  Запуск: `make run` (всё) или по частям `make etc` (db+s3+миграции), `make run-be`, `make run-fe`.
  Тестовые данные приходят миграцией при `make etc`.
- Порядок при изменении API: proto → `make gen` → `implementation` → `usecase` → `repository`
  → `fe/src/api/index.ts` → `make test`.
- Подробности — в `README.md`, особенности локальной машины — в `CLAUDE.local.md`.
