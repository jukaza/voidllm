---
title: "Roadmap triển khai"
description: "4 phase: dọn dẹp → data model → backend → frontend, kèm điểm chạm code cụ thể"
section: marketplace
order: 5
---
# Roadmap triển khai

## Phase 0 — Dọn dẹp (xoá trước khi thêm)

Mục tiêu: nhẹ `internal/app/app.go` (1539 dòng) trước khi cắm service mới.

1. **Gỡ license** (~29 call sites): xoá `internal/license/`, `internal/api/admin/license.go`, gate trong app.go/models.go/routes.go/audit; UI xoá `LicensePage`, `useLicense`, gate trong `Sidebar.tsx`, `App.tsx` (route `/license`), i18n keys.
2. **Gỡ SSO/OIDC**: `internal/sso/`, route `/auth/oidc/*` (routes.go:26-27), `oidc.go`, `org_sso.go`; UI: `SSOConfigPage`, `OrgDetailSSOTab`, `CallbackPage`.
3. **Gỡ MCP** (phần nặng nhất): `internal/mcp/`, `internal/proxy/mcp_*_cache.go`, `mcp_*` trong api/admin và db (chỉ code, **giữ migration + bảng DB**), gỡ dây app.go:75-87, 247+, 491+; UI: 4 trang MCP + routes + `MCPUsagePage`.
4. **Gỡ audit doanh nghiệp**: giữ code ghi log đơn giản, xoá `AuditLogPage` + endpoint nếu không dùng.
5. Chạy `go build ./... && go test ./...` sau mỗi bước — không gộp 4 bước vào 1 commit.

**Definition of done**: build sạch, test pass, UI không còn menu License/SSO/MCP/Audit.

## Phase 1 — Data model (migrations 0015-0019)

Theo [04-database.md](04-database.md): providers, giá bán/is_public trên models, wallets + topup_requests + transactions, mở rộng usage_events, data-migration role. Kèm querier Go trong `internal/db/` (providers.go, wallets.go, transactions.go, topups.go) theo pattern file hiện có + test.

## Phase 2 — Backend

1. **WalletService** (`internal/wallet/`): in-memory balance theo pattern `TokenCounter`, seed từ DB, API Check/Deduct/Credit. Cắm vào `app.New()`.
2. **Hot path**: thêm check balance trong `checkLimits()` (handler.go:753) → 402 `insufficient_balance`; trừ ví trong usage logger pipeline (revenue = tokens × giá bán model).
3. **Cached tokens**: mở rộng `streamUsageExtractor` + non-stream parse đọc `prompt_tokens_details.cached_tokens`.
4. **RPM per kênh**: counter per deployment_id, router né kênh chạm trần (registry.go + router).
5. **Signup công khai**: `POST /api/v1/auth/register` (mẫu từ `invites/redeem`) — tạo user + org cá nhân + ví + auto-cấp key; throttle theo mẫu `login_throttle.go`.
6. **API topup**: customer tạo `topup_requests`; admin list/approve/reject → ghi transaction + cộng ví.
7. **Admin API**: CRUD providers, set giá bán, quản lý khách (khoá/mở, adjust ví), báo cáo doanh thu/lãi.

## Phase 3 — Frontend

1. **Storefront public** (route không auth, ngang `/login` trong App.tsx): landing, bảng giá (`GET /api/v1/public/models` mới — chỉ model `is_public`), trang đăng ký.
2. **Customer portal**: ví + nạp tiền, lịch sử giao dịch, keys (tái dùng `KeysPage`), usage của tôi (tái dùng `usage/me`), Playground (giữ nguyên).
3. **Admin console**: duyệt nạp tiền, quản lý providers + giá, quản lý khách, báo cáo doanh thu (nâng cấp từ `CostReportsPage`).
4. Sidebar 2 chế độ theo role: customer thấy Ví/Keys/Usage/Playground; admin thấy toàn bộ.
5. i18n cho toàn bộ trang mới (vi/en — hạ tầng đã có từ commit b6b9449).

## Phase 4 (sau MVP)

- Cổng thanh toán tự động (VNPay/Momo/Stripe) thay duyệt tay.
- Xác thực email khi signup; cảnh báo số dư thấp (email/webhook).
- Redis cho ví + RPM kênh khi scale ngang.
- Affiliate/CTV; hoá đơn xuất PDF.

## Thứ tự & ước lượng

| Phase | Nội dung | Ước lượng |
|---|---|---|
| 0 | Dọn license/SSO/MCP/audit | 2-3 ngày |
| 1 | Migrations + queriers | 1-2 ngày |
| 2 | Wallet, hot path, RPM kênh, signup, topup API | 4-6 ngày |
| 3 | Storefront + portal + admin UI | 4-6 ngày |

Rủi ro lớn nhất nằm ở Phase 2 mục 4 (RPM per kênh — logic router) và độ chính xác trừ ví với streaming (xem lưu ý trong [02-feasibility.md](02-feasibility.md)).
