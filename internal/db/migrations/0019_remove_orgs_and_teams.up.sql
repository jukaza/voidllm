-- Migration: 0019_remove_orgs_and_teams.up.sql
-- Description: Completely remove organizations, teams, memberships, and related columns.

DROP TABLE IF EXISTS org_memberships;
DROP TABLE IF EXISTS team_memberships;
DROP TABLE IF EXISTS team_model_access;
DROP TABLE IF EXISTS org_model_access;
DROP TABLE IF EXISTS service_accounts;
DROP TABLE IF EXISTS teams;
DROP TABLE IF EXISTS organizations;

-- Alter api_keys to drop B2B scope columns
ALTER TABLE api_keys DROP COLUMN org_id;
ALTER TABLE api_keys DROP COLUMN team_id;
ALTER TABLE api_keys DROP COLUMN service_account_id;

-- Alter model_aliases to drop scope columns
DROP INDEX IF EXISTS idx_model_aliases_unique;
ALTER TABLE model_aliases DROP COLUMN scope_type;
ALTER TABLE model_aliases DROP COLUMN org_id;
ALTER TABLE model_aliases DROP COLUMN team_id;

CREATE UNIQUE INDEX idx_model_aliases_unique ON model_aliases (alias);
