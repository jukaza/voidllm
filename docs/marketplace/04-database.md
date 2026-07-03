---
title: "Thiết kế database"
description: "Bảng mới (providers, wallets, topup, transactions) và cột bổ sung cho bảng cũ"
section: marketplace
order: 4
---
# Thiết kế database

Tuân theo convention hiện có (`0001_initial_schema.up.sql`): UUIDv7 TEXT PK, timestamp ISO 8601 UTC, soft-delete qua `deleted_at`, 0 = unlimited. **Chỉ thêm migration mới (0015+), không sửa migration cũ.**

## Migration 0015 — providers (nhà cung cấp là đối tác kinh doanh)

```sql
CREATE TABLE providers (
    id            TEXT PRIMARY KEY,
    name          TEXT NOT NULL,                -- tên đối tác
    contact_info  TEXT NOT NULL DEFAULT '',     -- email/telegram/ghi chú
    status        TEXT NOT NULL DEFAULT 'active', -- 'active' | 'paused'
    notes         TEXT NOT NULL DEFAULT '',
    created_at    TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at    TEXT
);

-- Kênh tham chiếu đối tác + giới hạn & giá vốn per kênh
ALTER TABLE model_deployments ADD COLUMN provider_id TEXT REFERENCES providers(id);
ALTER TABLE model_deployments ADD COLUMN rpm_limit INTEGER NOT NULL DEFAULT 0;   -- 0 = unlimited
ALTER TABLE model_deployments ADD COLUMN tpm_limit INTEGER NOT NULL DEFAULT 0;
ALTER TABLE model_deployments ADD COLUMN daily_request_limit INTEGER NOT NULL DEFAULT 0;
ALTER TABLE model_deployments ADD COLUMN cost_input_per_1m REAL;                 -- giá vốn
ALTER TABLE model_deployments ADD COLUMN cost_output_per_1m REAL;
```

## Migration 0016 — giá bán & publish trên model

```sql
ALTER TABLE models ADD COLUMN is_public INTEGER NOT NULL DEFAULT 0;   -- hiện trên bảng giá công khai
ALTER TABLE models ADD COLUMN sell_input_per_1m REAL;                 -- giá bán (trừ ví khách)
ALTER TABLE models ADD COLUMN sell_output_per_1m REAL;
ALTER TABLE models ADD COLUMN sell_cached_input_per_1m REAL;
-- input_price_per_1m/output_price_per_1m cũ giữ nguyên vai trò giá vốn mặc định
-- khi kênh không set giá vốn riêng
```

## Migration 0017 — ví & giao dịch

```sql
CREATE TABLE wallets (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL UNIQUE REFERENCES users(id),
    balance     REAL NOT NULL DEFAULT 0,        -- cache; nguồn sự thật là SUM(transactions)
    currency    TEXT NOT NULL DEFAULT 'VND',
    created_at  TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE topup_requests (
    id            TEXT PRIMARY KEY,
    user_id       TEXT NOT NULL REFERENCES users(id),
    amount        REAL NOT NULL,
    payment_ref   TEXT NOT NULL DEFAULT '',     -- mã giao dịch chuyển khoản khách nhập
    status        TEXT NOT NULL DEFAULT 'pending', -- 'pending'|'approved'|'rejected'
    reviewed_by   TEXT REFERENCES users(id),    -- admin duyệt
    reviewed_at   TEXT,
    note          TEXT NOT NULL DEFAULT '',
    created_at    TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Append-only, không FK tới usage (giống usage_events), không bao giờ UPDATE/DELETE
CREATE TABLE transactions (
    id           TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL,
    type         TEXT NOT NULL,                 -- 'topup'|'usage'|'adjustment'|'refund'
    amount       REAL NOT NULL,                 -- dương = cộng ví, âm = trừ ví
    balance_after REAL NOT NULL,                -- số dư sau giao dịch (đối soát)
    ref_id       TEXT NOT NULL DEFAULT '',      -- topup_request id / usage request_id
    description  TEXT NOT NULL DEFAULT '',
    created_at   TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_transactions_user ON transactions (user_id, created_at);
```

## Migration 0018 — mở rộng usage_events

```sql
ALTER TABLE usage_events ADD COLUMN cached_tokens INTEGER NOT NULL DEFAULT 0;
ALTER TABLE usage_events ADD COLUMN revenue REAL;          -- tiền trừ ví khách (giá bán)
ALTER TABLE usage_events ADD COLUMN deployment_id TEXT NOT NULL DEFAULT ''; -- kênh thực chạy (tính lãi theo kênh)
-- cost_estimate cũ giữ vai trò giá vốn
```

## Migration 0019 — role customer

```sql
-- Không đổi schema; data migration: map các role cũ về mô hình 2 vai trò
-- system_admin → giữ nguyên (admin)
-- org_admin / team_admin / member → 'member' trong org cá nhân (hiển thị là customer)
```

## Bảng/tính năng KHÔNG đụng tới

- Các bảng MCP (0004-0008), SSO, audit: **giữ nguyên trong DB**, chỉ xoá code sử dụng — an toàn rollback, không phá migration chain.
- `api_keys`, `models`, `model_aliases`, `model_deployments` (phần lõi): giữ nguyên cấu trúc, chỉ ADD COLUMN.
