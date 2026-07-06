-- Migration: 0036_remove_totp.down.sql
-- Description: Recreate TOTP schema objects from 0035 (no data restore).

ALTER TABLE users ADD COLUMN totp_secret_encrypted TEXT;
ALTER TABLE users ADD COLUMN totp_enabled_at TEXT;

CREATE TABLE IF NOT EXISTS user_totp_backup_codes (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id),
    code_hash   TEXT NOT NULL,
    used_at     TEXT,
    created_at  TEXT NOT NULL DEFAULT (CURRENT_TIMESTAMP)
);

CREATE INDEX IF NOT EXISTS idx_user_totp_backup_codes_user_unused
    ON user_totp_backup_codes (user_id)
    WHERE used_at IS NULL;