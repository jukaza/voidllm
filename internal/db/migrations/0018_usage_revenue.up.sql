-- Migration: 0018_usage_revenue.up.sql
-- Description: Marketplace accounting columns on usage_events.
--
-- cached_tokens: prompt tokens served from the provider's prompt cache
-- (usage.prompt_tokens_details.cached_tokens in OpenAI-format responses).
-- Charged at the model's sell_cached_input_per_1m rate when configured.
--
-- revenue: the amount debited from the customer's wallet for this request,
-- computed from the model's sell prices. NULL when the model has no sell
-- pricing configured (request was free). cost_estimate (existing column)
-- retains its role as the upstream cost estimate.
--
-- deployment_id: which channel actually served the request, so gross margin
-- can be reported per channel/provider. Empty string for single-deployment
-- (legacy) models.

ALTER TABLE usage_events ADD COLUMN cached_tokens INTEGER NOT NULL DEFAULT 0;
ALTER TABLE usage_events ADD COLUMN revenue REAL;
ALTER TABLE usage_events ADD COLUMN deployment_id TEXT NOT NULL DEFAULT '';
