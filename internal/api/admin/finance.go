package admin

import (
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/voidmind-io/voidllm/internal/apierror"
	"github.com/voidmind-io/voidllm/internal/auth"
	"github.com/voidmind-io/voidllm/internal/db"
	"github.com/voidmind-io/voidllm/internal/payment"
)

const financeTZNote = "Asia/Ho_Chi_Minh (UTC+7)"

// parseFinanceRangeRequired parses from/to query params; both are required.
func parseFinanceRangeRequired(c fiber.Ctx) (from, to time.Time, ok bool) {
	return parseFinanceRange(c, true)
}

// parseFinanceRangeOptional defaults to the last 90 days when from/to are omitted.
func parseFinanceRangeOptional(c fiber.Ctx) (from, to time.Time, ok bool) {
	return parseFinanceRange(c, false)
}

func parseFinanceRange(c fiber.Ctx, required bool) (from, to time.Time, ok bool) {
	fromStr := c.Query("from")
	toStr := c.Query("to")

	if fromStr == "" || toStr == "" {
		if required {
			_ = apierror.BadRequest(c, "from and to are required")
			return
		}
		to = time.Now().UTC()
		from = to.Add(-maxUsageRangeDays * 24 * time.Hour)
		ok = true
		return
	}

	var err error
	from, err = time.Parse(time.RFC3339, fromStr)
	if err != nil {
		_ = apierror.BadRequest(c, "from must be a valid RFC3339 timestamp")
		return
	}
	to, err = time.Parse(time.RFC3339, toStr)
	if err != nil {
		_ = apierror.BadRequest(c, "to must be a valid RFC3339 timestamp")
		return
	}
	if !from.Before(to) {
		_ = apierror.BadRequest(c, "from must be before to")
		return
	}
	if to.Sub(from) > maxUsageRangeDays*24*time.Hour {
		_ = apierror.BadRequest(c, "time range must not exceed 90 days")
		return
	}
	ok = true
	return
}

// FinanceSummary handles GET /api/v1/admin/finance/summary.
func (h *Handler) FinanceSummary(c fiber.Ctx) error {
	from, to, ok := parseFinanceRangeRequired(c)
	if !ok {
		return nil
	}

	summary, daily, err := h.DB.GetFinanceSummary(c.Context(), from, to)
	if err != nil {
		h.Log.ErrorContext(c.Context(), "finance summary", "error", err.Error())
		return apierror.InternalError(c, "failed to load finance summary")
	}

	dailyOut := make([]fiber.Map, len(daily))
	for i, b := range daily {
		dailyOut[i] = fiber.Map{
			"day":            b.Day,
			"topup_inflow":   b.TopupInflow,
			"usage_outflow":  b.UsageOutflow,
			"adjustment_net": b.AdjustmentNet,
			"refund_total":   b.RefundTotal,
			"net_flow":       b.NetFlow,
		}
	}

	return c.JSON(fiber.Map{
		"from":     from.UTC().Format(time.RFC3339),
		"to":       to.UTC().Format(time.RFC3339),
		"timezone": financeTZNote,
		"currency": "VND",
		"totals": fiber.Map{
			"wallet_liability":      summary.WalletLiability,
			"topup_inflow":          summary.TopupInflow,
			"topup_pay_amount":      summary.TopupPayAmount,
			"topup_bonus":           summary.TopupBonus,
			"usage_outflow":         summary.UsageOutflow,
			"adjustment_net":        summary.AdjustmentNet,
			"refund_total":          summary.RefundTotal,
			"pending_topup_count":   summary.PendingTopupCount,
			"completed_topup_count": summary.CompletedTopupCount,
		},
		"daily": dailyOut,
	})
}

// FinanceTopups handles GET /api/v1/admin/finance/topups.
func (h *Handler) FinanceTopups(c fiber.Ctx) error {
	status := c.Query("status", "")
	if status != "" && status != "pending" && status != "completed" && status != "expired" && status != "failed" {
		return apierror.BadRequest(c, "invalid status filter")
	}

	pg, err := parsePagination(c)
	if err != nil {
		return apierror.BadRequest(c, err.Error())
	}

	skipDate := status == "pending"
	var from, to time.Time
	if !skipDate {
		var ok bool
		from, to, ok = parseFinanceRangeOptional(c)
		if !ok {
			return nil
		}
	}

	reqs, err := h.DB.ListTopupRequestsAdmin(c.Context(), db.TopupListFilter{
		From:     from,
		To:       to,
		Status:   status,
		UserID:   c.Query("user_id", ""),
		Cursor:   pg.Cursor,
		Limit:    pg.Limit + 1,
		SkipDate: skipDate,
	})
	if err != nil {
		h.Log.ErrorContext(c.Context(), "finance topups", "error", err.Error())
		return apierror.InternalError(c, "failed to list top-up orders")
	}

	hasMore := len(reqs) > pg.Limit
	if hasMore {
		reqs = reqs[:pg.Limit]
	}
	items := make([]fiber.Map, len(reqs))
	for i := range reqs {
		items[i] = topupWithUserToJSON(&reqs[i])
	}
	cursor := ""
	if hasMore && len(reqs) > 0 {
		cursor = reqs[len(reqs)-1].ID
	}
	return c.JSON(fiber.Map{"data": items, "has_more": hasMore, "cursor": cursor})
}

type reviewFinanceTopupBody struct {
	Action string `json:"action"` // "approve" | "reject"
	Note   string `json:"note"`
}

// FinanceReviewTopup handles POST /api/v1/admin/finance/topups/:topup_id/review.
// Manual fallback when SePay webhook auto-complete fails.
func (h *Handler) FinanceReviewTopup(c fiber.Ctx) error {
	keyInfo := auth.KeyInfoFromCtx(c)
	if keyInfo == nil {
		return apierror.Send(c, fiber.StatusForbidden, "forbidden", "no identity")
	}

	var req reviewFinanceTopupBody
	if err := c.Bind().JSON(&req); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}
	if req.Action != "approve" && req.Action != "reject" {
		return apierror.BadRequest(c, `action must be "approve" or "reject"`)
	}

	topupID := c.Params("topup_id")
	tr, err := h.DB.GetTopupRequest(c.Context(), topupID)
	if err != nil {
		if isNotFound(err) {
			return apierror.NotFound(c, "top-up request not found")
		}
		return apierror.InternalError(c, "failed to load top-up request")
	}

	var creditAmount, bonusAmount float64
	var bonusDetail string
	if req.Action == "approve" {
		lockUserTopup(tr.UserID)
		defer unlockUserTopup(tr.UserID)

		cfg, loadErr := payment.Load(c.Context(), h.DB)
		if loadErr != nil {
			return apierror.InternalError(c, "failed to load payment settings")
		}
		quote, quoteErr := resolveTopupCredit(c.Context(), h.DB, cfg, tr)
		if quoteErr != nil {
			return apierror.InternalError(c, "failed to compute credit")
		}
		creditAmount = quote.CreditAmount
		bonusAmount = quote.BonusAmount
		bonusDetail = payment.MarshalBonusDetail(quote.BonusDetail)
	}

	newBalance, err := h.DB.ManualReviewTopup(c.Context(), topupID, keyInfo.UserID, req.Action, req.Note, creditAmount, bonusAmount, bonusDetail)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "top-up request not found or already processed")
		}
		h.Log.ErrorContext(c.Context(), "finance review topup", "error", err.Error())
		return apierror.InternalError(c, "failed to review top-up request")
	}

	if req.Action == "approve" && h.Wallet != nil {
		tr, getErr := h.DB.GetTopupRequest(c.Context(), topupID)
		if getErr == nil {
			h.Wallet.SetBalance(tr.UserID, newBalance)
		}
	}

	status := "failed"
	if req.Action == "approve" {
		status = "completed"
	}
	return c.JSON(fiber.Map{"status": status, "balance": newBalance})
}

// FinanceTransactions handles GET /api/v1/admin/finance/transactions.
func (h *Handler) FinanceTransactions(c fiber.Ctx) error {
	txType := c.Query("type", "")
	if txType != "" && txType != "topup" && txType != "usage" && txType != "adjustment" && txType != "refund" {
		return apierror.BadRequest(c, "invalid type filter")
	}

	from, to, ok := parseFinanceRangeOptional(c)
	if !ok {
		return nil
	}

	pg, err := parsePagination(c)
	if err != nil {
		return apierror.BadRequest(c, err.Error())
	}

	txs, err := h.DB.ListTransactionsAdmin(c.Context(), db.TransactionListFilter{
		From:   from,
		To:     to,
		Type:   txType,
		UserID: c.Query("user_id", ""),
		Cursor: pg.Cursor,
		Limit:  pg.Limit + 1,
	})
	if err != nil {
		h.Log.ErrorContext(c.Context(), "finance transactions", "error", err.Error())
		return apierror.InternalError(c, "failed to list transactions")
	}

	hasMore := len(txs) > pg.Limit
	if hasMore {
		txs = txs[:pg.Limit]
	}
	items := make([]fiber.Map, len(txs))
	for i := range txs {
		items[i] = transactionWithUserToJSON(&txs[i])
	}
	cursor := ""
	if hasMore && len(txs) > 0 {
		cursor = txs[len(txs)-1].ID
	}
	return c.JSON(fiber.Map{"data": items, "has_more": hasMore, "cursor": cursor})
}

func topupWithUserToJSON(t *db.TopupWithUser) fiber.Map {
	out := topupToJSON(&t.TopupRequest)
	out["user_email"] = t.UserEmail
	out["user_display_name"] = t.UserDisplayName
	return out
}

func transactionWithUserToJSON(t *db.TransactionWithUser) fiber.Map {
	out := fiber.Map{
		"id":                t.ID,
		"user_id":           t.UserID,
		"type":              t.Type,
		"amount":            t.Amount,
		"balance_after":     t.BalanceAfter,
		"ref_id":            t.RefID,
		"description":       t.Description,
		"created_at":        t.CreatedAt,
		"user_email":        t.UserEmail,
		"user_display_name": t.UserDisplayName,
	}
	if t.Type == "usage" && strings.TrimSpace(t.RefID) != "" {
		out["usage_log_url"] = "/analytics/logs?request_id=" + t.RefID
	}
	return out
}

