---
title: "Documentation"
description: "Tavo documentation home - guides, reference, and API docs"
section: root
order: 0
---
# Tavo Documentation

Privacy-first LLM proxy and AI gateway. Self-hosted, single binary, sub-500us overhead.

## Getting Started

- [Quick Start](getting-started.md) - from `docker run` to first proxied request
- [Configuration Reference](configuration.md) - all YAML settings with examples

## Deployment

- [Local Development](deployment/local-dev.md) - backend + UI dev server, encryption key & DB checklist
- [Binary](deployment/binary.md) - standalone binary, Linux/macOS/Windows
- [Docker](deployment/docker.md) - Docker and Docker Compose
- [Kubernetes](deployment/kubernetes.md) - Helm chart, Istio, health probes
- [Reverse Proxy](deployment/reverse-proxy.md) - Nginx, Caddy, Traefik
- [Database](deployment/database.md) - SQLite, PostgreSQL, migration

## Models

- [Provider Setup](models/providers.md) - OpenAI, Anthropic, Azure, Ollama, vLLM, custom
- [Load Balancing](models/load-balancing.md) - strategies, failover, circuit breakers
- [Aliases](models/aliases.md) - global logical names for models

## Security

- [RBAC](security/rbac.md) - system admin and member roles
- [Privacy](security/privacy.md) - zero-knowledge architecture, GDPR
- [Hardening](security/hardening.md) - security checklist, TLS, network policies

## API

- [Overview](api/overview.md) - authentication, endpoints, error codes
- [OpenAPI Spec](api/swagger.yaml) - full API specification

## Frontend (đang triển khai)

- [Cài đặt hệ thống & Tài khoản (FE)](fe-settings-account.md) - redesign `/settings` và `/account`, Live/Preview, roadmap backend (tiếng Việt)

## Resources

- [Troubleshooting](troubleshooting.md) - common issues and solutions
- [Blog](https://github.com/jukaza/tavo/tree/main/docs/blog) - architecture deep-dives, benchmarks, guides
- [FAQ](https://github.com/jukaza/tavo/tree/main/docs/faq) - frequently asked questions
- [GitHub](https://github.com/jukaza/tavo) - source code, issues, releases
- [Security Policy](https://github.com/jukaza/tavo/blob/main/SECURITY.md) - vulnerability reporting
