package admin

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/jukaza/tavo/internal/apierror"
	"github.com/jukaza/tavo/internal/auth"
	"github.com/jukaza/tavo/internal/db"
	"github.com/jukaza/tavo/internal/keys"
	"github.com/jukaza/tavo/internal/payment"
)

type publicSubscriptionPlanJSON struct {
	ID                    string   `json:"id"`
	Name                  string   `json:"name"`
	Description           string   `json:"description"`
	Price                 float64  `json:"price"`
	ValidityDays          int      `json:"validity_days"`
	MaxConcurrentBindings int      `json:"max_concurrent_bindings"`
	SlotsRemaining        *int     `json:"slots_remaining,omitempty"`
	DailyTokenLimit       int64    `json:"daily_token_limit"`
	MonthlyTokenLimit     int64    `json:"monthly_token_limit"`
	DailyRequestLimit     int      `json:"daily_request_limit"`
	MonthlyRequestLimit   int      `json:"monthly_request_limit"`
	AllowedModels         []string `json:"allowed_models"`
	QuotaExceededPolicy   string   `json:"quota_exceeded_policy"`
}

type publicSubscriptionPackageJSON struct {
	ID          string                       `json:"id"`
	Name        string                       `json:"name"`
	Description string                       `json:"description"`
	CoverType   string                       `json:"cover_type"`
	CoverValue  string                       `json:"cover_value"`
	Plans       []publicSubscriptionPlanJSON `json:"plans"`
}

// PublicSubscriptionCatalog handles GET /api/v1/public/subscription-packages.
func (h *Handler) PublicSubscriptionCatalog(c fiber.Ctx) error {
	pkgs, byPackage, err := h.DB.ListPublicSubscriptionCatalog(c.Context())
	if err != nil {
		h.Log.ErrorContext(c.Context(), "public subscription catalog", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to load subscription catalog")
	}
	items := make([]publicSubscriptionPackageJSON, 0, len(pkgs))
	for _, pkg := range pkgs {
		plans := byPackage[pkg.ID]
		planJSON := make([]publicSubscriptionPlanJSON, 0, len(plans))
		for i := range plans {
			j, jErr := h.publicPlanToJSON(c, &plans[i])
			if jErr != nil {
				continue
			}
			planJSON = append(planJSON, j)
		}
		if len(planJSON) == 0 {
			continue
		}
		items = append(items, publicSubscriptionPackageJSON{
			ID:          pkg.ID,
			Name:        pkg.Name,
			Description: pkg.Description,
			CoverType:   pkg.CoverType,
			CoverValue:  pkg.CoverValue,
			Plans:       planJSON,
		})
	}
	return c.JSON(fiber.Map{"data": items})
}

func (h *Handler) publicPlanToJSON(c fiber.Ctx, plan *db.SubscriptionPlan) (publicSubscriptionPlanJSON, error) {
	active, err := h.DB.CountActivePlanBindings(c.Context(), plan.ID)
	if err != nil {
		return publicSubscriptionPlanJSON{}, err
	}
	out := publicSubscriptionPlanJSON{
		ID:                    plan.ID,
		Name:                  plan.Name,
		Description:           plan.Description,
		Price:                 plan.Price,
		ValidityDays:          plan.ValidityDays,
		MaxConcurrentBindings: plan.MaxConcurrentBindings,
		DailyTokenLimit:       plan.DailyTokenLimit,
		MonthlyTokenLimit:     plan.MonthlyTokenLimit,
		DailyRequestLimit:     plan.DailyRequestLimit,
		MonthlyRequestLimit:   plan.MonthlyRequestLimit,
		AllowedModels:         keys.ParseModelLimits(plan.AllowedModels),
		QuotaExceededPolicy:   plan.QuotaExceededPolicy,
	}
	if plan.MaxConcurrentBindings > 0 {
		rem := plan.MaxConcurrentBindings - active
		if rem < 0 {
			rem = 0
		}
		out.SlotsRemaining = &rem
	}
	return out, nil
}

type createSubscriptionOrderRequest struct {
	PlanID string `json:"plan_id"`
}

// CreateSubscriptionOrder handles POST /api/v1/me/subscription-orders.
func (h *Handler) CreateSubscriptionOrder(c fiber.Ctx) error {
	keyInfo := auth.KeyInfoFromCtx(c)
	if keyInfo == nil || keyInfo.UserID == "" {
		return apierror.Send(c, fiber.StatusForbidden, "forbidden", "no user identity on this key")
	}

	var req createSubscriptionOrderRequest
	if err := c.Bind().JSON(&req); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}
	if strings.TrimSpace(req.PlanID) == "" {
		return apierror.BadRequest(c, "plan_id is required")
	}

	plan, err := h.DB.GetSubscriptionPlan(c.Context(), req.PlanID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "plan not found")
		}
		return apierror.InternalError(c, "failed to load plan")
	}
	if !plan.ForSale {
		return apierror.BadRequest(c, "plan is not available for purchase")
	}
	pkg, err := h.DB.GetSubscriptionPackage(c.Context(), plan.PackageID)
	if err != nil || !pkg.Enabled {
		return apierror.BadRequest(c, "plan is not available")
	}
	if plan.Price <= 0 {
		return apierror.BadRequest(c, "plan has no price")
	}

	wallet, walletErr := h.DB.GetWalletByUser(c.Context(), keyInfo.UserID)
	if walletErr == nil && wallet.Balance >= plan.Price {
		us, newBalance, purchaseErr := h.DB.PurchaseSubscriptionWithWallet(c.Context(), keyInfo.UserID, plan.ID)
		if purchaseErr != nil {
			if errors.Is(purchaseErr, db.ErrInsufficientBalance) {
				return apierror.Send(c, fiber.StatusPaymentRequired, "insufficient_balance", "insufficient wallet balance")
			}
			h.Log.ErrorContext(c.Context(), "wallet subscription purchase", slog.String("error", purchaseErr.Error()))
			return apierror.InternalError(c, "failed to purchase subscription")
		}
		if h.Wallet != nil {
			h.Wallet.SetBalance(keyInfo.UserID, newBalance)
		}
		pkgName := pkg.Name
		return c.Status(fiber.StatusCreated).JSON(subscriptionOrderResponse{
			PaymentMethod: "wallet",
			PayAmount:     plan.Price,
			Status:        "completed",
			Subscription:  ptrUserSubscriptionJSON(userSubscriptionToJSON(us, plan.Name, pkgName)),
		})
	}

	cfg, err := payment.Load(c.Context(), h.DB)
	if err != nil {
		return apierror.InternalError(c, "failed to load payment settings")
	}
	if err := validatePayAmount(cfg, plan.Price); err != nil {
		return apierror.BadRequest(c, err.Error())
	}
	if !cfg.Sepay.Enabled {
		return apierror.Send(c, fiber.StatusPaymentRequired, "insufficient_balance",
			"insufficient wallet balance — bank transfer payment is disabled")
	}
	if !cfg.Sepay.IsConfigured() {
		return apierror.Send(c, fiber.StatusPaymentRequired, "insufficient_balance",
			"insufficient wallet balance — configure SePay webhook in Settings → Payment, or top up your wallet")
	}

	tradeNo := generateSubscriptionTradeNo(keyInfo.UserID)
	ttl := cfg.Sepay.OrderTTLMinutes
	if ttl <= 0 {
		ttl = 15
	}
	expiresAt := time.Now().Add(time.Duration(ttl) * time.Minute).UTC().Format(time.RFC3339)

	tr, err := h.DB.CreateSepaySubscriptionOrder(c.Context(), db.CreateSepaySubscriptionParams{
		UserID:    keyInfo.UserID,
		PlanID:    plan.ID,
		TradeNo:   tradeNo,
		PayAmount: plan.Price,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		h.Log.ErrorContext(c.Context(), "create subscription order", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to create subscription order")
	}

	qrURL := buildVietQRURL(cfg.Sepay, plan.Price, tradeNo)
	return c.Status(fiber.StatusCreated).JSON(subscriptionOrderResponse{
		PaymentMethod: "sepay",
		TradeNo:       tr.TradeNo,
		PayAmount:     plan.Price,
		CreditAmount:  0,
		BonusAmount:   0,
		BankCode:      cfg.Sepay.BankCode,
		BankName:      bankDisplayName(cfg.Sepay.BankCode),
		AccountNumber: cfg.Sepay.AccountNumber,
		AccountName:   cfg.Sepay.AccountName,
		QRURL:         qrURL,
		ExpiresAt:     expiresAt,
		Status:        tr.Status,
	})
}

type subscriptionOrderResponse struct {
	PaymentMethod string                `json:"payment_method"`
	TradeNo       string                `json:"trade_no,omitempty"`
	PayAmount     float64               `json:"pay_amount"`
	CreditAmount  float64               `json:"credit_amount,omitempty"`
	BonusAmount   float64               `json:"bonus_amount,omitempty"`
	BankCode      string                `json:"bank_code,omitempty"`
	BankName      string                `json:"bank_name,omitempty"`
	AccountNumber string                `json:"account_number,omitempty"`
	AccountName   string                `json:"account_name,omitempty"`
	QRURL         string                `json:"qr_url,omitempty"`
	ExpiresAt     string                `json:"expires_at,omitempty"`
	Status        string                `json:"status"`
	Subscription  *userSubscriptionJSON `json:"subscription,omitempty"`
}

func ptrUserSubscriptionJSON(j userSubscriptionJSON) *userSubscriptionJSON {
	return &j
}

func generateSubscriptionTradeNo(userID string) string {
	shortUser := strings.ReplaceAll(userID, "-", "")
	if len(shortUser) > 8 {
		shortUser = shortUser[:8]
	}
	randPart := randomAlphaNum(6)
	return fmt.Sprintf("VLSUB%sNO%s%d", shortUser, randPart, time.Now().Unix()%1000000)
}