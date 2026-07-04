-- Migration: 0024_billing_min_per_request.down.sql

ALTER TABLE models DROP COLUMN sell_min_per_request;
ALTER TABLE models DROP COLUMN bill_min_per_request;