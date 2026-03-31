.PHONY: up down restart logs build migrate

up:
	docker-compose up -d

down:
	docker-compose down

restart:
	docker-compose restart backend

logs:
	docker-compose logs -f backend

build:
	docker-compose up -d --build

migrate:
	docker-compose exec backend go run cmd/migrate/main.go up