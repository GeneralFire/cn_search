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

- `be/` — Go. Контракт: `be/api/carsearch/v1/carsearch.proto` **без HTTP-аннотаций**.
  HTTP-ручки генерирует clay (`generate_unbound_methods` → `POST /api/carsearch.v1.CarSearchService/<Method>`).
  Сгенерированное — `be/pkg/**` и `carsearch.swagger.json`, руками не править: `make gen-go`.
  Логика: `internal/service` (реализация), `internal/cars` (домен, Postgres, MemoryStore для тестов),
  `internal/storage` (S3). JSON/ошибки ручек — `be/httpruntime.go`.
- `fe/` — TypeScript + Vite. TS-клиент `fe/src/api/generated.ts` генерируется из swagger: `make gen-ts`.
  Поиск «Номер И Ключ», бесконечная прокрутка + «Загрузить ещё», модалка «Уточнить» (заглушка).
- `docker-compose.yml`: db (Postgres), s3 (RustFS — minio/minio больше нет на Docker Hub), be, fe.
  Тестовые данные: `make seed`.
- Порядок при изменении API: proto → `make gen` → `internal/service` → `fe/src/api/index.ts` → `make test`.
- Подробности — в `README.md`, особенности локальной машины — в `CLAUDE.local.md`.
