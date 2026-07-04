-- Migration: 0027_cache_pricing.up.sql
-- Cache pricing on upstream inventory, cache-write sell price on products.

ALTER TABLE provider_upstream_models ADD COLUMN cost_cached_input_per_1m REAL;
ALTER TABLE provider_upstream_models ADD COLUMN cost_cache_write_per_1m REAL;
ALTER TABLE models ADD COLUMN sell_cache_write_per_1m REAL;