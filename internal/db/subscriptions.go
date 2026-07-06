package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
)

// CreateSepaySubscriptionParams describes a pending SePay subscription purchase order.
type CreateSepaySubscriptionParams struct {
	UserID    string
	PlanID    string
	TradeNo   string
	PayAmount float64
	ExpiresAt string
}

// SubscriptionPackage is an admin-defined subscription product family.
type SubscriptionPackage struct {
	ID          string
	Name        string
	Description string
	CoverType   string
	CoverValue  string
	Enabled     bool
	SortOrder   int
	CreatedAt   string
	UpdatedAt   string
	DeletedAt   *string
}

// SubscriptionPlan is a quota/pricing tier within a package.
type SubscriptionPlan struct {
	ID                      string
	PackageID               string
	Name                    string
	Description             string
	Price                   float64
	ValidityDays            int
	MaxConcurrentBindings   int
	DailyTokenLimit         int64
	MonthlyTokenLimit       int64
	DailyRequestLimit       int
	MonthlyRequestLimit     int
	RequestsPerMinute       int
	RequestsPerDay          int
	AllowedModels           string
	QuotaExceededPolicy     string
	ForSale                 bool
	SortOrder               int
	CreatedAt               string
	UpdatedAt               string
	DeletedAt               *string
}

// UserSubscription is an active entitlement for a user.
type UserSubscription struct {
	ID        string
	UserID    string
	PlanID    string
	Status    string
	StartsAt  string
	ExpiresAt string
	OrderID   string
	CreatedAt string
	UpdatedAt string
}

// KeySubscriptionBinding links an API key to a user subscription.
type KeySubscriptionBinding struct {
	ID                 string
	KeyID              string
	UserSubscriptionID string
	Status             string
	BoundAt            string
	ReleasedAt         *string
}

// KeySubscriptionContext is the resolved binding + plan for proxy hot path.
type KeySubscriptionContext struct {
	BindingID          string
	UserSubscriptionID string
	PlanID             string
	UserID             string
	AllowedModels      string
	DailyTokenLimit    int64
	MonthlyTokenLimit  int64
	DailyRequestLimit  int
	MonthlyRequestLimit int
	RequestsPerMinute  int
	RequestsPerDay     int
	QuotaExceededPolicy string
	ExpiresAt          time.Time
}

// CreateSubscriptionPackageParams holds input for creating a package.
type CreateSubscriptionPackageParams struct {
	Name        string
	Description string
	CoverType   string
	CoverValue  string
	Enabled     bool
	SortOrder   int
}

// UpdateSubscriptionPackageParams holds optional package update fields.
type UpdateSubscriptionPackageParams struct {
	Name        *string
	Description *string
	CoverType   *string
	CoverValue  *string
	Enabled     *bool
	SortOrder   *int
}

// CreateSubscriptionPlanParams holds input for creating a plan.
type CreateSubscriptionPlanParams struct {
	PackageID             string
	Name                  string
	Description           string
	Price                 float64
	ValidityDays          int
	MaxConcurrentBindings int
	DailyTokenLimit       int64
	MonthlyTokenLimit     int64
	DailyRequestLimit     int
	MonthlyRequestLimit   int
	RequestsPerMinute     int
	RequestsPerDay        int
	AllowedModels         string
	QuotaExceededPolicy   string
	ForSale               bool
	SortOrder             int
}

// UpdateSubscriptionPlanParams holds optional plan update fields.
type UpdateSubscriptionPlanParams struct {
	Name                  *string
	Description           *string
	Price                 *float64
	ValidityDays          *int
	MaxConcurrentBindings *int
	DailyTokenLimit       *int64
	MonthlyTokenLimit     *int64
	DailyRequestLimit     *int
	MonthlyRequestLimit   *int
	RequestsPerMinute     *int
	RequestsPerDay        *int
	AllowedModels         *string
	QuotaExceededPolicy   *string
	ForSale               *bool
	SortOrder             *int
}

// CreateUserSubscriptionParams holds input for granting a subscription.
type CreateUserSubscriptionParams struct {
	UserID    string
	PlanID    string
	StartsAt  string
	ExpiresAt string
	OrderID   string
}

var ErrSubscriptionSlotsFull = errors.New("subscription plan slots full")

func (d *DB) ListSubscriptionPackages(ctx context.Context, includeDisabled bool) ([]SubscriptionPackage, error) {
	q := "SELECT id, name, description, cover_type, cover_value, enabled, sort_order, created_at, updated_at, deleted_at " +
		"FROM subscription_packages WHERE deleted_at IS NULL"
	if !includeDisabled {
		q += " AND enabled = 1"
	}
	q += " ORDER BY sort_order ASC, name ASC"

	rows, err := d.sql.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("list subscription packages: %w", translateError(err))
	}
	defer rows.Close()

	var out []SubscriptionPackage
	for rows.Next() {
		var pkg SubscriptionPackage
		var enabled int
		if err := rows.Scan(
			&pkg.ID, &pkg.Name, &pkg.Description, &pkg.CoverType, &pkg.CoverValue,
			&enabled, &pkg.SortOrder, &pkg.CreatedAt, &pkg.UpdatedAt, &pkg.DeletedAt,
		); err != nil {
			return nil, fmt.Errorf("scan subscription package: %w", err)
		}
		pkg.Enabled = enabled == 1
		out = append(out, pkg)
	}
	return out, rows.Err()
}

func (d *DB) GetSubscriptionPackage(ctx context.Context, id string) (*SubscriptionPackage, error) {
	p := d.dialect.Placeholder
	var pkg SubscriptionPackage
	var enabled int
	err := d.sql.QueryRowContext(ctx,
		"SELECT id, name, description, cover_type, cover_value, enabled, sort_order, created_at, updated_at, deleted_at "+
			"FROM subscription_packages WHERE id = "+p(1)+" AND deleted_at IS NULL",
		id,
	).Scan(
		&pkg.ID, &pkg.Name, &pkg.Description, &pkg.CoverType, &pkg.CoverValue,
		&enabled, &pkg.SortOrder, &pkg.CreatedAt, &pkg.UpdatedAt, &pkg.DeletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get subscription package %s: %w", id, translateError(err))
	}
	pkg.Enabled = enabled == 1
	return &pkg, nil
}

func (d *DB) CreateSubscriptionPackage(ctx context.Context, params CreateSubscriptionPackageParams) (*SubscriptionPackage, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("create subscription package: %w", err)
	}
	p := d.dialect.Placeholder
	enabled := 0
	if params.Enabled {
		enabled = 1
	}
	coverType := params.CoverType
	if coverType == "" {
		coverType = "default"
	}
	_, err = d.sql.ExecContext(ctx,
		"INSERT INTO subscription_packages (id, name, description, cover_type, cover_value, enabled, sort_order, created_at, updated_at) "+
			"VALUES ("+p(1)+", "+p(2)+", "+p(3)+", "+p(4)+", "+p(5)+", "+p(6)+", "+p(7)+", CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)",
		id.String(), params.Name, params.Description, coverType, params.CoverValue, enabled, params.SortOrder,
	)
	if err != nil {
		return nil, fmt.Errorf("insert subscription package: %w", translateError(err))
	}
	return d.GetSubscriptionPackage(ctx, id.String())
}

func (d *DB) UpdateSubscriptionPackage(ctx context.Context, id string, params UpdateSubscriptionPackageParams) (*SubscriptionPackage, error) {
	setClauses := []string{"updated_at = CURRENT_TIMESTAMP"}
	args := []any{}
	argN := 1
	p := d.dialect.Placeholder

	add := func(col string, val any) {
		setClauses = append(setClauses, col+" = "+p(argN))
		args = append(args, val)
		argN++
	}

	if params.Name != nil {
		add("name", *params.Name)
	}
	if params.Description != nil {
		add("description", *params.Description)
	}
	if params.CoverType != nil {
		add("cover_type", *params.CoverType)
	}
	if params.CoverValue != nil {
		add("cover_value", *params.CoverValue)
	}
	if params.Enabled != nil {
		v := 0
		if *params.Enabled {
			v = 1
		}
		add("enabled", v)
	}
	if params.SortOrder != nil {
		add("sort_order", *params.SortOrder)
	}

	if len(setClauses) == 1 {
		return d.GetSubscriptionPackage(ctx, id)
	}

	args = append(args, id)
	query := "UPDATE subscription_packages SET " + strings.Join(setClauses, ", ") +
		" WHERE id = " + p(argN) + " AND deleted_at IS NULL"
	result, err := d.sql.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("update subscription package: %w", translateError(err))
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return nil, ErrNotFound
	}
	return d.GetSubscriptionPackage(ctx, id)
}

func (d *DB) DeleteSubscriptionPackage(ctx context.Context, id string) error {
	p := d.dialect.Placeholder
	result, err := d.sql.ExecContext(ctx,
		"UPDATE subscription_packages SET deleted_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP, enabled = 0 "+
			"WHERE id = "+p(1)+" AND deleted_at IS NULL",
		id,
	)
	if err != nil {
		return fmt.Errorf("delete subscription package: %w", translateError(err))
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (d *DB) ListSubscriptionPlans(ctx context.Context, packageID string) ([]SubscriptionPlan, error) {
	p := d.dialect.Placeholder
	q := "SELECT id, package_id, name, description, price, validity_days, max_concurrent_bindings, " +
		"daily_token_limit, monthly_token_limit, daily_request_limit, monthly_request_limit, " +
		"requests_per_minute, requests_per_day, allowed_models, quota_exceeded_policy, for_sale, sort_order, " +
		"created_at, updated_at, deleted_at FROM subscription_plans WHERE deleted_at IS NULL"
	args := []any{}
	if packageID != "" {
		q += " AND package_id = " + p(1)
		args = append(args, packageID)
	}
	q += " ORDER BY sort_order ASC, name ASC"

	rows, err := d.sql.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list subscription plans: %w", translateError(err))
	}
	defer rows.Close()
	return scanSubscriptionPlans(rows)
}

func scanSubscriptionPlan(row interface {
	Scan(dest ...any) error
}) (*SubscriptionPlan, error) {
	var plan SubscriptionPlan
	var forSale int
	if err := row.Scan(
		&plan.ID, &plan.PackageID, &plan.Name, &plan.Description, &plan.Price, &plan.ValidityDays,
		&plan.MaxConcurrentBindings,
		&plan.DailyTokenLimit, &plan.MonthlyTokenLimit,
		&plan.DailyRequestLimit, &plan.MonthlyRequestLimit,
		&plan.RequestsPerMinute, &plan.RequestsPerDay,
		&plan.AllowedModels, &plan.QuotaExceededPolicy, &forSale, &plan.SortOrder,
		&plan.CreatedAt, &plan.UpdatedAt, &plan.DeletedAt,
	); err != nil {
		return nil, err
	}
	plan.ForSale = forSale == 1
	return &plan, nil
}

func scanSubscriptionPlans(rows *sql.Rows) ([]SubscriptionPlan, error) {
	var out []SubscriptionPlan
	for rows.Next() {
		plan, err := scanSubscriptionPlan(rows)
		if err != nil {
			return nil, fmt.Errorf("scan subscription plan: %w", err)
		}
		out = append(out, *plan)
	}
	return out, rows.Err()
}

func (d *DB) GetSubscriptionPlan(ctx context.Context, id string) (*SubscriptionPlan, error) {
	p := d.dialect.Placeholder
	row := d.sql.QueryRowContext(ctx,
		"SELECT id, package_id, name, description, price, validity_days, max_concurrent_bindings, "+
			"daily_token_limit, monthly_token_limit, daily_request_limit, monthly_request_limit, "+
			"requests_per_minute, requests_per_day, allowed_models, quota_exceeded_policy, for_sale, sort_order, "+
			"created_at, updated_at, deleted_at FROM subscription_plans WHERE id = "+p(1)+" AND deleted_at IS NULL",
		id,
	)
	plan, err := scanSubscriptionPlan(row)
	if err != nil {
		return nil, fmt.Errorf("get subscription plan %s: %w", id, translateError(err))
	}
	return plan, nil
}

func (d *DB) CreateSubscriptionPlan(ctx context.Context, params CreateSubscriptionPlanParams) (*SubscriptionPlan, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	p := d.dialect.Placeholder
	forSale := 0
	if params.ForSale {
		forSale = 1
	}
	policy := params.QuotaExceededPolicy
	if policy == "" {
		policy = "fallback_wallet"
	}
	models := params.AllowedModels
	if models == "" {
		models = "[]"
	}
	_, err = d.sql.ExecContext(ctx,
		"INSERT INTO subscription_plans (id, package_id, name, description, price, validity_days, max_concurrent_bindings, "+
			"daily_token_limit, monthly_token_limit, daily_request_limit, monthly_request_limit, "+
			"requests_per_minute, requests_per_day, allowed_models, quota_exceeded_policy, for_sale, sort_order, created_at, updated_at) "+
			"VALUES ("+p(1)+", "+p(2)+", "+p(3)+", "+p(4)+", "+p(5)+", "+p(6)+", "+p(7)+", "+
			p(8)+", "+p(9)+", "+p(10)+", "+p(11)+", "+p(12)+", "+p(13)+", "+p(14)+", "+p(15)+", "+p(16)+", "+p(17)+
			", CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)",
		id.String(), params.PackageID, params.Name, params.Description, params.Price, params.ValidityDays,
		params.MaxConcurrentBindings,
		params.DailyTokenLimit, params.MonthlyTokenLimit,
		params.DailyRequestLimit, params.MonthlyRequestLimit,
		params.RequestsPerMinute, params.RequestsPerDay,
		models, policy, forSale, params.SortOrder,
	)
	if err != nil {
		return nil, fmt.Errorf("insert subscription plan: %w", translateError(err))
	}
	return d.GetSubscriptionPlan(ctx, id.String())
}

func (d *DB) UpdateSubscriptionPlan(ctx context.Context, id string, params UpdateSubscriptionPlanParams) (*SubscriptionPlan, error) {
	setClauses := []string{"updated_at = CURRENT_TIMESTAMP"}
	args := []any{}
	argN := 1
	p := d.dialect.Placeholder
	add := func(col string, val any) {
		setClauses = append(setClauses, col+" = "+p(argN))
		args = append(args, val)
		argN++
	}

	if params.Name != nil {
		add("name", *params.Name)
	}
	if params.Description != nil {
		add("description", *params.Description)
	}
	if params.Price != nil {
		add("price", *params.Price)
	}
	if params.ValidityDays != nil {
		add("validity_days", *params.ValidityDays)
	}
	if params.MaxConcurrentBindings != nil {
		add("max_concurrent_bindings", *params.MaxConcurrentBindings)
	}
	if params.DailyTokenLimit != nil {
		add("daily_token_limit", *params.DailyTokenLimit)
	}
	if params.MonthlyTokenLimit != nil {
		add("monthly_token_limit", *params.MonthlyTokenLimit)
	}
	if params.DailyRequestLimit != nil {
		add("daily_request_limit", *params.DailyRequestLimit)
	}
	if params.MonthlyRequestLimit != nil {
		add("monthly_request_limit", *params.MonthlyRequestLimit)
	}
	if params.RequestsPerMinute != nil {
		add("requests_per_minute", *params.RequestsPerMinute)
	}
	if params.RequestsPerDay != nil {
		add("requests_per_day", *params.RequestsPerDay)
	}
	if params.AllowedModels != nil {
		add("allowed_models", *params.AllowedModels)
	}
	if params.QuotaExceededPolicy != nil {
		add("quota_exceeded_policy", *params.QuotaExceededPolicy)
	}
	if params.ForSale != nil {
		v := 0
		if *params.ForSale {
			v = 1
		}
		add("for_sale", v)
	}
	if params.SortOrder != nil {
		add("sort_order", *params.SortOrder)
	}

	if len(setClauses) == 1 {
		return d.GetSubscriptionPlan(ctx, id)
	}
	args = append(args, id)
	query := "UPDATE subscription_plans SET " + strings.Join(setClauses, ", ") +
		" WHERE id = " + p(argN) + " AND deleted_at IS NULL"
	result, err := d.sql.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("update subscription plan: %w", translateError(err))
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return nil, ErrNotFound
	}
	return d.GetSubscriptionPlan(ctx, id)
}

func (d *DB) DeleteSubscriptionPlan(ctx context.Context, id string) error {
	p := d.dialect.Placeholder
	result, err := d.sql.ExecContext(ctx,
		"UPDATE subscription_plans SET deleted_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP, for_sale = 0 "+
			"WHERE id = "+p(1)+" AND deleted_at IS NULL",
		id,
	)
	if err != nil {
		return fmt.Errorf("delete subscription plan: %w", translateError(err))
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (d *DB) CountActivePlanBindings(ctx context.Context, planID string) (int, error) {
	p := d.dialect.Placeholder
	now := time.Now().UTC().Format(time.RFC3339)
	var count int
	err := d.sql.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM key_subscription_bindings b "+
			"JOIN user_subscriptions u ON u.id = b.user_subscription_id "+
			"WHERE u.plan_id = "+p(1)+" AND b.status = 'active' AND u.status = 'active' AND u.expires_at > "+p(2),
		planID, now,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count plan bindings: %w", translateError(err))
	}
	return count, nil
}

func (d *DB) ListUserSubscriptions(ctx context.Context, userID string, activeOnly bool) ([]UserSubscription, error) {
	p := d.dialect.Placeholder
	q := "SELECT id, user_id, plan_id, status, starts_at, expires_at, order_id, created_at, updated_at " +
		"FROM user_subscriptions WHERE user_id = " + p(1)
	args := []any{userID}
	if activeOnly {
		q += " AND status = 'active' AND expires_at > " + p(2)
		args = append(args, time.Now().UTC().Format(time.RFC3339))
	}
	q += " ORDER BY expires_at DESC"

	rows, err := d.sql.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list user subscriptions: %w", translateError(err))
	}
	defer rows.Close()

	var out []UserSubscription
	for rows.Next() {
		var us UserSubscription
		if err := rows.Scan(
			&us.ID, &us.UserID, &us.PlanID, &us.Status, &us.StartsAt, &us.ExpiresAt, &us.OrderID, &us.CreatedAt, &us.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan user subscription: %w", err)
		}
		out = append(out, us)
	}
	return out, rows.Err()
}

func (d *DB) GetUserSubscription(ctx context.Context, id string) (*UserSubscription, error) {
	p := d.dialect.Placeholder
	var us UserSubscription
	err := d.sql.QueryRowContext(ctx,
		"SELECT id, user_id, plan_id, status, starts_at, expires_at, order_id, created_at, updated_at "+
			"FROM user_subscriptions WHERE id = "+p(1),
		id,
	).Scan(
		&us.ID, &us.UserID, &us.PlanID, &us.Status, &us.StartsAt, &us.ExpiresAt, &us.OrderID, &us.CreatedAt, &us.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get user subscription %s: %w", id, translateError(err))
	}
	return &us, nil
}

func (d *DB) CreateUserSubscription(ctx context.Context, params CreateUserSubscriptionParams) (*UserSubscription, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	p := d.dialect.Placeholder
	_, err = d.sql.ExecContext(ctx,
		"INSERT INTO user_subscriptions (id, user_id, plan_id, status, starts_at, expires_at, order_id, created_at, updated_at) "+
			"VALUES ("+p(1)+", "+p(2)+", "+p(3)+", 'active', "+p(4)+", "+p(5)+", "+p(6)+", CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)",
		id.String(), params.UserID, params.PlanID, params.StartsAt, params.ExpiresAt, params.OrderID,
	)
	if err != nil {
		return nil, fmt.Errorf("insert user subscription: %w", translateError(err))
	}
	return d.GetUserSubscription(ctx, id.String())
}

func (d *DB) UpdateUserSubscriptionStatus(ctx context.Context, id, status string) (*UserSubscription, error) {
	p := d.dialect.Placeholder
	result, err := d.sql.ExecContext(ctx,
		"UPDATE user_subscriptions SET status = "+p(1)+", updated_at = CURRENT_TIMESTAMP WHERE id = "+p(2),
		status, id,
	)
	if err != nil {
		return nil, fmt.Errorf("update user subscription status: %w", translateError(err))
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return nil, ErrNotFound
	}
	if status != "active" {
		_, _ = d.sql.ExecContext(ctx,
			"UPDATE key_subscription_bindings SET status = 'released', released_at = CURRENT_TIMESTAMP "+
				"WHERE user_subscription_id = "+p(1)+" AND status = 'active'",
			id,
		)
	}
	return d.GetUserSubscription(ctx, id)
}

func (d *DB) GetKeySubscriptionBinding(ctx context.Context, keyID string) (*KeySubscriptionBinding, error) {
	p := d.dialect.Placeholder
	var b KeySubscriptionBinding
	err := d.sql.QueryRowContext(ctx,
		"SELECT id, key_id, user_subscription_id, status, bound_at, released_at "+
			"FROM key_subscription_bindings WHERE key_id = "+p(1)+" AND status = 'active'",
		keyID,
	).Scan(&b.ID, &b.KeyID, &b.UserSubscriptionID, &b.Status, &b.BoundAt, &b.ReleasedAt)
	if err != nil {
		return nil, fmt.Errorf("get key subscription binding: %w", translateError(err))
	}
	return &b, nil
}

func (d *DB) LoadAllActiveKeySubscriptionContexts(ctx context.Context) (map[string]KeySubscriptionContext, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	p := d.dialect.Placeholder
	rows, err := d.sql.QueryContext(ctx,
		"SELECT b.id, b.key_id, u.id, u.plan_id, u.user_id, p.allowed_models, "+
			"p.daily_token_limit, p.monthly_token_limit, p.daily_request_limit, p.monthly_request_limit, "+
			"p.requests_per_minute, p.requests_per_day, p.quota_exceeded_policy, u.expires_at "+
			"FROM key_subscription_bindings b "+
			"JOIN user_subscriptions u ON u.id = b.user_subscription_id "+
			"JOIN subscription_plans p ON p.id = u.plan_id "+
			"WHERE b.status = 'active' AND u.status = 'active' AND u.expires_at > "+p(1)+" AND p.deleted_at IS NULL",
		now,
	)
	if err != nil {
		return nil, fmt.Errorf("load key subscription contexts: %w", translateError(err))
	}
	defer rows.Close()

	out := make(map[string]KeySubscriptionContext)
	for rows.Next() {
		var ctx KeySubscriptionContext
		var keyID, expiresAt string
		if err := rows.Scan(
			&ctx.BindingID, &keyID, &ctx.UserSubscriptionID, &ctx.PlanID, &ctx.UserID,
			&ctx.AllowedModels,
			&ctx.DailyTokenLimit, &ctx.MonthlyTokenLimit,
			&ctx.DailyRequestLimit, &ctx.MonthlyRequestLimit,
			&ctx.RequestsPerMinute, &ctx.RequestsPerDay,
			&ctx.QuotaExceededPolicy, &expiresAt,
		); err != nil {
			return nil, fmt.Errorf("scan key subscription context: %w", err)
		}
		if t, err := time.Parse(time.RFC3339, expiresAt); err == nil {
			ctx.ExpiresAt = t
		}
		out[keyID] = ctx
	}
	return out, rows.Err()
}

func (d *DB) BindKeySubscription(ctx context.Context, keyID, userSubscriptionID, userID string) (*KeySubscriptionBinding, error) {
	tx, err := d.sql.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback() //nolint:errcheck

	p := d.dialect.Placeholder

	var subUserID, planID, subStatus, expiresAt string
	err = tx.QueryRowContext(ctx,
		"SELECT user_id, plan_id, status, expires_at FROM user_subscriptions WHERE id = "+p(1),
		userSubscriptionID,
	).Scan(&subUserID, &planID, &subStatus, &expiresAt)
	if err != nil {
		return nil, fmt.Errorf("bind: load subscription: %w", translateError(err))
	}
	if subUserID != userID {
		return nil, ErrNotFound
	}
	if subStatus != "active" {
		return nil, fmt.Errorf("subscription is not active")
	}
	if expiresAt <= time.Now().UTC().Format(time.RFC3339) {
		return nil, fmt.Errorf("subscription has expired")
	}

	var keyUserID sql.NullString
	err = tx.QueryRowContext(ctx,
		"SELECT user_id FROM api_keys WHERE id = "+p(1)+" AND deleted_at IS NULL",
		keyID,
	).Scan(&keyUserID)
	if err != nil {
		return nil, fmt.Errorf("bind: load key: %w", translateError(err))
	}
	if !keyUserID.Valid || keyUserID.String != userID {
		return nil, ErrNotFound
	}

	var existing int
	_ = tx.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM key_subscription_bindings WHERE key_id = "+p(1)+" AND status = 'active'",
		keyID,
	).Scan(&existing)
	if existing > 0 {
		return nil, fmt.Errorf("key already has an active subscription binding")
	}

	var maxSlots int
	err = tx.QueryRowContext(ctx,
		"SELECT max_concurrent_bindings FROM subscription_plans WHERE id = "+p(1)+" AND deleted_at IS NULL",
		planID,
	).Scan(&maxSlots)
	if err != nil {
		return nil, fmt.Errorf("bind: load plan: %w", translateError(err))
	}
	if maxSlots > 0 {
		var used int
		err = tx.QueryRowContext(ctx,
			"SELECT COUNT(*) FROM key_subscription_bindings b "+
				"JOIN user_subscriptions u ON u.id = b.user_subscription_id "+
				"WHERE u.plan_id = "+p(1)+" AND b.status = 'active' AND u.status = 'active' AND u.expires_at > "+p(2),
			planID, time.Now().UTC().Format(time.RFC3339),
		).Scan(&used)
		if err != nil {
			return nil, err
		}
		if used >= maxSlots {
			return nil, ErrSubscriptionSlotsFull
		}
	}

	bindingID, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	_, err = tx.ExecContext(ctx,
		"INSERT INTO key_subscription_bindings (id, key_id, user_subscription_id, status, bound_at) "+
			"VALUES ("+p(1)+", "+p(2)+", "+p(3)+", 'active', CURRENT_TIMESTAMP)",
		bindingID.String(), keyID, userSubscriptionID,
	)
	if err != nil {
		return nil, fmt.Errorf("bind: insert: %w", translateError(err))
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return d.GetKeySubscriptionBinding(ctx, keyID)
}

func (d *DB) ReleaseKeySubscriptionBinding(ctx context.Context, keyID, userID string) error {
	p := d.dialect.Placeholder
	var subUserID string
	err := d.sql.QueryRowContext(ctx,
		"SELECT u.user_id FROM key_subscription_bindings b "+
			"JOIN user_subscriptions u ON u.id = b.user_subscription_id "+
			"WHERE b.key_id = "+p(1)+" AND b.status = 'active'",
		keyID,
	).Scan(&subUserID)
	if err != nil {
		return fmt.Errorf("release binding: %w", translateError(err))
	}
	if subUserID != userID {
		return ErrNotFound
	}
	result, err := d.sql.ExecContext(ctx,
		"UPDATE key_subscription_bindings SET status = 'released', released_at = CURRENT_TIMESTAMP "+
			"WHERE key_id = "+p(1)+" AND status = 'active'",
		keyID,
	)
	if err != nil {
		return fmt.Errorf("release binding: %w", translateError(err))
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (d *DB) ResolveKeySubscriptionContext(ctx context.Context, keyID string) (*KeySubscriptionContext, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	p := d.dialect.Placeholder
	var subCtx KeySubscriptionContext
	var expiresAt string
	err := d.sql.QueryRowContext(ctx,
		"SELECT b.id, u.id, u.plan_id, u.user_id, p.allowed_models, "+
			"p.daily_token_limit, p.monthly_token_limit, p.daily_request_limit, p.monthly_request_limit, "+
			"p.requests_per_minute, p.requests_per_day, p.quota_exceeded_policy, u.expires_at "+
			"FROM key_subscription_bindings b "+
			"JOIN user_subscriptions u ON u.id = b.user_subscription_id "+
			"JOIN subscription_plans p ON p.id = u.plan_id "+
			"WHERE b.key_id = "+p(1)+" AND b.status = 'active' AND u.status = 'active' AND u.expires_at > "+p(2)+" AND p.deleted_at IS NULL",
		keyID, now,
	).Scan(
		&subCtx.BindingID, &subCtx.UserSubscriptionID, &subCtx.PlanID, &subCtx.UserID,
		&subCtx.AllowedModels,
		&subCtx.DailyTokenLimit, &subCtx.MonthlyTokenLimit,
		&subCtx.DailyRequestLimit, &subCtx.MonthlyRequestLimit,
		&subCtx.RequestsPerMinute, &subCtx.RequestsPerDay,
		&subCtx.QuotaExceededPolicy, &expiresAt,
	)
	if err != nil {
		return nil, fmt.Errorf("resolve key subscription: %w", translateError(err))
	}
	if t, err := time.Parse(time.RFC3339, expiresAt); err == nil {
		subCtx.ExpiresAt = t
	}
	return &subCtx, nil
}

// CreateSepaySubscriptionOrder records a pending SePay subscription purchase.
func (d *DB) CreateSepaySubscriptionOrder(ctx context.Context, params CreateSepaySubscriptionParams) (*TopupRequest, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	p := d.dialect.Placeholder
	insertQuery := "INSERT INTO topup_requests (id, user_id, amount, payment_ref, status, created_at, " +
		"trade_no, pay_amount, credit_amount, bonus_amount, bonus_detail, expires_at, order_kind, plan_id) " +
		"VALUES (" + p(1) + ", " + p(2) + ", " + p(3) + ", " + p(4) + ", 'pending', CURRENT_TIMESTAMP, " +
		p(5) + ", " + p(6) + ", 0, 0, '', " + p(7) + ", 'subscription', " + p(8) + ")"
	selectQuery := "SELECT " + topupSelectColumns + " FROM topup_requests WHERE id = " + p(1)

	var tr *TopupRequest
	err = d.WithTx(ctx, func(q Querier) error {
		if _, execErr := q.ExecContext(ctx, insertQuery,
			id.String(), params.UserID, params.PayAmount, params.TradeNo,
			params.TradeNo, params.PayAmount, params.ExpiresAt, params.PlanID,
		); execErr != nil {
			return translateError(execErr)
		}
		row := q.QueryRowContext(ctx, selectQuery, id.String())
		var scanErr error
		tr, scanErr = scanTopupRequest(row)
		return scanErr
	})
	if err != nil {
		return nil, fmt.Errorf("create sepay subscription order: %w", err)
	}
	return tr, nil
}

// CompleteSepaySubscription marks the order completed and grants user subscription.
func (d *DB) CompleteSepaySubscription(ctx context.Context, tradeNo, sepayTxID string, paidAmount float64) (*UserSubscription, error) {
	p := d.dialect.Placeholder
	lockQuery := "SELECT id, user_id, plan_id, pay_amount, status FROM topup_requests WHERE trade_no = " + p(1) +
		" AND order_kind = 'subscription'"
	updateQuery := "UPDATE topup_requests SET status = 'completed', sepay_tx_id = " + p(1) +
		", completed_at = CURRENT_TIMESTAMP WHERE trade_no = " + p(2) +
		" AND status IN ('pending', 'expired') AND order_kind = 'subscription'"

	var us *UserSubscription
	err := d.WithTx(ctx, func(q Querier) error {
		var requestID, userID, planID, status string
		var payAmount float64
		if scanErr := q.QueryRowContext(ctx, lockQuery, tradeNo).Scan(&requestID, &userID, &planID, &payAmount, &status); scanErr != nil {
			return translateError(scanErr)
		}
		if status != "pending" && status != "expired" {
			return ErrNotFound
		}
		if math.Abs(paidAmount-payAmount) > 1.0 {
			return ErrAmountMismatch
		}
		plan, planErr := d.getSubscriptionPlanInTx(ctx, q, planID)
		if planErr != nil {
			return planErr
		}

		res, execErr := q.ExecContext(ctx, updateQuery, sepayTxID, tradeNo)
		if execErr != nil {
			return translateError(execErr)
		}
		affected, raErr := res.RowsAffected()
		if raErr != nil {
			return raErr
		}
		if affected == 0 {
			return ErrNotFound
		}

		now := time.Now().UTC()
		expires := now.Add(time.Duration(plan.ValidityDays) * 24 * time.Hour)
		subID, idErr := uuid.NewV7()
		if idErr != nil {
			return idErr
		}
		_, execErr = q.ExecContext(ctx,
			"INSERT INTO user_subscriptions (id, user_id, plan_id, status, starts_at, expires_at, order_id, created_at, updated_at) "+
				"VALUES ("+p(1)+", "+p(2)+", "+p(3)+", 'active', "+p(4)+", "+p(5)+", "+p(6)+", CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)",
			subID.String(), userID, planID, now.Format(time.RFC3339), expires.Format(time.RFC3339), requestID,
		)
		if execErr != nil {
			return translateError(execErr)
		}
		row := q.QueryRowContext(ctx,
			"SELECT id, user_id, plan_id, status, starts_at, expires_at, order_id, created_at, updated_at "+
				"FROM user_subscriptions WHERE id = "+p(1),
			subID.String(),
		)
		var created UserSubscription
		if scanErr := row.Scan(
			&created.ID, &created.UserID, &created.PlanID, &created.Status,
			&created.StartsAt, &created.ExpiresAt, &created.OrderID, &created.CreatedAt, &created.UpdatedAt,
		); scanErr != nil {
			return scanErr
		}
		us = &created
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("CompleteSepaySubscription %s: %w", tradeNo, err)
	}
	return us, nil
}

func (d *DB) getSubscriptionPlanInTx(ctx context.Context, q Querier, planID string) (*SubscriptionPlan, error) {
	p := d.dialect.Placeholder
	row := q.QueryRowContext(ctx,
		"SELECT id, package_id, name, description, price, validity_days, max_concurrent_bindings, "+
			"daily_token_limit, monthly_token_limit, daily_request_limit, monthly_request_limit, "+
			"requests_per_minute, requests_per_day, allowed_models, quota_exceeded_policy, for_sale, sort_order, "+
			"created_at, updated_at, deleted_at FROM subscription_plans WHERE id = "+p(1)+" AND deleted_at IS NULL AND for_sale = 1",
		planID,
	)
	plan, err := scanSubscriptionPlan(row)
	if err != nil {
		return nil, fmt.Errorf("subscription plan %s: %w", planID, translateError(err))
	}
	return plan, nil
}

// ListPublicSubscriptionCatalog returns enabled packages with for-sale plans.
func (d *DB) ListPublicSubscriptionCatalog(ctx context.Context) ([]SubscriptionPackage, map[string][]SubscriptionPlan, error) {
	pkgs, err := d.ListSubscriptionPackages(ctx, false)
	if err != nil {
		return nil, nil, err
	}
	plans, err := d.ListSubscriptionPlans(ctx, "")
	if err != nil {
		return nil, nil, err
	}
	byPackage := make(map[string][]SubscriptionPlan)
	for i := range plans {
		if !plans[i].ForSale {
			continue
		}
		byPackage[plans[i].PackageID] = append(byPackage[plans[i].PackageID], plans[i])
	}
	var out []SubscriptionPackage
	for _, pkg := range pkgs {
		if len(byPackage[pkg.ID]) > 0 {
			out = append(out, pkg)
		}
	}
	return out, byPackage, nil
}

// PurchaseSubscriptionWithWallet debits the user's wallet and grants an active subscription.
func (d *DB) PurchaseSubscriptionWithWallet(ctx context.Context, userID, planID string) (*UserSubscription, float64, error) {
	p := d.dialect.Placeholder
	var us *UserSubscription
	var newBalance float64

	err := d.WithTx(ctx, func(q Querier) error {
		plan, planErr := d.getSubscriptionPlanInTx(ctx, q, planID)
		if planErr != nil {
			return planErr
		}
		var pkgEnabled bool
		if scanErr := q.QueryRowContext(ctx,
			"SELECT enabled FROM subscription_packages WHERE id = "+p(1)+" AND deleted_at IS NULL",
			plan.PackageID,
		).Scan(&pkgEnabled); scanErr != nil {
			return translateError(scanErr)
		}
		if !pkgEnabled {
			return ErrNotFound
		}
		price := plan.Price
		if price <= 0 {
			return fmt.Errorf("plan has no price")
		}

		debitQuery := "UPDATE wallets SET balance = balance - " + p(1) + ", updated_at = CURRENT_TIMESTAMP" +
			" WHERE user_id = " + p(2) + " AND balance >= " + p(3)
		res, execErr := q.ExecContext(ctx, debitQuery, price, userID, price)
		if execErr != nil {
			return translateError(execErr)
		}
		affected, raErr := res.RowsAffected()
		if raErr != nil {
			return raErr
		}
		if affected == 0 {
			return ErrInsufficientBalance
		}
		if scanErr := q.QueryRowContext(ctx, "SELECT balance FROM wallets WHERE user_id = "+p(1), userID).Scan(&newBalance); scanErr != nil {
			return scanErr
		}

		orderID, idErr := uuid.NewV7()
		if idErr != nil {
			return idErr
		}
		shortUser := strings.ReplaceAll(userID, "-", "")
		if len(shortUser) > 8 {
			shortUser = shortUser[:8]
		}
		tradeNo := fmt.Sprintf("VLWAL%s%d", shortUser, time.Now().Unix()%1000000)
		_, execErr = q.ExecContext(ctx,
			"INSERT INTO topup_requests (id, user_id, amount, payment_ref, status, created_at, "+
				"trade_no, pay_amount, credit_amount, bonus_amount, bonus_detail, completed_at, order_kind, plan_id) "+
				"VALUES ("+p(1)+", "+p(2)+", "+p(3)+", "+p(4)+", 'completed', CURRENT_TIMESTAMP, "+
				p(5)+", "+p(6)+", 0, 0, '', CURRENT_TIMESTAMP, 'subscription', "+p(7)+")",
			orderID.String(), userID, price, tradeNo, tradeNo, price, planID,
		)
		if execErr != nil {
			return translateError(execErr)
		}

		txID, idErr := uuid.NewV7()
		if idErr != nil {
			return idErr
		}
		desc := fmt.Sprintf("Subscription purchase: %s", plan.Name)
		_, execErr = q.ExecContext(ctx,
			"INSERT INTO transactions (id, user_id, type, amount, balance_after, ref_id, description, created_at) "+
				"VALUES ("+p(1)+", "+p(2)+", 'usage', "+p(3)+", "+p(4)+", "+p(5)+", "+p(6)+", CURRENT_TIMESTAMP)",
			txID.String(), userID, -price, newBalance, orderID.String(), desc,
		)
		if execErr != nil {
			return translateError(execErr)
		}

		now := time.Now().UTC()
		expires := now.Add(time.Duration(plan.ValidityDays) * 24 * time.Hour)
		subID, idErr := uuid.NewV7()
		if idErr != nil {
			return idErr
		}
		_, execErr = q.ExecContext(ctx,
			"INSERT INTO user_subscriptions (id, user_id, plan_id, status, starts_at, expires_at, order_id, created_at, updated_at) "+
				"VALUES ("+p(1)+", "+p(2)+", "+p(3)+", 'active', "+p(4)+", "+p(5)+", "+p(6)+", CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)",
			subID.String(), userID, planID, now.Format(time.RFC3339), expires.Format(time.RFC3339), orderID.String(),
		)
		if execErr != nil {
			return translateError(execErr)
		}
		row := q.QueryRowContext(ctx,
			"SELECT id, user_id, plan_id, status, starts_at, expires_at, order_id, created_at, updated_at "+
				"FROM user_subscriptions WHERE id = "+p(1),
			subID.String(),
		)
		var created UserSubscription
		if scanErr := row.Scan(
			&created.ID, &created.UserID, &created.PlanID, &created.Status,
			&created.StartsAt, &created.ExpiresAt, &created.OrderID, &created.CreatedAt, &created.UpdatedAt,
		); scanErr != nil {
			return scanErr
		}
		us = &created
		return nil
	})
	if err != nil {
		return nil, 0, fmt.Errorf("PurchaseSubscriptionWithWallet user=%s plan=%s: %w", userID, planID, err)
	}
	return us, newBalance, nil
}