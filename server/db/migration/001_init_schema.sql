-- =================================================================
--          FINAL DATABASE SCHEMA FOR PROJECT "CONCORDIA"
-- =================================================================

-- I. Определение Пользовательских Типов (ENUMs)
-- -----------------------------------------------------------------

CREATE TYPE channel_type AS ENUM ('TEXT', 'VOICE');
CREATE TYPE friendship_status AS ENUM ('PENDING', 'ACCEPTED', 'BLOCKED');
CREATE TYPE user_status AS ENUM ('ONLINE', 'IDLE', 'DND', 'OFFLINE');
CREATE TYPE mention_type AS ENUM ('USER', 'ROLE');
CREATE TYPE submission_status AS ENUM ('ASSIGNED', 'TURNED_IN', 'LATE', 'RETURNED');


-- II. Основные Сущности (Core Entities)
-- -----------------------------------------------------------------

-- Пользователи
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(32) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255),
    avatar_url TEXT,
    about_me TEXT,
    social_links JSONB,
    status user_status DEFAULT 'OFFLINE',
    custom_status TEXT,
    provider VARCHAR(50) NOT NULL DEFAULT 'email',
    provider_id VARCHAR(255),
    settings JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Рабочие пространства (Серверы/Команды)
CREATE TABLE workspaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    icon_url TEXT,
    is_public BOOLEAN DEFAULT false,
    description TEXT,
    tags TEXT[],
    welcome_channel_id UUID, -- Может ссылаться на channels(id) после создания
    welcome_message_template TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Каналы внутри пространств
CREATE TABLE channels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    type channel_type NOT NULL DEFAULT 'TEXT',
    topic VARCHAR(255),
    position INT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Сообщения в текстовых каналах
CREATE TABLE messages (
    id BIGSERIAL PRIMARY KEY,
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    author_id UUID REFERENCES users(id) ON DELETE SET NULL,
    content TEXT,
    parent_id BIGINT REFERENCES messages(id) ON DELETE SET NULL, -- Для тредов
    attachments JSONB,
    reactions JSONB,
    is_scheduled BOOLEAN DEFAULT false,
    scheduled_for TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);


-- III. Модерация и Управление Доступом
-- -----------------------------------------------------------------

-- Роли
CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name VARCHAR(50) NOT NULL,
    permissions BIGINT NOT NULL DEFAULT 0, -- Битовая маска прав
    UNIQUE(workspace_id, name)
);

-- Участники пространств (связь User <-> Workspace)
CREATE TABLE members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    role_id UUID REFERENCES roles(id) ON DELETE SET NULL,
    nickname VARCHAR(32),
    notification_settings JSONB,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, workspace_id)
);

-- Журнал аудита действий
CREATE TABLE audit_logs (
    id BIGSERIAL PRIMARY KEY,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    actor_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action_type VARCHAR(50) NOT NULL,
    target_type VARCHAR(50),
    target_id TEXT,
    changes JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Управление приглашениями
CREATE TABLE invites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(16) NOT NULL UNIQUE,
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    creator_id UUID REFERENCES users(id) ON DELETE SET NULL,
    max_uses INT DEFAULT 0, -- 0 = бесконечно
    current_uses INT DEFAULT 0,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);


-- IV. Социальные и Коммуникационные Функции
-- -----------------------------------------------------------------

-- Система дружбы
CREATE TABLE friendships (
    user_one_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_two_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status friendship_status NOT NULL,
    action_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (user_one_id, user_two_id)
);

-- Личные чаты (DM) - группировка
CREATE TABLE conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    is_group BOOLEAN NOT NULL DEFAULT false,
    name VARCHAR(100)
);

-- Личные чаты - участники
CREATE TABLE conversation_participants (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, conversation_id)
);

-- Личные чаты - сообщения
CREATE TABLE direct_messages (
    id BIGSERIAL PRIMARY KEY,
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    author_id UUID REFERENCES users(id) ON DELETE SET NULL,
    content TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- @-Упоминания
CREATE TABLE mentions (
    message_id BIGINT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    mentioned_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    mention_type mention_type NOT NULL DEFAULT 'USER',
    PRIMARY KEY (message_id, mentioned_user_id)
);

-- Статусы прочтения
CREATE TABLE message_read_status (
    message_id BIGINT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    read_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (message_id, user_id)
);

-- Метаданные записей звонков
CREATE TABLE recordings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_id UUID REFERENCES channels(id) ON DELETE SET NULL,
    scheduled_event_id UUID, -- Будет ссылаться на scheduled_events
    started_by_id UUID REFERENCES users(id) ON DELETE SET NULL,
    file_url TEXT NOT NULL,
    file_size_bytes BIGINT,
    duration_seconds INT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);


-- V. Продуктивность, Организация и Обучение
-- -----------------------------------------------------------------

-- Календарь и запланированные события
CREATE TABLE scheduled_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    channel_id UUID REFERENCES channels(id) ON DELETE SET NULL,
    creator_id UUID REFERENCES users(id) ON DELETE SET NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL
);

-- Добавляем внешний ключ в recordings после создания scheduled_events
ALTER TABLE recordings ADD CONSTRAINT fk_scheduled_event
FOREIGN KEY (scheduled_event_id) REFERENCES scheduled_events(id) ON DELETE SET NULL;

-- Закрепленные сообщения
CREATE TABLE pinned_messages (
    message_id BIGINT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    pinned_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    pinned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (message_id, channel_id)
);

-- Вебхуки для интеграций
CREATE TABLE webhooks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    token_hash VARCHAR(255) NOT NULL,
    avatar_url TEXT
);

-- Опросы
CREATE TABLE polls (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id BIGINT NOT NULL REFERENCES messages(id) ON DELETE CASCADE UNIQUE
);

CREATE TABLE poll_options (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    poll_id UUID NOT NULL REFERENCES polls(id) ON DELETE CASCADE,
    option_text TEXT NOT NULL
);

CREATE TABLE poll_votes (
    option_id UUID NOT NULL REFERENCES poll_options(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (option_id, user_id)
);

-- Персональные задачи
CREATE TABLE tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    is_completed BOOLEAN NOT NULL DEFAULT false,
    due_date DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Система заданий (Assignments)
CREATE TABLE assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    creator_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    instructions TEXT,
    points_possible SMALLINT,
    due_date TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Сданные работы
CREATE TABLE assignment_submissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    assignment_id UUID NOT NULL REFERENCES assignments(id) ON DELETE CASCADE,
    student_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status submission_status NOT NULL DEFAULT 'ASSIGNED',
    submitted_at TIMESTAMPTZ,
    attachments JSONB,
    grade SMALLINT,
    feedback TEXT,
    UNIQUE (assignment_id, student_id)
);


-- VI. Индексы для Производительности
-- -----------------------------------------------------------------

CREATE INDEX idx_messages_channel_id ON messages(channel_id);
CREATE INDEX idx_members_user_id ON members(user_id);
CREATE INDEX idx_members_workspace_id ON members(workspace_id);
CREATE INDEX idx_audit_logs_workspace_id ON audit_logs(workspace_id);
CREATE INDEX idx_tasks_user_id ON tasks(user_id);
CREATE INDEX idx_mentions_mentioned_user_id ON mentions(mentioned_user_id);
CREATE INDEX idx_assignments_channel_id ON assignments(channel_id);
CREATE INDEX idx_submissions_student_id ON assignment_submissions(student_id);
CREATE INDEX idx_submissions_assignment_id ON assignment_submissions(assignment_id);