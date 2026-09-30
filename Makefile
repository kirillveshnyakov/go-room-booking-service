.DEFAULT_GOAL := help

.PHONY: help postgres migrate app up down clean test test-unit test-repository test-race test-cover

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

test: ## Запустить тесты без PostgreSQL repository-набора и кеширования результатов.
	cd room-booking-service && go test -v ./... -count=1

test-unit: ## Запустить unit-тесты usecase-слоя.
	cd room-booking-service && go test -v ./internal/usecase/... -count=1

test-repository: ## Поднять тестовую PostgreSQL и запустить repository integration-тесты.
	docker compose --profile test up -d --wait postgres-test
	docker compose --profile test run --rm --no-deps -v "$$(go env GOMODCACHE):/go/pkg/mod:ro" repository-tests

test-race: ## Запустить все тесты с race detector (нужны CGO и C-компилятор).
	cd room-booking-service && CGO_ENABLED=1 go test -v -race ./... -count=1

test-cover: ## Запустить все тесты и показать покрытие по пакетам.
	cd room-booking-service && go test -v ./... -cover -count=1

clean: ## Остановить контейнеры и удалить PostgreSQL volume со всеми данными.
	docker compose down --volumes --remove-orphans
