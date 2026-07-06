-- Migration: 0040_subscription_orders.up.sql
-- Extend topup_requests for subscription purchase orders (shared SePay flow).

ALTER TABLE topup_requests ADD COLUMN order_kind TEXT NOT NULL DEFAULT 'wallet';
ALTER TABLE topup_requests ADD COLUMN plan_id TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_topup_requests_order_kind
    ON topup_requests (order_kind, status, created_at);