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
  `cmd/` — main (`be serve|seed`) и `httpruntime.go` (JSON/ошибки ручек).
  `internal/implementation` — реализация сервиса, один RPC — один файл (`search.go`, `get.go`;
  руками, clay её не генерирует), `internal/models` — доменные модели,
  `internal/repository` — БД (только Postgres; plain SQL через `s.Select(ctx, &buf, query, args...)` / `Get` / `Exec` и `*Sql`-варианты
  для построителей — `executor.go`; модели с тегами `db`; таблицы/SQL-фрагменты — `repository_entities.go`; генератор данных).
  В тестах сервиса и ручек — заглушка репозитория в `_test.go`,
  `internal/s3` — клиент S3.
- Миграции — только goose: `be/migrations/NNNN_название.sql` (подряд с 0001, `-- +goose Up/Down`).
  Применяет сервис `migrate` в docker-compose (при `make etc` / `make run`), вручную — `make migrate [CMD=status|down|…]`.
  В коде `be` миграций нет. Применённые не менять, новая — `make migration NAME=…`.
- БД: `plates` (id uuid, plate, meta jsonb) и `plate_sightings` (фиксации, `plate_id → plates.id`).
  Поиск — `to_tsvector('simple', p.plate) @@ plainto_tsquery('simple', <строка из rpc>)` с GIN-индексом,
  без pg_trgm и без обработки ввода в Go. `Get(id)` — все фиксации того же номера.
- Ошибки ручек: `{"code": "<имя gRPC-кода>", "message": …}` (`cmd/httpruntime.go`).
- Данные БД и S3 — bind mount в `./.data` (не коммитится), удаляет `make reset`.
- `fe/` — TypeScript + Vite. TS-клиент `fe/src/api/generated.ts` генерируется из swagger: `make gen-ts`.
  Поиск по частям номера, бесконечная прокрутка + «Загрузить ещё», модалка «Уточнить» (заглушка).
- `docker-compose.yml`: db (Postgres), s3 (RustFS — minio/minio больше нет на Docker Hub), be, fe.
  Запуск: `make run` (всё) или по частям `make etc` (db+s3+миграции), `make run-be`, `make run-fe`.
  Тестовые данные: `make seed`.
- Порядок при изменении API: proto → `make gen` → `internal/implementation` → `fe/src/api/index.ts` → `make test`.
- Подробности — в `README.md`, особенности локальной машины — в `CLAUDE.local.md`.
