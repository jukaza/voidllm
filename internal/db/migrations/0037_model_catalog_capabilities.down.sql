-- Migration: 0037_model_catalog_capabilities.down.sql

ALTER TABLE models DROP COLUMN supports_vision;
ALTER TABLE models DROP COLUMN supports_tools;