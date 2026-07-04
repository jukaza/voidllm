-- Migration: 0029_usage_log_enhancements.up.sql
-- Description: Per-request log metadata, error classification, and richer rollups.

ALTER TABLE usage_events ADD COLUMN log_type TEXT NOT NULL DEFAULT 'consume';
ALTER TABLE usage_events ADD COLUMN is_stream INTEGER NOT NULL DEFAULT 0;
ALTER TABLE usage_events ADD COLUMN cache_write_tokens INTEGER NOT NULL DEFAULT 0;
ALTER TABLE usage_events ADD COLUMN meta TEXT NOT NULL DEFAULT '{}';

ALTER TABLE usage_hourly ADD COLUMN cached_tokens INTEGER NOT NULL DEFAULT 0;
ALTER TABLE usage_hourly ADD COLUMN revenue_sum REAL NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_usage_events_time_id ON usage_events (created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_usage_events_user_time ON usage_events (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_usage_events_deployment_time ON usage_events (deployment_id, created_at DESC);