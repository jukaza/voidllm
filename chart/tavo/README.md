# Tavo Helm Chart

Privacy-first LLM proxy and AI gateway. Zero knowledge of your prompts.

## Quick Start

```bash
helm repo add tavo https://jukaza.github.io/tavo
helm repo update
helm install tavo tavo/tavo \
  --set secrets.adminKey=$(openssl rand -base64 32) \
  --set secrets.encryptionKey=$(openssl rand -base64 32)
```

Open the service URL - Tavo prints bootstrap credentials to the pod logs:

```bash
kubectl logs deploy/tavo | grep "BOOTSTRAP"
```

## Features

- **OpenAI-compatible proxy** - route to OpenAI, Anthropic, Azure, Ollama, vLLM
- **Sub-500us overhead** - Go + Fiber, in-memory auth and model resolution
- **Built-in admin UI** - embedded in the binary, no separate deployment
- **RBAC** - system admin and member roles with per-key rate limits and token budgets
- **Load balancing** - round-robin, least-latency, weighted, priority with automatic failover
- **Zero-knowledge** - never stores prompt or response content

## Configuration

See [values.yaml](values.yaml) for all configuration options.

### Minimal

```yaml
secrets:
  adminKey: "your-admin-key-at-least-32-characters"
  encryptionKey: "base64-encoded-32-byte-key"
```

### With models

```yaml
config:
  models:
    - name: gpt-4o
      provider: openai
      base_url: https://api.openai.com/v1
      api_key: ${OPENAI_KEY}
      aliases: [default]

secrets:
  extraEnv:
    OPENAI_KEY: "sk-..."
```

### With PostgreSQL

```yaml
postgresql:
  enabled: true
  auth:
    password: "your-db-password"
```

### With Redis (multi-instance)

```yaml
redis:
  enabled: true

replicaCount: 3

autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 10
```

### With Istio

```yaml
istio:
  enabled: true
  virtualService:
    hosts:
      - tavo.example.com
  gateway:
    servers:
      - port:
          number: 443
          name: https
          protocol: HTTPS
        tls:
          mode: SIMPLE
          credentialName: tavo-tls
        hosts:
          - tavo.example.com
```

## Persistence

SQLite data is stored in a PersistentVolumeClaim by default. When using PostgreSQL, persistence can be disabled:

```yaml
persistence:
  enabled: false
```

## Links

- [GitHub](https://github.com/jukaza/tavo)
- [Documentation](https://github.com/jukaza/tavo/tree/main/docs)
- [Website](https://github.com/jukaza/tavo/tree/main/docs)
- [Blog](https://github.com/jukaza/tavo/tree/main/docs/blog)
