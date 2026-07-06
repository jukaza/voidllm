---
title: "RBAC"
description: "Role-based access control for the API key marketplace"
section: security
order: 1
---
# RBAC (Role-Based Access Control)

Tavo uses a simple two-role model for the API key reseller marketplace.

## Roles

| Role | Scope | Can do |
|---|---|---|
| `system_admin` | System-wide | Manage models, global aliases, provider connections, all users, system usage, and wallet operations. |
| `member` | Own account | Manage own API keys, view own usage, use the proxy with issued keys. |

`system_admin` satisfies any `member` requirement.

## Key Types

| Prefix | Type | Created by |
|---|---|---|
| `vl_uk_` | User API key | User or system admin |
| `vl_sk_` | Session key | System (on login, 24h TTL) |

## Limits

Rate limits and token budgets are configured **per API key**:

- Requests per minute / per day
- Daily and monthly token budgets

There is no org- or team-level inheritance. Each key enforces its own limits.

## Model Access

All active models and global aliases are available to authenticated keys. Model routing is controlled by the system admin through the model registry and alias configuration.

## User Onboarding

1. **Public signup** — `POST /api/v1/auth/register` creates a customer account, wallet, and first API key
2. **Manual creation** — system admin creates users from the admin UI
3. **Bootstrap** — first-run setup creates the system admin and wallet