# 🌌 Lumen

![Lumen Banner](<img width="2000" height="2000" alt="lumen" src="https://github.com/user-attachments/assets/3521d5b9-956f-475d-893b-8ee9c68bd9e7" />)

**Lumen** — современная self-hosted чат-платформа, вдохновленная Discord.  
Проект находится в стадии активной разработки (MVP).

## 🚀 Текущий статус

На данный момент реализовано ядро бэкенда:
- **Auth:** middleware для проверки JWT и защиты маршрутов.
- **Guilds:** создание серверов и вступление по инвайт-коду.
- **Messages API:** чтение истории сообщений с проверкой доступа к гильдии.
- **WebSocket Gateway:** события реального времени (`MESSAGE_CREATE`, `TYPING_START`, `PRESENCE_UPDATE`, `VOICE_STATE_UPDATE`).
- **Real-time Engine:** Redis Pub/Sub + presence-статусы через Redis TTL.
- **Voice:** эндпоинт генерации LiveKit join-токена.
- **Database:** SQL-миграции в `backend/migrations`.

**В разработке:**
- полный цикл авторизации (регистрация и логин);
- CRUD текстовых и голосовых каналов;
- UI/API для управления гильдиями;
- фронтенд на Next.js 15.

## 🛠 Технологический стек

- **Backend:** Go (Fiber), PostgreSQL (GORM), Redis.
- **Real-time:** WebSocket Gateway.
- **Voice/Video:** LiveKit (интеграция токенов).
- **DevOps:** Docker Compose, Makefile.

## 📦 Быстрый старт

```bash
# 1. Клонируйте репозиторий
git clone https://github.com/moongrammic/lumen.git
cd lumen

# 2. Настройте конфигурацию (JWT_SECRET, ключи LiveKit и т.д.)
cp backend/.env.example backend/.env

# 3. Запустите инфраструктуру
make up

# 4. Примените миграции БД
make migrate

# 5. Просмотр логов бэкенда
make logs
```

Доступные адреса:
- Backend API + WebSocket: `http://localhost:8080`
- Frontend: в разработке

## 📋 Полезные команды

- `make up` — запустить сервисы в Docker.
- `make down` — остановить сервисы.
- `make build` — пересобрать контейнеры.
- `make migrate` — применить SQL-миграции (up).
- `make migrate-down` — откатить миграции (down).

## ⚙️ Переменные окружения

Используйте `backend/.env.example` как источник актуальных переменных для локального запуска.

---
Lumen — Built for privacy, engineered for speed.