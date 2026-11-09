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
- Тестовые данные — `be/internal/test_utils` (goose-миграция `migrations/` с отдельной таблицей версий +
  `s3/upload.sh` через aws-cli), заливает `make sample-data`. `fe` ничего из `be` не импортирует.
- БД: `plates` (id uuid, plate, meta jsonb) и `plate_sightings` (id uuid, `plate_id → plates.id`).
  Поиск — `to_tsvector('simple', p.plate) @@ plainto_tsquery('simple', <строка из rpc>)` с GIN-индексом,
  без pg_trgm и без обработки ввода в Go. `Get(id)` — все фиксации того же номера.
- Данные БД и S3 — bind mount в `./.data` (не коммитится), удаляет `make reset`.
- `fe/` — TypeScript + Vite. TS-клиент `fe/src/api/generated.ts` генерируется из swagger: `make gen-ts`.
  Поиск по номеру, бесконечная прокрутка + «Загрузить ещё», модалка «Уточнить» (заглушка).
- `docker-compose.yml`: db (Postgres), s3 (RustFS — minio/minio больше нет на Docker Hub), be, fe.
  Запуск: `make run` (всё) или по частям `make etc` (db+s3+миграции), `make run-be`, `make run-fe`.
  Тестовые данные: `make sample-data`.
- Порядок при изменении API: proto → `make gen` → `implementation` → `usecase` → `repository`
  → `fe/src/api/index.ts` → `make test`.
- Подробности — в `README.md`, особенности локальной машины — в `CLAUDE.local.md`.
