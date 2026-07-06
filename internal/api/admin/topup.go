package admin

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/big"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/jukaza/tavo/internal/apierror"
	"github.com/jukaza/tavo/internal/auth"
	"github.com/jukaza/tavo/internal/db"
	"github.com/jukaza/tavo/internal/payment"
)

var orderLocks sync.Map
var userTopupLocks sync.Map

type refCountedMutex struct {
	mu       sync.Mutex
	refCount int
}

var createOrderLock sync.Mutex

func lockRefCounted(store *sync.Map, key string) {
	createOrderLock.Lock()
	defer createOrderLock.Unlock()
	var rcm *refCountedMutex
	if v, ok := store.Load(key); ok {
		rcm = v.(*refCountedMutex)
	} else {
		rcm = &refCountedMutex{}
		store.Store(key, rcm)
	}
	rcm.refCount++
	rcm.mu.Lock()
}

func unlockRefCounted(store *sync.Map, key string) {
	createOrderLock.Lock()
	defer createOrderLock.Unlock()
	v, ok := store.Load(key)
	if !ok {
		return
	}
	rcm := v.(*refCountedMutex)
	rcm.mu.Unlock()
	rcm.refCount--
	if rcm.refCount <= 0 {
		store.Delete(key)
	}
}

func lockOrder(tradeNo string)  { lockRefCounted(&orderLocks, tradeNo) }
func unlockOrder(tradeNo string) { unlockRefCounted(&orderLocks, tradeNo) }

// lockUserTopup serializes bonus recomputation and wallet credit per user.
func lockUserTopup(userID string)  { lockRefCounted(&userTopupLocks, userID) }
func unlockUserTopup(userID string) { unlockRefCounted(&userTopupLocks, userID) }

type topupAmountBody struct {
	Amount float64 `json:"amount"`
}

type sepayOrderResponse struct {
	TradeNo       string  `json:"trade_no"`
	PayAmount     float64 `json:"pay_amount"`
	CreditAmount  float64 `json:"credit_amount"`
	BonusAmount   float64 `json:"bonus_amount"`
	BankCode      string  `json:"bank_code"`
	BankName      string  `json:"bank_name"`
	AccountNumber string  `json:"account_number"`
	AccountName   string  `json:"account_name"`
	QRURL         string  `json:"qr_url"`
	ExpiresAt     string  `json:"expires_at"`
	Status        string  `json:"status"`
}

// GetPublicTopupConfig handles GET /api/v1/public/topup-config.
func (h *Handler) GetPublicTopupConfig(c fiber.Ctx) error {
	cfg, err := payment.Load(c.Context(), h.DB)
	if err != nil {
		h.Log.ErrorContext(c.Context(), "topup config", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to load top-up config")
	}
	hasCompleted := false
	if keyInfo := auth.KeyInfoFromCtx(c); keyInfo != nil && keyInfo.UserID != "" {
		hasCompleted, err = h.DB.HasCompletedTopup(c.Context(), keyInfo.UserID)
		if err != nil {
			h.Log.ErrorContext(c.Context(), "topup config: prior topups", slog.String("error", err.Error()))
		}
	}
	return c.JSON(payment.PublicFromConfig(cfg, time.Now(), hasCompleted))
}

// QuoteTopup handles POST /api/v1/me/topups/quote.
func (h *Handler) QuoteTopup(c fiber.Ctx) error {
	keyInfo := auth.KeyInfoFromCtx(c)
	if keyInfo == nil || keyInfo.UserID == "" {
		return apierror.Send(c, fiber.StatusForbidden, "forbidden", "no user identity on this key")
	}
	var req topupAmountBody
	if err := c.Bind().JSON(&req); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}
	cfg, err := payment.Load(c.Context(), h.DB)
	if err != nil {
		return apierror.InternalError(c, "failed to load payment settings")
	}
	if err := validatePayAmount(cfg, req.Amount); err != nil {
		return apierror.BadRequest(c, err.Error())
	}
	hasCompleted, err := h.DB.HasCompletedTopup(c.Context(), keyInfo.UserID)
	if err != nil {
		return apierror.InternalError(c, "failed to check top-up history")
	}
	return c.JSON(payment.ComputeQuote(cfg, req.Amount, time.Now(), hasCompleted))
}

// CreateMyTopup handles POST /api/v1/me/topups — creates a SePay order.
func (h *Handler) CreateMyTopup(c fiber.Ctx) error {
	keyInfo := auth.KeyInfoFromCtx(c)
	if keyInfo == nil || keyInfo.UserID == "" {
		return apierror.Send(c, fiber.StatusForbidden, "forbidden", "no user identity on this key")
	}

	var req topupAmountBody
	if err := c.Bind().JSON(&req); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}

	cfg, err := payment.Load(c.Context(), h.DB)
	if err != nil {
		return apierror.InternalError(c, "failed to load payment settings")
	}
	if !cfg.Sepay.Enabled {
		return apierror.Send(c, fiber.StatusServiceUnavailable, "topup_disabled", "automatic top-up is disabled")
	}
	if !cfg.Sepay.IsConfigured() {
		return apierror.Send(c, fiber.StatusServiceUnavailable, "topup_not_configured", "payment gateway is not configured")
	}
	if err := validatePayAmount(cfg, req.Amount); err != nil {
		return apierror.BadRequest(c, err.Error())
	}

	hasCompleted, err := h.DB.HasCompletedTopup(c.Context(), keyInfo.UserID)
	if err != nil {
		return apierror.InternalError(c, "failed to check top-up history")
	}
	quote := payment.ComputeQuote(cfg, req.Amount, time.Now(), hasCompleted)
	tradeNo := generateTradeNo(keyInfo.UserID)
	ttl := cfg.Sepay.OrderTTLMinutes
	if ttl <= 0 {
		ttl = 15
	}
	expiresAt := time.Now().Add(time.Duration(ttl) * time.Minute).UTC().Format(time.RFC3339)

	tr, err := h.DB.CreateSepayTopupOrder(c.Context(), db.CreateSepayTopupParams{
		UserID:       keyInfo.UserID,
		TradeNo:      tradeNo,
		PayAmount:    quote.PayAmount,
		CreditAmount: quote.CreditAmount,
		BonusAmount:  quote.BonusAmount,
		BonusDetail:  payment.MarshalBonusDetail(quote.BonusDetail),
		ExpiresAt:    expiresAt,
	})
	if err != nil {
		h.Log.ErrorContext(c.Context(), "create sepay topup", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to create top-up order")
	}

	qrURL := buildVietQRURL(cfg.Sepay, quote.PayAmount, tradeNo)
	return c.Status(fiber.StatusCreated).JSON(sepayOrderResponse{
		TradeNo:       tr.TradeNo,
		PayAmount:     quote.PayAmount,
		CreditAmount:  quote.CreditAmount,
		BonusAmount:   quote.BonusAmount,
		BankCode:      cfg.Sepay.BankCode,
		BankName:      bankDisplayName(cfg.Sepay.BankCode),
		AccountNumber: cfg.Sepay.AccountNumber,
		AccountName:   cfg.Sepay.AccountName,
		QRURL:         qrURL,
		ExpiresAt:     expiresAt,
		Status:        tr.Status,
	})
}

// GetTopupStatus handles GET /api/v1/me/topups/:trade_no/status.
func (h *Handler) GetTopupStatus(c fiber.Ctx) error {
	keyInfo := auth.KeyInfoFromCtx(c)
	if keyInfo == nil || keyInfo.UserID == "" {
		return apierror.Send(c, fiber.StatusForbidden, "forbidden", "no user identity on this key")
	}
	tradeNo := c.Params("trade_no")
	tr, err := h.DB.ExpireTopupIfNeeded(c.Context(), tradeNo)
	if err != nil {
		if isNotFound(err) {
			return apierror.NotFound(c, "top-up order not found")
		}
		return apierror.InternalError(c, "failed to load top-up order")
	}
	if tr.UserID != keyInfo.UserID {
		return apierror.NotFound(c, "top-up order not found")
	}
	return c.JSON(fiber.Map{
		"status":        tr.Status,
		"trade_no":      tr.TradeNo,
		"pay_amount":    tr.PayAmount,
		"credit_amount": tr.CreditAmount,
		"bonus_amount":  tr.BonusAmount,
	})
}

// SepayWebhook handles POST /api/v1/webhooks/sepay — public SePay callback.
// SePay expects HTTP 200/201 with {"success": true} within 30 seconds.
func (h *Handler) SepayWebhook(c fiber.Ctx) error {
	cfg, err := payment.Load(c.Context(), h.DB)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"success": false})
	}
	if !cfg.Sepay.Enabled {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"success": false, "message": "top-up disabled"})
	}
	if !cfg.Sepay.WebhookAuthConfigured() {
		h.Log.WarnContext(c.Context(), "sepay webhook rejected: auth not configured",
			slog.String("client_ip", c.IP()),
		)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"success": false, "message": "webhook not configured"})
	}
	if cfg.Sepay.WebhookIPCheck && !payment.IsSepayWebhookIP(c.IP()) {
		h.Log.WarnContext(c.Context(), "sepay webhook rejected: ip not allowed",
			slog.String("client_ip", c.IP()),
		)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"success": false, "message": "ip not allowed"})
	}

	rawBody := c.Body()
	if err := payment.VerifyWebhookAuth(cfg.Sepay, payment.SepayWebhookAuthHeaders{
		Authorization: c.Get("Authorization"),
		Signature:     c.Get("X-SePay-Signature"),
		Timestamp:     c.Get("X-SePay-Timestamp"),
	}, rawBody, time.Now()); err != nil {
		h.Log.WarnContext(c.Context(), "sepay webhook unauthorized",
			slog.String("client_ip", c.IP()),
			slog.String("error", err.Error()),
		)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"success": false, "message": "Unauthorized"})
	}

	var payload payment.SepayWebhookPayload
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"success": false, "error": err.Error()})
	}

	h.Log.InfoContext(c.Context(), "sepay webhook received",
		slog.Int64("transaction_id", payload.ID),
		slog.String("gateway", payload.Gateway),
		slog.String("transfer_type", payload.TransferType),
		slog.Float64("amount", payload.TransferAmount),
		slog.String("content", payload.Content),
	)

	if payload.TransferType != "" && payload.TransferType != "in" {
		return c.JSON(fiber.Map{"success": true, "message": "ignored non-incoming transfer"})
	}

	sepayTxID := fmt.Sprintf("%d", payload.ID)
	if payload.ID > 0 {
		if existing, dupErr := h.DB.GetCompletedTopupBySepayTxID(c.Context(), sepayTxID); dupErr == nil && existing != nil {
			h.Log.InfoContext(c.Context(), "sepay webhook duplicate transaction id",
				slog.String("sepay_tx_id", sepayTxID),
				slog.String("trade_no", existing.TradeNo),
			)
			return c.JSON(fiber.Map{"success": true})
		}
	}

	if cfg.Sepay.AccountNumber != "" && payload.AccountNumber != "" &&
		!payment.AccountNumberMatches(cfg.Sepay.AccountNumber, payload.AccountNumber) {
		h.Log.WarnContext(c.Context(), "sepay webhook account mismatch",
			slog.String("expected", cfg.Sepay.AccountNumber),
			slog.String("received", payload.AccountNumber),
		)
		return c.JSON(fiber.Map{"success": false, "message": "account mismatch"})
	}

	tradeNo := payment.ExtractTradeNo(payload.Content, payload.Code)
	if tradeNo == "" {
		return c.JSON(fiber.Map{"success": false, "message": "trade number not found"})
	}

	lockOrder(tradeNo)
	defer unlockOrder(tradeNo)

	tr, err := h.DB.ExpireTopupIfNeeded(c.Context(), tradeNo)
	if err != nil {
		if isNotFound(err) {
			h.Log.WarnContext(c.Context(), "sepay webhook unknown order", slog.String("trade_no", tradeNo))
			return c.JSON(fiber.Map{"success": false, "message": "order not found"})
		}
		return c.JSON(fiber.Map{"success": false, "message": "database error"})
	}
	if tr.Status == "completed" {
		return c.JSON(fiber.Map{"success": true})
	}
	if tr.Status != "pending" && tr.Status != "expired" {
		return c.JSON(fiber.Map{"success": false, "message": "order not completable"})
	}
	if math.Abs(payload.TransferAmount-tr.PayAmount) > 1.0 {
		h.Log.WarnContext(c.Context(), "sepay amount mismatch",
			slog.String("trade_no", tradeNo),
			slog.Float64("expected", tr.PayAmount),
			slog.Float64("paid", payload.TransferAmount),
		)
		return c.JSON(fiber.Map{"success": false, "message": "amount mismatch"})
	}

	lockUserTopup(tr.UserID)
	defer unlockUserTopup(tr.UserID)

	quote, err := resolveTopupCredit(c.Context(), h.DB, cfg, tr)
	if err != nil {
		h.Log.ErrorContext(c.Context(), "sepay resolve credit", slog.String("error", err.Error()))
		return c.JSON(fiber.Map{"success": false, "message": "failed to compute credit"})
	}

	newBalance, err := h.DB.CompleteSepayTopup(
		c.Context(), tradeNo, sepayTxID, payload.TransferAmount,
		quote.CreditAmount, quote.BonusAmount, payment.MarshalBonusDetail(quote.BonusDetail),
	)
	if err != nil {
		if isNotFound(err) {
			return c.JSON(fiber.Map{"success": false, "message": "order already processed"})
		}
		if errors.Is(err, db.ErrAmountMismatch) {
			return c.JSON(fiber.Map{"success": false, "message": "amount mismatch"})
		}
		h.Log.ErrorContext(c.Context(), "sepay complete topup", slog.String("error", err.Error()))
		return c.JSON(fiber.Map{"success": false, "message": "failed to credit wallet"})
	}
	if h.Wallet != nil {
		h.Wallet.SetBalance(tr.UserID, newBalance)
	}

	h.Log.InfoContext(c.Context(), "sepay topup completed",
		slog.String("trade_no", tradeNo),
		slog.String("user_id", tr.UserID),
		slog.Float64("credit_amount", quote.CreditAmount),
	)
	return c.JSON(fiber.Map{"success": true})
}

func resolveTopupCredit(ctx context.Context, database *db.DB, cfg payment.Config, tr *db.TopupRequest) (payment.Quote, error) {
	hasCompleted, err := database.HasCompletedTopupExcluding(ctx, tr.UserID, tr.ID)
	if err != nil {
		return payment.Quote{}, err
	}
	return payment.ComputeQuote(cfg, tr.PayAmount, time.Now(), hasCompleted), nil
}

func validatePayAmount(cfg payment.Config, amount float64) error {
	if amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}
	if amount < cfg.Sepay.MinAmount {
		return fmt.Errorf("amount must be at least %.0f", cfg.Sepay.MinAmount)
	}
	if cfg.Sepay.MaxAmount > 0 && amount > cfg.Sepay.MaxAmount {
		return fmt.Errorf("amount exceeds maximum %.0f", cfg.Sepay.MaxAmount)
	}
	return nil
}

func generateTradeNo(userID string) string {
	shortUser := strings.ReplaceAll(userID, "-", "")
	if len(shortUser) > 8 {
		shortUser = shortUser[:8]
	}
	randPart := randomAlphaNum(6)
	return fmt.Sprintf("VL%sNO%s%d", shortUser, randPart, time.Now().Unix()%1000000)
}

func randomAlphaNum(n int) string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	out := make([]byte, n)
	for i := range out {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			out[i] = alphabet[i%len(alphabet)]
			continue
		}
		out[i] = alphabet[idx.Int64()]
	}
	return string(out)
}

func buildVietQRURL(sepay payment.SepayConfig, amount float64, tradeNo string) string {
	return fmt.Sprintf(
		"https://img.vietqr.io/image/%s-%s-compact2.png?amount=%.0f&addInfo=%s&accountName=%s",
		url.PathEscape(sepay.BankCode),
		url.PathEscape(sepay.AccountNumber),
		amount,
		url.QueryEscape(tradeNo),
		url.QueryEscape(sepay.AccountName),
	)
}

func bankDisplayName(code string) string {
	for _, bank := range payment.SupportedBanks {
		if bank.Code == code {
			return bank.Name
		}
	}
	return code
}

func isNotFound(err error) bool {
	return errors.Is(err, db.ErrNotFound) || (err != nil && strings.Contains(err.Error(), db.ErrNotFound.Error()))
}

func topupToJSON(t *db.TopupRequest) fiber.Map {
	out := fiber.Map{
		"id":            t.ID,
		"user_id":       t.UserID,
		"amount":        t.Amount,
		"payment_ref":   t.PaymentRef,
		"status":        t.Status,
		"reviewed_by":   t.ReviewedBy,
		"reviewed_at":   t.ReviewedAt,
		"note":          t.Note,
		"created_at":    t.CreatedAt,
		"trade_no":      t.TradeNo,
		"pay_amount":    t.PayAmount,
		"credit_amount": t.CreditAmount,
		"bonus_amount":  t.BonusAmount,
		"expires_at":    t.ExpiresAt,
		"completed_at":  t.CompletedAt,
		"sepay_tx_id":   t.SepayTxID,
	}
	if strings.TrimSpace(t.BonusDetail) != "" {
		var detail any
		if err := json.Unmarshal([]byte(t.BonusDetail), &detail); err == nil {
			out["bonus_detail"] = detail
		}
	}
	return out
}