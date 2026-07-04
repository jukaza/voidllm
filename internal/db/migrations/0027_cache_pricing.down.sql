-- Migration: 0027_cache_pricing.down.sql

ALTER TABLE models DROP COLUMN sell_cache_write_per_1m;
ALTER TABLE provider_upstream_models DROP COLUMN cost_cache_write_per_1m;
ALTER TABLE provider_upstream_models DROP COLUMN cost_cached_input_per_1m;