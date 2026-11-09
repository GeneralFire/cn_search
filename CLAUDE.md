# CLAUDE.md — car-search

## ⚠️ Pet-проект

Личный проект, к работодателю отношения не имеет. Репозиторий публичный.

- **Зависимости — только из публичных реестров:**
  - npm → `https://registry.npmjs.org/` (`fe/.npmrc`); в `fe/package-lock.json` — только `registry.npmjs.org`
  - Go → `https://proxy.golang.org`, `GOSUMDB=sum.golang.org`
  - Docker-образы → Docker Hub
- Машина разработчика может быть глобально настроена на корпоративные реестры (`~/.npmrc`, `go env`).
  Перед `npm install` / `go get` подключайте `.env`: `set -a; source .env; set +a`
  (содержимое — в README, «Зависимости и окружение»; агенту `.env*` закрыт правилами доступа).
  Проверка: `grep -oE '"resolved": "https://[^/]+' fe/package-lock.json | sort -u` → только npmjs.
- Корпоративные конвенции и внутренние пакеты не используются.
- **В коммитящиеся файлы не писать** внутренние хосты, детали корпоративной инфраструктуры,
  рабочую почту. Такое — только в `CLAUDE.local.md` (не коммитится).

## Устройство

- `fe/` — TypeScript + Vite, без фреймворков. Данные в `fe/src/data.ts`.
- `be/` — Go, stdlib. **Пока заглушка** (`GET /api/health`), бэкенд не разрабатываем.
- `docker-compose.yml` — fe (hot-reload, порт 7173) + be (127.0.0.1:8080).
- Подробности — в `README.md`, особенности локальной машины — в `CLAUDE.local.md`.
