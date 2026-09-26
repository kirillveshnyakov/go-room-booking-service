COMPOSE ?= docker compose

.DEFAULT_GOAL := help

.PHONY: help postgres migrate app up down clean

help: ## Показать доступные команды.
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ {printf "%-12s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

postgres: ## Запустить только PostgreSQL и дождаться его готовности.
	$(COMPOSE) up -d --wait postgres

migrate: postgres ## Собрать и применить миграции к запущенной PostgreSQL.
	$(COMPOSE) build migrate
	$(COMPOSE) run --rm --no-deps migrate

app: ## Запустить API; Compose автоматически поднимет зависимости и миграции.
	$(COMPOSE) up -d --build app

up: ## Собрать и запустить PostgreSQL, миграции и API целиком.
	$(COMPOSE) up -d --build

down: ## Остановить и удалить контейнеры, сохранив данные PostgreSQL.
	$(COMPOSE) down --remove-orphans

clean: ## Остановить контейнеры и удалить PostgreSQL volume со всеми данными.
	$(COMPOSE) down --volumes --remove-orphans
