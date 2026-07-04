-- Migration: 0026_drop_legacy_b2b.up.sql
-- Description: Remove orphaned B2B/MCP/invite tables and org/team columns from
-- usage and audit tables after the single-tenant marketplace pivot.

-- MCP gateway stack (no API handlers remain).
DROP TABLE IF EXISTS output_schemas;
DROP TABLE IF EXISTS mcp_server_tools;
DROP TABLE IF EXISTS mcp_tool_blocklist;
DROP TABLE IF EXISTS mcp_tool_calls;
DROP TABLE IF EXISTS key_mcp_access;
DROP TABLE IF EXISTS team_mcp_access;
DROP TABLE IF EXISTS org_mcp_access;
DROP TABLE IF EXISTS mcp_servers;

-- Org-scoped features removed in 0019.
DROP TABLE IF EXISTS invite_tokens;
DROP TABLE IF EXISTS org_sso_config;
DROP TABLE IF EXISTS key_model_access;

-- Indexes tied to dropped columns.
DROP INDEX IF EXISTS idx_usage_org_time;
DROP INDEX IF EXISTS idx_usage_team_time;
DROP INDEX IF EXISTS idx_usage_hourly_org;
DROP INDEX IF EXISTS idx_usage_hourly_team;
DROP INDEX IF EXISTS idx_audit_logs_org_time;

-- usage_events: drop B2B scope columns.
ALTER TABLE usage_events DROP COLUMN org_id;
ALTER TABLE usage_events DROP COLUMN team_id;
ALTER TABLE usage_events DROP COLUMN service_account_id;

-- usage_hourly: drop org/team rollup dimensions.
ALTER TABLE usage_hourly DROP COLUMN org_id;
ALTER TABLE usage_hourly DROP COLUMN team_id;

-- audit_logs: drop org context column.
ALTER TABLE audit_logs DROP COLUMN org_id;