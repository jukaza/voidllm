---
title: "Kubernetes Deployment"
description: "Deploy Tavo with Helm, Istio, and health probes"
section: deployment
order: 2
---
# Kubernetes Deployment (Helm)

## Basic Installation

```bash
helm repo add tavo https://jukaza.github.io/tavo
helm repo update

helm install tavo tavo/tavo \
  --set secrets.adminKey=$(openssl rand -base64 32) \
  --set secrets.encryptionKey=$(openssl rand -base64 32)
```

Check bootstrap credentials in the pod logs:

```bash
kubectl logs deploy/tavo | grep "BOOTSTRAP"
```

## With PostgreSQL

The Helm chart includes a Bitnami PostgreSQL subchart. When enabled, Tavo automatically switches from SQLite to PostgreSQL - no manual config needed.

```bash
helm install tavo tavo/tavo \
  --set postgresql.enabled=true \
  --set postgresql.auth.password=$(openssl rand -base64 16) \
  --set secrets.adminKey=$(openssl rand -base64 32) \
  --set secrets.encryptionKey=$(openssl rand -base64 32)
```

The password must be set explicitly - Tavo and the PostgreSQL subchart share this value to authenticate. Default username is `tavo`, default database is `tavo`.

Pod-to-pod traffic within the cluster is unencrypted (`sslmode=disable`). If you need encrypted database connections, use an external PostgreSQL with a custom DSN:

```bash
helm install tavo tavo/tavo \
  --set config.database.driver=postgres \
  --set config.database.dsn="postgres://user:pass@external-db:5432/tavo?sslmode=require"
```

## With Redis (Multi-Instance)

Redis enables distributed rate limiting and instant cache invalidation. Requires an Enterprise license. Without Redis, run only one replica.

```bash
helm install tavo tavo/tavo \
  --set postgresql.enabled=true \
  --set postgresql.auth.password=$(openssl rand -base64 16) \
  --set redis.enabled=true \
  --set replicaCount=3 \
  --set secrets.license="eyJ..." \
  --set secrets.adminKey=$(openssl rand -base64 32) \
  --set secrets.encryptionKey=$(openssl rand -base64 32)
```

Multi-instance requires both PostgreSQL (shared state) and Redis (distributed rate limiting + cache invalidation).

**Note:** Schema migrations currently run on every pod startup. With multiple replicas, pods may briefly race during rolling updates. PostgreSQL's transaction isolation prevents corruption, but you may see transient errors in logs. A dedicated migration hook is planned ([#48](https://github.com/jukaza/tavo/issues/48)).

## Enterprise Features

Enterprise features are disabled by default and must be explicitly enabled. Add these to your existing `helm install` or `helm upgrade`:

**Audit Logging:**
```bash
--set secrets.license="eyJ..." \
--set config.settings.audit.enabled=true
```

**OpenTelemetry Tracing:**
```bash
--set config.settings.otel.enabled=true \
--set config.settings.otel.endpoint=tempo:4317
```

See the full [values.yaml](https://github.com/jukaza/tavo/blob/main/chart/tavo/values.yaml) for all Helm configuration options.

## Istio Support

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

## Health Probes

```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 8080
  initialDelaySeconds: 5

readinessProbe:
  httpGet:
    path: /readyz
    port: 8080
  initialDelaySeconds: 5
```

| Endpoint | Purpose | Expected |
|---|---|---|
| `GET /healthz` | Liveness | 200 always |
| `GET /readyz` | Readiness | 200 (503 during graceful drain) |
| `GET /metrics` | Prometheus | Prometheus text format |

## Graceful Shutdown

Tavo supports phased graceful shutdown for zero-downtime deployments:

1. **SIGTERM received** - `/readyz` returns 503 (K8s stops routing new traffic)
2. **Drain period** (configurable, default 25s) - in-flight requests complete
3. **Force cancel** - remaining requests aborted if drain times out
4. **Cleanup** - flush usage/audit buffers, close DB, stop background tasks

```yaml
server:
  proxy:
    drain_timeout: 25s    # Must be less than K8s terminationGracePeriodSeconds
```
