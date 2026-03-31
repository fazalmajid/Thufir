-- +goose Up

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Auth: Users (created first — referenced by data tables)
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    display_name VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Auth: WebAuthn credentials (one per device/passkey)
CREATE TABLE credentials (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credential_id TEXT NOT NULL UNIQUE,   -- WebAuthn credential ID (base64url)
    public_key BYTEA NOT NULL,            -- COSE-encoded public key
    sign_count BIGINT NOT NULL DEFAULT 0, -- Replay-attack counter
    transports TEXT[],                    -- e.g. internal, usb, ble, nfc
    device_name VARCHAR(100),             -- Optional label, e.g. "MacBook Touch ID"
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Auth: Sessions
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Areas: Top-level life areas (per user)
CREATE TABLE areas (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    color VARCHAR(7),  -- Hex color code
    icon VARCHAR(50),  -- Icon identifier
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Projects: Collections of tasks (per user)
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    notes TEXT,
    area_id UUID REFERENCES areas(id) ON DELETE SET NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- active, completed, archived
    deadline DATE,
    tags VARCHAR(50)[],
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);

-- Tasks: Individual to-do items (per user)
CREATE TABLE tasks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(500) NOT NULL,
    notes TEXT,

    -- Hierarchy
    project_id UUID REFERENCES projects(id) ON DELETE SET NULL,
    area_id UUID REFERENCES areas(id) ON DELETE SET NULL,
    parent_task_id UUID REFERENCES tasks(id) ON DELETE CASCADE,

    -- Status & timing
    status VARCHAR(20) NOT NULL DEFAULT 'inbox', -- inbox, today, upcoming, anytime, someday, completed
    is_completed BOOLEAN NOT NULL DEFAULT FALSE,
    completed_at TIMESTAMPTZ,

    -- Dates
    start_date DATE,
    deadline DATE,
    scheduled_date DATE,

    -- Time
    start_time TIME,
    reminder_time TIMESTAMPTZ,

    -- Flags & priority
    is_flagged BOOLEAN NOT NULL DEFAULT FALSE,
    priority INTEGER DEFAULT 0, -- 0=none, 1=low, 2=medium, 3=high

    -- Tags
    tags VARCHAR(50)[],

    -- Ordering
    sort_order INTEGER NOT NULL DEFAULT 0,

    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Tags: Standalone tag definitions (global, not per-user)
CREATE TABLE tags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(50) NOT NULL UNIQUE,
    color VARCHAR(7),
    usage_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── Indexes ────────────────────────────────────────────────────────────────────

CREATE INDEX idx_credentials_user_id ON credentials(user_id);
CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

CREATE INDEX idx_areas_user_id ON areas(user_id);
CREATE INDEX idx_projects_user_id ON projects(user_id);
CREATE INDEX idx_projects_area_id ON projects(area_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_tasks_user_id ON tasks(user_id);
CREATE INDEX idx_tasks_project_id ON tasks(project_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_tasks_area_id ON tasks(area_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_tasks_parent_task_id ON tasks(parent_task_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_tasks_status ON tasks(user_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_tasks_scheduled_date ON tasks(user_id, scheduled_date) WHERE deleted_at IS NULL;
CREATE INDEX idx_tasks_updated_at ON tasks(user_id, updated_at, id);
CREATE INDEX idx_projects_updated_at ON projects(user_id, updated_at, id);
CREATE INDEX idx_areas_updated_at ON areas(user_id, updated_at, id);

-- ── Updated_at triggers ────────────────────────────────────────────────────────

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_areas_updated_at    BEFORE UPDATE ON areas    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_projects_updated_at BEFORE UPDATE ON projects FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_tasks_updated_at    BEFORE UPDATE ON tasks    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_tags_updated_at     BEFORE UPDATE ON tags     FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- +goose Down

DROP TRIGGER IF EXISTS update_tags_updated_at     ON tags;
DROP TRIGGER IF EXISTS update_tasks_updated_at    ON tasks;
DROP TRIGGER IF EXISTS update_projects_updated_at ON projects;
DROP TRIGGER IF EXISTS update_areas_updated_at    ON areas;
DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TABLE IF EXISTS tags;
DROP TABLE IF EXISTS tasks;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS areas;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS credentials;
DROP TABLE IF EXISTS users;
