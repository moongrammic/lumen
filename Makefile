.PHONY: up down restart logs build migrate migrate-down frontend-dev frontend-build

up:
	docker compose up -d --build

down:
	docker compose down

restart:
	docker compose restart backend

logs:
	docker compose logs -f backend frontend

build:
	docker compose up -d --build

frontend-dev:
	docker compose up --build frontend

frontend-build:
	cd frontend && npm run build

migrate:
	docker compose exec -T backend go run ./cmd/migrate/main.go up

migrate-down:
	docker compose exec -T backend go run ./cmd/migrate/main.go down
