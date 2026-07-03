package admin

import (
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v3"

	"github.com/voidmind-io/voidllm/internal/apierror"
	"github.com/voidmind-io/voidllm/internal/auth"
	"github.com/voidmind-io/voidllm/internal/db"
)

// providerRequest is the JSON body for creating/updating a provider.
type providerRequest struct {
	Name        *string `json:"name"`
	ContactInfo *string `json:"contact_info"`
	Status      *string `json:"status"`
	Notes       *string `json:"notes"`
}

func providerToJSON(p *db.Provider) fiber.Map {
	return fiber.Map{
		"id":           p.ID,
		"name":         p.Name,
		"contact_info": p.ContactInfo,
		"status":       p.Status,
		"notes":        p.Notes,
		"created_at":   p.CreatedAt,
		"updated_at":   p.UpdatedAt,
	}
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
	contact, notes := "", ""
	if req.ContactInfo != nil {
		contact = *req.ContactInfo
	}
	if req.Notes != nil {
		notes = *req.Notes
	}

	provider, err := h.DB.CreateProvider(c.Context(), db.CreateProviderParams{
		Name: *req.Name, ContactInfo: contact, Status: status, Notes: notes,
	})
	if err != nil {
		h.Log.ErrorContext(c.Context(), "create provider", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to create provider")
	}
	return c.Status(fiber.StatusCreated).JSON(providerToJSON(provider))
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

	provider, err := h.DB.UpdateProvider(c.Context(), c.Params("provider_id"), db.UpdateProviderParams{
		Name: req.Name, ContactInfo: req.ContactInfo, Status: req.Status, Notes: req.Notes,
	})
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "provider not found")
		}
		h.Log.ErrorContext(c.Context(), "update provider", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to update provider")
	}
	return c.JSON(providerToJSON(provider))
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
