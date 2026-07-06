DROP INDEX IF EXISTS idx_api_keys_user_status;

ALTER TABLE api_keys DROP COLUMN model_limits;
ALTER TABLE api_keys DROP COLUMN model_limits_enabled;
ALTER TABLE api_keys DROP COLUMN ip_blacklist;
ALTER TABLE api_keys DROP COLUMN ip_whitelist;
ALTER TABLE api_keys DROP COLUMN spend_used;
ALTER TABLE api_keys DROP COLUMN spend_cap;
ALTER TABLE api_keys DROP COLUMN status;