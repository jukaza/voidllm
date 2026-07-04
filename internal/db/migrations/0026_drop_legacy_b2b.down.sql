-- Migration: 0026_drop_legacy_b2b.down.sql
-- Description: Best-effort reverse of 0026_drop_legacy_b2b.up.sql.
-- Recreates dropped columns with empty defaults; does not restore dropped tables.

ALTER TABLE audit_logs ADD COLUMN org_id TEXT NOT NULL DEFAULT '';
ALTER TABLE usage_hourly ADD COLUMN team_id TEXT NOT NULL DEFAULT '';
ALTER TABLE usage_hourly ADD COLUMN org_id TEXT NOT NULL DEFAULT '';
ALTER TABLE usage_events ADD COLUMN service_account_id TEXT;
ALTER TABLE usage_events ADD COLUMN team_id TEXT;
ALTER TABLE usage_events ADD COLUMN org_id TEXT NOT NULL DEFAULT '';