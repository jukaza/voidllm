-- Migration: 0021_model_pricing_modes.up.sql
-- Description: Dual-mode customer billing — per-token and/or per-request.
--
-- bill_per_token / bill_per_request let admins enable one or both billing modes.
-- sell_per_request is the flat USD charge per API call when bill_per_request=1.
-- Existing models default to token billing only (bill_per_token=1).

ALTER TABLE models ADD COLUMN bill_per_token INTEGER NOT NULL DEFAULT 1 CHECK (bill_per_token IN (0, 1));
ALTER TABLE models ADD COLUMN bill_per_request INTEGER NOT NULL DEFAULT 0 CHECK (bill_per_request IN (0, 1));
ALTER TABLE models ADD COLUMN sell_per_request REAL;