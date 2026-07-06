---
title: "Local Development"
description: "Run Tavo backend + UI dev server locally without losing providers, login, or API keys after restart"
section: deployment
order: 2
---
# Local Development

Hướng dẫn chạy Tavo trên máy dev (backend Go + UI Vite). Đọc phần **Checklist** trước khi restart — tránh các lỗi hay gặp: login fail, `invalid email or password`, Playground báo `upstream temporarily unavailable`.

## Prerequisites

- Go 1.23+
- Node.js 20+
- Binary `tavo` đã build (hoặc `go run ./cmd/tavo`)

```bash
go build -o tavo ./cmd/tavo
```

## One-time secrets (ghi nhớ)

Tạo file env local (không commit):

```bash
# .env.local — add to .gitignore
export TAVO_ENCRYPTION_KEY='dev-encryption-key-32chars-long!!'
export TAVO_ADMIN_KEY='my-admin-key-at-least-32-chars!!'   # chỉ dùng lần bootstrap đầu
```

| Biến | Bắt buộc | Ghi chú |
|---|---|---|
| `TAVO_ENCRYPTION_KEY` | **Có** | **Không đổi** sau khi đã lưu provider API key trong DB |
| `TAVO_ADMIN_KEY` | Lần đầu | Chỉ dùng khi DB trống; sau bootstrap có thể bỏ |
| `TAVO_DATABASE_DSN` | Không | Chỉ có hiệu lực khi **không** dùng `-config`; với `tavo.dev.yaml` hãy set `database.dsn` trong file |

Load env trước mỗi phiên dev:

```bash
set -a && source .env.local && set +a
```

### Encryption key — quy tắc vàng

- Provider / connection API key được **mã hóa AES-256-GCM** trong DB (`api_key_encrypted`).
- Server **bắt buộc** có `TAVO_ENCRYPTION_KEY` — không thể tắt mã hóa.
- **Đổi key = không giải mã được key cũ** → Playground/proxy lỗi dù kênh vẫn “Sẵn sàng” trên UI.
- Repo dev dùng DB legacy `voidllm.db` (từ VoidLLM): default trong `tavo.dev.yaml` là `dev-encryption-key-32chars-long!!`.
- DB mới (`tavo.db`): generate key một lần và **lưu vĩnh viễn**:

```bash
export TAVO_ENCRYPTION_KEY=$(openssl rand -base64 32)
echo "$TAVO_ENCRYPTION_KEY"   # copy vào password manager / .env.local
```

Nếu đã đổi nhầm key: vào **Providers → Connections**, nhập lại API key từng kênh (hoặc restore DB backup).

## Config file

Dev dùng `tavo.dev.yaml` (không phải `tavo.yaml` production):

```yaml
database:
  dsn: ${TAVO_DATABASE_DSN:-./voidllm.db}   # DB có data dev hiện tại

settings:
  encryption_key: "${TAVO_ENCRYPTION_KEY:-dev-encryption-key-32chars-long!!}"
```

**Lưu ý:** Khi chạy `./tavo -config tavo.dev.yaml`, giá trị `database.dsn` lấy từ YAML (có hỗ trợ `${VAR:-default}`). Export `TAVO_DATABASE_DSN` **không** override YAML trừ khi YAML tham chiếu biến đó.

## Start servers

### Terminal 1 — Backend

```bash
set -a && source .env.local && set +a
cd /path/to/tavo
./tavo -dev -config tavo.dev.yaml
```

- API + embedded UI: http://127.0.0.1:8080
- `-dev`: CORS `*`, debug log, pprof `:6060`

### Terminal 2 — UI (hot reload)

```bash
cd ui
npm install
npm run dev
```

- Dev UI: http://127.0.0.1:5173
- Vite proxy `/api` → `http://localhost:8080`

**Luôn mở http://127.0.0.1:5173** khi dev frontend. Port 8080 phục vụ UI build embed (có thể cũ hơn source `ui/`).

### Restart nhanh

```bash
fuser -k 8080/tcp 5173/tcp 2>/dev/null; sleep 1
# rồi start lại hai terminal như trên (cùng .env.local)
```

## Phiên đăng nhập (sessions)

### Vì sao restart server bị logout?

Token login (`sk-...` trong `localStorage`) được hash bằng **HMAC secret** dẫn xuất từ `TAVO_ENCRYPTION_KEY`. Đổi encryption key → hash không khớp DB → 401 → UI đẩy về `/login`.

**Cách tránh:** giữ cố định `TAVO_ENCRYPTION_KEY` (xem phần trên). Cùng key + cùng DB → restart **không** cần login lại (token còn hạn 24h–720h tùy policy).

### Vì sao nhiều “thiết bị” cùng một trình duyệt?

Trong **Settings → Security**, nếu bật **“Cho phép nhiều thiết bị đăng nhập cùng lúc”** (`allow_multiple: true`), mỗi lần login tạo **session mới** trong DB thay vì thay thế session cũ.

- `Browser on Unknown OS` = thường là session từ **curl** / script (không phải Chrome)
- `Chrome on Linux` = session trình duyệt thật

**Dev gợi ý:** tắt “nhiều thiết bị” → mỗi login chỉ giữ 1 phiên. Hoặc giữ bật: từ bản code mới, login lại **cùng IP + User-Agent** sẽ tự thay session cũ của trình duyệt đó.

Dọn session rác: **Account → Sessions → Thu hồi các phiên khác**.

## Checklist sau mỗi lần restart

- [ ] `TAVO_ENCRYPTION_KEY` **giống lúc tạo/lưu kênh** (xem `.env.local`)
- [ ] Backend log không có `failed to decrypt` / `combo hop: failed to decrypt`
- [ ] `database.dsn` trỏ đúng file DB (`voidllm.db` vs `tavo.db`)
- [ ] Health: `curl -s http://127.0.0.1:8080/health` → 200
- [ ] Login UI hoạt động (user trong DB, không phải DB trống)
- [ ] Playground: gửi 1 message → có reply (không phải `upstream temporarily unavailable`)

## Login dev (voidllm.db mẫu)

Nếu dùng DB dev sẵn có trong repo, email có thể **không** dạng `admin@...` — kiểm tra bảng `users`:

```bash
sqlite3 voidllm.db "SELECT email, display_name FROM users WHERE deleted_at IS NULL;"
```

Mật khẩu trong DB là **bcrypt hash** — không đọc plaintext từ DB. Quên pass:

- System admin reset qua API `PATCH /api/v1/users/:id` với `password`, hoặc
- Dev: ghi đè hash trong SQLite (chỉ local), hoặc
- Xóa DB và bootstrap lại (mất data)

## Triệu chứng → nguyên nhân

| Triệu chứng | Nguyên nhân thường gặp |
|---|---|
| `invalid email or password` (đúng pass) | Backend đọc **DB trống** (`tavo.db`) thay vì `voidllm.db` |
| `upstream temporarily unavailable` / `circuit_open` | **`TAVO_ENCRYPTION_KEY` sai** → không decrypt API key kênh (UI vẫn hiện kênh OK) |
| Màn hình đen sau login | Thường do frontend lỗi JS hoặc mở nhầm port 8080 thay vì 5173 |
| Provider “active” nhưng proxy fail | Key mã hóa bằng encryption key khác; re-enter API key trên UI |

## Verify encryption key (debug)

```bash
# Decrypt OK → in "decrypt OK"
TAVO_ENCRYPTION_KEY='your-key-here' TAVO_DATABASE_DSN=./voidllm.db go run ./scripts/check_decrypt/
```

Script in danh sách connection active và kết quả giải mã.

## Production deploy

Local dev **không** dùng cho production. Xem:

- [Binary](binary.md) — systemd, secrets cố định
- [Docker](docker.md) — volume `/data`, env trong compose
- [Kubernetes](kubernetes.md) — Secret `TAVO_ENCRYPTION_KEY`
- [Hardening](../security/hardening.md) — backup key cùng DB

**Production checklist thêm:**

- [ ] `TAVO_ENCRYPTION_KEY` trong Secret / vault, **backup cùng DB**
- [ ] Volume DB persistent (`tavo.db` hoặc Postgres)
- [ ] Không dùng `-dev` flag
- [ ] Document key rotation: re-enter tất cả provider keys sau khi rotate