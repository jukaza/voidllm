---
title: "Model Aliases"
description: "Global logical model names for clients"
section: models
order: 3
---
# Model Aliases

Aliases let clients use logical names like `default` or `fast` instead of specific model names. When you swap providers, clients don't need to change their code.

## How It Works

```yaml
models:
  - name: gpt-4o
    provider: openai
    base_url: https://api.openai.com/v1
    api_key: ${OPENAI_KEY}
    aliases: [default, smart]
```

A client sends `model: "default"` — VoidLLM resolves it to `gpt-4o` and routes accordingly. Later, if you switch to Claude, update the config and `default` now points to a different model. Zero client changes.

## Global Aliases

Aliases are **global** in the marketplace model. They can be set in:

- YAML config (`aliases` field on each model)
- Admin UI (Create/Edit Model)
- Admin API (`POST /api/v1/model-aliases`)

When a client sends `model: "default"`, VoidLLM looks up the global alias table first, then falls back to a model name match.

## API

```bash
curl -X POST https://voidllm.example.com/api/v1/model-aliases \
  -H "Authorization: Bearer vl_uk_..." \
  -H "Content-Type: application/json" \
  -d '{"alias": "default", "model_name": "claude-sonnet"}'
```

## Common Patterns

| Alias | Use case |
|---|---|
| `default` | General purpose, the model most people should use |
| `fast` | Low-latency, cheaper model for quick tasks |
| `smart` | High-capability model for complex reasoning |
| `embedding` | Text embedding model |
| `code` | Code-optimized model |