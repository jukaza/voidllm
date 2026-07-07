-- Migration: 0042_user_roles.down.sql

UPDATE users SET is_system_admin = 1 WHERE role = 'root';
UPDATE users SET is_system_admin = 0 WHERE role != 'root';

DROP INDEX IF EXISTS idx_users_status;
DROP INDEX IF EXISTS idx_users_role;