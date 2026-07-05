---
title: "FE: Cài đặt hệ thống & Tài khoản"
description: "Giới thiệu redesign UI /settings và /account — trạng thái Live/Preview, cấu trúc file, việc cần làm backend"
section: fe
order: 0
---
# Frontend: `/settings` (admin) và `/account` (user)

Tài liệu này ghi lại redesign UI đã triển khai (tháng 7/2026) để phiên code sau biết **đã xong gì**, **pattern nào**, và **backend còn thiếu gì**.

> **Lưu ý:** VoidLLM **không có** groups/subscriptions như sub2api. Không copy toàn bộ admin settings từ sub2api — chỉ lấy UX pattern phù hợp marketplace (ví trả trước, member + system_admin).

---

## Tổng quan

| Trang | Route | Đối tượng | Tab |
|-------|-------|-----------|-----|
| Cài đặt hệ thống | `/settings` | `system_admin` | 6 tab |
| Tài khoản | `/account` | mọi user đăng nhập | 4 tab |

Cả hai trang dùng chung:

- `TabbedPageLayout` — sidebar tab (desktop) + `TabSwitcher` (mobile)
- Deep link: `?tab=<key>` (vd. `/settings?tab=payment`, `/account?tab=security`)
- Badge **Live** / **Preview** trên từng section
- i18n `en` + `vi` qua `ui/src/lib/i18n.tsx`

---

## Pattern Live vs Preview

| Badge | Ý nghĩa | Lưu ở đâu |
|-------|---------|-----------|
| **Live** | Gọi API backend thật, có hiệu lực sau Save | Server / React Query |
| **Preview** | UI + localStorage draft, chưa có API | `useSettingsDraft` hoặc `useAccountDraft` |

**Admin draft:** `localStorage` key `voidllm.admin_settings.draft.v1`  
**Account draft:** `localStorage` key `voidllm.account.draft.v1`

Khi wire backend cho trường Preview: thêm API → bỏ draft field → đổi badge sang Live → xóa hint footer preview.

---

## `/settings` — Admin System Settings

**Orchestrator:** `ui/src/pages/settings/SystemSettingsPage.tsx`

### 6 tab

| Tab | File | Live API | Preview (draft) |
|-----|------|----------|-----------------|
| General | `GeneralSettingsTab.tsx` | `GET/PUT /admin/settings/site` + `POST/DELETE /admin/settings/site/logo` (upload logo, tên, phụ đề, footer, homepage, Zalo/Telegram, docs, API base) | — (tab Live hoàn toàn) |
| Security | `SecuritySettingsTab.tsx` | `register_enabled`, Turnstile, OAuth, policy 2FA/session/password | — |
| Features | `FeaturesSettingsTab.tsx` | — | `enforce_balance`, `initial_wallet_balance`, catalog/playground toggles |
| Payment | `PaymentSettingsTab.tsx` | `GET/PUT /admin/settings/payment` | — |
| Legal & Notice | `LegalNoticeSettingsTab.tsx` | Site legal + notice list API | — |
| Backup | `BackupSettingsTab.tsx` | Export gọi site + payment API | Import preview, schedule S3 (disabled) |

**Đã bỏ:** tab **Email** (SMTP) — xem mục [Đã gỡ tính năng email](#đã-gỡ-tính-năng-email) bên dưới.

### Shared components (`ui/src/pages/settings/components/`)

- `SettingsLayout.tsx` — bọc `TabbedPageLayout`
- `SettingsSectionCard.tsx` — card có title + badge
- `PreviewBadge.tsx` / `LiveBadge`
- `SettingsTabFooter.tsx` — sticky save + hint live/preview
- `SettingsStatusStrip.tsx` — chip Register + SePay (không còn SMTP)
- `SetupGuideBlock.tsx` — hướng dẫn từng bước (Payment tab)

### Config tab

- `settingsTabs.ts` — keys + `isSettingsTabKey`
- `settingsDraftTypes.ts` — shape draft admin
- `useSettingsDraft.ts` — đọc/ghi localStorage

---

## `/account` — User Account

**Orchestrator:** `ui/src/pages/account/AccountSettingsPage.tsx`  
**Route:** `App.tsx` → `/account` (redirect `/profile` → `/account`)

**Layout:** `PageHeader` → `AccountHero` → `AccountStatusStrip` → `AccountLayout` (4 tab)  
**Max width:** `max-w-6xl`

### 4 tab

| Tab | File | Live | Preview |
|-----|------|------|---------|
| Profile | `ProfileTab.tsx` | Email read-only (`GET /me`) | Display name (`PATCH /me` chưa có) |
| Security | `SecurityTab.tsx` | Mật khẩu, 2FA TOTP, phiên đăng nhập (API Live) | — |
| Connections | `ConnectionsTab.tsx` | OAuth bind Google/GitHub | — |
| Preferences | `PreferencesTab.tsx` | Ngôn ngữ `setLanguage` | `record_ip` |

### Hero & stats (Live)

`AccountHero.tsx` gọi:

- `GET /me`
- `GET /me/wallet`
- `GET /usage/me` (30 ngày)
- `GET /keys` (đếm key)

Link nhanh: Wallet, Analytics, Keys.

### Dialogs (`ui/src/pages/account/components/`)

- `ChangePasswordDialog.tsx` — **Live** (`POST /me/password`)
- `SetPasswordDialog.tsx` — **Live** (`POST /me/password/set`, OAuth-only)
- `TwoFactorDialog.tsx` — **Live** (`POST /me/2fa/*`, khi admin bật `allow_user_enable`)
- `SessionsDialog.tsx` — **Live** (`GET/DELETE /me/sessions`)
- `SecurityActionTiles.tsx` — tiles mở dialog

### Đã bỏ so với bản demo cũ

- IP whitelist
- Sessions trùng lặp
- Footer text “frontend demo”
- Thông báo email (high usage / new login)

---

## Đã gỡ tính năng email

Quyết định: **không triển khai SMTP / gửi mail / thông báo email** trong giai đoạn FE-first.

### Backend đã xóa

- Package `internal/email/`
- `internal/api/admin/email_settings.go`
- Routes `GET/PUT /api/v1/admin/settings/email`

### Frontend đã xóa

- Tab Email trong `/settings`
- Hook `useEmailSettings.ts`
- Toggle xác minh email / reset mật khẩu qua email (Security)
- Toggle thông báo email trong `/account` Preferences
- Export backup không còn gồm block `email`

### Vẫn giữ (không phải “tính năng email”)

- **Email đăng nhập** — login, register, profile read-only, cột user trong admin/finance
- **`support_zalo` / `support_telegram`** — kênh hỗ trợ cộng đồng trên landing footer (không email)

---

## Backend còn thiếu — ưu tiên phiên sau

### Account (`/account`)

| Tính năng | API đề xuất | Ghi chú |
|-----------|-------------|---------|
| Sửa tên hiển thị | `PATCH /me` `{ display_name }` | Hiện chỉ draft localStorage |
| ~~2FA~~ | `POST /me/2fa/setup`, `POST /me/2fa/verify`, `DELETE /me/2fa` | **Đã Live** |
| ~~Sessions~~ | `GET /me/sessions`, `DELETE /me/sessions/:id` | **Đã Live** |
| Login 2FA | `POST /auth/login` → `requires_2fa`, `POST /auth/login/2fa` | **Đã Live** |
| Đặt mật khẩu OAuth | `POST /me/password/set` | **Đã Live** |
| OAuth bind | Flow redirect + callback per provider | Admin bật OAuth ở Security trước |
| `record_ip` preference | `PATCH /me/preferences` hoặc user settings table | Preview |

### Admin Settings (draft → Live)

| Trường draft | Gợi ý |
|--------------|-------|
| ~~`site_subtitle`, contact fields~~ | **Đã Live** — `site_subtitle`, `support_zalo`, `support_telegram`, `doc_url` |
| Turnstile / OAuth | Endpoint security settings riêng |
| `enforce_balance`, `initial_wallet_balance` | Config server / wallet bootstrap |
| Backup import | `POST /admin/settings/import` |
| Backup schedule | Service cron + S3 — phase sau |

### Không làm lại (đã chốt bỏ)

- SMTP, email verification signup, password reset qua email
- Email notification preferences

---

## i18n

Tất cả text UI mới dùng prefix:

- `settings.*` — admin settings (~100+ keys)
- `account.*` — user account (~90 keys)

Thêm key: cập nhật **cả** `en` và `vi` trong `ui/src/lib/i18n.tsx`.  
Type `TranslationKey` suy ra từ object `en` — thiếu key sẽ lỗi TypeScript.

---

## Cây thư mục chính

```
ui/src/
├── components/layout/TabbedPageLayout.tsx    # shared layout
├── pages/settings/
│   ├── SystemSettingsPage.tsx
│   ├── GeneralSettingsTab.tsx
│   ├── SecuritySettingsTab.tsx
│   ├── FeaturesSettingsTab.tsx
│   ├── PaymentSettingsTab.tsx
│   ├── LegalNoticeSettingsTab.tsx
│   ├── BackupSettingsTab.tsx
│   ├── settingsTabs.ts
│   ├── settingsDraftTypes.ts
│   ├── useSettingsDraft.ts
│   └── components/...
└── pages/account/
    ├── AccountSettingsPage.tsx
    ├── ProfileTab.tsx
    ├── SecurityTab.tsx
    ├── ConnectionsTab.tsx
    ├── PreferencesTab.tsx
    ├── accountTabs.ts
    ├── accountDraftTypes.ts
    ├── useAccountDraft.ts
    └── components/...
```

---

## Tham chiếu UX (không copy 1:1)

- **new-api** `web/default/src/features/profile/` — hero, stats, security tiles
- **sub2api** admin settings — tham khảo độ phủ tab; voidllm đã **rút gọn** (không groups, không email SMTP, backup lite)

---

## Kiểm tra build

```bash
cd ui && npm run build
go build ./...
```

Cả hai đã pass sau redesign + gỡ email.