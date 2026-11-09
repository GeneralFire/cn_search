# Поиск по авто

Поиск автомобиля по госномеру или ключу.

> **Pet-проект.** Зависимости — только из публичных реестров
> (npmjs.org, proxy.golang.org, Docker Hub). Подробнее — в
> [«Зависимости и окружение»](#зависимости-и-окружение) и `CLAUDE.md`.

| Папка | Что внутри | Статус |
|---|---|---|
| `fe/` | Фронтенд: TypeScript + Vite, без фреймворков | в работе |
| `be/` | Бэкенд: Go, только стандартная библиотека | **заглушка** — отвечает лишь на `GET /api/health` |
| `mockups/` | Статичные HTML-макеты дизайна (вариант A — светлый, B — тёмный) | для обсуждения |

```
car-search/
├── CLAUDE.md              # памятка: pet-проект, публичные реестры
├── CLAUDE.local.md        # локальные заметки о машине (не коммитится)
├── docker-compose.yml     # поднимает fe + be
├── .env                   # перекрывает глобальные реестры (создаётся вручную, не коммитится)
├── fe/
│   ├── .npmrc             # registry=https://registry.npmjs.org/
│   ├── Dockerfile         # dev-образ с hot-reload
│   ├── vite.config.ts     # порт 7173, host, polling
│   ├── index.html
│   └── src/
│       ├── main.ts        # UI и поиск
│       ├── data.ts        # данные (пока захардкожены)
│       └── style.css
├── be/
│   ├── Dockerfile
│   ├── go.mod
│   └── main.go            # заглушка
└── mockups/               # variant-a.html, variant-b.html
```

## Быстрый старт (Docker)

Требуется Docker (Docker Desktop, Colima, OrbStack…) и Docker Compose.

```bash
docker-compose up -d --build  # собрать и поднять fe + be
```

| Сервис | Адрес | Кому доступен |
|---|---|---|
| fe | http://localhost:7173 · `http://<ваш-IP>:7173` | вам и всем в локальной сети |
| be | http://127.0.0.1:8080/api/health | только вам (проброшен на localhost) |

Свой IP: `ipconfig getifaddr en0`.

Полезные команды:
```bash
docker-compose logs -f fe     # логи фронта (видно hmr update)
docker-compose ps             # статус
docker-compose down           # остановить
docker-compose up -d --build fe   # пересобрать fe (после изменения package.json)
```

> Команды ниже — через `docker-compose`; с плагином Compose v2 то же самое работает как `docker compose`.

### Hot-reload фронтенда

Папка `fe/` монтируется в контейнер (`./fe:/app`), Vite внутри следит за файлами —
правки в `fe/src/**` применяются в браузере сразу, без перезапуска контейнера.

- **Polling.** В Docker на macOS (Colima, Docker Desktop) события файловой системы из
  bind-mount доходят до контейнера ненадёжно, поэтому включён опрос раз в 300 мс
  (`VITE_USE_POLLING=true` в `docker-compose.yml` → `server.watch.usePolling` в `vite.config.ts`).
- **`node_modules`.** Берутся из образа, а не с хоста (анонимный том `/app/node_modules`):
  бинарники под macOS в Linux-контейнере не работают.
  Поэтому после изменения `package.json` нужно пересобрать образ:
  `docker-compose up -d --build fe`.
- Изменения `vite.config.ts` требуют перезапуска: `docker-compose restart fe`.

### Бэкенд

Пока заглушка: `be/main.go` отвечает только на `GET /api/health` → `{"status":"ok"}`.
Фронтенд к бэкенду не обращается — данные лежат в `fe/src/data.ts`.
После изменения Go-кода: `docker-compose up -d --build be`.

## Запуск без Docker

```bash
cd fe && npm install && npm run dev       # http://localhost:7173
cd be && go run .                          # http://localhost:8080/api/health
```

Сборка фронта: `npm run build` (проверка типов + сборка в `fe/dist/`),
просмотр сборки: `npm run preview` (порт 7174).

## Зависимости и окружение

Если на машине глобально настроены корпоративные/приватные реестры (`~/.npmrc`,
`go env -w GOPROXY=...`), они легко просачиваются в lock-файлы. Защита:

| Где | Как защищено |
|---|---|
| npm | `fe/.npmrc` → `registry=https://registry.npmjs.org/`; lock-файл собран из npmjs |
| Go в Docker | `ENV GOPROXY=https://proxy.golang.org,direct`, `GOSUMDB=sum.golang.org` в `be/Dockerfile` |
| Docker-образы | Docker Hub (`node`, `golang`, `alpine`) |
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

Подключать перед `npm install` / `go get` в текущем шелле:

```bash
set -a; source .env; set +a
```

Переменные окружения имеют приоритет над `~/.npmrc` и над `go env -w`, так что в этом
шелле глобальные реестры не используются. `.env` в `.gitignore`.

**Проверка**, что ничего корпоративного не просочилось:

```bash
grep -oE '"resolved": "https://[^/]+' fe/package-lock.json | sort -u   # → только registry.npmjs.org
(cd fe && npm config get registry)          # → https://registry.npmjs.org/
go env GOPROXY GOSUMDB                       # → proxy.golang.org…, sum.golang.org (после source .env)
```

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
   подсети (`192.168.1.x`); у терминала/IDE, из которого запущен сервер без Docker, включено
   «Системные настройки → Конфиденциальность и безопасность → Локальная сеть».

Dev-сервер виден всем в сети — в публичных Wi-Fi так не делайте.
Чужие (корпоративные) правила файрвола не отключайте — выберите разрешённый порт.

## Данные

`fe/src/data.ts`, тип `Car`:

| Поле | Тип | Описание |
|---|---|---|
| `id` | `number` | идентификатор |
| `photo?` | `string` | URL фото (необязательно) |
| `address` | `string` | адрес |
| `plate` | `string` | госномер кириллицей, `"А123ВС 777"` |
| `key` | `string` | ключ, `"KEY-0451"` |
| `date` | `string` | дата `YYYY-MM-DD` |

### Поиск
- Без учёта регистра и пробелов: `а123вс777` найдёт `А123ВС 777`.
- Номер можно вводить латиницей: `A123BC` найдёт `А123ВС` (12 букв, одинаковых в обоих алфавитах).
- Пустой запрос показывает все записи.
- Дата форматируется в UTC, чтобы в западных часовых поясах не сдвигалась на день назад.
