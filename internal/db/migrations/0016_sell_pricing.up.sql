-- Migration: 0016_sell_pricing.up.sql
-- Description: Public visibility flag and sell-side pricing on models.
--
-- is_public: 1 = the model appears on the public storefront price list and is
-- purchasable by customers. 0 (default) = internal only.
--
-- sell_* prices are what customers are charged per 1M tokens. They are
-- independent of the cost prices (input_price_per_1m / output_price_per_1m on
-- models, cost_input_per_1m / cost_output_per_1m on deployments) so that gross
-- margin can be computed per request.
--
-- sell_cached_input_per_1m applies to prompt tokens served from the upstream
-- provider's prompt cache (usage.prompt_tokens_details.cached_tokens).

ALTER TABLE models ADD COLUMN is_public INTEGER NOT NULL DEFAULT 0 CHECK (is_public IN (0, 1));
ALTER TABLE models ADD COLUMN sell_input_per_1m REAL;
ALTER TABLE models ADD COLUMN sell_output_per_1m REAL;
ALTER TABLE models ADD COLUMN sell_cached_input_per_1m REAL;
