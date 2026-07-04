-- Migration: 0030_usage_daily.up.sql
-- Description: Daily usage rollups for fast analytics over multi-day windows.

CREATE TABLE usage_daily (
    day               TEXT NOT NULL,
    user_id           TEXT NOT NULL DEFAULT '',
    requests          INTEGER NOT NULL DEFAULT 0,
    prompt_tokens     INTEGER NOT NULL DEFAULT 0,
    completion_tokens INTEGER NOT NULL DEFAULT 0,
    cached_tokens     INTEGER NOT NULL DEFAULT 0,
    cost_sum          REAL NOT NULL DEFAULT 0,
    revenue_sum       REAL NOT NULL DEFAULT 0,
    by_model          TEXT NOT NULL DEFAULT '{}',
    by_deployment     TEXT NOT NULL DEFAULT '{}',
    PRIMARY KEY (day, user_id)
);

CREATE INDEX idx_usage_daily_day ON usage_daily (day);