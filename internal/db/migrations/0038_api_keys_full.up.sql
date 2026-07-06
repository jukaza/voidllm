-- API key management: status, spend cap, IP ACL, model allowlist.

ALTER TABLE api_keys ADD COLUMN status TEXT NOT NULL DEFAULT 'active';
ALTER TABLE api_keys ADD COLUMN spend_cap REAL NOT NULL DEFAULT 0;
ALTER TABLE api_keys ADD COLUMN spend_used REAL NOT NULL DEFAULT 0;
ALTER TABLE api_keys ADD COLUMN ip_whitelist TEXT NOT NULL DEFAULT '';
ALTER TABLE api_keys ADD COLUMN ip_blacklist TEXT NOT NULL DEFAULT '';
ALTER TABLE api_keys ADD COLUMN model_limits_enabled INTEGER NOT NULL DEFAULT 0;
ALTER TABLE api_keys ADD COLUMN model_limits TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_api_keys_user_status
    ON api_keys (user_id, status)
    WHERE deleted_at IS NULL AND key_type = 'user_key';