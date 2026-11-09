# car-search — основные команды. `make help` — список.
#
# Docker Compose: по умолчанию отдельный бинарник docker-compose;
# с плагином Compose v2 — `make DC="docker compose" …`.
DC ?= docker-compose

BE        := be
FE        := fe
BIN       := $(abspath $(BE)/bin)
PROTO_DIR := $(BE)/api
PROTO     := $(PROTO_DIR)/carsearch/v1/carsearch.proto
SWAGGER   := $(BE)/schema/carsearch.swagger.json
TS_CLIENT := $(FE)/src/api/generated.ts

# Только публичные реестры (машина может быть глобально настроена на приватные).
export GOPROXY   := https://proxy.golang.org,direct
export GOSUMDB   := sum.golang.org
export GOPRIVATE :=
export GONOPROXY :=
export GONOSUMDB :=
export GOFLAGS   :=
export NPM_CONFIG_USERCONFIG := /dev/null

CLAY_VERSION     := v3.2.0
PROTOC_GEN_GO    := v1.36.12
PROTOC_GEN_GRPC  := v1.6.2

.DEFAULT_GOAL := help

.PHONY: help
help: ## список команд
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

# ---------------------------------------------------------------- запуск

.PHONY: up
up: ## собрать и поднять всё (db, s3, be, fe)
	$(DC) up -d --build

.PHONY: down
down: ## остановить (данные сохраняются в томах)
	$(DC) down

.PHONY: reset
reset: ## остановить и УДАЛИТЬ данные БД и S3 (./.data)
	$(DC) down -v
	rm -rf .data

.PHONY: logs
logs: ## логи всех сервисов
	$(DC) logs -f --tail=100

.PHONY: ps
ps: ## статус сервисов
	$(DC) ps

# ---------------------------------------------------------------- данные

.PHONY: migrate
migrate: ## миграции goose вручную: make migrate [CMD=up|status|down|redo|version|up-to ARG=2]
	$(DC) up -d --wait db
	$(DC) run --rm -e GOOSE_COMMAND=$(or $(CMD),up) -e GOOSE_COMMAND_ARG=$(ARG) migrate

.PHONY: migration
migration: ## создать миграцию: make migration NAME=add_vin
	@test -n "$(NAME)" || { echo "укажите имя: make migration NAME=add_vin"; exit 1; }
	@echo "$(NAME)" | grep -Eq '^[a-z0-9_]+$$' || { echo "NAME: строчные латинские буквы, цифры, _"; exit 1; }
	@n=$$(ls $(BE)/migrations/[0-9][0-9][0-9][0-9]_*.sql 2>/dev/null | wc -l | tr -d ' '); \
	f=$(BE)/migrations/$$(printf '%04d' $$((n+1)))_$(NAME).sql; \
	printf -- '-- +goose Up\n\n-- +goose Down\n' > $$f; echo "создан $$f"

.PHONY: seed
seed: ## залить тестовые данные: машины в БД, картинки в S3 (SEED_TOTAL=600)
	$(DC) up -d --wait db s3
	$(DC) run --rm migrate
	$(DC) run --rm --build -e SEED_TOTAL=$(or $(SEED_TOTAL),600) be seed

.PHONY: psql
psql: ## консоль БД
	$(DC) exec db psql -U $${POSTGRES_USER:-carsearch} carsearch

# ---------------------------------------------------------------- генерация

.PHONY: tools
tools: ## поставить генераторы protoc-gen-* в be/bin
	GOBIN=$(BIN) go install google.golang.org/protobuf/cmd/protoc-gen-go@$(PROTOC_GEN_GO)
	GOBIN=$(BIN) go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@$(PROTOC_GEN_GRPC)
	GOBIN=$(BIN) go install github.com/utrack/clay/v3/cmd/protoc-gen-goclay@$(CLAY_VERSION)
	cd $(FE) && npm install

.PHONY: gen
gen: gen-go gen-ts ## сгенерировать всё из proto: Go, swagger.json, TS-клиент

.PHONY: gen-go
gen-go: ## proto → Go (protobuf, gRPC, ручки clay) + swagger.json
	@command -v protoc >/dev/null || { echo "нужен protoc: brew install protobuf"; exit 1; }
	@test -x $(BIN)/protoc-gen-goclay || { echo "нет генераторов: make tools"; exit 1; }
	rm -rf $(BE)/pkg/carsearch
	mkdir -p $(BE)/pkg
	PATH=$(BIN):$$PATH protoc -I $(PROTO_DIR) --unsafe_allow_out_dir_escape \
		--go_out=$(BE)/pkg --go_opt=paths=source_relative \
		--go-grpc_out=$(BE)/pkg --go-grpc_opt=paths=source_relative \
		--goclay_out=$(BE)/pkg --goclay_opt=paths=source_relative,generate_unbound_methods=true,impl=false,tests=false,swagger_path=../schema \
		$(PROTO)
	@# clay пишет swagger в <goclay_out>/<swagger_path>/<имя proto>.json = be/schema/carsearch.json
	@# (выход за goclay_out требует --unsafe_allow_out_dir_escape); имя — по нашему соглашению.
	mv $(BE)/schema/carsearch.json $(SWAGGER)

.PHONY: gen-ts
gen-ts: ## swagger.json → TypeScript-клиент (fe/src/api/generated.ts)
	cd $(FE) && npx swagger-typescript-api generate \
		--path ../$(SWAGGER) \
		--output src/api \
		--name generated.ts \
		--api-class-name CarSearchApi \
		--extract-request-body --extract-response-body \
		--sort-types --silent

# ---------------------------------------------------------------- проверки

.PHONY: test
test: ## тесты бэка и проверка типов фронта
	cd $(BE) && go test -race -count=1 ./...
	cd $(FE) && npx tsc

.PHONY: check-registries
check-registries: ## убедиться, что зависимости только из публичных реестров
	@! grep -oE '"resolved": "https://[^/]+' $(FE)/package-lock.json | sort -u | grep -v registry.npmjs.org
	@echo "npm: только registry.npmjs.org"
