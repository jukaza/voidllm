-- Migration: 0020_provider_slug_routes.up.sql
-- Description: Reseller routing upgrade.
--
-- providers gains connection defaults so a provider acts as a reusable
-- "source" for the setup wizard:
--   slug              - short unique handle ('openai', 'ds', 'or'), used in
--                       admin UI route labels like "ds / deepseek-chat"
--   protocol          - default wire protocol for deployments created from
--                       this provider: 'openai' | 'anthropic' | 'gemini' |
--                       'azure' | 'vertex' | 'vllm' | 'ollama' | 'custom'
--   logo              - logo URL or asset path shown in admin UI
--   base_url          - default upstream base URL
--   api_key_encrypted - default upstream API key (AES-256-GCM, AAD 'provider:<id>')
--
-- models gains:
--   logo - customer-facing logo URL. Empty = FE falls back to the logo of the
--          provider on the first active route.
--
-- model_deployments gains:
--   upstream_model            - model name sent upstream. Empty = use the
--                               canonical model name (existing behaviour).
--                               Non-empty enables cross-model routes, e.g.
--                               product 'coding-pro' -> 'deepseek-chat'.
--   cost_cached_input_per_1m  - cost price USD/1M for cache-hit prompt tokens
--   cost_cache_write_per_1m   - cost price USD/1M for cache-write tokens
--                               (Anthropic prompt caching bills writes above
--                               the base input rate)

ALTER TABLE providers ADD COLUMN slug TEXT;
ALTER TABLE providers ADD COLUMN protocol TEXT NOT NULL DEFAULT 'openai';
ALTER TABLE providers ADD COLUMN logo TEXT NOT NULL DEFAULT '';
ALTER TABLE providers ADD COLUMN base_url TEXT NOT NULL DEFAULT '';
ALTER TABLE providers ADD COLUMN api_key_encrypted TEXT;

-- Unique among live rows only, so a deleted provider frees its slug.
CREATE UNIQUE INDEX idx_providers_slug
    ON providers (slug)
    WHERE slug IS NOT NULL AND deleted_at IS NULL;

ALTER TABLE models ADD COLUMN logo TEXT NOT NULL DEFAULT '';

ALTER TABLE model_deployments ADD COLUMN upstream_model TEXT NOT NULL DEFAULT '';
ALTER TABLE model_deployments ADD COLUMN cost_cached_input_per_1m REAL;
ALTER TABLE model_deployments ADD COLUMN cost_cache_write_per_1m REAL;
