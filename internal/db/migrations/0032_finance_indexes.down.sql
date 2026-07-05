-- Migration: 0032_finance_indexes.down.sql

DROP INDEX IF EXISTS idx_topup_requests_status_completed;
DROP INDEX IF EXISTS idx_transactions_type_created;