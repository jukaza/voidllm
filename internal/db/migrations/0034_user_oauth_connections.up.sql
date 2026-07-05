CREATE TABLE IF NOT EXISTS user_oauth_connections (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id),
    provider    TEXT NOT NULL,
    external_id TEXT NOT NULL,
    label       TEXT,
    created_at  TEXT NOT NULL DEFAULT (CURRENT_TIMESTAMP),
    UNIQUE (provider, external_id),
    UNIQUE (user_id, provider)
);

CREATE INDEX IF NOT EXISTS idx_user_oauth_connections_user_id
    ON user_oauth_connections (user_id);