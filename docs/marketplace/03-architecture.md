---
title: "Kiến trúc & thiết kế"
description: "Tầng Provider → Kênh → Model bán, routing, định giá, luồng request"
section: marketplace
order: 3
---
# Kiến trúc & thiết kế

## Mô hình 3 tầng: Provider → Kênh → Model bán

```
Nhà cung cấp (providers — MỚI)
  │  thông tin đối tác, nhiều API key gốc, trạng thái hợp tác
  ▼
Kênh (model_deployments — CÓ SẴN, mở rộng)
  │  endpoint cụ thể + key, weight, priority
  │  + MỚI: rpm_limit, tpm_limit, giá vốn per kênh
  ▼
Model bán "E" (models — CÓ SẴN, mở rộng)
     tên công khai khách gọi
     strategy: round-robin | weighted | priority   (migration 0003)
     fallback_model_id → model khác                (migration 0011)
     + MỚI: is_public, giá bán input/output/cached per 1M token
```

Ví dụ: model công khai `E` gồm 3 kênh — OpenAI (weight=3, rẻ), Azure (priority=2), custom vLLM (fallback). Khách luôn trả **1 giá theo model E** bất kể kênh nào phục vụ; admin tối ưu chi phí phía sau bằng weight/priority.

## Luồng request (hot path)

```
Request khách (Bearer vl_uk_...)
  1. Auth middleware        → xác thực key, load KeyInfo          [có sẵn]
  2. checkLimits()          → rate limit + token budget           [có sẵn]
     + KIỂM TRA SỐ DƯ VÍ    → balance ≤ 0 → HTTP 402              [MỚI]
  3. resolveModel()         → alias → model E                     [có sẵn]
  4. Chọn kênh theo strategy, né kênh:
     - circuit breaker mở (chết)                                  [có sẵn]
     - CHẠM TRẦN RPM/TPM kênh                                     [MỚI]
  5. Gọi upstream; lỗi → kênh kế → hết kênh → fallback model      [có sẵn]
  6. Response (stream/non-stream) → extract usage tokens          [có sẵn]
  7. Async usage logger:
     - ghi usage_events (+ cached_tokens, revenue, deployment_id) [mở rộng]
     - TRỪ VÍ = tokens × giá bán model E                          [MỚI]
     - cộng counter RPM kênh, cộng chi phí vốn theo kênh          [MỚI]
```

## Thiết kế ví (WalletService)

Theo đúng pattern `TokenCounter` (`internal/ratelimit/token_counter.go`):

- **In-memory atomic balance** per customer, seed từ DB lúc khởi động.
- **Check trước request**: lock-free read; balance ≤ 0 → từ chối 402 `insufficient_balance`.
- **Trừ sau response**: async qua usage logger pipeline; ghi bản ghi `transactions` append-only (nguồn sự thật), balance in-memory là cache.
- **Nạp tiền**: admin duyệt `topup_requests` → ghi transaction `type=topup` → cộng balance.
- Chấp nhận âm nhẹ giữa 2 flush (chuẩn ngành với prepaid API); tuỳ chọn "soft floor" chặn khi balance < chi phí ước tính của request.

## Định giá

- **Giá bán** nằm ở model công khai: `sell_input_per_1m`, `sell_output_per_1m`, `sell_cached_input_per_1m` (USD hoặc VND — chọn 1 currency hệ thống).
- **Giá vốn** nằm ở kênh: `cost_input_per_1m`, `cost_output_per_1m` — mỗi kênh mua mỗi nơi giá khác.
- Lãi gộp = revenue (theo giá bán model) − cost (theo giá vốn kênh thực chạy) → báo cáo theo model / nhà cung cấp / kênh.
- Cached tokens: cần đọc `prompt_tokens_details.cached_tokens` từ response provider (hiện chưa có — bổ sung vào extractor).

## RPM/TPM per kênh (upstream limit)

- Counter atomic per `deployment_id` (tái dùng khung `rate_limiter.go`), window 1 phút + 1 ngày.
- Router coi "chạm trần" như "unhealthy tạm thời": bỏ qua kênh đó trong vòng chọn, giống circuit breaker skip.
- Multi-instance: chuyển counter sang Redis (khung `redis_checker.go` có sẵn).

## Phân quyền rút gọn

- `admin` (= `system_admin` cũ): toàn quyền vận hành.
- `customer`: chỉ thấy ví/key/usage của mình.
- Giữ nguyên hạ tầng org: mỗi customer = 1 org cá nhân ẩn (tự tạo lúc signup) — tránh phải viết lại toàn bộ scoping theo org_id đang xuyên suốt DB/API. UI không hiển thị khái niệm org cho khách.
