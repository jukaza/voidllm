-- Migration: 0028_wallets_vnd_default.down.sql

UPDATE wallets SET currency = 'USD' WHERE currency = 'VND';