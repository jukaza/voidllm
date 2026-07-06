---
title: "Chuyển đổi sang Web bán API (Marketplace)"
description: "Tài liệu dự án chuyển Tavo thành nền tảng bán lại API theo mô hình ví trả trước"
section: marketplace
order: 0
---
# Chuyển đổi Tavo → Web bán API

Bộ tài liệu này mô tả kế hoạch chuyển Tavo (LLM gateway tự host cho doanh nghiệp nội bộ) thành **nền tảng bán lại API**: nhiều nhà cung cấp đưa API custom vào hệ thống, admin đóng gói thành các model công khai và bán cho khách hàng cuối theo mô hình **ví trả trước**.

## Danh mục tài liệu

- [Tổng quan & mô hình kinh doanh](01-overview.md) — mục tiêu, quy trình khách hàng/admin, danh sách tính năng
- [Phân tích khả thi codebase](02-feasibility.md) — đánh giá từng điểm tích hợp/gỡ bỏ trên code hiện tại
- [Kiến trúc & thiết kế](03-architecture.md) — tầng Provider → Kênh → Model bán, routing, định giá
- [Thiết kế database](04-database.md) — bảng mới (ví, nạp tiền, giao dịch, giá bán) và thay đổi bảng cũ
- [Roadmap triển khai](05-roadmap.md) — 4 phase: dọn dẹp → data model → backend → frontend

## Các quyết định đã chốt

| Chủ đề | Quyết định |
|---|---|
| Mô hình tính phí | Ví trả trước — nạp tiền, trừ theo usage thực tế |
| Nạp tiền | Thủ công (admin duyệt) trước; cổng thanh toán tự động làm sau |
| Tính năng doanh nghiệp | Xoá: license Pro/Enterprise, SSO/OIDC, MCP Gateway, audit log |
| Phân quyền | Rút gọn còn 2 vai trò: `admin` và `customer` |
| Giá bán | Khách trả 1 giá thống nhất theo model công khai, bất kể kênh upstream nào phục vụ |
