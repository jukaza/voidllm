---
title: "Tổng quan & mô hình kinh doanh"
description: "Mục tiêu chuyển đổi, quy trình khách hàng/admin, danh sách tính năng theo khu vực"
section: marketplace
order: 1
---
# Tổng quan & mô hình kinh doanh

## Mục tiêu

Chuyển Tavo từ **LLM gateway tự host cho nội bộ doanh nghiệp** thành **nền tảng bán lại API**:

- Nhiều nhà cung cấp (provider) đưa API custom của họ vào hệ thống.
- Admin đóng gói các endpoint đó thành **model công khai** với giá bán thống nhất.
- Khách hàng cuối đăng ký, nạp tiền vào ví, nhận API key, và gọi API qua endpoint OpenAI-compatible.
- Hệ thống trừ ví real-time theo token thực dùng; hết tiền thì key tự khoá.

## Quy trình khách hàng (customer flow)

1. Vào trang bán hàng công khai → xem bảng giá model → **đăng ký** (email + password, tự động).
2. Đăng nhập dashboard → hệ thống **tự cấp sẵn 1 API key** (`vl_uk_...`).
3. **Nạp tiền**: chọn số tiền → chuyển khoản (QR/số TK hiển thị) → nhập mã giao dịch → trạng thái "chờ duyệt".
4. Admin đối chiếu → duyệt → tiền cộng vào ví.
5. Khách gọi `/v1/chat/completions` bằng key → hệ thống **trừ ví theo giá bán** của model.
6. Khách xem: số dư, lịch sử giao dịch, usage theo model, tạo/thu hồi key phụ.
7. Ví về 0 → key tự khoá, request bị từ chối tới khi nạp thêm.

## Quy trình admin (operator flow)

1. Thêm **nhà cung cấp** — nhập API key gốc (mã hoá AES-256-GCM như hiện tại).
2. Tạo **model bán** — gộp nhiều kênh (deployment) từ nhiều nhà cung cấp dưới 1 tên công khai, chọn strategy (xoay vòng / weighted / priority), set RPM per kênh, fallback model.
3. Set **giá bán** (input/output/cached per 1M token) — tách biệt giá vốn per kênh.
4. **Duyệt nạp tiền**, quản lý khách hàng (khoá/mở, chỉnh ví thủ công).
5. Theo dõi doanh thu, lãi gộp theo model/nhà cung cấp, usage toàn hệ thống.

## Tính năng theo khu vực

### Trang công khai (Storefront)
- Landing page, bảng giá model, đăng ký / đăng nhập.

### Dashboard khách hàng (Customer Portal)
- Số dư ví + nút nạp tiền; lịch sử giao dịch (nạp/trừ).
- Quản lý API key (tạo/xoá/thu hồi); usage theo model/ngày.
- Playground thử API (giữ từ `PlaygroundPage` hiện có); hồ sơ cá nhân.

### Dashboard admin (Operator Console)
- Quản lý nhà cung cấp & model (giữ nguyên engine hiện tại), giá bán/markup.
- Duyệt nạp tiền; quản lý khách hàng; báo cáo doanh thu & usage; thu hồi key khẩn cấp.

### Lõi kỹ thuật (giữ nguyên phần lớn)
- Proxy OpenAI-compatible đa provider, load balancing + failover, circuit breaker.
- Rate limit theo key; usage tracking (token/cost/latency).
- **Mới**: trừ ví real-time, khoá key khi hết tiền, RPM per kênh upstream, giá cached tokens.

### Sẽ xoá khỏi bản hiện tại
- License Pro/Enterprise, SSO/OIDC, MCP Gateway + Code Mode, audit log doanh nghiệp, RBAC 4 cấp (rút còn `admin` / `customer`).
