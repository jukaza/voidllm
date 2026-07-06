-- Migration: 0036_remove_totp.up.sql
-- Description: Remove TOTP 2FA tables/columns added in 0035. Keeps api_keys session metadata.

UPDATE users SET totp_secret_encrypted = NULL, totp_enabled_at = NULL
WHERE totp_secret_encrypted IS NOT NULL OR totp_enabled_at IS NOT NULL;

DROP INDEX IF EXISTS idx_user_totp_backup_codes_user_unused;
DROP TABLE IF EXISTS user_totp_backup_codes;

ALTER TABLE users DROP COLUMN totp_secret_encrypted;
ALTER TABLE users DROP COLUMN totp_enabled_at;

DELETE FROM settings WHERE key LIKE 'security.two_fa.%';