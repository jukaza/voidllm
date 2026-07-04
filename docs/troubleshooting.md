---
title: "Troubleshooting"
description: "Common issues and solutions for VoidLLM"
section: root
order: 3
---
# Troubleshooting

## Startup Issues

### "admin key must be at least 32 characters"
Your `VOIDLLM_ADMIN_KEY` is too short. Generate one:
```bash
export VOIDLLM_ADMIN_KEY=$(openssl rand -base64 32)
```

### "VOIDLLM_ADMIN_KEY is set but database already has keys, ignoring"
This is normal on subsequent starts. The admin key is only used on first boot to create the bootstrap user. After that, it's ignored.

### "VOIDLLM_ENCRYPTION_KEY" missing
The encryption key is required for all deployments. It encrypts upstream API keys in the database:
```bash
export VOIDLLM_ENCRYPTION_KEY=$(openssl rand -base64 32)
```

### Can't find bootstrap credentials
VoidLLM prints credentials to stdout on first start only. Check container logs:
```bash
docker logs voidllm | grep "BOOTSTRAP"
kubectl logs deploy/voidllm | grep "BOOTSTRAP"
```

If you missed them, delete the database and restart to re-bootstrap.

## Proxy Issues

### 401 Unauthorized
- API key is wrong, expired, or revoked
- Check key format: must start with `vl_uk_` or `vl_sk_`
- Session keys (`vl_sk_`) expire after 24 hours

### 404 Model not found
- The model name or alias doesn't exist in VoidLLM
- Check available models: `GET /api/v1/models` or the UI Models page
- Global model aliases are shared across all customers

### 502 Upstream unavailable
- The upstream LLM provider is unreachable
- Check the model's `base_url` in configuration
- Test connectivity: the model's health status on the Models page
- If using load balancing, check individual deployment health

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

## Database Issues

### SQLite "database is locked"
- Only one VoidLLM instance can write to SQLite at a time
- For multi-instance deployments, use PostgreSQL
- Check that no other process is accessing the database file

### PostgreSQL connection errors
- Verify DSN format: `postgres://user:pass@host:5432/dbname?sslmode=require`
- Check network connectivity between VoidLLM and PostgreSQL
- Verify credentials and database permissions

## Performance

### High latency
- Check the `/metrics` endpoint for proxy latency percentiles
- VoidLLM adds < 500us overhead - if latency is high, the upstream provider is slow
- Check circuit breaker status on the Models page - a tripped breaker adds retry latency

### High memory usage
- Check for large request/response bodies (VoidLLM buffers bodies in memory during proxying)

## Getting Help

- [GitHub Issues](https://github.com/voidmind-io/voidllm/issues) - bug reports and feature requests
- [Security](mailto:security@voidmind.io) - vulnerability reports
- [Contact](mailto:hello@voidmind.io) - general inquiries