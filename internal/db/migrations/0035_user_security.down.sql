DROP INDEX IF EXISTS idx_user_totp_backup_codes_user_unused;
DROP TABLE IF EXISTS user_totp_backup_codes;

-- SQLite does not support DROP COLUMN; down migration is best-effort for dev rollback.