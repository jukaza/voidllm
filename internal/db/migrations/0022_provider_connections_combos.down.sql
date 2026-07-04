-- Migration: 0022_provider_connections_combos.down.sql

DROP TABLE IF EXISTS model_route_steps;
DROP TABLE IF EXISTS provider_upstream_models;
DROP TABLE IF EXISTS provider_connections;

ALTER TABLE models DROP COLUMN routing_strategy;
ALTER TABLE models DROP COLUMN sticky_round_robin_limit;

ALTER TABLE providers DROP COLUMN connection_strategy;
ALTER TABLE providers DROP COLUMN sticky_round_robin_limit;