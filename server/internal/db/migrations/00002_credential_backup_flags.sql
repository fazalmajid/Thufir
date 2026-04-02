-- +goose Up
ALTER TABLE credential
    ADD COLUMN IF NOT EXISTS backup_eligible boolean NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS backup_state    boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE credential
    DROP COLUMN IF EXISTS backup_eligible,
    DROP COLUMN IF EXISTS backup_state;
