-- Migration: 0032_finance_indexes.up.sql
-- Description: Indexes for admin finance aggregate and list queries.

CREATE INDEX IF NOT EXISTS idx_transactions_type_created
    ON transactions (type, created_at);

CREATE INDEX IF NOT EXISTS idx_topup_requests_status_completed
    ON topup_requests (status, completed_at);