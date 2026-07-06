---
title: "Reverse Proxy"
description: "Configure Nginx, Caddy, or Traefik in front of Tavo"
section: deployment
order: 3
---
# Reverse Proxy Configuration

Tavo works behind any reverse proxy (Nginx, Traefik, Caddy, K8s Ingress).

## Nginx

```nginx
location /v1/ {
    proxy_pass http://tavo:8080;
    proxy_http_version 1.1;
    proxy_set_header Connection "";
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_buffering off;              # Required for SSE streaming
}

location / {
    proxy_pass http://tavo:8080;
    proxy_http_version 1.1;
    proxy_set_header Connection "";
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_buffering off;
}
```

## Caddy

```
tavo.example.com {
    reverse_proxy tavo:8080
}
```

Caddy handles TLS automatically via Let's Encrypt.

## Traefik

```yaml
http:
  routers:
    tavo:
      rule: "Host(`tavo.example.com`)"
      service: tavo
      tls:
        certResolver: letsencrypt
  services:
    tavo:
      loadBalancer:
        servers:
          - url: "http://tavo:8080"
```

## Important Notes

- **Streaming:** Ensure your reverse proxy does not buffer responses. SSE streaming requires `proxy_buffering off` (Nginx) or equivalent.
- **Timeouts:** Set upstream timeouts high enough for LLM responses (60s+). Short timeouts will kill streaming responses.
- **WebSocket:** Not required. Tavo uses HTTP POST for all proxy requests.
- **TLS:** Terminate TLS at the reverse proxy or ingress level. Tavo supports TLS on the admin port (`server.admin.tls`) but not on the proxy port.
