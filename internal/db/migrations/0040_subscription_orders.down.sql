DROP INDEX IF EXISTS idx_topup_requests_order_kind;
ALTER TABLE topup_requests DROP COLUMN plan_id;
ALTER TABLE topup_requests DROP COLUMN order_kind;