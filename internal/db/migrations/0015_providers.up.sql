-- Migration: 0015_providers.up.sql
-- Description: Business-level provider (upstream partner) entity plus
-- per-deployment upstream limits and cost pricing for the API-resell model.
--
-- providers: a partner supplying upstream API capacity. Deployments reference
-- a provider so revenue/cost can be reported per partner.
--
-- model_deployments gains:
--   provider_id          - optional FK to providers (NULL = unattributed)
--   rpm_limit            - max requests/minute sent to this channel (0 = unlimited)
--   tpm_limit            - max tokens/minute sent to this channel (0 = unlimited)
--   daily_request_limit  - max requests/day sent to this channel (0 = unlimited)
--   cost_input_per_1m    - cost price USD per 1M input tokens (NULL = fall back to model price)
--   cost_output_per_1m   - cost price USD per 1M output tokens

CREATE TABLE providers (
    id            TEXT PRIMARY KEY,
    name          TEXT NOT NULL,
    contact_info  TEXT NOT NULL DEFAULT '',
    status        TEXT NOT NULL DEFAULT 'active',   -- 'active' | 'paused'
    notes         TEXT NOT NULL DEFAULT '',
    created_at    TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at    TEXT
);

ALTER TABLE model_deployments ADD COLUMN provider_id TEXT REFERENCES providers(id);
ALTER TABLE model_deployments ADD COLUMN rpm_limit INTEGER NOT NULL DEFAULT 0;
ALTER TABLE model_deployments ADD COLUMN tpm_limit INTEGER NOT NULL DEFAULT 0;
ALTER TABLE model_deployments ADD COLUMN daily_request_limit INTEGER NOT NULL DEFAULT 0;
ALTER TABLE model_deployments ADD COLUMN cost_input_per_1m REAL;
ALTER TABLE model_deployments ADD COLUMN cost_output_per_1m REAL;

CREATE INDEX idx_model_deployments_provider
    ON model_deployments (provider_id)
    WHERE provider_id IS NOT NULL;
