-- Migration: 0041_api_key_encrypted.up.sql
-- Description: Store AES-encrypted API keys so owners can reveal/copy from any device.

ALTER TABLE api_keys ADD COLUMN key_encrypted TEXT;