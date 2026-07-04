-- Migration: 0028_wallets_vnd_default.up.sql
-- Description: Switch system currency default from USD to VND.
-- Instance is greenfield; this normalizes any seeded wallets and documents
-- the single-currency policy. Application code also defaults to VND.

UPDATE wallets SET currency = 'VND' WHERE currency = 'USD';