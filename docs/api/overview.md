---
title: "API Reference"
description: "Authentication, endpoints, error codes, and Swagger UI"
section: api
order: 1
---
# API Reference

VoidLLM exposes two API surfaces: the **Proxy API** (OpenAI-compatible, for LLM requests) and the **Admin API** (for management).

## Authentication

All requests require an API key in the `Authorization` header:

```
Authorization: Bearer vl_uk_...
```

Key types: `vl_uk_` (user API key), `vl_sk_` (session key, 24h TTL).

## Proxy API (`/v1/*`)

The proxy forwards requests to upstream LLM providers. Any OpenAI-compatible endpoint works:

| Endpoint | Description |
|---|---|
| `POST /v1/chat/completions` | Chat completions (streaming supported) |
| `POST /v1/completions` | Text completions |
| `POST /v1/embeddings` | Text embeddings |
| `POST /v1/images/generations` | Image generation |
| `POST /v1/audio/transcriptions` | Audio transcription |
| `POST /v1/audio/speech` | Text to speech |
| `GET /v1/models` | List available models |

VoidLLM does not validate request bodies beyond extracting the `model` field. The upstream provider handles validation.

## Admin API (`/api/v1/*`)

Management endpoints for keys, models, usage, wallet, and users.

### Auth
| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/api/v1/auth/login` | Email/password login |
| `POST` | `/api/v1/auth/register` | Public customer signup |
| `GET` | `/api/v1/auth/me` | Current user profile |
| `GET` | `/api/v1/auth/providers` | Available login methods |

### API Keys
| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/api/v1/keys` | List keys |
| `POST` | `/api/v1/keys` | Create key |
| `DELETE` | `/api/v1/keys/:id` | Revoke key |
| `POST` | `/api/v1/keys/:id/rotate` | Rotate key (24h grace) |

### Models
| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/api/v1/models` | List models |
| `POST` | `/api/v1/models` | Create model |
| `PATCH` | `/api/v1/models/:id` | Update model |
| `DELETE` | `/api/v1/models/:id` | Delete model |
| `POST` | `/api/v1/models/:id/test` | Test upstream connectivity |

### Model Aliases
| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/api/v1/model-aliases` | List global aliases |
| `POST` | `/api/v1/model-aliases` | Create alias |
| `DELETE` | `/api/v1/model-aliases/:id` | Delete alias |

### Usage
| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/api/v1/usage` | System-wide usage (system admin) |
| `GET` | `/api/v1/usage/me` | Current user's usage |

### Users (system admin)
| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/api/v1/users` | List users |
| `POST` | `/api/v1/users` | Create user |
| `DELETE` | `/api/v1/users/:id` | Delete user |

## Health Endpoints

| Endpoint | Auth | Description |
|---|---|---|
| `GET /healthz` | No | Liveness probe (always 200) |
| `GET /readyz` | No | Readiness probe (503 during drain) |
| `GET /metrics` | No | Prometheus metrics |

## Swagger UI

When VoidLLM is running, the full OpenAPI spec is available at:

- **Swagger UI:** `http://localhost:8080/api/docs`
- **OpenAPI JSON:** `http://localhost:8080/api/docs/swagger.json`

A static copy of the spec is also available in the repository: [swagger.yaml](https://github.com/voidmind-io/voidllm/blob/main/docs/api/swagger.yaml)

## Error Format

All Admin API errors follow a consistent JSON envelope:

```json
{
  "error": "unauthorized",
  "message": "invalid API key"
}
```

Common HTTP status codes: `400` (bad request), `401` (unauthorized), `403` (forbidden), `404` (not found), `409` (conflict), `429` (rate limited), `500` (internal error).