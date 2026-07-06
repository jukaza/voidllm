# Tavo

[![CI](https://github.com/jukaza/tavo/actions/workflows/ci.yml/badge.svg)](https://github.com/jukaza/tavo/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/jukaza/tavo/graph/badge.svg?token=1OCK31RDMG)](https://codecov.io/gh/jukaza/tavo)
[![Go Report Card](https://goreportcard.com/badge/github.com/jukaza/tavo)](https://goreportcard.com/report/github.com/jukaza/tavo)
[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/tavo)](https://artifacthub.io/packages/search?repo=tavo)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/jukaza/tavo/badge)](https://securityscorecards.dev/viewer/?uri=github.com/jukaza/tavo)
[![Snyk](https://snyk.io/test/github/jukaza/tavo/badge.svg)](https://snyk.io/test/github/jukaza/tavo)
[![Release](https://img.shields.io/github/v/release/jukaza/tavo)](https://github.com/jukaza/tavo/releases/latest)
[![Go](https://img.shields.io/github/go-mod/go-version/jukaza/tavo)](go.mod)
[![License: BSL 1.1](https://img.shields.io/badge/License-BSL_1.1-blue.svg)](LICENSE)

**Self-hosted LLM API marketplace — sell OpenAI-compatible access with prepaid wallets.**

Tavo is a single binary that combines a **public storefront**, **OpenAI-compatible proxy**, and **operator admin UI**. Customers sign up, top up a prepaid wallet, and call your models with reseller API keys. You configure upstream providers, set sell prices, approve top-ups, and monitor usage — without handing customers your raw provider keys.

Sub-2ms proxy overhead. Zero-knowledge by architecture: prompts and responses never touch disk.

> **Current phase:** API routing and key issuance first. Wallet top-up enforcement is **off by default** (`settings.wallet.enforce_balance: false`) so customers can call the proxy immediately after signup. Turn billing on when you are ready to go live with prepaid balances.

![Tavo Dashboard](docs/screenshots/Tavo-Dashboard.jpg)

<details>
<summary>More screenshots</summary>

![Usage Analytics](docs/screenshots/Tavo-Usage.jpg)
![API Keys](docs/screenshots/Tavo-Keys.jpg)
![Playground](docs/screenshots/Tavo-Playground.jpg)

</details>

---

## What you get

| Layer | What it does |
|---|---|
| **Storefront** | Public landing page with model catalog and pricing (`/`, `/register`) |
| **Proxy** | OpenAI-compatible `/v1/*` — chat, embeddings, images, audio, streaming |
| **Billing** | Prepaid wallet per customer; requests debit balance; empty wallet → `402` |
| **Keys** | Per-customer API keys (`vl_uk_`) with RPM/RPD and token budgets |
| **Admin** | Provider catalog, top-up review, wallet adjustments, models, usage |

One deployment. No separate frontend server — the UI is embedded in the Go binary.

## Customer journey

1. **Sign up** at `/register` — account, wallet, and first API key are created automatically
2. **Get an API key** — issued on signup; create more from the dashboard
3. **Call the API** — point any OpenAI SDK at your Tavo base URL

When you enable `settings.wallet.enforce_balance: true`, add a top-up step: customers request credits, you approve them in the admin queue, and empty wallets receive `402`.

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer vl_uk_..." \
  -H "Content-Type: application/json" \
  -d '{"model":"default","messages":[{"role":"user","content":"hello"}]}'
```

## Operator journey

1. **Bootstrap** — first start creates a `system_admin` account (credentials printed once)
2. **Add providers** — connect OpenAI, Anthropic, Azure, Ollama, vLLM, or custom upstreams
3. **Publish catalog** — set sell prices (per-token or per-request) and global model aliases
4. **Run the business** — approve top-ups, adjust wallets, rotate keys, watch usage and margins

## How it works

```mermaid
flowchart LR
    Visitor[Visitor] -->|signup| Store[Storefront UI]
    Store --> Wallet[(Prepaid wallet)]
    App[Customer app / SDK] -->|OpenAI API| Proxy[Tavo proxy]
    Proxy --> Check[Key auth + limits + balance]
    Check --> Route[Alias + load balancing]
    Route --> Upstream[OpenAI / Anthropic / Azure<br/>Ollama / vLLM / custom]
    Proxy -. usage metadata .-> DB[(SQLite / PostgreSQL)]
    Admin[system_admin] -. top-ups, catalog .-> Wallet
```

Authentication happens on every request. Rate limits and token budgets are enforced per API key. The wallet is checked before forwarding; usage is logged as metadata only (key, model, tokens, cost, latency) — never prompt or response content.

## Privacy

Tavo is zero-knowledge by architecture, not by configuration:

- No request or response bodies in logs, database, or persistent storage
- No prompt caching — content passes through memory only
- Usage events contain: key/user, model, tokens, cost estimate, latency

## Quick Start

```bash
export TAVO_ADMIN_KEY=$(openssl rand -base64 32)
export TAVO_ENCRYPTION_KEY=$(openssl rand -base64 32)

docker run -p 8080:8080 \
  -e TAVO_ADMIN_KEY -e TAVO_ENCRYPTION_KEY \
  -v tavo_data:/data \
  ghcr.io/jukaza/tavo:latest
```

On first start, Tavo prints bootstrap credentials to stdout:

```
========================================
 BOOTSTRAP COMPLETE - COPY THESE NOW
========================================
  API Key:    vl_uk_a3f2...
  Email:      admin@tavo.local
  Password:   <random>
========================================
```

| URL | Who | Purpose |
|---|---|---|
| `http://localhost:8080/` | Public | Landing + pricing catalog |
| `http://localhost:8080/register` | Public | Customer signup |
| `http://localhost:8080/login` | Anyone | Login (admin or customer) |
| `http://localhost:8080/dashboard` | Authenticated | Admin or customer dashboard |

Log in as `system_admin`, add models and providers, then open `/` in a private window to test the customer signup flow.

### Binary

```bash
curl -sL https://github.com/jukaza/tavo/releases/latest/download/tavo-linux-amd64.tar.gz | tar xz
export TAVO_ADMIN_KEY=$(openssl rand -base64 32)
export TAVO_ENCRYPTION_KEY=$(openssl rand -base64 32)
./tavo
```

Available for Linux, macOS, and Windows (amd64 and arm64).

### One-Click Deploy

[![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/wild-pure?referralCode=fw9l7c)

## Features

| Area | Details |
|---|---|
| **Marketplace** | Public catalog, self-service signup, prepaid wallet, manual top-up review |
| **Proxy** | OpenAI-compatible `/v1/*`, streaming, sub-2ms overhead |
| **Routing** | Global model aliases, multi-deployment load balancing, circuit breakers |
| **Billing** | Per-token or per-request sell pricing; wallet debit on each call |
| **Keys** | Per-key RPM/RPD and daily/monthly token budgets |
| **Roles** | `system_admin` (operator) and `member` (customer) |
| **Observability** | Usage dashboards, cost reports, Prometheus `/metrics` |
| **Deployment** | Single binary, embedded UI, Docker, Helm, SQLite or PostgreSQL |

## Configuration

```yaml
server:
  proxy:
    port: 8080

models:
  - name: gpt-4o
    provider: openai
    base_url: https://api.openai.com/v1
    api_key: ${OPENAI_KEY}
    aliases: [default]
    bill_per_token: true
    pricing:
      input_per_1m: 2.50    # sell price (USD)
      output_per_1m: 10.00

  - name: llama-local
    provider: ollama
    base_url: http://localhost:11434/v1
    aliases: [fast]
    bill_per_request: 0.001

settings:
  admin_key: ${TAVO_ADMIN_KEY}
  encryption_key: ${TAVO_ENCRYPTION_KEY}
  bootstrap:
    admin_email: admin@tavo.local
  wallet:
    enforce_balance: false   # true = require positive balance (402 when empty)
```

Supported upstream providers: `openai` · `anthropic` · `azure` · `vllm` · `ollama` · `custom`

Secrets use `${VAR}` interpolation — never hardcode API keys in config files.

## Deployment

### Docker Compose

```bash
cp tavo.yaml.example tavo.yaml
export TAVO_ADMIN_KEY=$(openssl rand -base64 32)
export TAVO_ENCRYPTION_KEY=$(openssl rand -base64 32)
docker-compose up
```

### Kubernetes (Helm)

```bash
helm install tavo chart/tavo/ \
  --set secrets.adminKey=$(openssl rand -base64 32) \
  --set secrets.encryptionKey=$(openssl rand -base64 32)
```

PostgreSQL and Redis are optional subcharts for production multi-replica setups.

### From Source

```bash
# Prerequisites: Go 1.23+, Node 20+
cd ui && npm ci && npm run build && cd ..
go run ./cmd/tavo --config tavo.yaml
```

## Production Checklist

- Use **PostgreSQL** instead of SQLite for production
- Terminate **TLS** at a reverse proxy (Nginx, Caddy, Traefik)
- Keep `TAVO_ENCRYPTION_KEY` in a secrets manager (encrypts upstream API keys at rest)
- Do not use the bootstrap admin key as a customer API key
- Scrape **`/metrics`** with Prometheus
- Configure **database backups**
- For multiple replicas, enable **Redis** so rate limits stay consistent across pods

See [Security Hardening](docs/security/hardening.md) for the full list.

## Documentation

**[Full docs](docs/index.md)** · **[Blog](https://github.com/jukaza/tavo/tree/main/docs/blog)** · **[FAQ](https://github.com/jukaza/tavo/tree/main/docs/faq)**

| Topic | Guide |
|---|---|
| Getting Started | [Quick Start](docs/getting-started.md) |
| Configuration | [YAML reference](docs/configuration.md) |
| API | [Endpoints](docs/api/overview.md) · [OpenAPI](docs/api/swagger.yaml) |
| RBAC | [Roles](docs/security/rbac.md) |
| Privacy | [Zero-knowledge](docs/security/privacy.md) |
| Deployment | [Docker](docs/deployment/docker.md) · [Kubernetes](docs/deployment/kubernetes.md) |
| Models | [Providers](docs/models/providers.md) · [Load balancing](docs/models/load-balancing.md) · [Aliases](docs/models/aliases.md) |

## CLI

```bash
# Migrate between SQLite and PostgreSQL
tavo migrate --from sqlite:///data/tavo.db --to postgres://user:pass@host/db
```

## Project

- [Contributing](CONTRIBUTING.md)
- [Changelog](CHANGELOG.md)
- [Security](SECURITY.md) — report vulnerabilities to security@tavo.io.vn
- [Code of Conduct](CODE_OF_CONDUCT.md)

## License

[Business Source License 1.1](LICENSE) — source available, self-hosting permitted,
competing hosted services prohibited. Converts to Apache 2.0 four years after each release.

---

Built by [jukaza](https://github.com/jukaza) · [Docs](https://github.com/jukaza/tavo/tree/main/docs) · API: `api.tavo.io.vn`