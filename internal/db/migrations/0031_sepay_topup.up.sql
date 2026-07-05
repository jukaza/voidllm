-- Migration: 0031_sepay_topup.up.sql
-- Description: SePay auto top-up fields on topup_requests.

ALTER TABLE topup_requests ADD COLUMN trade_no TEXT NOT NULL DEFAULT '';
ALTER TABLE topup_requests ADD COLUMN pay_amount REAL;
ALTER TABLE topup_requests ADD COLUMN credit_amount REAL;
ALTER TABLE topup_requests ADD COLUMN bonus_amount REAL NOT NULL DEFAULT 0;
ALTER TABLE topup_requests ADD COLUMN bonus_detail TEXT NOT NULL DEFAULT '';
ALTER TABLE topup_requests ADD COLUMN expires_at TEXT;
ALTER TABLE topup_requests ADD COLUMN sepay_tx_id TEXT NOT NULL DEFAULT '';
ALTER TABLE topup_requests ADD COLUMN completed_at TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS idx_topup_requests_trade_no
    ON topup_requests (trade_no)
    WHERE trade_no != '';

CREATE INDEX IF NOT EXISTS idx_topup_requests_user_status
    ON topup_requests (user_id, status, created_at);

-- Normalize legacy manual-review statuses to SePay vocabulary.
UPDATE topup_requests
SET status = 'completed',
    completed_at = COALESCE(NULLIF(completed_at, ''), reviewed_at)
WHERE status = 'approved';

UPDATE topup_requests SET status = 'failed' WHERE status = 'rejected';