-- Migration: 0039_subscriptions.up.sql
-- Subscription packages, plans, user entitlements, and key bindings.

CREATE TABLE subscription_packages (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    cover_type TEXT NOT NULL DEFAULT 'default',
    cover_value TEXT NOT NULL DEFAULT '',
    enabled INTEGER NOT NULL DEFAULT 1,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (CURRENT_TIMESTAMP),
    updated_at TEXT NOT NULL DEFAULT (CURRENT_TIMESTAMP),
    deleted_at TEXT
);

CREATE TABLE subscription_plans (
    id TEXT PRIMARY KEY,
    package_id TEXT NOT NULL REFERENCES subscription_packages(id),
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    price REAL NOT NULL DEFAULT 0,
    validity_days INTEGER NOT NULL DEFAULT 30,
    max_concurrent_bindings INTEGER NOT NULL DEFAULT 0,
    daily_token_limit INTEGER NOT NULL DEFAULT 0,
    monthly_token_limit INTEGER NOT NULL DEFAULT 0,
    daily_request_limit INTEGER NOT NULL DEFAULT 0,
    monthly_request_limit INTEGER NOT NULL DEFAULT 0,
    requests_per_minute INTEGER NOT NULL DEFAULT 0,
    requests_per_day INTEGER NOT NULL DEFAULT 0,
    allowed_models TEXT NOT NULL DEFAULT '[]',
    quota_exceeded_policy TEXT NOT NULL DEFAULT 'fallback_wallet',
    for_sale INTEGER NOT NULL DEFAULT 1,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (CURRENT_TIMESTAMP),
    updated_at TEXT NOT NULL DEFAULT (CURRENT_TIMESTAMP),
    deleted_at TEXT
);

CREATE INDEX idx_subscription_plans_package ON subscription_plans (package_id);

CREATE TABLE user_subscriptions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    plan_id TEXT NOT NULL REFERENCES subscription_plans(id),
    status TEXT NOT NULL DEFAULT 'active',
    starts_at TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    order_id TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (CURRENT_TIMESTAMP),
    updated_at TEXT NOT NULL DEFAULT (CURRENT_TIMESTAMP)
);

CREATE INDEX idx_user_subscriptions_user ON user_subscriptions (user_id, status);
CREATE INDEX idx_user_subscriptions_plan ON user_subscriptions (plan_id, status);

CREATE TABLE key_subscription_bindings (
    id TEXT PRIMARY KEY,
    key_id TEXT NOT NULL,
    user_subscription_id TEXT NOT NULL REFERENCES user_subscriptions(id),
    status TEXT NOT NULL DEFAULT 'active',
    bound_at TEXT NOT NULL DEFAULT (CURRENT_TIMESTAMP),
    released_at TEXT
);

CREATE UNIQUE INDEX idx_key_sub_binding_active_key
    ON key_subscription_bindings (key_id)
    WHERE status = 'active';

CREATE INDEX idx_key_sub_binding_plan_lookup
    ON key_subscription_bindings (user_subscription_id, status);