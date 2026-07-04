# VoidLLM

[![CI](https://github.com/voidmind-io/voidllm/actions/workflows/ci.yml/badge.svg)](https://github.com/voidmind-io/voidllm/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/voidmind-io/voidllm/graph/badge.svg?token=1OCK31RDMG)](https://codecov.io/gh/voidmind-io/voidllm)
[![Go Report Card](https://goreportcard.com/badge/github.com/voidmind-io/voidllm)](https://goreportcard.com/report/github.com/voidmind-io/voidllm)
[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/voidllm)](https://artifacthub.io/packages/search?repo=voidllm)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/voidmind-io/voidllm/badge)](https://securityscorecards.dev/viewer/?uri=github.com/voidmind-io/voidllm)
[![Snyk](https://snyk.io/test/github/voidmind-io/voidllm/badge.svg)](https://snyk.io/test/github/voidmind-io/voidllm)
[![Release](https://img.shields.io/github/v/release/voidmind-io/voidllm)](https://github.com/voidmind-io/voidllm/releases/latest)
[![Go](https://img.shields.io/github/go-mod/go-version/voidmind-io/voidllm)](go.mod)
[![License: BSL 1.1](https://img.shields.io/badge/License-BSL_1.1-blue.svg)](LICENSE)

**A privacy-first LLM proxy and API key marketplace for resellers.**

VoidLLM is a self-hosted LLM proxy that sits between your applications and LLM providers — OpenAI, Anthropic, Azure, Ollama, vLLM, or any custom endpoint. It provides prepaid wallet billing, per-key rate limits, usage tracking, global model aliases, and multi-deployment load balancing. One Go binary, sub-2ms proxy overhead, zero knowledge of your prompts.

![VoidLLM Dashboard](docs/screenshots/VoidLLM-Dashboard.jpg)

<details>
<summary>More screenshots</summary>

![Usage Analytics](docs/screenshots/VoidLLM-Usage.jpg)
![API Keys](docs/screenshots/VoidLLM-Keys.jpg)
![Playground](docs/screenshots/VoidLLM-Playground.jpg)

</details>

> **Privacy-First by Design:** VoidLLM is a zero-knowledge LLM proxy - it never stores, logs, or persists any prompt or response content. Not as a setting you can toggle - by architecture. Only metadata is tracked: who made the request, which model, how many tokens, how long it took. Your data stays yours.

---

## Why VoidLLM?

| Problem | How VoidLLM solves it |
|---|---|
| Customers share raw provider keys | Reseller API keys with per-key limits and wallet balances |
| No visibility into who's spending what | Per-key usage tracking + cost estimation |
| One runaway script burns the monthly budget | Per-key rate limits + token budgets enforced by the proxy |
| Switching providers means changing every app | Model aliases - clients call `default`, the proxy routes it anywhere |
| Provider goes down, everything breaks | Multi-deployment load balancing with automatic failover |
| Existing proxies log your prompts | Zero-knowledge proxy architecture - content never touches disk |

## How it works

```mermaid
flowchart LR
    App[Your app / SDK] -->|OpenAI-compatible| Proxy[VoidLLM proxy]
    Proxy --> Gate[API key + RBAC<br/>rate limits + budgets]
    Gate --> Router[Model alias + load balancing]
    Router --> Providers[OpenAI / Anthropic / Azure<br/>Ollama / vLLM / custom]
    Proxy -. metadata only .-> DB[(SQLite / PostgreSQL)]
```

Your apps speak the OpenAI API to VoidLLM. It authenticates the key, applies RBAC, rate limits and budgets, resolves the model alias, and routes to the right provider. Only **metadata** - who called, which model, token counts, latency - is written to the database. Prompt and response content passes through memory and is never persisted.

## Who it's for

**A good fit if you:**
- Self-host LLM infrastructure (vLLM, Ollama) or use managed providers and need one control plane
- Cannot log prompts or responses for privacy or compliance reasons
- Need per-customer API keys, wallet billing, and model routing
- Run multiple providers and want aliases, load balancing, and failover

**Not the right fit if you:**
- Want a hosted SaaS gateway - VoidLLM is self-hosted by design
- Need full prompt/response logging or content-level observability - it's zero-knowledge by architecture

## Quick Start

```bash
# Generate required keys
export VOIDLLM_ADMIN_KEY=$(openssl rand -base64 32)
export VOIDLLM_ENCRYPTION_KEY=$(openssl rand -base64 32)

# Start the proxy - no config file needed, VoidLLM boots with sensible defaults
docker run -p 8080:8080 \
  -e VOIDLLM_ADMIN_KEY -e VOIDLLM_ENCRYPTION_KEY \
  -v voidllm_data:/data \
  ghcr.io/voidmind-io/voidllm:latest
```

On first start VoidLLM prints bootstrap credentials to stdout (shown below) - no config file is required. Add models in the UI, or mount a `voidllm.yaml` to declare them (see [Configuration](#configuration)).

### Binary (no Docker needed)

Download the latest binary for your platform from the [releases page](https://github.com/voidmind-io/voidllm/releases/latest):

```bash
# Linux
curl -sL https://github.com/voidmind-io/voidllm/releases/latest/download/voidllm-linux-amd64.tar.gz | tar xz
export VOIDLLM_ADMIN_KEY=$(openssl rand -base64 32)
export VOIDLLM_ENCRYPTION_KEY=$(openssl rand -base64 32)
./voidllm
```

Available for: Linux (amd64, arm64), Windows (amd64, arm64), macOS (amd64, arm64).

On first start, VoidLLM prints your credentials to stdout:

```
========================================
 BOOTSTRAP COMPLETE - COPY THESE NOW
========================================
  API Key:    vl_uk_a3f2...
  Email:      admin@voidllm.local
  Password:   <random>
========================================
```

Open `http://localhost:8080`, log in with the email and password above, and start proxying. The API key is used for SDK calls (`Authorization: Bearer vl_uk_...`). These credentials are shown once - save them.

### One-Click Deploy

[![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/wild-pure?referralCode=fw9l7c)

Keys are auto-generated. Open the URL Railway gives you and start adding models.

```bash
# Your apps just point at the proxy instead of the provider
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer vl_uk_..." \
  -H "Content-Type: application/json" \
  -d '{"model":"default","messages":[{"role":"user","content":"hello"}]}'
```

Any OpenAI-compatible SDK works out of the box - just change the base URL to your VoidLLM proxy.

## Features

| Feature | Details |
|---|---|
| OpenAI-compatible proxy | `/v1/chat/completions`, embeddings, images, audio, streaming |
| Multi-provider routing | OpenAI, Anthropic, Azure, Ollama, vLLM, any custom endpoint |
| Load balancing | Round-robin, least-latency, weighted, priority across deployments |
| Automatic failover | Retry on 5xx/timeout, circuit breakers, health-aware routing |
| Web UI | Dashboard, playground, API keys, models, usage, wallet, settings |
| RBAC | `system_admin` and `member` roles |
| Rate limits | Per-key requests per minute/day |
| Token budgets | Daily/monthly limits, real-time enforcement |
| Usage tracking | Tokens, cost, duration, TTFT per request |
| Usage export | CSV / JSON download |
| Model aliases | Clients call `default`, you control where it routes |
| Prepaid wallet | Customer balance tracking and deduction per request |
| Prometheus metrics | Latency, tokens, active streams, routing, health |
| Database | SQLite (default) or PostgreSQL |
| Deployment | Docker, Helm chart, graceful shutdown |

## Documentation

**[Full documentation](docs/index.md)** | **[Blog](https://voidllm.ai/blog)** | **[FAQ](https://voidllm.ai/faq)**

| Topic | Guide |
|---|---|
| Getting Started | [Quick Start](docs/getting-started.md) |
| Configuration | [All YAML settings](docs/configuration.md) |
| Docker | [Docker deployment](docs/deployment/docker.md) |
| Kubernetes | [Helm chart](docs/deployment/kubernetes.md) |
| Providers | [OpenAI, Anthropic, Azure, Ollama, vLLM](docs/models/providers.md) |
| Load Balancing | [Strategies, failover, circuit breakers](docs/models/load-balancing.md) |
| RBAC | [Roles and permissions](docs/security/rbac.md) |
| Privacy | [Zero-knowledge architecture](docs/security/privacy.md) |
| API Reference | [Endpoints and error codes](docs/api/overview.md) |
| Troubleshooting | [Common issues](docs/troubleshooting.md) |

## Configuration

```yaml
server:
  proxy:
    port: 8080

models:
  # Single endpoint
  - name: dolphin-mistral
    provider: ollama
    base_url: http://localhost:11434/v1
    timeout: 30s
    aliases: [default]
    pricing:
      input_per_1m: 0.15
      output_per_1m: 0.60

  # Load balanced - multiple deployments with failover
  - name: gpt-4o
    strategy: round-robin
    aliases: [smart]
    deployments:
      - name: azure-east
        provider: azure
        base_url: https://eastus.openai.azure.com
        api_key: ${AZURE_EAST_KEY}
        azure_deployment: gpt-4o
        priority: 1
      - name: openai-fallback
        provider: openai
        base_url: https://api.openai.com/v1
        api_key: ${OPENAI_KEY}
        priority: 2

settings:
  admin_key: ${VOIDLLM_ADMIN_KEY}
  encryption_key: ${VOIDLLM_ENCRYPTION_KEY}
```

Supported providers: `openai` · `anthropic` · `azure` · `vllm` · `ollama` · `custom`

Environment variables are interpolated with `${VAR}` syntax. Secrets never hardcoded.

## Deployment

### Docker Compose

```bash
cp voidllm.yaml.example voidllm.yaml
export VOIDLLM_ADMIN_KEY=$(openssl rand -base64 32)
export VOIDLLM_ENCRYPTION_KEY=$(openssl rand -base64 32)
docker-compose up
```

### Kubernetes (Helm)

```bash
helm install voidllm chart/voidllm/ \
  --set secrets.adminKey=$(openssl rand -base64 32) \
  --set secrets.encryptionKey=$(openssl rand -base64 32) \
  --set config.models[0].name=my-model \
  --set config.models[0].provider=ollama \
  --set config.models[0].base_url=http://ollama:11434/v1
```

PostgreSQL and Redis are available as optional subcharts for production deployments.

### From Source

```bash
# Prerequisites: Go 1.23+, Node 20+
cd ui && npm ci && npm run build && cd ..
go run ./cmd/voidllm --config voidllm.yaml
```

## Production Checklist

- Use **PostgreSQL** instead of SQLite
- Put VoidLLM behind **TLS / a reverse proxy**
- **Isolate the admin UI/API** from public proxy traffic (separate `admin.port`, with TLS)
- Use a strong `VOIDLLM_ENCRYPTION_KEY` (32+ bytes) and keep it in a **secrets manager**
- Don't use `VOIDLLM_ADMIN_KEY` as a production API key - it's for bootstrap only
- Set **resource limits** and **network policies** in Kubernetes
- Scrape **`/metrics`** with Prometheus
- Configure **database backups**
- For multiple replicas, use **Redis** so rate limits and budgets are shared (they are per-process without it) - Enterprise

See the [Security Hardening guide](docs/security/hardening.md) for the full list.

## Privacy

This is not a feature toggle. It's an architectural decision that makes VoidLLM a privacy-first LLM proxy.

- **No request body** in logs, DB, or any persistent storage
- **No response body** in logs, DB, or any persistent storage
- **No prompt caching** - content passes through memory only
- **Usage events** contain only: who (key/user), what (model), how much (tokens/cost)
- There is no `enable_content_logging` option. It doesn't exist.
- Designed to support GDPR compliance - no personal data in prompts is stored or processed

## CLI Tools

```bash
# Bidirectional database migration
voidllm migrate --from sqlite:///data/voidllm.db --to postgres://user:pass@host/db

# License management (for Enterprise)
voidllm license verify < license.jwt
```

## Project

- [Contributing](CONTRIBUTING.md)
- [Changelog](CHANGELOG.md)
- [Security Policy](SECURITY.md) - report vulnerabilities privately to security@voidmind.io
- [Code of Conduct](CODE_OF_CONDUCT.md)

## License

[Business Source License 1.1](LICENSE) - source available, self-hosting permitted,
competing hosted services prohibited. Converts to Apache 2.0 four years after each release.

---

Built by [VoidMind](https://voidmind.io) · [voidllm.ai](https://voidllm.ai)

This project was built with significant assistance from AI (Claude by Anthropic).
