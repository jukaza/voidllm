-- Migration: 0025_provider_model_rpm.up.sql
-- Per-provider and per-product-model request-per-minute caps (0 = unlimited).

ALTER TABLE providers ADD COLUMN rpm_limit INTEGER NOT NULL DEFAULT 0;
ALTER TABLE models ADD COLUMN rpm_limit INTEGER NOT NULL DEFAULT 0;