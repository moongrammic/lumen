# Lumen — современный self-hosted аналог Discord

![Lumen Banner](https://via.placeholder.com/1200x400/5865F2/FFFFFF?text=Lumen+—+Chat+Reimagined)  
*(Замени на реальный баннер позже — сделай в Figma или Canva)*

**Lumen** — это открытый, высокопроизводительный и полностью self-hosted чат-приложение, вдохновлённое Discord.  
Создавай серверы, общайся в реальном времени, проводи голосовые и видеозвонки — всё под твоим полным контролем.

### Почему Lumen?

- **Высокая производительность** — бэкенд на Go (Fiber/Echo + Gorilla WebSocket) + PostgreSQL.
- **Современный UI** — полностью на Next.js 15 (App Router) + TypeScript + Tailwind + shadcn/ui.
- **Реал-тайм** — WebSocket + Redis Pub/Sub.
- **Голос и видео** — LiveKit (нативно на Go).
- **Полная приватность** — никаких облачных сервисов по умолчанию, всё self-hosted.
- **Масштабируемость** — готов к запуску на одном сервере или в кластере.

### Текущий статус (MVP Roadmap)

**Сделано:**
- Базовая структура Go-бэкенда (модели, миграции, JWT auth — обнови список по факту).

**В процессе / Планы на ближайшие 2–4 недели:**
- Полноценная система серверов, каналов, сообщений (текст + вложения).
- Реал-тайм чат через WebSocket.
- Next.js фронтенд с современным Discord-подобным интерфейсом (тёмная/светлая тема, responsive).
- Базовая модерация (kick/ban, роли).

**Дальше (в порядке приоритета):**
1. Permissions & role overrides
2. LiveKit voice/video + screen sharing
3. Приглашения, друзья, DM
4. Поиск сообщений, эмодзи, реакции
5. Мобильная адаптация + PWA
6. Docker + Kubernetes готовность

### Технологический стек

**Backend**
- Go 1.23+
- PostgreSQL + GORM
- Fiber / Echo (роутинг)
- Gorilla WebSocket + Redis Pub/Sub
- LiveKit (voice/video)
- JWT + bcrypt
- MinIO (S3-совместимое хранилище файлов)

**Frontend**
- Next.js 15 (App Router)
- TypeScript
- Tailwind CSS + shadcn/ui
- TanStack Query + Zustand
- LiveKit React SDK

**DevOps**
- Docker + Docker Compose
- goose / golang-migrate (миграции БД)
- Makefile для удобства

### Быстрый старт (локальная разработка)

```bash
# 1. Клонируем репозиторий
git clone https://github.com/moongrammic/lumen.git
cd lumen

# 2. Запускаем всё через Docker Compose
docker compose up -d --build

# 3. Применяем миграции БД (внутри backend контейнера)
docker compose exec backend go run cmd/migrate/main.go up

# 4. Открываем
Frontend: http://localhost:3000
Backend API: http://localhost:8080
