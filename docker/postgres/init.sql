-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Areas: Top-level life areas
CREATE TABLE areas (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    color VARCHAR(7), -- Hex color code
    icon VARCHAR(50), -- Icon identifier
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ -- Soft delete
);

-- Projects: Collections of tasks
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    notes TEXT,
    area_id UUID REFERENCES areas(id) ON DELETE SET NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- active, completed, archived
    deadline DATE,
    tags VARCHAR(50)[], -- Array of tag names for quick filtering
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);

-- Tasks: Individual to-do items
CREATE TABLE tasks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(500) NOT NULL,
    notes TEXT,

    -- Hierarchy
    project_id UUID REFERENCES projects(id) ON DELETE SET NULL,
    area_id UUID REFERENCES areas(id) ON DELETE SET NULL, -- For tasks without project
    parent_task_id UUID REFERENCES tasks(id) ON DELETE CASCADE, -- For subtasks

    -- Status & timing
    status VARCHAR(20) NOT NULL DEFAULT 'inbox', -- inbox, today, upcoming, anytime, someday, completed
    is_completed BOOLEAN NOT NULL DEFAULT FALSE,
    completed_at TIMESTAMPTZ,

    -- Dates
    start_date DATE, -- When task becomes available
    deadline DATE,
    scheduled_date DATE, -- Explicitly scheduled for a date

    -- Time
    start_time TIME, -- Optional time of day
    reminder_time TIMESTAMPTZ,

    -- Flags & priority
    is_flagged BOOLEAN NOT NULL DEFAULT FALSE,
    priority INTEGER DEFAULT 0, -- 0=none, 1=low, 2=medium, 3=high

    -- Tags
    tags VARCHAR(50)[], -- Denormalized for quick filtering

    -- Ordering
    sort_order INTEGER NOT NULL DEFAULT 0,

    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Tags: Standalone tag definitions
CREATE TABLE tags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(50) NOT NULL UNIQUE,
    color VARCHAR(7),
    usage_count INTEGER NOT NULL DEFAULT 0, -- Denormalized count
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_tasks_project_id ON tasks(project_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_tasks_area_id ON tasks(area_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_tasks_parent_task_id ON tasks(parent_task_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_tasks_status ON tasks(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_tasks_scheduled_date ON tasks(scheduled_date) WHERE deleted_at IS NULL;
CREATE INDEX idx_tasks_deadline ON tasks(deadline) WHERE deleted_at IS NULL;
CREATE INDEX idx_tasks_is_completed ON tasks(is_completed) WHERE deleted_at IS NULL;
CREATE INDEX idx_tasks_tags ON tasks USING GIN(tags) WHERE deleted_at IS NULL;
CREATE INDEX idx_projects_area_id ON projects(area_id) WHERE deleted_at IS NULL;

-- Updated_at triggers
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_areas_updated_at BEFORE UPDATE ON areas
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_projects_updated_at BEFORE UPDATE ON projects
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_tasks_updated_at BEFORE UPDATE ON tasks
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_tags_updated_at BEFORE UPDATE ON tags
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Auth: Users
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
    public_key BYTEA NOT NULL,            -- Stored public key
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

CREATE INDEX idx_credentials_user_id ON credentials(user_id);
CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

-- Insert some sample data for testing
INSERT INTO areas (id, name, color, sort_order) VALUES
    ('00000000-0000-0000-0000-000000000001', 'Personal', '#3b82f6', 1),
    ('00000000-0000-0000-0000-000000000002', 'Work', '#10b981', 2);

INSERT INTO projects (id, name, area_id, sort_order) VALUES
    ('00000000-0000-0000-0000-000000000011', 'Home Improvement', '00000000-0000-0000-0000-000000000001', 1),
    ('00000000-0000-0000-0000-000000000012', 'Q4 Planning', '00000000-0000-0000-0000-000000000002', 1);

INSERT INTO tasks (id, title, status, project_id, sort_order) VALUES
    ('00000000-0000-0000-0000-000000000101', 'Buy groceries', 'today', NULL, 1),
    ('00000000-0000-0000-0000-000000000102', 'Review Q4 goals', 'inbox', '00000000-0000-0000-0000-000000000012', 2),
    ('00000000-0000-0000-0000-000000000103', 'Paint living room', 'upcoming', '00000000-0000-0000-0000-000000000011', 3);
