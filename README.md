# Lumen

**Lumen** — self-hosted чат-платформа в стиле Discord.

Репозиторий — **монорепозиторий**: в корне **`backend/`** (Go API), **`frontend/`** (Next.js), общие **`docker-compose.yml`** и **`Makefile`**. Клонирование и Docker — из корня; зависимости и скрипты фронтенда — из **`frontend/`**.

Текущий этап: **MVP backend** и **frontend** с чатом (история по HTTP, live по WebSocket), auth и защитой маршрутов через **Next.js `proxy.ts`**.

## Что уже реализовано

- **Auth:** `POST /api/auth/register`, `POST /api/auth/login`, JWT middleware, защищённые роуты.
- **Guilds и Channels:** создание гильдий, вступление по invite, создание/список каналов.
- **Сообщения:** отправка и чтение истории (`GET/POST /api/channels/:channelID/messages`); ответ истории в формате `MessagePayload` (как в WS).
- **WebSocket Gateway:** после upgrade клиент шлёт **`IDENTIFY`** с JWT в теле; сервер отвечает **`READY`**, затем подписки `SUBSCRIBE_CHANNEL`, события `MESSAGE_CREATE` и т.д. Токен **не** передаётся в query string.
- **Права:** проверка `PermSendMessages` при отправке сообщений.
- **Rate limit:** Redis-ограничение отправки сообщений (по умолчанию `10` сообщений за `10` секунд на `user+channel`).
- **Voice:** выдача LiveKit join token.
- **Инфраструктура:** Docker Compose, миграции `golang-migrate`, конфиг через `cleanenv`.
- **Frontend (`frontend/`):** Next.js 16, **`proxy.ts`** (редиректы auth вместо устаревшего `middleware.ts`), прокси HTTP API через rewrites, JWT в httpOnly + in-memory для Bearer, WS на `NEXT_PUBLIC_WS_URL` с **`IDENTIFY` → `READY`**, список сообщений (TanStack Query + Zustand), базовый **Markdown** в теле сообщения (`react-markdown`), auto-scroll, loading/error UI.

## Технологии

- **Backend:** Go, Fiber, GORM, PostgreSQL.
- **Frontend:** Next.js 16 (App Router), TypeScript, Tailwind, TanStack Query, Zustand, shadcn/ui, react-markdown.
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

После этого API: `http://localhost:8080`, при поднятом сервисе **`frontend`** в Compose — UI: `http://localhost:3000`.

### Вариант 2 (если `make` не установлен, например Windows PowerShell)

```powershell
git clone https://github.com/moongrammic/lumen.git
cd lumen
Copy-Item backend/.env.example backend/.env
docker compose up -d --build
docker compose exec -T backend go run ./cmd/migrate/main.go up
docker compose logs -f backend frontend
```

### Локальная разработка frontend (монорепо)

```bash
cd frontend
npm install
npm run dev
```

Убедитесь, что backend доступен по тем же URL, что ожидает фронт (по умолчанию `NEXT_PUBLIC_API_URL` и `NEXT_PUBLIC_WS_URL` указывают на `localhost:8080`). При необходимости задайте переменные в `frontend/.env.local` (см. ниже).

## Smoke/E2E тесты

- HTTP smoke тест:

```powershell
./scripts/test-api.ps1
```

- WebSocket e2e тест (IDENTIFY → READY → подписки → таргетированная доставка):

```powershell
./scripts/test-ws.ps1
```

Оба теста должны завершаться `PASSED`.

## WebSocket: контракт (кратко)

1. Подключение к `GET /ws` **без** JWT в URL (HTTP-маршрут не требует `Authorization` на upgrade).
2. Первое сообщение от клиента: `{ "op": 2, "event": "IDENTIFY", "payload": { "token": "<jwt>" } }`.
3. При успехе сервер шлёт `{ "event": "READY", ... }`, затем обрабатываются `SUBSCRIBE_CHANNEL`, `MESSAGE_CREATE` и остальное по `internal/service/chat.go` и `cmd/api/main.go`.
4. При ошибке идентификации: `{ "event": "ERROR", "payload": { "code": "IDENTIFY_FAILED", "message": "..." } }`, соединение закрывается со стороны клиента после обработки.

## Ключевые переменные окружения

### Backend

См. `backend/.env.example`.

Основные:

- `JWT_SECRET` — секрет подписи JWT.
- `DB_*` — параметры PostgreSQL.
- `REDIS_*` — параметры Redis и канал Pub/Sub.
- `RATE_LIMIT_MESSAGES_PER_10S` — лимит сообщений на `10` секунд.
- `LIVEKIT_*` — параметры интеграции LiveKit.

### Frontend

Для локального `npm run dev` удобно положить в **`frontend/.env.local`** (файл не коммитить):

- `NEXT_PUBLIC_API_URL` — база HTTP API (например `http://localhost:8080/api`).
- `NEXT_PUBLIC_WS_URL` — WebSocket (например `ws://localhost:8080/ws`).
- `BACKEND_URL` — origin backend **без** `/api` (для server-side rewrites в `next.config.ts`, в Docker Compose задаётся автоматически).

В **Docker Compose** для сервиса `frontend` те же переменные задаются в `docker-compose.yml`.

## Полезные команды

- `make up` / `make down` / `make restart`
- `make build`
- `make migrate` / `make migrate-down`
- `make logs` — логи `backend` и `frontend`
- `make frontend-dev` — поднять только frontend через Compose (с пересборкой)
- `make frontend-build` — production-сборка Next в `frontend/`

## Ближайшие шаги

- Подгрузка гильдий/каналов с API вместо заглушек в UI.
- Typing/presence в UI, LiveKit.
- Ужесточение лимитов/наблюдаемость на «голом» `/ws` (upgrade без pre-auth).
- Дальнейшая унификация формата ошибок WS/HTTP.
