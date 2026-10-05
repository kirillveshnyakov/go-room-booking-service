.DEFAULT_GOAL := help

.PHONY: help postgres migrate app up down clean install-mockgen generate-mocks generate-sqlc test test-unit test-db-up test-db-down test-repository test-concurrency test-integration test-race test-cover

help: ## Показать доступные команды.
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ {printf "%-18s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

postgres: ## Запустить только PostgreSQL и дождаться его готовности.
	docker compose up -d --wait postgres

migrate: postgres ## Собрать и применить миграции к запущенной PostgreSQL.
	docker compose build migrate
	docker compose run --rm --no-deps migrate

app: ## Запустить API; Compose автоматически поднимет зависимости и миграции.
	docker compose up -d --build app

up: ## Собрать и запустить PostgreSQL, миграции и API целиком.
	docker compose up -d --build

down: ## Остановить и удалить контейнеры, сохранив данные PostgreSQL.
	docker compose down --remove-orphans

install-mockgen: ## Установить mockgen версии, используемой в проекте.
	cd room-booking-service && go install go.uber.org/mock/mockgen@v0.6.0

generate-mocks: ## Перегенерировать моки usecase-слоя (требуется mockgen в PATH).
	cd room-booking-service && go generate ./internal/usecase/...

generate-sqlc: ## Перегенерировать PostgreSQL-код из SQL (требуется sqlc в PATH).
	cd room-booking-service && sqlc generate

test: ## Поднять тестовую PostgreSQL и запустить все тесты без кеширования результатов.
	@set -e; \
	trap '$(MAKE) test-db-down' EXIT; \
	$(MAKE) test-db-up; \
	docker compose --profile test run --rm --no-deps -v "$$(go env GOMODCACHE):/go/pkg/mod:ro" go-tests go test -v ./... -count=1

test-unit: ## Запустить unit-тесты usecase-слоя.
	cd room-booking-service && go test -v ./internal/usecase/... -count=1

test-db-up: ## Поднять тестовую PostgreSQL и дождаться её готовности.
	docker compose --profile test up -d --wait postgres-test

test-db-down: ## Остановить тестовую PostgreSQL.
	docker compose --profile test stop postgres-test

test-repository: ## Поднять тестовую PostgreSQL и запустить repository integration-тесты.
	@set -e; \
	trap '$(MAKE) test-db-down' EXIT; \
	$(MAKE) test-db-up; \
	docker compose --profile test run --rm --no-deps -v "$$(go env GOMODCACHE):/go/pkg/mod:ro" go-tests go test -v ./internal/repository/postgres/... -count=1

test-concurrency: ## Запустить конкурентные repository-тесты с race detector.
	@set -e; \
	trap '$(MAKE) test-db-down' EXIT; \
	$(MAKE) test-db-up; \
	docker compose --profile test run --rm --no-deps -v "$$(go env GOMODCACHE):/go/pkg/mod:ro" go-tests go test -v -race -run '^TestConcurrency_' ./internal/repository/postgres/... -count=100

test-integration: ## Поднять тестовую PostgreSQL и запустить HTTP integration-тесты.
	@set -e; \
	trap '$(MAKE) test-db-down' EXIT; \
	$(MAKE) test-db-up; \
	docker compose --profile test run --rm --no-deps -v "$$(go env GOMODCACHE):/go/pkg/mod:ro" go-tests go test -v ./tests/integration/... -count=1

test-race: ## Поднять тестовую PostgreSQL и запустить все тесты с race detector.
	@set -e; \
	trap '$(MAKE) test-db-down' EXIT; \
	$(MAKE) test-db-up; \
	docker compose --profile test run --rm --no-deps -e CGO_ENABLED=1 -v "$$(go env GOMODCACHE):/go/pkg/mod:ro" go-tests go test -v -race ./... -count=1

test-cover: ## Поднять тестовую PostgreSQL, запустить все тесты и показать покрытие.
	@set -e; \
	trap '$(MAKE) test-db-down' EXIT; \
	$(MAKE) test-db-up; \
	docker compose --profile test run --rm --no-deps -v "$$(go env GOMODCACHE):/go/pkg/mod:ro" go-tests go test -v ./... -cover -count=1

clean: ## Остановить контейнеры и удалить PostgreSQL volume со всеми данными.
	docker compose down --volumes --remove-orphans
