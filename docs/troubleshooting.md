---
title: "Troubleshooting"
description: "Common issues and solutions for Tavo"
section: root
order: 3
---
# Troubleshooting

## Startup Issues

### "admin key must be at least 32 characters"
Your `TAVO_ADMIN_KEY` is too short. Generate one:
```bash
export TAVO_ADMIN_KEY=$(openssl rand -base64 32)
```

### "TAVO_ADMIN_KEY is set but database already has keys, ignoring"
This is normal on subsequent starts. The admin key is only used on first boot to create the bootstrap user. After that, it's ignored.

### "TAVO_ENCRYPTION_KEY" missing
The encryption key is required for all deployments. It encrypts upstream API keys in the database:
```bash
export TAVO_ENCRYPTION_KEY=$(openssl rand -base64 32)
```

Save the value permanently. You need the **same** key on every restart and on every replica that shares the database.

### Provider works in UI but proxy returns `upstream temporarily unavailable`
The UI can show a channel as active while the proxy cannot decrypt its stored API key.

**Cause:** `TAVO_ENCRYPTION_KEY` changed since the key was saved (common after rename VoidLLM → Tavo or regenerating dev secrets).

**Fix:**
1. Restore the original encryption key in env / config, **or**
2. Re-enter the provider API key in **Providers → Connections** for each affected channel

**Verify:**
```bash
TAVO_ENCRYPTION_KEY='your-key' TAVO_DATABASE_DSN=./your.db go run ./scripts/check_decrypt/
```
Look for `decrypt OK` on active connections.

See [Local Development](deployment/local-dev.md) for the full dev checklist.

### Login returns `invalid email or password` (credentials are correct)
**Cause:** The backend is using a different database file than the one that contains your user — e.g. `tavo.db` (empty) instead of `voidllm.db`.

When using `-config tavo.dev.yaml`, `database.dsn` in the YAML wins. `TAVO_DATABASE_DSN` in the shell does **not** override a literal path in YAML unless the YAML uses `${TAVO_DATABASE_DSN:-...}`.

**Fix:** Point `database.dsn` at the correct file and restart the backend.

### Can't find bootstrap credentials
Tavo prints credentials to stdout on first start only. Check container logs:
```bash
docker logs tavo | grep "BOOTSTRAP"
kubectl logs deploy/tavo | grep "BOOTSTRAP"
```

If you missed them, delete the database and restart to re-bootstrap.

## Proxy Issues

### 401 Unauthorized
- API key is wrong, expired, or revoked
- Check key format: must start with `vl_uk_` or `vl_sk_`
- Session keys (`vl_sk_`) expire after 24 hours

### 404 Model not found
- The model name or alias doesn't exist in Tavo
- Check available models: `GET /api/v1/models` or the UI Models page
- Global model aliases are shared across all customers

### 502 Upstream unavailable
- The upstream LLM provider is unreachable
- Check the model's `base_url` in configuration
- Test connectivity: the model's health status on the Models page
- If using load balancing, check individual deployment health
- If the error code is `circuit_open` with message `upstream temporarily unavailable`, also check encryption key mismatch (see above) — this message is misleading when decrypt fails

### 429 Rate limit exceeded
- The caller has exceeded their per-key rate limit (RPM/RPD) or token budget
- Check limits on the API key in the Keys page

### Streaming responses cut off
- Reverse proxy may be buffering responses - set `proxy_buffering off` in Nginx
- Upstream timeout may be too short - increase `write_timeout` in server config
- Check per-model timeout if set

## UI Issues

### Can't log in
- Check email and password (case-sensitive)
- Session keys expire after 24 hours - you may need to log in again

### Logged out after every server restart (dev)
Session tokens are validated with an HMAC derived from `TAVO_ENCRYPTION_KEY`. If the key changes between restarts, existing browser tokens stop working even though they are still in `localStorage`.

**Fix:** Use the same `TAVO_ENCRYPTION_KEY` every time. See [Local Development](deployment/local-dev.md).

### Many duplicate devices in Account → Sessions
**Cause:** `security.session.allow_multiple` is enabled (Settings → Security). Each login creates a new session row. CLI tools (`curl`) appear as `Browser on Unknown OS` or `curl CLI`.

**Fix:**
- Disable “Allow multiple devices signed in at once” for single-session behavior, or
- Use “Revoke all others” on the Sessions page
- New logins from the same browser replace the previous session for that browser when `allow_multiple` is on

## Database Issues

### SQLite "database is locked"
- Only one Tavo instance can write to SQLite at a time
- For multi-instance deployments, use PostgreSQL
- Check that no other process is accessing the database file

### PostgreSQL connection errors
- Verify DSN format: `postgres://user:pass@host:5432/dbname?sslmode=require`
- Check network connectivity between Tavo and PostgreSQL
- Verify credentials and database permissions

## Performance

### High latency
- Check the `/metrics` endpoint for proxy latency percentiles
- Tavo adds < 500us overhead - if latency is high, the upstream provider is slow
- Check circuit breaker status on the Models page - a tripped breaker adds retry latency

### High memory usage
- Check for large request/response bodies (Tavo buffers bodies in memory during proxying)

## Getting Help

- [GitHub Issues](https://github.com/jukaza/tavo/issues) - bug reports and feature requests
- [Security](mailto:security@tavo.io.vn) - vulnerability reports
- [Contact](mailto:hello@tavo.io.vn) - general inquiries