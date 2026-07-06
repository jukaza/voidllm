package admin

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/jukaza/tavo/internal/apierror"
	"github.com/jukaza/tavo/internal/auth"
	"github.com/jukaza/tavo/internal/db"
	"github.com/jukaza/tavo/internal/keys"
	subsvc "github.com/jukaza/tavo/internal/subscription"
)

type subscriptionPackageJSON struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CoverType   string `json:"cover_type"`
	CoverValue  string `json:"cover_value"`
	Enabled     bool   `json:"enabled"`
	SortOrder   int    `json:"sort_order"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type subscriptionPlanJSON struct {
	ID                    string   `json:"id"`
	PackageID             string   `json:"package_id"`
	Name                  string   `json:"name"`
	Description           string   `json:"description"`
	Price                 float64  `json:"price"`
	ValidityDays          int      `json:"validity_days"`
	MaxConcurrentBindings int      `json:"max_concurrent_bindings"`
	ActiveBindings        int      `json:"active_bindings"`
	SlotsRemaining        *int     `json:"slots_remaining,omitempty"`
	DailyTokenLimit       int64    `json:"daily_token_limit"`
	MonthlyTokenLimit     int64    `json:"monthly_token_limit"`
	DailyRequestLimit     int      `json:"daily_request_limit"`
	MonthlyRequestLimit   int      `json:"monthly_request_limit"`
	RequestsPerMinute       int      `json:"requests_per_minute"`
	RequestsPerDay          int      `json:"requests_per_day"`
	AllowedModels         []string `json:"allowed_models"`
	QuotaExceededPolicy   string   `json:"quota_exceeded_policy"`
	ForSale               bool     `json:"for_sale"`
	SortOrder             int      `json:"sort_order"`
	CreatedAt             string   `json:"created_at"`
	UpdatedAt             string   `json:"updated_at"`
}

type userSubscriptionJSON struct {
	ID         string `json:"id"`
	UserID     string `json:"user_id"`
	PlanID     string `json:"plan_id"`
	PlanName   string `json:"plan_name,omitempty"`
	PackageName string `json:"package_name,omitempty"`
	Status     string `json:"status"`
	StartsAt   string `json:"starts_at"`
	ExpiresAt  string `json:"expires_at"`
	OrderID    string `json:"order_id,omitempty"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

type keySubscriptionBindingJSON struct {
	KeyID              string `json:"key_id"`
	UserSubscriptionID string `json:"user_subscription_id"`
	PlanID             string `json:"plan_id,omitempty"`
	PlanName           string `json:"plan_name,omitempty"`
	PackageName        string `json:"package_name,omitempty"`
	ExpiresAt          string `json:"expires_at,omitempty"`
	Status             string `json:"status"`
}

func packageToJSON(p *db.SubscriptionPackage) subscriptionPackageJSON {
	return subscriptionPackageJSON{
		ID:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		CoverType:   p.CoverType,
		CoverValue:  p.CoverValue,
		Enabled:     p.Enabled,
		SortOrder:   p.SortOrder,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

func (h *Handler) planToJSON(ctx context.Context, p *db.SubscriptionPlan) (subscriptionPlanJSON, error) {
	active, err := h.DB.CountActivePlanBindings(ctx, p.ID)
	if err != nil {
		return subscriptionPlanJSON{}, err
	}
	out := subscriptionPlanJSON{
		ID:                    p.ID,
		PackageID:             p.PackageID,
		Name:                  p.Name,
		Description:           p.Description,
		Price:                 p.Price,
		ValidityDays:          p.ValidityDays,
		MaxConcurrentBindings: p.MaxConcurrentBindings,
		ActiveBindings:        active,
		DailyTokenLimit:       p.DailyTokenLimit,
		MonthlyTokenLimit:     p.MonthlyTokenLimit,
		DailyRequestLimit:     p.DailyRequestLimit,
		MonthlyRequestLimit:   p.MonthlyRequestLimit,
		RequestsPerMinute:     p.RequestsPerMinute,
		RequestsPerDay:        p.RequestsPerDay,
		AllowedModels:         keys.ParseModelLimits(p.AllowedModels),
		QuotaExceededPolicy:   p.QuotaExceededPolicy,
		ForSale:               p.ForSale,
		SortOrder:             p.SortOrder,
		CreatedAt:             p.CreatedAt,
		UpdatedAt:             p.UpdatedAt,
	}
	if p.MaxConcurrentBindings > 0 {
		rem := p.MaxConcurrentBindings - active
		if rem < 0 {
			rem = 0
		}
		out.SlotsRemaining = &rem
	}
	return out, nil
}

// ListSubscriptionPackages handles GET /api/v1/admin/subscription-packages.
func (h *Handler) ListSubscriptionPackages(c fiber.Ctx) error {
	includeDisabled := c.Query("include_disabled", "") == "1"
	pkgs, err := h.DB.ListSubscriptionPackages(c.Context(), includeDisabled)
	if err != nil {
		h.Log.ErrorContext(c.Context(), "list subscription packages", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to list subscription packages")
	}
	items := make([]subscriptionPackageJSON, len(pkgs))
	for i := range pkgs {
		items[i] = packageToJSON(&pkgs[i])
	}
	return c.JSON(fiber.Map{"data": items})
}

type createPackageRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	CoverType   string `json:"cover_type"`
	CoverValue  string `json:"cover_value"`
	Enabled     *bool  `json:"enabled"`
	SortOrder   int    `json:"sort_order"`
}

// CreateSubscriptionPackage handles POST /api/v1/admin/subscription-packages.
func (h *Handler) CreateSubscriptionPackage(c fiber.Ctx) error {
	var req createPackageRequest
	if err := c.Bind().JSON(&req); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}
	if strings.TrimSpace(req.Name) == "" {
		return apierror.BadRequest(c, "name is required")
	}
	coverType := req.CoverType
	if coverType == "" {
		coverType = "default"
	}
	if !subsvc.ValidCoverType(coverType) {
		return apierror.BadRequest(c, "invalid cover_type")
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	pkg, err := h.DB.CreateSubscriptionPackage(c.Context(), db.CreateSubscriptionPackageParams{
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
		CoverType:   coverType,
		CoverValue:  strings.TrimSpace(req.CoverValue),
		Enabled:     enabled,
		SortOrder:   req.SortOrder,
	})
	if err != nil {
		h.Log.ErrorContext(c.Context(), "create subscription package", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to create subscription package")
	}
	return c.Status(fiber.StatusCreated).JSON(packageToJSON(pkg))
}

// UpdateSubscriptionPackage handles PATCH /api/v1/admin/subscription-packages/:id.
func (h *Handler) UpdateSubscriptionPackage(c fiber.Ctx) error {
	id := c.Params("id")
	var req createPackageRequest
	if err := c.Bind().JSON(&req); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}
	params := db.UpdateSubscriptionPackageParams{}
	if req.Name != "" {
		v := strings.TrimSpace(req.Name)
		params.Name = &v
	}
	if req.Description != "" || c.Get("Content-Type") != "" {
		v := strings.TrimSpace(req.Description)
		params.Description = &v
	}
	if req.CoverType != "" {
		if !subsvc.ValidCoverType(req.CoverType) {
			return apierror.BadRequest(c, "invalid cover_type")
		}
		params.CoverType = &req.CoverType
	}
	if req.CoverValue != "" || req.CoverType == "url" || req.CoverType == "default" {
		v := strings.TrimSpace(req.CoverValue)
		params.CoverValue = &v
	}
	if req.Enabled != nil {
		params.Enabled = req.Enabled
	}
	if req.SortOrder != 0 {
		params.SortOrder = &req.SortOrder
	}
	pkg, err := h.DB.UpdateSubscriptionPackage(c.Context(), id, params)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "subscription package not found")
		}
		h.Log.ErrorContext(c.Context(), "update subscription package", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to update subscription package")
	}
	return c.JSON(packageToJSON(pkg))
}

// DeleteSubscriptionPackage handles DELETE /api/v1/admin/subscription-packages/:id.
func (h *Handler) DeleteSubscriptionPackage(c fiber.Ctx) error {
	id := c.Params("id")
	pkg, err := h.DB.GetSubscriptionPackage(c.Context(), id)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "subscription package not found")
		}
		return apierror.InternalError(c, "failed to load subscription package")
	}
	if err := h.DB.DeleteSubscriptionPackage(c.Context(), id); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "subscription package not found")
		}
		return apierror.InternalError(c, "failed to delete subscription package")
	}
	if subsvc.IsUploadedCover(pkg.CoverValue) {
		_ = subsvc.RemovePackageCover(h.DataDir, id)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// UploadSubscriptionPackageCover handles POST /api/v1/admin/subscription-packages/:id/cover.
func (h *Handler) UploadSubscriptionPackageCover(c fiber.Ctx) error {
	id := c.Params("id")
	if _, err := h.DB.GetSubscriptionPackage(c.Context(), id); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "subscription package not found")
		}
		return apierror.InternalError(c, "failed to load subscription package")
	}
	file, err := c.FormFile("cover")
	if err != nil {
		return apierror.BadRequest(c, "cover file is required")
	}
	publicPath, err := subsvc.SavePackageCover(h.DataDir, id, file)
	if err != nil {
		if strings.Contains(err.Error(), "exceeds") ||
			strings.Contains(err.Error(), "unsupported") ||
			strings.Contains(err.Error(), "required") ||
			strings.Contains(err.Error(), "empty") {
			return apierror.BadRequest(c, err.Error())
		}
		h.Log.ErrorContext(c.Context(), "subscription cover upload", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to save cover")
	}
	coverType := "upload"
	pkg, err := h.DB.UpdateSubscriptionPackage(c.Context(), id, db.UpdateSubscriptionPackageParams{
		CoverType:  &coverType,
		CoverValue: &publicPath,
	})
	if err != nil {
		return apierror.InternalError(c, "failed to update package cover")
	}
	return c.JSON(packageToJSON(pkg))
}

// ListSubscriptionPlans handles GET /api/v1/admin/subscription-plans.
func (h *Handler) ListSubscriptionPlans(c fiber.Ctx) error {
	plans, err := h.DB.ListSubscriptionPlans(c.Context(), c.Query("package_id", ""))
	if err != nil {
		return apierror.InternalError(c, "failed to list subscription plans")
	}
	items := make([]subscriptionPlanJSON, 0, len(plans))
	for i := range plans {
		j, err := h.planToJSON(c.Context(), &plans[i])
		if err != nil {
			return apierror.InternalError(c, "failed to enrich subscription plan")
		}
		items = append(items, j)
	}
	return c.JSON(fiber.Map{"data": items})
}

type createPlanRequest struct {
	PackageID             string   `json:"package_id"`
	Name                  string   `json:"name"`
	Description           string   `json:"description"`
	Price                 float64  `json:"price"`
	ValidityDays          int      `json:"validity_days"`
	MaxConcurrentBindings int      `json:"max_concurrent_bindings"`
	DailyTokenLimit       int64    `json:"daily_token_limit"`
	MonthlyTokenLimit     int64    `json:"monthly_token_limit"`
	DailyRequestLimit     int      `json:"daily_request_limit"`
	MonthlyRequestLimit   int      `json:"monthly_request_limit"`
	RequestsPerMinute     int      `json:"requests_per_minute"`
	RequestsPerDay        int      `json:"requests_per_day"`
	AllowedModels         []string `json:"allowed_models"`
	QuotaExceededPolicy   string   `json:"quota_exceeded_policy"`
	ForSale               *bool    `json:"for_sale"`
	SortOrder             int      `json:"sort_order"`
}

// CreateSubscriptionPlan handles POST /api/v1/admin/subscription-plans.
func (h *Handler) CreateSubscriptionPlan(c fiber.Ctx) error {
	var req createPlanRequest
	if err := c.Bind().JSON(&req); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}
	if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.PackageID) == "" {
		return apierror.BadRequest(c, "name and package_id are required")
	}
	if _, err := h.DB.GetSubscriptionPackage(c.Context(), req.PackageID); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.BadRequest(c, "package not found")
		}
		return apierror.InternalError(c, "failed to verify package")
	}
	policy := req.QuotaExceededPolicy
	if policy == "" {
		policy = "fallback_wallet"
	}
	if !subsvc.ValidQuotaPolicy(policy) {
		return apierror.BadRequest(c, "invalid quota_exceeded_policy")
	}
	validity := req.ValidityDays
	if validity <= 0 {
		validity = 30
	}
	forSale := true
	if req.ForSale != nil {
		forSale = *req.ForSale
	}
	plan, err := h.DB.CreateSubscriptionPlan(c.Context(), db.CreateSubscriptionPlanParams{
		PackageID:             req.PackageID,
		Name:                  strings.TrimSpace(req.Name),
		Description:           strings.TrimSpace(req.Description),
		Price:                 req.Price,
		ValidityDays:          validity,
		MaxConcurrentBindings: req.MaxConcurrentBindings,
		DailyTokenLimit:       req.DailyTokenLimit,
		MonthlyTokenLimit:     req.MonthlyTokenLimit,
		DailyRequestLimit:     req.DailyRequestLimit,
		MonthlyRequestLimit:   req.MonthlyRequestLimit,
		RequestsPerMinute:     req.RequestsPerMinute,
		RequestsPerDay:        req.RequestsPerDay,
		AllowedModels:         formatModelLimitsForDB(req.AllowedModels),
		QuotaExceededPolicy:   policy,
		ForSale:               forSale,
		SortOrder:             req.SortOrder,
	})
	if err != nil {
		return apierror.InternalError(c, "failed to create subscription plan")
	}
	j, err := h.planToJSON(c.Context(), plan)
	if err != nil {
		return apierror.InternalError(c, "failed to enrich subscription plan")
	}
	return c.Status(fiber.StatusCreated).JSON(j)
}

// UpdateSubscriptionPlan handles PATCH /api/v1/admin/subscription-plans/:id.
func (h *Handler) UpdateSubscriptionPlan(c fiber.Ctx) error {
	id := c.Params("id")
	var req createPlanRequest
	if err := c.Bind().JSON(&req); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}
	params := db.UpdateSubscriptionPlanParams{}
	if req.Name != "" {
		v := strings.TrimSpace(req.Name)
		params.Name = &v
	}
	if req.Description != "" {
		v := strings.TrimSpace(req.Description)
		params.Description = &v
	}
	if req.Price != 0 {
		params.Price = &req.Price
	}
	if req.ValidityDays > 0 {
		params.ValidityDays = &req.ValidityDays
	}
	params.MaxConcurrentBindings = &req.MaxConcurrentBindings
	if req.DailyTokenLimit >= 0 {
		params.DailyTokenLimit = &req.DailyTokenLimit
	}
	if req.MonthlyTokenLimit >= 0 {
		params.MonthlyTokenLimit = &req.MonthlyTokenLimit
	}
	if req.DailyRequestLimit >= 0 {
		params.DailyRequestLimit = &req.DailyRequestLimit
	}
	if req.MonthlyRequestLimit >= 0 {
		params.MonthlyRequestLimit = &req.MonthlyRequestLimit
	}
	if req.RequestsPerMinute >= 0 {
		params.RequestsPerMinute = &req.RequestsPerMinute
	}
	if req.RequestsPerDay >= 0 {
		params.RequestsPerDay = &req.RequestsPerDay
	}
	if req.AllowedModels != nil {
		v := formatModelLimitsForDB(req.AllowedModels)
		params.AllowedModels = &v
	}
	if req.QuotaExceededPolicy != "" {
		if !subsvc.ValidQuotaPolicy(req.QuotaExceededPolicy) {
			return apierror.BadRequest(c, "invalid quota_exceeded_policy")
		}
		params.QuotaExceededPolicy = &req.QuotaExceededPolicy
	}
	if req.ForSale != nil {
		params.ForSale = req.ForSale
	}
	if req.SortOrder != 0 {
		params.SortOrder = &req.SortOrder
	}
	plan, err := h.DB.UpdateSubscriptionPlan(c.Context(), id, params)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "subscription plan not found")
		}
		return apierror.InternalError(c, "failed to update subscription plan")
	}
	j, err := h.planToJSON(c.Context(), plan)
	if err != nil {
		return apierror.InternalError(c, "failed to enrich subscription plan")
	}
	return c.JSON(j)
}

// DeleteSubscriptionPlan handles DELETE /api/v1/admin/subscription-plans/:id.
func (h *Handler) DeleteSubscriptionPlan(c fiber.Ctx) error {
	id := c.Params("id")
	if err := h.DB.DeleteSubscriptionPlan(c.Context(), id); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "subscription plan not found")
		}
		return apierror.InternalError(c, "failed to delete subscription plan")
	}
	return c.SendStatus(fiber.StatusNoContent)
}

type grantSubscriptionRequest struct {
	UserID string `json:"user_id"`
	PlanID string `json:"plan_id"`
	Days   int    `json:"days"`
}

// GrantUserSubscription handles POST /api/v1/admin/user-subscriptions.
func (h *Handler) GrantUserSubscription(c fiber.Ctx) error {
	var req grantSubscriptionRequest
	if err := c.Bind().JSON(&req); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}
	if strings.TrimSpace(req.UserID) == "" || strings.TrimSpace(req.PlanID) == "" {
		return apierror.BadRequest(c, "user_id and plan_id are required")
	}
	if _, err := h.DB.GetUser(c.Context(), req.UserID); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.BadRequest(c, "user not found")
		}
		return apierror.InternalError(c, "failed to verify user")
	}
	plan, err := h.DB.GetSubscriptionPlan(c.Context(), req.PlanID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.BadRequest(c, "plan not found")
		}
		return apierror.InternalError(c, "failed to verify plan")
	}
	days := req.Days
	if days <= 0 {
		days = plan.ValidityDays
	}
	now := time.Now().UTC()
	expires := now.Add(time.Duration(days) * 24 * time.Hour)
	us, err := h.DB.CreateUserSubscription(c.Context(), db.CreateUserSubscriptionParams{
		UserID:    req.UserID,
		PlanID:    req.PlanID,
		StartsAt:  now.Format(time.RFC3339),
		ExpiresAt: expires.Format(time.RFC3339),
	})
	if err != nil {
		return apierror.InternalError(c, "failed to grant subscription")
	}
	return c.Status(fiber.StatusCreated).JSON(userSubscriptionToJSON(us, plan.Name, ""))
}

// ListAdminUserSubscriptions handles GET /api/v1/admin/user-subscriptions.
func (h *Handler) ListAdminUserSubscriptions(c fiber.Ctx) error {
	userID := c.Query("user_id", "")
	if userID == "" {
		return apierror.BadRequest(c, "user_id is required")
	}
	subs, err := h.DB.ListUserSubscriptions(c.Context(), userID, false)
	if err != nil {
		return apierror.InternalError(c, "failed to list subscriptions")
	}
	items := make([]userSubscriptionJSON, 0, len(subs))
	for i := range subs {
		plan, _ := h.DB.GetSubscriptionPlan(c.Context(), subs[i].PlanID)
		planName := ""
		pkgName := ""
		if plan != nil {
			planName = plan.Name
			if pkg, err := h.DB.GetSubscriptionPackage(c.Context(), plan.PackageID); err == nil {
				pkgName = pkg.Name
			}
		}
		items = append(items, userSubscriptionToJSON(&subs[i], planName, pkgName))
	}
	return c.JSON(fiber.Map{"data": items})
}

func userSubscriptionToJSON(us *db.UserSubscription, planName, packageName string) userSubscriptionJSON {
	return userSubscriptionJSON{
		ID:          us.ID,
		UserID:      us.UserID,
		PlanID:      us.PlanID,
		PlanName:    planName,
		PackageName: packageName,
		Status:      us.Status,
		StartsAt:    us.StartsAt,
		ExpiresAt:   us.ExpiresAt,
		OrderID:     us.OrderID,
		CreatedAt:   us.CreatedAt,
		UpdatedAt:   us.UpdatedAt,
	}
}

// MySubscriptions handles GET /api/v1/my-subscriptions.
func (h *Handler) MySubscriptions(c fiber.Ctx) error {
	keyInfo, ok := requireAuth(c)
	if !ok {
		return nil
	}
	subs, err := h.DB.ListUserSubscriptions(c.Context(), keyInfo.UserID, true)
	if err != nil {
		return apierror.InternalError(c, "failed to list subscriptions")
	}
	items := make([]userSubscriptionJSON, 0, len(subs))
	for i := range subs {
		plan, _ := h.DB.GetSubscriptionPlan(c.Context(), subs[i].PlanID)
		planName := ""
		pkgName := ""
		if plan != nil {
			planName = plan.Name
			if pkg, err := h.DB.GetSubscriptionPackage(c.Context(), plan.PackageID); err == nil {
				pkgName = pkg.Name
			}
		}
		items = append(items, userSubscriptionToJSON(&subs[i], planName, pkgName))
	}
	return c.JSON(fiber.Map{"data": items})
}

type bindSubscriptionRequest struct {
	UserSubscriptionID string `json:"user_subscription_id"`
}

// BindKeySubscription handles POST /api/v1/keys/:key_id/subscription-binding.
func (h *Handler) BindKeySubscription(c fiber.Ctx) error {
	keyInfo, ok := requireAuth(c)
	if !ok {
		return nil
	}
	keyID := c.Params("key_id")
	apiKey, err := h.DB.GetAPIKey(c.Context(), keyID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "api key not found")
		}
		return apierror.InternalError(c, "failed to load api key")
	}
	if !apiKeyVisibleToCallerKey(apiKey, keyInfo) {
		return apierror.NotFound(c, "api key not found")
	}

	var req bindSubscriptionRequest
	if err := c.Bind().JSON(&req); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}
	if strings.TrimSpace(req.UserSubscriptionID) == "" {
		return apierror.BadRequest(c, "user_subscription_id is required")
	}

	_, err = h.DB.BindKeySubscription(c.Context(), keyID, req.UserSubscriptionID, keyInfo.UserID)
	if err != nil {
		if errors.Is(err, db.ErrSubscriptionSlotsFull) {
			return apierror.Send(c, fiber.StatusConflict, "slots_full", "subscription plan has no available slots")
		}
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "subscription not found")
		}
		return apierror.BadRequest(c, err.Error())
	}
	if err := h.refreshKeySubscriptionInCache(c.Context(), apiKey); err != nil {
		h.Log.ErrorContext(c.Context(), "refresh key subscription cache", slog.String("error", err.Error()))
	}
	return h.GetKeySubscriptionBinding(c)
}

// ReleaseKeySubscription handles DELETE /api/v1/keys/:key_id/subscription-binding.
func (h *Handler) ReleaseKeySubscription(c fiber.Ctx) error {
	keyInfo, ok := requireAuth(c)
	if !ok {
		return nil
	}
	keyID := c.Params("key_id")
	apiKey, err := h.DB.GetAPIKey(c.Context(), keyID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "api key not found")
		}
		return apierror.InternalError(c, "failed to load api key")
	}
	if !apiKeyVisibleToCallerKey(apiKey, keyInfo) {
		return apierror.NotFound(c, "api key not found")
	}
	if err := h.DB.ReleaseKeySubscriptionBinding(c.Context(), keyID, keyInfo.UserID); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "no active subscription binding")
		}
		return apierror.InternalError(c, "failed to release subscription binding")
	}
	if err := h.refreshKeySubscriptionInCache(c.Context(), apiKey); err != nil {
		h.Log.ErrorContext(c.Context(), "refresh key subscription cache", slog.String("error", err.Error()))
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// GetKeySubscriptionBinding handles GET /api/v1/keys/:key_id/subscription-binding.
func (h *Handler) GetKeySubscriptionBinding(c fiber.Ctx) error {
	keyInfo, ok := requireAuth(c)
	if !ok {
		return nil
	}
	keyID := c.Params("key_id")
	apiKey, err := h.DB.GetAPIKey(c.Context(), keyID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "api key not found")
		}
		return apierror.InternalError(c, "failed to load api key")
	}
	if !apiKeyVisibleToCallerKey(apiKey, keyInfo) {
		return apierror.NotFound(c, "api key not found")
	}
	binding, err := h.DB.GetKeySubscriptionBinding(c.Context(), keyID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return c.JSON(fiber.Map{"data": nil})
		}
		return apierror.InternalError(c, "failed to load subscription binding")
	}
	us, err := h.DB.GetUserSubscription(c.Context(), binding.UserSubscriptionID)
	if err != nil {
		return apierror.InternalError(c, "failed to load subscription")
	}
	plan, _ := h.DB.GetSubscriptionPlan(c.Context(), us.PlanID)
	out := keySubscriptionBindingJSON{
		KeyID:              keyID,
		UserSubscriptionID: binding.UserSubscriptionID,
		PlanID:             us.PlanID,
		ExpiresAt:          us.ExpiresAt,
		Status:             binding.Status,
	}
	if plan != nil {
		out.PlanName = plan.Name
		if pkg, err := h.DB.GetSubscriptionPackage(c.Context(), plan.PackageID); err == nil {
			out.PackageName = pkg.Name
		}
	}
	return c.JSON(fiber.Map{"data": out})
}

func (h *Handler) refreshKeySubscriptionInCache(ctx context.Context, apiKey *db.APIKey) error {
	if apiKey == nil {
		return nil
	}
	cached, ok := h.KeyCache.Get(apiKey.KeyHash)
	if !ok {
		return nil
	}
	subCtx, err := h.DB.ResolveKeySubscriptionContext(ctx, apiKey.ID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			cached.Subscription = nil
			h.KeyCache.Set(apiKey.KeyHash, cached)
			return nil
		}
		return err
	}
	cached.Subscription = &auth.SubscriptionBinding{
		UserSubscriptionID:  subCtx.UserSubscriptionID,
		PlanID:              subCtx.PlanID,
		AllowedModels:       subCtx.AllowedModels,
		DailyTokenLimit:     subCtx.DailyTokenLimit,
		MonthlyTokenLimit:   subCtx.MonthlyTokenLimit,
		DailyRequestLimit:   subCtx.DailyRequestLimit,
		MonthlyRequestLimit: subCtx.MonthlyRequestLimit,
		RequestsPerMinute:   subCtx.RequestsPerMinute,
		RequestsPerDay:      subCtx.RequestsPerDay,
		QuotaExceededPolicy: subCtx.QuotaExceededPolicy,
		ExpiresAt:           subCtx.ExpiresAt,
	}
	h.KeyCache.Set(apiKey.KeyHash, cached)
	return nil
}

