-- User 2FA fields, session metadata on api_keys, backup codes table.

ALTER TABLE users ADD COLUMN totp_secret_encrypted TEXT;
ALTER TABLE users ADD COLUMN totp_enabled_at TEXT;

ALTER TABLE api_keys ADD COLUMN login_ip TEXT;
ALTER TABLE api_keys ADD COLUMN user_agent TEXT;

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