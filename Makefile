.PHONY: help up ps logs down reset-db migrate-up migrate-down migrate-status migrate-new run test fmt vet tidy test-race lint check-env

# Имя сервиса базы из compose.yaml
DB_SERVICE ?= postgres
# Сервис для logs: по умолчанию база, можно передать s=...
s ?= $(DB_SERVICE)
help: ## Показать список команд
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

ps: check-env ## Поднятые контейнеры проекта
	docker compose ps
logs: check-env ## Логи сервиса (по умолчанию база): make logs s=app
	docker compose logs -f $(s)
up: check-env ## Поднять докер на фоне
	docker compose up -d --build
down: check-env ## Остановить контейнеры (данные сохраняются)
	docker compose down
reset-db: check-env ## УДАЛИТЬ базу и поднять чистую: make reset-db confirm=yes
	@[ "$(confirm)" = "yes" ] || (echo "Warning: This will delete all database data. Run: make reset-db confirm=yes";exit 1)
	@docker compose down -v
	@$(MAKE) up

migrate-up: check-env ## Применить все новые миграции
	goose up
migrate-down: check-env ## Откатить последнюю миграцию
	goose down
migrate-status: check-env ## Показать статус миграций
	goose status
migrate-new: check-env ## Создать миграцию: make migrate-new name=create_users
	@if [ -z "$(name)" ]; then echo "Usage: make migrate-new name=create_users"; exit 1; fi
	@goose -s create $(name) sql

run: ## Запустить приложение
	go run ./cmd/api
fmt: ## Форматировать go файлы
	go fmt ./...
vet: ## go vet
	go vet ./...
tidy: ## go mod tidy
	go mod tidy
test: ## Запустить тесты
	go test ./...
test-race: ## Запустить тесты с детектором гонок
	go test ./... -race
lint: ## Проверить код линтером
	golangci-lint run

check-env:
	@[ -f .env ] || (echo ".env file is not found. Create: cp .env.example .env"; exit 1)

