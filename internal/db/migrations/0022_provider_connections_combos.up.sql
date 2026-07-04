-- Migration: 0022_provider_connections_combos.up.sql
-- Description: Multi-key provider connections, per-provider upstream model
-- inventory, and product combo route steps. Product models gain combo routing
-- fields; providers gain per-provider connection selection defaults.

CREATE TABLE provider_connections (
    id                    TEXT PRIMARY KEY,
    provider_id           TEXT NOT NULL REFERENCES providers(id),
    name                  TEXT NOT NULL,
    auth_type             TEXT NOT NULL DEFAULT 'apikey',
    priority              INTEGER NOT NULL DEFAULT 1,
    is_active             INTEGER NOT NULL DEFAULT 1 CHECK (is_active IN (0, 1)),
    api_key_encrypted     TEXT,
    test_status           TEXT NOT NULL DEFAULT 'unknown',
    last_error            TEXT NOT NULL DEFAULT '',
    error_code            INTEGER,
    last_error_at         TEXT,
    backoff_level         INTEGER NOT NULL DEFAULT 0,
    locked_until          TEXT,
    model_locks           TEXT NOT NULL DEFAULT '{}',
    last_used_at          TEXT,
    consecutive_use_count INTEGER NOT NULL DEFAULT 0,
    created_at            TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at            TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at            TEXT
);

CREATE INDEX idx_provider_connections_provider
    ON provider_connections (provider_id, priority)
    WHERE deleted_at IS NULL;

CREATE TABLE provider_upstream_models (
    id                 TEXT PRIMARY KEY,
    provider_id        TEXT NOT NULL REFERENCES providers(id),
    upstream_id        TEXT NOT NULL,
    display_name       TEXT NOT NULL DEFAULT '',
    is_enabled         INTEGER NOT NULL DEFAULT 1 CHECK (is_enabled IN (0, 1)),
    cost_input_per_1m  REAL,
    cost_output_per_1m REAL,
    metadata           TEXT NOT NULL DEFAULT '{}',
    created_at         TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at         TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (provider_id, upstream_id)
);

CREATE INDEX idx_provider_upstream_models_provider
    ON provider_upstream_models (provider_id);

CREATE TABLE model_route_steps (
    id              TEXT PRIMARY KEY,
    model_id        TEXT NOT NULL REFERENCES models(id) ON DELETE CASCADE,
    position        INTEGER NOT NULL,
    provider_id     TEXT NOT NULL REFERENCES providers(id),
    upstream_model  TEXT NOT NULL,
    is_enabled      INTEGER NOT NULL DEFAULT 1 CHECK (is_enabled IN (0, 1)),
    created_at      TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (model_id, position)
);

CREATE INDEX idx_model_route_steps_model
    ON model_route_steps (model_id, position);

ALTER TABLE models ADD COLUMN routing_strategy TEXT NOT NULL DEFAULT 'fallback';
ALTER TABLE models ADD COLUMN sticky_round_robin_limit INTEGER NOT NULL DEFAULT 1;

ALTER TABLE providers ADD COLUMN connection_strategy TEXT NOT NULL DEFAULT 'fill-first';
ALTER TABLE providers ADD COLUMN sticky_round_robin_limit INTEGER NOT NULL DEFAULT 3;