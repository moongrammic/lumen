# Lumen

**Lumen** - self-hosted чат-платформа в стиле Discord.  
Текущий этап: **MVP backend готов к интеграции с frontend**.

## Что уже реализовано

- **Auth:** `POST /api/auth/register`, `POST /api/auth/login`, JWT middleware, защищенные роуты.
- **Guilds и Channels:** создание гильдий, вступление по invite, создание/список каналов.
- **Сообщения:** отправка и чтение истории (`/api/channels/:channelID/messages`).
- **WebSocket Gateway:** подписки на каналы, `MESSAGE_CREATE`, таргетированная доставка.
- **Права:** проверка `PermSendMessages` при отправке сообщений.
- **Rate limit:** Redis-ограничение отправки сообщений (по умолчанию `10` сообщений за `10` секунд на `user+channel`).
- **Voice:** выдача LiveKit join token.
- **Инфраструктура:** Docker Compose, миграции `golang-migrate`, конфиг через `cleanenv`.

## Технологии

- **Backend:** Go, Fiber, GORM, PostgreSQL.
- **Realtime:** WebSocket + Redis Pub/Sub.
- **Voice:** LiveKit.
- **Infra:** Docker, Docker Compose, Makefile.

## Быстрый старт

### Вариант 1 (через Makefile)

```bash
git clone https://github.com/moongrammic/lumen.git
cd lumen
cp backend/.env.example backend/.env
make up
make migrate
make logs
```

### Вариант 2 (если `make` не установлен, например Windows PowerShell)

```powershell
git clone https://github.com/moongrammic/lumen.git
cd lumen
Copy-Item backend/.env.example backend/.env
docker compose up -d --build
docker compose exec -T backend go run ./cmd/migrate/main.go up
docker compose logs -f backend
```

## Smoke/E2E тесты

- HTTP smoke тест:

```powershell
./scripts/test-api.ps1
```

- WebSocket e2e тест (подписки + таргетированная доставка):

```powershell
./scripts/test-ws.ps1
```

Оба теста должны завершаться `PASSED`.

## Ключевые переменные окружения

См. `backend/.env.example`.

Основные:
- `JWT_SECRET` - секрет подписи JWT.
- `DB_*` - параметры PostgreSQL.
- `REDIS_*` - параметры Redis и канал Pub/Sub.
- `RATE_LIMIT_MESSAGES_PER_10S` - лимит сообщений на `10` секунд.
- `LIVEKIT_*` - параметры интеграции LiveKit.

## Полезные команды

- `make up` / `make down` / `make restart`
- `make build`
- `make migrate` / `make migrate-down`
- `make logs`

## Ближайшие шаги

- Запуск `frontend/` (Next.js) поверх текущего API/WS.
- Улучшение WS reconnection handling.
- Дальнейшая унификация формата ошибок WS/HTTP.