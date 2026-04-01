.PHONY: up down restart logs build migrate migrate-down

up:
	docker compose up -d

down:
	docker compose down

restart:
	docker compose restart backend

logs:
	docker compose logs -f backend

build:
	docker compose up -d --build

migrate:
	docker compose exec -T backend go run ./cmd/migrate/main.go up

migrate-down:
	docker compose exec -T backend go run ./cmd/migrate/main.go down
