-- Migration: 0023_model_name_tombstone.up.sql
-- Description: Rename soft-deleted product models so their API names can be reused.

UPDATE models
SET name = name || '@deleted:' || id
WHERE deleted_at IS NOT NULL
  AND name NOT LIKE '%@deleted:%';