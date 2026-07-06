---
title: "Phân tích khả thi codebase"
description: "Đánh giá từng điểm tích hợp/gỡ bỏ trên code hiện tại, kèm file:line cụ thể"
section: marketplace
order: 2
---
# Phân tích khả thi codebase

**Kết luận chung: KHẢ THI, độ khó trung bình.** Kiến trúc hiện tại có sẵn ~70% phần lõi kỹ thuật (proxy đa provider, load balancing, fallback, usage tracking, per-key limit). Phần thiếu là tầng thương mại (ví, giá bán, signup công khai) — đều có điểm móc nối rõ ràng, không cần đập kiến trúc.

## 1. Điểm móc ví tiền vào proxy hot path — DỄ ✅

Vòng đời request trong `internal/proxy/handler.go`:

- **Auth**: middleware `internal/auth` chạy trước `Handle()`, key info lấy qua `auth.KeyInfoFromCtx(c)` (handler.go:189).
- **Kiểm tra trước khi forward**: `checkLimits()` (handler.go:753-813) đã kiểm rate limit + token budget theo key/team/org. → **Chỗ kiểm tra số dư ví đặt ngay tại đây**, cùng pattern.
- **Ghi nhận sau response**: usage logger chạy **async** (buffered channel + batch flush, `internal/usage/logger.go:26-89`) — không chặn hot path. Streaming đã có `streamUsageExtractor` (handler.go:123-152) đọc usage object từ chunk SSE cuối.

**Điểm vàng**: `TokenCounter` (`internal/ratelimit/token_counter.go`) là counter in-memory atomic, seed từ DB lúc khởi động, cộng ngay khi có usage event, kiểm tra trước mỗi request. **Ví tiền dùng đúng pattern này** — thay "token budget" bằng "money balance": kiểm balance trước request (in-memory, lock-free), trừ tiền async sau response, ghi transaction vào DB qua batch flush. Rủi ro âm ví nhẹ giữa 2 lần flush là chấp nhận được với mô hình trả trước (giống mọi nhà cung cấp API thực tế).

## 2. Usage events — DỄ, cần bổ sung cột ✅

`internal/usage/event.go` đã có: KeyID, OrgID, ModelName, RequestedModelName (phân biệt fallback), Prompt/Completion/TotalTokens, `CostEstimate` (USD), DurationMS, TTFT, StatusCode, RequestID.

Thiếu: **cached tokens** (grep `cached_tokens`/`prompt_tokens_details` toàn repo = 0 kết quả) và **giá bán/doanh thu**. Cần thêm cột `cached_tokens`, `revenue`, `deployment_id` (kênh thực chạy — để tính lãi theo kênh).

## 3. Gỡ license — DỄ ✅

Coupling thấp: 29 call sites ngoài package license, tập trung ở `internal/app/app.go` (15), `internal/api/admin/license.go` (8 — xoá cả file), còn lại rải rác 1-2 chỗ ở models.go, routes.go, handler.go, config.go, audit/middleware.go. UI: ~10 file (LicensePage, useLicense hook, Sidebar, các trang gated). Gỡ theo kiểu "mở khoá hết rồi xoá gate" — không đụng logic nghiệp vụ.

## 4. Gỡ MCP — TRUNG BÌNH ⚠️

`internal/mcp` tự thân độc lập, nhưng dây vào `internal/app/app.go` khá nhiều (caches, MCPLogger, MCPHealthChecker — app.go:75-87, 247-254, 491-499+) và có các file `internal/proxy/mcp_*_cache.go`. DB tables MCP (migration 0004-0008) **giữ nguyên migration, không rollback** — chỉ xoá code đọc/ghi. UI xoá 4 trang MCP + route. Ước lượng: 1-2 ngày công, chủ yếu là gỡ dây trong app.go (file 1539 dòng, DI thủ công).

## 5. Signup công khai — DỄ ✅

Đã có sẵn pattern: `/api/v1/invites/redeem` là **endpoint public tạo user** (routes.go:22) — signup chỉ là biến thể bỏ bước invite token. Session = token 24h (revoke session cũ khi login lại, `internal/api/admin/auth.go`). Thêm `POST /api/v1/auth/register`: tạo user + org cá nhân + ví + auto-cấp key `vl_uk_` (tái dùng `pkg/keygen` + `CreateAPIKey` sẵn có). Cần thêm: throttle đăng ký (đã có `login_throttle.go` làm mẫu), xác thực email (có thể để phase sau).

## 6. API key & per-key limit — CÓ SẴN ✅

Key đã có per-key `DailyTokenLimit`/`MonthlyTokenLimit` enforce trong hot path (handler.go:766-767). Cơ chế khoá key (soft-delete/revoke, rotate) đầy đủ. "Khoá key khi ví = 0" không cần đụng bảng key — chỉ là check balance trả 402/429 trong `checkLimits`.

## 7. Rate limit upstream (RPM per kênh) — CẦN LÀM MỚI ⚠️

`internal/ratelimit` hiện chỉ **client-side** (theo key/team/org), backing in-memory hoặc Redis (`redis_checker.go` — hỗ trợ multi-instance sẵn). **Chưa có** RPM/TPM per deployment phía upstream. Cần thêm counter per deployment_id (tái dùng `rate_limiter.go`) và dạy router né kênh chạm trần — sửa logic chọn deployment trong `internal/proxy/registry.go` + `internal/router`. Đây là phần việc mới lớn nhất về kỹ thuật, nhưng nền tảng (atomic counter, circuit breaker skip-unhealthy) đã có mẫu.

## 8. Database — DỄ ✅

Hỗ trợ SQLite + PostgreSQL (`internal/db/dialect.go`, docs/deployment/database.md). Hệ migration đánh số tuần tự (hiện tới 0014) — thêm bảng mới chỉ là viết migration 0015+. Convention rõ (UUIDv7, soft-delete, ISO timestamp).

## 9. UI storefront — DỄ ✅

React Router đã tách route public (`/login`, `/invite/:token`) và protected (`RequireAuth` wrapper — App.tsx:79-82). Thêm section storefront không cần auth là thêm route ngang hàng. i18n vừa được thêm (commit b6b9449) — bảng giá/landing đa ngôn ngữ làm được ngay.

## 10. Wiring service mới — TRUNG BÌNH ⚠️

`internal/app/app.go` là DI thủ công 1539 dòng trong `New()`. WalletService/PricingService cắm vào đây theo pattern hiện có (như UsageLogger, TokenCounter). Không khó nhưng file này là điểm nghẽn mọi thay đổi — nên dọn MCP/license trước để nhẹ bớt rồi mới thêm.

## Bảng tổng hợp

| Hạng mục | Độ khó | Ghi chú |
|---|---|---|
| Móc ví vào hot path | Dễ | Copy pattern TokenCounter |
| Bổ sung usage events | Dễ | Thêm cột migration |
| Gỡ license | Dễ | 29 call sites, tập trung |
| Gỡ MCP | Trung bình | Dây nhiều trong app.go |
| Signup công khai | Dễ | Pattern invites/redeem sẵn |
| Auto-cấp key | Dễ | Tái dùng keygen + CreateAPIKey |
| RPM per kênh upstream | Trung bình | Phần mới lớn nhất, có mẫu |
| Migration DB mới | Dễ | Hệ migration chuẩn |
| UI storefront | Dễ | Route public đã có pattern |
| Rút gọn RBAC | Trung bình | Nhiều endpoint gate theo 4 role |

## Rủi ro / lưu ý (landmines)

1. **Đừng xoá migration cũ** — chỉ thêm migration mới; DB đang chạy (`tavo.db`) sẽ hỏng nếu đổi lịch sử migration.
2. **Race âm ví khi trừ tiền async** — chấp nhận với trả trước; có thể siết bằng "soft floor" (chặn request khi balance < ước tính chi phí request).
3. **Streaming không trả usage** với vài provider custom nếu client không set `stream_options.include_usage` — cần fallback ước tính token hoặc ép include_usage phía proxy.
4. **RBAC rút gọn** đụng tất cả `RequireRole` trong routes.go — nên map `org_admin/team_admin/member` → `customer` bằng migration thay vì sửa từng chỗ ngay, xoá dần sau.
5. **Multi-instance**: TokenCounter/ví in-memory chỉ đúng single-instance; scale ngang cần chuyển sang RedisChecker pattern (đã có sẵn khung).
