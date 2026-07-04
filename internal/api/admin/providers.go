package admin

import (
	"errors"
	"log/slog"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/voidmind-io/voidllm/internal/apierror"
	"github.com/voidmind-io/voidllm/internal/auth"
	"github.com/voidmind-io/voidllm/internal/db"
	"github.com/voidmind-io/voidllm/internal/provider"
	"github.com/voidmind-io/voidllm/pkg/crypto"
)

// providerRequest is the JSON body for creating/updating a provider.
type providerRequest struct {
	Name        *string `json:"name"`
	ContactInfo *string `json:"contact_info"`
	Status      *string `json:"status"`
	Notes       *string `json:"notes"`
	// Slug is a short unique handle used to label routes ("openai", "ds").
	// Set to empty string to clear.
	Slug *string `json:"slug"`
	// Protocol is the default wire protocol for routes created from this
	// provider. One of provider.ValidProviders.
	Protocol *string `json:"protocol"`
	Logo     *string `json:"logo"`
	BaseURL  *string `json:"base_url"`
	// APIKey is the default upstream API key in plaintext; it is encrypted
	// before storage and never returned. Set to empty string to clear.
	APIKey *string `json:"api_key"`
}

// slugRe constrains slugs to short lowercase identifiers safe for display
// and for use as route labels.
var slugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,31}$`)

func providerAAD(id string) []byte {
	return []byte("provider:" + id)
}

func providerToJSON(p *db.Provider) fiber.Map {
	slug := ""
	if p.Slug != nil {
		slug = *p.Slug
	}
	return fiber.Map{
		"id":           p.ID,
		"name":         p.Name,
		"contact_info": p.ContactInfo,
		"status":       p.Status,
		"notes":        p.Notes,
		"slug":         slug,
		"protocol":     p.Protocol,
		"logo":         p.Logo,
		"base_url":     p.BaseURL,
		"has_api_key":  p.APIKeyEncrypted != nil,
		"created_at":   p.CreatedAt,
		"updated_at":   p.UpdatedAt,
	}
}

// validateProviderFields checks slug/protocol values shared by create and update.
func validateProviderFields(req *providerRequest) string {
	if req.Slug != nil && *req.Slug != "" && !slugRe.MatchString(*req.Slug) {
		return "slug must be 1-32 lowercase letters, digits, or hyphens"
	}
	if req.Protocol != nil && !provider.ValidProviders[*req.Protocol] {
		return "protocol must be one of: " + strings.Join(provider.Names(), ", ")
	}
	return ""
}

// CreateProvider handles POST /api/v1/providers (system_admin).
func (h *Handler) CreateProvider(c fiber.Ctx) error {
	var req providerRequest
	if err := c.Bind().JSON(&req); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}
	if req.Name == nil || *req.Name == "" {
		return apierror.BadRequest(c, "name is required")
	}
	status := "active"
	if req.Status != nil {
		if *req.Status != "active" && *req.Status != "paused" {
			return apierror.BadRequest(c, `status must be "active" or "paused"`)
		}
		status = *req.Status
	}
	if msg := validateProviderFields(&req); msg != "" {
		return apierror.BadRequest(c, msg)
	}
	contact, notes, protocol, logo, baseURL := "", "", "openai", "", ""
	if req.ContactInfo != nil {
		contact = *req.ContactInfo
	}
	if req.Notes != nil {
		notes = *req.Notes
	}
	if req.Protocol != nil {
		protocol = *req.Protocol
	}
	if req.Logo != nil {
		logo = *req.Logo
	}
	if req.BaseURL != nil {
		baseURL = *req.BaseURL
	}
	var slug *string
	if req.Slug != nil && *req.Slug != "" {
		slug = req.Slug
	}

	created, err := h.DB.CreateProvider(c.Context(), db.CreateProviderParams{
		Name: *req.Name, ContactInfo: contact, Status: status, Notes: notes,
		Slug: slug, Protocol: protocol, Logo: logo, BaseURL: baseURL,
	})
	if err != nil {
		if errors.Is(err, db.ErrConflict) {
			return apierror.Send(c, fiber.StatusConflict, "conflict", "slug already in use")
		}
		h.Log.ErrorContext(c.Context(), "create provider", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to create provider")
	}

	// The API key is encrypted with the provider ID as AAD, so it can only be
	// stored after the row exists and its ID is known.
	if req.APIKey != nil && *req.APIKey != "" {
		enc, encErr := crypto.EncryptString(*req.APIKey, h.EncryptionKey, providerAAD(created.ID))
		if encErr != nil {
			h.Log.ErrorContext(c.Context(), "encrypt provider api key", slog.String("error", encErr.Error()))
			return apierror.InternalError(c, "failed to store api key")
		}
		created, err = h.DB.UpdateProvider(c.Context(), created.ID, db.UpdateProviderParams{APIKeyEncrypted: &enc})
		if err != nil {
			h.Log.ErrorContext(c.Context(), "store provider api key", slog.String("error", err.Error()))
			return apierror.InternalError(c, "failed to store api key")
		}
	}
	return c.Status(fiber.StatusCreated).JSON(providerToJSON(created))
}

// ListProviders handles GET /api/v1/providers (system_admin).
func (h *Handler) ListProviders(c fiber.Ctx) error {
	pg, err := parsePagination(c)
	if err != nil {
		return apierror.BadRequest(c, err.Error())
	}
	providers, err := h.DB.ListProviders(c.Context(), pg.Cursor, pg.Limit+1)
	if err != nil {
		h.Log.ErrorContext(c.Context(), "list providers", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to list providers")
	}
	hasMore := len(providers) > pg.Limit
	if hasMore {
		providers = providers[:pg.Limit]
	}
	items := make([]fiber.Map, len(providers))
	for i := range providers {
		items[i] = providerToJSON(&providers[i])
	}
	cursor := ""
	if hasMore && len(providers) > 0 {
		cursor = providers[len(providers)-1].ID
	}
	return c.JSON(fiber.Map{"data": items, "has_more": hasMore, "cursor": cursor})
}

// GetProvider handles GET /api/v1/providers/:provider_id (system_admin).
func (h *Handler) GetProvider(c fiber.Ctx) error {
	provider, err := h.DB.GetProvider(c.Context(), c.Params("provider_id"))
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "provider not found")
		}
		h.Log.ErrorContext(c.Context(), "get provider", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to get provider")
	}
	return c.JSON(providerToJSON(provider))
}

// UpdateProvider handles PATCH /api/v1/providers/:provider_id (system_admin).
func (h *Handler) UpdateProvider(c fiber.Ctx) error {
	var req providerRequest
	if err := c.Bind().JSON(&req); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}
	if req.Status != nil && *req.Status != "active" && *req.Status != "paused" {
		return apierror.BadRequest(c, `status must be "active" or "paused"`)
	}
	if msg := validateProviderFields(&req); msg != "" {
		return apierror.BadRequest(c, msg)
	}

	providerID := c.Params("provider_id")
	params := db.UpdateProviderParams{
		Name: req.Name, ContactInfo: req.ContactInfo, Status: req.Status, Notes: req.Notes,
		Slug: req.Slug, Protocol: req.Protocol, Logo: req.Logo, BaseURL: req.BaseURL,
	}
	if req.APIKey != nil {
		if *req.APIKey == "" {
			empty := ""
			params.APIKeyEncrypted = &empty // clears the stored key
		} else {
			enc, encErr := crypto.EncryptString(*req.APIKey, h.EncryptionKey, providerAAD(providerID))
			if encErr != nil {
				h.Log.ErrorContext(c.Context(), "encrypt provider api key", slog.String("error", encErr.Error()))
				return apierror.InternalError(c, "failed to store api key")
			}
			params.APIKeyEncrypted = &enc
		}
	}

	updated, err := h.DB.UpdateProvider(c.Context(), providerID, params)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "provider not found")
		}
		if errors.Is(err, db.ErrConflict) {
			return apierror.Send(c, fiber.StatusConflict, "conflict", "slug already in use")
		}
		h.Log.ErrorContext(c.Context(), "update provider", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to update provider")
	}
	return c.JSON(providerToJSON(updated))
}

// DeleteProvider handles DELETE /api/v1/providers/:provider_id (system_admin).
func (h *Handler) DeleteProvider(c fiber.Ctx) error {
	if err := h.DB.DeleteProvider(c.Context(), c.Params("provider_id")); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "provider not found")
		}
		h.Log.ErrorContext(c.Context(), "delete provider", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to delete provider")
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ListTopups handles GET /api/v1/topups?status=pending (system_admin) —
// the admin review queue for manual top-ups.
func (h *Handler) ListTopups(c fiber.Ctx) error {
	pg, err := parsePagination(c)
	if err != nil {
		return apierror.BadRequest(c, err.Error())
	}
	status := c.Query("status", "")
	if status != "" && status != "pending" && status != "approved" && status != "rejected" {
		return apierror.BadRequest(c, "invalid status filter")
	}

	reqs, err := h.DB.ListTopupRequests(c.Context(), status, c.Query("user_id", ""), pg.Cursor, pg.Limit+1)
	if err != nil {
		h.Log.ErrorContext(c.Context(), "list topups", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to list top-up requests")
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

// reviewTopupRequestBody is the JSON body for POST /topups/:topup_id/review.
type reviewTopupRequestBody struct {
	Status string `json:"status"` // "approved" | "rejected"
	Note   string `json:"note"`
}

// ReviewTopup handles POST /api/v1/topups/:topup_id/review (system_admin).
// Approval credits the customer's wallet atomically with the ledger entry.
func (h *Handler) ReviewTopup(c fiber.Ctx) error {
	keyInfo := auth.KeyInfoFromCtx(c)
	if keyInfo == nil {
		return apierror.Send(c, fiber.StatusForbidden, "forbidden", "no identity")
	}

	var req reviewTopupRequestBody
	if err := c.Bind().JSON(&req); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}
	if req.Status != "approved" && req.Status != "rejected" {
		return apierror.BadRequest(c, `status must be "approved" or "rejected"`)
	}

	topupID := c.Params("topup_id")
	newBalance, err := h.DB.ReviewTopupRequest(c.Context(), topupID, keyInfo.UserID, req.Status, req.Note)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "top-up request not found or already reviewed")
		}
		h.Log.ErrorContext(c.Context(), "review topup", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to review top-up request")
	}

	// Sync the in-memory wallet cache with the credited balance.
	if req.Status == "approved" && h.Wallet != nil {
		tr, getErr := h.DB.GetTopupRequest(c.Context(), topupID)
		if getErr == nil {
			h.Wallet.SetBalance(tr.UserID, newBalance)
		}
	}

	return c.JSON(fiber.Map{"status": req.Status, "balance": newBalance})
}

// adjustWalletRequestBody is the JSON body for POST /users/:user_id/wallet/adjust.
type adjustWalletRequestBody struct {
	Amount      float64 `json:"amount"` // positive credits, negative debits
	Description string  `json:"description"`
}

// AdjustWallet handles POST /api/v1/users/:user_id/wallet/adjust
// (system_admin) — manual balance correction, positive or negative.
func (h *Handler) AdjustWallet(c fiber.Ctx) error {
	var req adjustWalletRequestBody
	if err := c.Bind().JSON(&req); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}
	if req.Amount == 0 {
		return apierror.BadRequest(c, "amount must be non-zero")
	}

	userID := c.Params("user_id")
	newBalance, err := h.DB.ApplyTransaction(c.Context(), db.ApplyTransactionParams{
		UserID:      userID,
		Type:        "adjustment",
		Amount:      req.Amount,
		Description: req.Description,
	})
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "user has no wallet")
		}
		h.Log.ErrorContext(c.Context(), "adjust wallet", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to adjust wallet")
	}
	if h.Wallet != nil {
		h.Wallet.SetBalance(userID, newBalance)
	}
	return c.JSON(fiber.Map{"balance": newBalance})
}

// GetUserWallet handles GET /api/v1/users/:user_id/wallet (system_admin).
func (h *Handler) GetUserWallet(c fiber.Ctx) error {
	w, err := h.DB.GetWalletByUser(c.Context(), c.Params("user_id"))
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "user has no wallet")
		}
		h.Log.ErrorContext(c.Context(), "get user wallet", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to load wallet")
	}
	return c.JSON(walletResponse{Balance: w.Balance, Currency: w.Currency})
}
