-- Migration: 0037_model_catalog_capabilities.up.sql
-- Description: Explicit catalog capability flags for tool calling and vision.

ALTER TABLE models ADD COLUMN supports_tools INTEGER NOT NULL DEFAULT 0 CHECK (supports_tools IN (0, 1));
ALTER TABLE models ADD COLUMN supports_vision INTEGER NOT NULL DEFAULT 0 CHECK (supports_vision IN (0, 1));