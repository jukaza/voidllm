# Design: Full Account Security + Admin Security Policies

**Status:** Draft  
**Date:** 2026-07-06  
**Scope:** `/account?tab=security`, `/settings?tab=security`, login 2FA flow

---

## Overview

Account Security tab hiện lẫn Live/Preview: đổi mật khẩu Live nhưng 2FA/sessions chỉ ghi `localStorage`. Admin Security chỉ có đăng ký, Turnstile, OAuth. Thiết kế này triển khai đầy đủ: **sessions thật**, **2FA TOTP**, **policy admin**, và sửa UI misleading.

## Goals

- User quản lý mật khẩu, 2FA, phiên đăng nhập qua API thật
- Admin cấu hình policy: 2FA availability, session TTL/multi-device, password min length
- Login email+password hỗ trợ bước 2FA khi user đã bật
- `AccountStatusStrip` + Security tab dùng cùng nguồn dữ liệu API
- Audit log cho hành động bảo mật quan trọng

## Non-Goals

- SMTP, email verification, password reset qua email
- OIDC (chỉ Google + GitHub)
- WebAuthn / SMS 2FA (phase sau)
- Bắt buộc 2FA toàn site (phase 1 chỉ `allow_user_enable`; `require_system_admin` phase 2)
- Admin xem/revoke session của user khác

---

## Key Decisions

| # | Decision | Rationale |
|---|----------|-----------|
| 1 | TOTP secret lưu `users.totp_secret_encrypted` (AES-256-GCM qua `pkg/crypto`) | Tái dùng `EncryptionKey` handler; không plaintext |
| 2 | Backup codes: bảng `user_totp_backup_codes` (hash SHA-256, single-use) | Không lưu plaintext; revoke từng code |
| 3 | Pending 2FA setup: Redis/in-memory `totp-pending:{userID}` TTL 10m | Tránh lưu secret chưa verify vào DB |
| 4 | Login 2FA: `temp_token` Redis TTL 5m sau password OK | Không issue session trước khi verify TOTP |
| 5 | Session metadata trên `api_keys`: `login_ip`, `user_agent`, `last_seen_at` | Không bảng sessions riêng; session = `key_type=session_key` |
| 6 | Policy `allow_multiple=false` (default) giữ hành vi hiện tại | `RevokeUserSessions` trước mỗi login |
| 7 | QR render FE từ `otpauth_url` (`qrcode` npm) | Giảm dependency SVG server |
| 8 | OAuth login **không** qua 2FA (phase 1) | IdP đã xác thực; đơn giản hóa |
| 9 | Mở rộng `GET /me` thay vì route `/me/security` riêng | Ít endpoint; FE một hook |
| 10 | `pquerna/otp` cho TOTP | Đã dùng trong `new-api`/`sub2api` cùng org |

---

## Data Model (Migration 0035)

```sql
-- users
ALTER TABLE users ADD COLUMN totp_secret_encrypted TEXT;      -- NULL = 2FA off
ALTER TABLE users ADD COLUMN totp_enabled_at TEXT;            -- RFC3339 when verified

-- api_keys (session metadata)
ALTER TABLE api_keys ADD COLUMN login_ip TEXT;
ALTER TABLE api_keys ADD COLUMN user_agent TEXT;
-- last_used_at already exists; use as last_seen_at

CREATE TABLE user_totp_backup_codes (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL REFERENCES users(id),
    code_hash   TEXT NOT NULL,           -- SHA-256(hex code)
    used_at     TEXT,                    -- NULL = unused
    created_at  TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_backup_codes_user ON user_totp_backup_codes(user_id) WHERE used_at IS NULL;
```

**Settings keys** (`internal/security/keys.go`):

```
security.two_fa.allow_user_enable          bool, default false
security.two_fa.require_system_admin       bool, default false (phase 2)
security.session.ttl_hours                 int,  default 24
security.session.allow_multiple            bool, default false
security.session.max_concurrent            int,  default 5
security.password.min_length               int,  default 8
security.password.allow_oauth_set_password bool, default true
```

---

## API Changes

### Extended `GET /api/v1/me`

```json
{
  "id": "...",
  "email": "...",
  "display_name": "...",
  "role": "member",
  "is_system_admin": false,
  "has_password": true,
  "auth_provider": "local",
  "two_fa_enabled": false,
  "two_fa_available": true,
  "active_session_count": 1
}
```

### Sessions

| Method | Route | Behavior |
|--------|-------|----------|
| GET | `/me/sessions` | List session keys for user; mark `current` by matching `keyInfo.ID` |
| DELETE | `/me/sessions/:id` | Revoke one (not current) |
| DELETE | `/me/sessions` | Revoke all except current |

Response item:

```json
{
  "id": "uuid",
  "ip": "203.0.113.45",
  "user_agent": "Mozilla/5.0...",
  "device_label": "Chrome on macOS",
  "created_at": "2026-07-06T10:00:00Z",
  "last_seen_at": "2026-07-06T12:00:00Z",
  "current": true
}
```

### 2FA

| Method | Route | Body | Response |
|--------|-------|------|----------|
| POST | `/me/2fa/setup` | — | `{ secret, otpauth_url }` (pending until verify) |
| POST | `/me/2fa/verify` | `{ code }` | `{ backup_codes: ["abcd-efgh", ...] }` |
| DELETE | `/me/2fa` | `{ password?, code? }` | 204 |

Guards: `two_fa_available` from admin config; reject if already enabled on setup.

### Password

| Method | Route | Notes |
|--------|-------|-------|
| POST | `/me/password` | Existing change password |
| POST | `/me/password/set` | OAuth-only first password; requires `allow_oauth_set_password` |

### Login 2FA

**`POST /auth/login`** — when password OK and `totp_enabled_at` set:

```json
{ "requires_2fa": true, "temp_token": "...", "expires_in": 300 }
```

**`POST /auth/login/2fa`**:

```json
{ "temp_token": "...", "code": "123456" }
```

→ standard `loginResponse` with session token.

Accept TOTP code or unused backup code (backup marks `used_at`).

### Admin `PUT /admin/settings/security`

Extend `UpdateInput`:

```json
{
  "two_fa": { "allow_user_enable": true },
  "session": { "ttl_hours": 24, "allow_multiple": true, "max_concurrent": 5 },
  "password": { "min_length": 8, "allow_oauth_set_password": true }
}
```

### Public `GET /public/auth-config`

```json
{ "two_fa": { "available": true }, ... }
```

---

## Backend Implementation Notes

### `issueUserSession` (`internal/api/admin/session.go`)

1. Load session policy from `security.LoadInternal`
2. If `!allow_multiple` → `RevokeUserSessions` (current behavior)
3. Else → enforce `max_concurrent`: delete oldest sessions if at limit
4. TTL from `ttl_hours` instead of hardcoded 24h
5. Store `login_ip = c.IP()`, `user_agent = c.Get("User-Agent")`

### `auth.Middleware` — optional hardening (PR11)

On cache miss, fallback `DB.LookupKeyByHash` → populate cache. Fixes logout sau restart khi DB còn session nhưng cache trống.

### TOTP package (`internal/totp/`)

- `GenerateSecret(email, issuer)` → secret + otpauth URL
- `Validate(secret, code)` — window ±1 step (30s)
- `EncryptSecret` / `DecryptSecret` via `pkg/crypto` with AAD `userID`

### Rate limits

Reuse login brute-force limits for `/auth/login/2fa` per IP + per `temp_token`.

---

## Frontend

### New hooks

- `useMe` — extend `MeResponse` type
- `useSessions`, `useRevokeSession`, `useRevokeOtherSessions`
- `useTwoFASetup`, `useTwoFAVerify`, `useTwoFADisable`
- `useSetPassword` (OAuth-only)

### `SecurityTab.tsx` layout

1. **Overview card** (Live) — chips từ `useMe` + `useConnections`
2. **Password card** (Live) — `ChangePasswordDialog` hoặc `SetPasswordDialog`
3. **2FA card** (Live when `two_fa_available`) — wizard dialog
4. **Sessions card** (Live) — inline list hoặc dialog

Remove `useAccountDraft` from security components. Delete fake defaults in `accountDraftTypes.ts` (sessions, oauth_github bound).

### `TwoFactorDialog.tsx` wizard

1. Intro + warning backup codes
2. QR (`qrcode` → canvas) + manual secret copy
3. Verify code input
4. Success: show backup codes once, checkbox "Đã lưu"

### `LoginPage.tsx`

State machine: `idle` → `requires_2fa` → show `TwoFactorLoginStep` component.

### `SecuritySettingsTab.tsx` — 3 section mới

- **2FA policy** — toggle `allow_user_enable`
- **Session policy** — TTL select, allow multiple, max concurrent
- **Password policy** — min length, allow OAuth set password

### `AccountStatusStrip.tsx`

```ts
const { data: me } = useMe()
const { data: connections } = useConnections()
// 2FA: me.two_fa_enabled
// OAuth: count linked providers
// Sessions: me.active_session_count
```

### i18n

Thêm keys `account.*`, `settings.*`, `login.twofa_*` — cả `en` và `vi`.

---

## Security Considerations

| Threat | Mitigation |
|--------|------------|
| TOTP secret leak | AES-GCM encrypted at rest |
| Backup code brute force | Rate limit; codes hashed |
| temp_token replay | Single-use consume; short TTL |
| Session fixation | New session per login; revoke policy |
| User enumeration on 2FA disable | Generic errors; bcrypt burn on password path |
| QR phishing | Show issuer "VoidLLM" + site name from config |

Audit events: `2fa.enabled`, `2fa.disabled`, `session.revoked`, `session.revoked_all`, `password.set`.

---

## PR Plan

### PR1 — Migration + security policy config (backend)
**Files:** `0035_user_security.up.sql`, `internal/security/keys.go`, `config.go`, tests  
**Deps:** none  
Add DB columns, settings keys, load/update policy in `security` package.

### PR2 — Session policy + metadata + APIs
**Files:** `session.go`, `internal/db/api_keys.go`, `sessions.go` (new), `routes.go`, tests  
**Deps:** PR1  
Policy-aware `issueUserSession`, `GET/DELETE /me/sessions`, update `last_seen_at` on authenticated requests (middleware hook or handler helper).

### PR3 — Extend GET /me + public auth config
**Files:** `auth.go`, `auth_config.go`, `security/config.go`  
**Deps:** PR1, PR2  
Return `has_password`, `two_fa_*`, `active_session_count`, `two_fa.available` on public config.

### PR4 — Account FE: real data + sessions Live
**Files:** `useMe.ts`, `useSessions.ts` (new), `AccountStatusStrip.tsx`, `SecurityTab.tsx`, `SessionsDialog.tsx`, `accountDraftTypes.ts`, `i18n.tsx`  
**Deps:** PR2, PR3  
Fix status strip, wire sessions, restructure Security tab badges, remove draft defaults.

### PR5 — Admin Security FE: policy sections
**Files:** `SecuritySettingsTab.tsx`, `useSecuritySettings.ts`, `i18n.tsx`  
**Deps:** PR1  
UI for 2FA/session/password policy toggles.

### PR6 — TOTP backend (setup/verify/disable)
**Files:** `internal/totp/`, `two_fa.go` (new handlers), `go.mod` (+ `pquerna/otp`), Redis pending store, tests  
**Deps:** PR1, PR3  

### PR7 — Login 2FA challenge flow
**Files:** `auth.go`, `login_2fa.go`, `LoginPage.tsx`, `TwoFactorLoginStep.tsx`, `i18n.tsx`, tests  
**Deps:** PR6  

### PR8 — Account 2FA wizard UI
**Files:** `TwoFactorDialog.tsx` (rewrite), `package.json` (+ `qrcode`), `i18n.tsx`  
**Deps:** PR6, PR7  

### PR9 — OAuth set password
**Files:** `set_password.go`, `SetPasswordDialog.tsx`, `SecurityTab.tsx`, tests  
**Deps:** PR1, PR3  

### PR10 — Auth cache DB fallback (hardening)
**Files:** `internal/auth/auth.go`, `internal/db/api_keys.go`, `loader.go`  
**Deps:** none (independent)  
Lookup key on cache miss; reduces false logout on restart.

### PR11 — Tests, docs, cleanup
**Files:** `docs/fe-settings-account.md`, integration tests, remove `useAccountDraft` security fields entirely  
**Deps:** all above  

---

## Rollout

1. Deploy PR1–3 (backend policy + sessions) — FE vẫn hoạt động, sessions mới có metadata
2. Deploy PR4–5 (FE account + admin policy)
3. Deploy PR6–8 (2FA) — admin bật `allow_user_enable` khi sẵn sàng
4. PR9–11 anytime after PR3

Default: `allow_user_enable=false` → 2FA UI ẩn cho đến khi admin bật.

---

## Open Questions

1. **`require_system_admin` 2FA** — làm trong PR6 hay defer PR12? → **Defer phase 2**
2. **Session `last_seen_at`** — update mỗi request hay throttle 5 phút? → **Throttle 5 phút** (giảm DB writes)
3. **Backup codes count** — 8 hay 10? → **10 codes**, format `xxxx-xxxx`