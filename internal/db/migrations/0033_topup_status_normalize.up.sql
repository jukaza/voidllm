-- Migration: 0033_topup_status_normalize.up.sql
-- Description: Remap legacy topup_requests statuses for databases that applied 0031 before this fix.

UPDATE topup_requests
SET status = 'completed',
    completed_at = COALESCE(NULLIF(completed_at, ''), reviewed_at)
WHERE status = 'approved';

UPDATE topup_requests SET status = 'failed' WHERE status = 'rejected';