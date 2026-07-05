package admin

import (
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"golang.org/x/crypto/bcrypt"

	"github.com/voidmind-io/voidllm/internal/apierror"
	"github.com/voidmind-io/voidllm/internal/auth"
	"github.com/voidmind-io/voidllm/internal/db"
	"github.com/voidmind-io/voidllm/internal/site"
	"github.com/voidmind-io/voidllm/pkg/keygen"
)

// registerRequest is the JSON body accepted by Register.
type registerRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
	AcceptTerms bool   `json:"accept_terms"`
}

// registerResponse is returned on successful public signup. It carries a
// session token so the UI can log the customer straight in.
type registerResponse struct {
	Token     string     `json:"token"`
	ExpiresAt string     `json:"expires_at"`
	User      meResponse `json:"user"`
}

// Register handles POST /api/v1/auth/register — public self-signup for the
// marketplace. It creates the user, an empty wallet, and a session.
//
// @Summary      Public customer signup
// @Description  Creates a customer account with a prepaid wallet and session.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Success      201  {object}  registerResponse
// @Failure      400  {object}  swaggerErrorResponse
// @Failure      409  {object}  swaggerErrorResponse
// @Router       /auth/register [post]
func (h *Handler) Register(c fiber.Ctx) error {
	var req registerRequest
	if err := c.Bind().JSON(&req); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}

	req.Email = strings.TrimSpace(strings.ToLower(req.Email))
	req.DisplayName = strings.TrimSpace(req.DisplayName)

	if req.Email == "" || !strings.Contains(req.Email, "@") {
		return apierror.BadRequest(c, "a valid email is required")
	}
	if len(req.Password) < 8 {
		return apierror.BadRequest(c, "password must be at least 8 characters")
	}
	if req.DisplayName == "" {
		return apierror.BadRequest(c, "display_name is required")
	}

	ctx := c.Context()

	siteCfg, err := site.Load(ctx, h.DB)
	if err != nil {
		h.Log.ErrorContext(ctx, "register: load site settings", slog.String("error", err.Error()))
		return apierror.InternalError(c, "signup failed")
	}
	if !siteCfg.RegisterEnabled {
		return apierror.Send(c, fiber.StatusForbidden, "forbidden", "registration is disabled")
	}
	if siteCfg.UserAgreementEnabled && !req.AcceptTerms {
		return apierror.BadRequest(c, "you must accept the terms of service")
	}

	// Reuse the login throttle for signup abuse protection: one bucket per IP
	// plus a per-email lockout, same thresholds as login brute-force.
	if h.LoginThrottle != nil {
		if err := h.LoginThrottle.Allow(c.IP(), req.Email); err != nil {
			return apierror.Send(c, fiber.StatusTooManyRequests, "too_many_requests",
				"too many signup attempts, try again later")
		}
	}

	passwordHashBytes, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		h.Log.ErrorContext(ctx, "register: hash password", slog.String("error", err.Error()))
		return apierror.InternalError(c, "signup failed")
	}
	passwordHash := string(passwordHashBytes)

	user, err := h.DB.CreateUser(ctx, db.CreateUserParams{
		Email:        req.Email,
		DisplayName:  req.DisplayName,
		PasswordHash: &passwordHash,
		AuthProvider: "local",
	})
	if err != nil {
		if errors.Is(err, db.ErrConflict) {
			if h.LoginThrottle != nil {
				h.LoginThrottle.RecordFailure(req.Email)
			}
			return apierror.Conflict(c, "email already registered")
		}
		h.Log.ErrorContext(ctx, "register: create user", slog.String("error", err.Error()))
		return apierror.InternalError(c, "signup failed")
	}

	// Empty prepaid wallet; debits apply only when settings.wallet.enforce_balance is true.
	if _, err := h.DB.CreateWallet(ctx, user.ID, ""); err != nil {
		h.Log.ErrorContext(ctx, "register: create wallet", slog.String("error", err.Error()))
		return apierror.InternalError(c, "signup failed")
	}
	if h.Wallet != nil {
		h.Wallet.Register(user.ID)
	}

	// Open a 24h session so the UI can log the customer in immediately.
	sessionKey, err := keygen.Generate(keygen.KeyTypeSession)
	if err != nil {
		h.Log.ErrorContext(ctx, "register: generate session key", slog.String("error", err.Error()))
		return apierror.InternalError(c, "signup failed")
	}
	sessionHash := keygen.Hash(sessionKey, h.HMACSecret)
	expiresAt := time.Now().UTC().Add(24 * time.Hour)
	expiresAtStr := expiresAt.Format(time.RFC3339)
	sessionRec, err := h.DB.CreateAPIKey(ctx, db.CreateAPIKeyParams{
		KeyHash:   sessionHash,
		KeyHint:   keygen.Hint(sessionKey),
		KeyType:   keygen.KeyTypeSession,
		Name:      "Login session",
		UserID:    &user.ID,
		ExpiresAt: &expiresAtStr,
		CreatedBy: user.ID,
	})
	if err != nil {
		h.Log.ErrorContext(ctx, "register: create session", slog.String("error", err.Error()))
		return apierror.InternalError(c, "signup failed")
	}
	h.KeyCache.Set(sessionHash, auth.KeyInfo{
		ID:        sessionRec.ID,
		KeyType:   keygen.KeyTypeSession,
		Role:      auth.RoleMember,
		UserID:    user.ID,
		Name:      "Login session",
		ExpiresAt: &expiresAt,
	})

	if h.LoginThrottle != nil {
		h.LoginThrottle.RecordSuccess(req.Email)
	}

	h.Log.InfoContext(ctx, "customer registered",
		slog.String("user_id", user.ID),
	)

	return c.Status(fiber.StatusCreated).JSON(registerResponse{
		Token:     sessionKey,
		ExpiresAt: expiresAtStr,
		User: meResponse{
			ID:          user.ID,
			Email:       user.Email,
			DisplayName: user.DisplayName,
			Role:        auth.RoleMember,
		},
	})
}

// walletResponse is the JSON shape for GET /me/wallet.
type walletResponse struct {
	Balance  float64 `json:"balance"`
	Currency string  `json:"currency"`
}

// MyWallet handles GET /api/v1/me/wallet — the customer's own balance.
func (h *Handler) MyWallet(c fiber.Ctx) error {
	keyInfo := auth.KeyInfoFromCtx(c)
	if keyInfo == nil || keyInfo.UserID == "" {
		return apierror.Send(c, fiber.StatusForbidden, "forbidden", "no user identity on this key")
	}

	w, err := h.DB.GetWalletByUser(c.Context(), keyInfo.UserID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return c.JSON(walletResponse{Balance: 0, Currency: "VND"})
		}
		h.Log.ErrorContext(c.Context(), "my wallet", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to load wallet")
	}
	return c.JSON(walletResponse{Balance: w.Balance, Currency: w.Currency})
}

// transactionItem is one row in the transaction history response.
type transactionItem struct {
	ID           string  `json:"id"`
	Type         string  `json:"type"`
	Amount       float64 `json:"amount"`
	BalanceAfter float64 `json:"balance_after"`
	RefID        string  `json:"ref_id"`
	Description  string  `json:"description"`
	CreatedAt    string  `json:"created_at"`
}

// MyTransactions handles GET /api/v1/me/transactions — the customer's ledger,
// newest first, keyset-paginated by transaction ID.
func (h *Handler) MyTransactions(c fiber.Ctx) error {
	keyInfo := auth.KeyInfoFromCtx(c)
	if keyInfo == nil || keyInfo.UserID == "" {
		return apierror.Send(c, fiber.StatusForbidden, "forbidden", "no user identity on this key")
	}

	pg, err := parsePagination(c)
	if err != nil {
		return apierror.BadRequest(c, err.Error())
	}

	txs, err := h.DB.ListTransactions(c.Context(), keyInfo.UserID, pg.Cursor, pg.Limit+1)
	if err != nil {
		h.Log.ErrorContext(c.Context(), "my transactions", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to load transactions")
	}

	hasMore := len(txs) > pg.Limit
	if hasMore {
		txs = txs[:pg.Limit]
	}
	items := make([]transactionItem, len(txs))
	for i, t := range txs {
		items[i] = transactionItem{
			ID: t.ID, Type: t.Type, Amount: t.Amount, BalanceAfter: t.BalanceAfter,
			RefID: t.RefID, Description: t.Description, CreatedAt: t.CreatedAt,
		}
	}
	cursor := ""
	if hasMore && len(items) > 0 {
		cursor = items[len(items)-1].ID
	}
	return c.JSON(fiber.Map{"data": items, "has_more": hasMore, "cursor": cursor})
}

// MyTopups handles GET /api/v1/me/topups — the customer's own top-up history.
func (h *Handler) MyTopups(c fiber.Ctx) error {
	keyInfo := auth.KeyInfoFromCtx(c)
	if keyInfo == nil || keyInfo.UserID == "" {
		return apierror.Send(c, fiber.StatusForbidden, "forbidden", "no user identity on this key")
	}

	pg, err := parsePagination(c)
	if err != nil {
		return apierror.BadRequest(c, err.Error())
	}

	reqs, err := h.DB.ListTopupRequests(c.Context(), "", keyInfo.UserID, pg.Cursor, pg.Limit+1)
	if err != nil {
		h.Log.ErrorContext(c.Context(), "my topups", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to load top-up requests")
	}

	hasMore := len(reqs) > pg.Limit
	if hasMore {
		reqs = reqs[:pg.Limit]
	}
	items := make([]fiber.Map, len(reqs))
	for i := range reqs {
		items[i] = topupToJSON(&reqs[i])
	}
	cursor := ""
	if hasMore && len(reqs) > 0 {
		cursor = reqs[len(reqs)-1].ID
	}
	return c.JSON(fiber.Map{"data": items, "has_more": hasMore, "cursor": cursor})
}

// catalogModelItem is one row of the public model catalog (sell prices only).
type catalogModelItem struct {
	Name                 string   `json:"name"`
	Type                 string   `json:"type"`
	Logo                 string   `json:"logo,omitempty"`
	MaxContextTokens     int      `json:"max_context_tokens,omitempty"`
	BillPerToken         bool     `json:"bill_per_token"`
	BillPerRequest       bool     `json:"bill_per_request"`
	SellInputPer1M       *float64 `json:"sell_input_per_1m,omitempty"`
	SellOutputPer1M      *float64 `json:"sell_output_per_1m,omitempty"`
	SellCachedInputPer1M *float64 `json:"sell_cached_input_per_1m,omitempty"`
	SellCacheWritePer1M  *float64 `json:"sell_cache_write_per_1m,omitempty"`
	SellPerRequest       *float64 `json:"sell_per_request,omitempty"`
}

func modelToCatalogItem(m db.Model) catalogModelItem {
	modelType := m.ModelType
	if modelType == "" {
		modelType = "chat"
	}
	return catalogModelItem{
		Name:                 m.Name,
		Type:                 modelType,
		Logo:                 m.Logo,
		MaxContextTokens:     m.MaxContextTokens,
		BillPerToken:         m.BillPerToken,
		BillPerRequest:       m.BillPerRequest,
		SellInputPer1M:       m.SellInputPer1M,
		SellOutputPer1M:      m.SellOutputPer1M,
		SellCachedInputPer1M: m.SellCachedInputPer1M,
		SellCacheWritePer1M:  m.SellCacheWritePer1M,
		SellPerRequest:       m.SellPerRequest,
	}
}

// PublicCatalog handles GET /api/v1/public/catalog — the unauthenticated model
// price catalog. Returns active models with at least one configured sell price.
func (h *Handler) PublicCatalog(c fiber.Ctx) error {
	models, err := h.DB.ListCatalogModels(c.Context())
	if err != nil {
		h.Log.ErrorContext(c.Context(), "public catalog", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to load catalog")
	}

	items := make([]catalogModelItem, len(models))
	for i, m := range models {
		items[i] = modelToCatalogItem(m)
	}
	return c.JSON(fiber.Map{"data": items})
}

// PublicModels handles GET /api/v1/public/models — deprecated alias for the
// public catalog. New clients should use GET /api/v1/public/catalog.
func (h *Handler) PublicModels(c fiber.Ctx) error {
	return h.PublicCatalog(c)
}
