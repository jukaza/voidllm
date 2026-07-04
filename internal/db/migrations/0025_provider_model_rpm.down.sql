-- Migration: 0025_provider_model_rpm.down.sql

ALTER TABLE models DROP COLUMN rpm_limit;
ALTER TABLE providers DROP COLUMN rpm_limit;