# 🌌 Lumen — Self-Hosted Discord Alternative

![Lumen Banner](https://via.placeholder.com/1200x400/5865F2/FFFFFF?text=Lumen+—+High+Performance+Chat+Engine)

**Lumen** — это современная, высокопроизводительная платформа для общения с открытым исходным кодом. Полный контроль над данными, безопасность и скорость Go в сочетании с гибкостью Next.js 15.

---

### 🚀 Технологический Стек

| Слой | Технологии |
| :--- | :--- |
| **Backend** | **Go 1.23+**, Fiber v2, GORM v2 (PostgreSQL), Redis (Pub/Sub) |
| **Real-time** | WebSocket (Gateway pattern), LiveKit (Voice/Video) |
| **Frontend** | **Next.js 15 (App Router)**, TypeScript, Tailwind CSS, shadcn/ui |
| **State** | TanStack Query v5, Zustand |
| **Infrastructure** | Docker & Compose, **golang-migrate**, S3 (MinIO) |

---

### ✅ Что уже реализовано (Backend Core)

- [x] **Clean Architecture:** Четкое разделение на `cmd`, `internal` (domain, service, repository) и `pkg`.
- [x] **Auth System:** JWT-авторизация с Claims (ID, Exp) и Middleware защитой доступа.
- [x] **Scalable Real-time:** WebSocket Hub с синхронизацией через **Redis Pub/Sub** (горизонтальное масштабирование).
- [x] **Database Engine:** PostgreSQL + GORM с поддержкой Soft Delete и автоматических миграций.
- [x] **Chat Logic:**
    - Создание гильдий (серверов) и вступление по уникальным инвайт-кодам.
    - Текстовые каналы и персистентное хранение истории сообщений.
    - **Cursor-based Pagination** для эффективной подгрузки истории (параметры `before`, `limit`).
- [x] **Stability & DX:** Структурированное логирование (`slog`), валидация DTO (`validator/v10`) и типизированные API ошибки.
- [x] **DevOps:** Готовый `docker-compose.yml` для локального запуска всей инфраструктуры.

---

### 🛠 В процессе (Ближайшие задачи)

- [ ] **Gateway Events:** Переход к типизированным событиям (`MESSAGE_CREATE`, `PRESENCE_UPDATE`, `TYPING_START`).
- [ ] **Voice Engine:** Интеграция **LiveKit** (генерация токенов доступа для голосовых комнат).
- [ ] **Permissions:** Система ролей и битовых масок прав доступа (Manage Channels, Send Messages и т.д.).
- [ ] **Migration System:** Переход с AutoMigrate на контролируемые SQL-миграции (`golang-migrate`).
- [ ] **Frontend Bootstrap:** Инициализация Next.js приложения и базовый интерфейс чата.

---

### 📈 Roadmap

1.  **Phase 1: Foundation (Current)** — Стабильный API, Auth, Базовый чат и Websocket Gateway.
2.  **Phase 2: Media & Voice** — Интеграция LiveKit, загрузка вложений в S3-совместимое хранилище (MinIO).
3.  **Phase 3: Social & UX** — Список друзей, личные сообщения (DM), реакции, эмодзи и поиск.
4.  **Phase 4: Scaling** — Модерация (Kick/Ban), аудит-логи, поддержка Kubernetes и мобильная адаптация.

---

### 📦 Быстрый старт

**1. Подготовка окружения:**
Создайте файл `.env` в папке `backend/` на основе примера:
```env
DB_HOST=postgres
DB_USER=lumen
DB_PASSWORD=lumen
DB_NAME=lumen
JWT_SECRET=your_super_secret_key
REDIS_ADDR=redis:6379
