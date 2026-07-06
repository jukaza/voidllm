---
title: "Getting Started"
description: "From docker run to your first proxied LLM request in 3 minutes"
section: root
order: 1
---
# Getting Started

Tavo runs as a single binary with the admin UI embedded. No separate frontend server, no Node.js, no extra containers.

## Quick Start (Docker)

```bash
docker run -p 8080:8080 \
  -v tavo_data:/data \
  -e TAVO_ENCRYPTION_KEY=$(openssl rand -base64 32) \
  -e TAVO_ADMIN_KEY=my-admin-key-at-least-32-chars!! \
  ghcr.io/jukaza/tavo:latest
```

## Quick Start (Binary)

Download the latest binary for your platform from the [releases page](https://github.com/jukaza/tavo/releases/latest):

```bash
# Linux (amd64)
curl -sL https://github.com/jukaza/tavo/releases/latest/download/tavo-linux-amd64.tar.gz | tar xz
export TAVO_ADMIN_KEY=$(openssl rand -base64 32)
export TAVO_ENCRYPTION_KEY=$(openssl rand -base64 32)
./tavo
```

Available for: Linux (amd64, arm64), Windows (amd64, arm64), macOS (amd64, arm64).

The database defaults to `./tavo.db` in the current directory. No config file required - Tavo starts with sensible defaults and the bootstrap wizard handles initial setup.

On first start, Tavo prints your credentials to stdout:

```
========================================
 BOOTSTRAP COMPLETE - COPY THESE NOW
========================================
  API Key:    vl_uk_a3f2...
  Email:      admin@tavo.local
  Password:   <random>
========================================
```

- **Email + Password** - for logging into the UI at `http://localhost:8080`
- **API Key** (`vl_uk_...`) - for SDK and API calls
- These are shown once - save them

## Add a Model

Edit `tavo.yaml` or use the UI (Models -> Create Model):

```yaml
models:
  - name: gpt-4o
    provider: openai
    base_url: https://api.openai.com/v1
    api_key: ${OPENAI_KEY}
    aliases: [default]
```

See [Provider Setup](models/providers.md) for all supported providers.

## Send Your First Request

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer vl_uk_your_key_here" \
  -H "Content-Type: application/json" \
  -d '{"model": "default", "messages": [{"role": "user", "content": "hello"}]}'
```

Tavo resolves `default` to whatever model you configured with that alias, forwards the request, and streams the response back. Under 500 microseconds of overhead.

## Connect Your IDE

### Cursor / Windsurf (LLM Proxy)

Change the base URL to your Tavo instance:
```
Base URL: http://localhost:8080/v1
API Key: vl_uk_...
```

## Explore the UI

Open `http://localhost:8080` and explore:

- **Dashboard** - request stats, token usage, model health
- **Keys** - create and manage API keys
- **Models** - add models, configure aliases, view health
- **Usage** - track consumption by user, model, and key
- **Playground** - test models directly in the browser

## Next Steps

- [Configuration Reference](configuration.md) - all YAML settings
- [Deployment Guide](deployment/docker.md) - Docker, Kubernetes, PostgreSQL
- [Load Balancing](models/load-balancing.md) - multi-deployment failover
- [RBAC](security/rbac.md) - system admin and member roles
