-- Migration: 0020_provider_slug_routes.down.sql

DROP INDEX IF EXISTS idx_providers_slug;
ALTER TABLE providers DROP COLUMN slug;
ALTER TABLE providers DROP COLUMN protocol;
ALTER TABLE providers DROP COLUMN logo;
ALTER TABLE providers DROP COLUMN base_url;
ALTER TABLE providers DROP COLUMN api_key_encrypted;

ALTER TABLE models DROP COLUMN logo;

ALTER TABLE model_deployments DROP COLUMN upstream_model;
ALTER TABLE model_deployments DROP COLUMN cost_cached_input_per_1m;
ALTER TABLE model_deployments DROP COLUMN cost_cache_write_per_1m;
