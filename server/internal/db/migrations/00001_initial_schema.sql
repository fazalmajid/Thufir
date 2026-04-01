-- +goose Up

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Auth: Users (created first — referenced by data tables)
CREATE TABLE name (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    display_name VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Auth: WebAuthn credentials (one per device/passkey)
CREATE TABLE credential (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES name(id) ON DELETE CASCADE,
    credential_id TEXT NOT NULL UNIQUE,   -- WebAuthn credential ID (base64url)
    public_key BYTEA NOT NULL,            -- COSE-encoded public key
    sign_count BIGINT NOT NULL DEFAULT 0, -- Replay-attack counter
    transports TEXT[],                    -- e.g. internal, usb, ble, nfc
    device_name VARCHAR(100),             -- Optional label, e.g. "MacBook Touch ID"
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Auth: Sessions
CREATE TABLE session (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES name(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Areas: Top-level life areas (per user)
CREATE TABLE area (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES name(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    color VARCHAR(7),  -- Hex color code
    icon VARCHAR(50),  -- Icon identifier
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Projects: Collections of tasks (per user)
CREATE TABLE project (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES name(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    notes TEXT,
    area_id UUID REFERENCES area(id) ON DELETE SET NULL,
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
CREATE TABLE task (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES name(id) ON DELETE CASCADE,
    title VARCHAR(500) NOT NULL,
    notes TEXT,

    -- Hierarchy
    project_id UUID REFERENCES project(id) ON DELETE SET NULL,
    area_id UUID REFERENCES area(id) ON DELETE SET NULL,
    parent_task_id UUID REFERENCES task(id) ON DELETE CASCADE,

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
CREATE TABLE tag (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(50) NOT NULL UNIQUE,
    color VARCHAR(7),
    usage_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ── Indexes ────────────────────────────────────────────────────────────────────

CREATE INDEX idx_credential_user_id ON credential(user_id);
CREATE INDEX idx_session_user_id ON session(user_id);
CREATE INDEX idx_session_expires_at ON session(expires_at);

CREATE INDEX idx_area_user_id ON area(user_id);
CREATE INDEX idx_project_user_id ON project(user_id);
CREATE INDEX idx_project_area_id ON project(area_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_task_user_id ON task(user_id);
CREATE INDEX idx_task_project_id ON task(project_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_task_area_id ON task(area_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_task_parent_task_id ON task(parent_task_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_task_status ON task(user_id, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_task_scheduled_date ON task(user_id, scheduled_date) WHERE deleted_at IS NULL;
CREATE INDEX idx_task_updated_at ON task(user_id, updated_at, id);
CREATE INDEX idx_project_updated_at ON project(user_id, updated_at, id);
CREATE INDEX idx_area_updated_at ON area(user_id, updated_at, id);

-- ── Updated_at triggers ────────────────────────────────────────────────────────

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER update_area_updated_at    BEFORE UPDATE ON area    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_project_updated_at BEFORE UPDATE ON project FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_task_updated_at    BEFORE UPDATE ON task    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_tag_updated_at     BEFORE UPDATE ON tag     FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- +goose Down

DROP TRIGGER IF EXISTS update_tag_updated_at     ON tag;
DROP TRIGGER IF EXISTS update_task_updated_at    ON task;
DROP TRIGGER IF EXISTS update_project_updated_at ON project;
DROP TRIGGER IF EXISTS update_area_updated_at    ON area;
DROP FUNCTION IF EXISTS update_updated_at_column();

DROP TABLE IF EXISTS tag;
DROP TABLE IF EXISTS task;
DROP TABLE IF EXISTS project;
DROP TABLE IF EXISTS area;
DROP TABLE IF EXISTS session;
DROP TABLE IF EXISTS credential;
DROP TABLE IF EXISTS name;
