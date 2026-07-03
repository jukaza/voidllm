-- Migration: 0017_wallets.up.sql
-- Description: Prepaid wallet, top-up requests, and append-only transaction ledger.
--
-- wallets: one per user. balance is a cached aggregate; the source of truth is
-- SUM(transactions.amount) for the user. Currency is a display concern only —
-- all arithmetic happens in one system currency.
--
-- topup_requests: manual top-up flow. A customer submits amount + payment_ref
-- (bank transfer reference); an admin approves or rejects. Approval writes a
-- 'topup' transaction and credits the wallet.
--
-- transactions: append-only. Never UPDATE or DELETE. amount is positive for
-- credits (topup/adjustment/refund) and negative for debits (usage).
-- balance_after records the wallet balance immediately after this transaction
-- for reconciliation. No FK on user_id (mirrors usage_events convention: rows
-- survive user deletion for accounting).

CREATE TABLE wallets (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL UNIQUE REFERENCES users(id),
    balance     REAL NOT NULL DEFAULT 0,
    currency    TEXT NOT NULL DEFAULT 'USD',
    created_at  TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE topup_requests (
    id           TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL REFERENCES users(id),
    amount       REAL NOT NULL CHECK (amount > 0),
    payment_ref  TEXT NOT NULL DEFAULT '',
    status       TEXT NOT NULL DEFAULT 'pending',  -- 'pending' | 'approved' | 'rejected'
    reviewed_by  TEXT REFERENCES users(id),
    reviewed_at  TEXT,
    note         TEXT NOT NULL DEFAULT '',
    created_at   TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_topup_requests_status ON topup_requests (status, created_at);
CREATE INDEX idx_topup_requests_user ON topup_requests (user_id, created_at);

CREATE TABLE transactions (
    id            TEXT PRIMARY KEY,
    user_id       TEXT NOT NULL,
    type          TEXT NOT NULL,               -- 'topup' | 'usage' | 'adjustment' | 'refund'
    amount        REAL NOT NULL,
    balance_after REAL NOT NULL,
    ref_id        TEXT NOT NULL DEFAULT '',    -- topup_request id or usage request_id
    description   TEXT NOT NULL DEFAULT '',
    created_at    TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_transactions_user ON transactions (user_id, created_at);
