-- Migration: 0024_billing_min_per_request.up.sql
-- Description: Optional minimum USD charge per request when billing per token.
-- Also ensures models do not enable both token and per-request billing modes.

UPDATE models SET bill_per_request = 0 WHERE bill_per_token = 1 AND bill_per_request = 1;

ALTER TABLE models ADD COLUMN bill_min_per_request INTEGER NOT NULL DEFAULT 0 CHECK (bill_min_per_request IN (0, 1));
ALTER TABLE models ADD COLUMN sell_min_per_request REAL;