-- Migration: 0042_user_roles.up.sql
-- Add role + status columns; migrate is_system_admin -> role.
-- is_system_admin is left in place (SQLite FK makes table-rebuild unsafe in-tx).

ALTER TABLE users ADD COLUMN role TEXT NOT NULL DEFAULT 'member';
ALTER TABLE users ADD COLUMN status TEXT NOT NULL DEFAULT 'active';

UPDATE users SET role = 'root' WHERE is_system_admin = 1;

CREATE INDEX IF NOT EXISTS idx_users_role ON users (role) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_users_status ON users (status) WHERE deleted_at IS NULL;