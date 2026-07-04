-- Migration: 0021_model_pricing_modes.down.sql

ALTER TABLE models DROP COLUMN bill_per_token;
ALTER TABLE models DROP COLUMN bill_per_request;
ALTER TABLE models DROP COLUMN sell_per_request;