package admin

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/voidmind-io/voidllm/internal/apierror"
	"github.com/voidmind-io/voidllm/internal/audit"
	"github.com/voidmind-io/voidllm/internal/auth"
	"github.com/voidmind-io/voidllm/internal/db"
	"github.com/voidmind-io/voidllm/internal/security"
	totplib "github.com/voidmind-io/voidllm/internal/totp"
)

const totpPendingTTL = 10 * time.Minute

type twoFASetupResponse struct {
	Secret     string `json:"secret"`
	OTPAuthURL string `json:"otpauth_url"`
}

type twoFAVerifyRequest struct {
	Code string `json:"code"`
}

type twoFAVerifyResponse struct {
	BackupCodes []string `json:"backup_codes"`
}

type twoFADisableRequest struct {
	Password string `json:"password"`
	Code     string `json:"code"`
}

type totpPendingPayload struct {
	Secret string `json:"secret"`
}

var (
	totpPendingMu sync.Mutex
	totpPending   = map[string]totpPendingPayload{}
)

func (h *Handler) storeTOTPPending(c fiber.Ctx, userID, secret string) error {
	payload, err := json.Marshal(totpPendingPayload{Secret: secret})
	if err != nil {
		return err
	}
	if h.Redis != nil {
		return h.Redis.SetTOTPPending(c.Context(), userID, payload, totpPendingTTL)
	}
	totpPendingMu.Lock()
	totpPending[userID] = totpPendingPayload{Secret: secret}
	totpPendingMu.Unlock()
	return nil
}

func (h *Handler) loadTOTPPending(c fiber.Ctx, userID string) (string, bool) {
	if h.Redis != nil {
		payload, ok, err := h.Redis.GetTOTPPending(c.Context(), userID)
		if err != nil || !ok {
			return "", false
		}
		var p totpPendingPayload
		if err := json.Unmarshal(payload, &p); err != nil {
			return "", false
		}
		return p.Secret, true
	}
	totpPendingMu.Lock()
	defer totpPendingMu.Unlock()
	p, ok := totpPending[userID]
	if !ok {
		return "", false
	}
	return p.Secret, true
}

func (h *Handler) clearTOTPPending(c fiber.Ctx, userID string) {
	if h.Redis != nil {
		_ = h.Redis.DeleteTOTPPending(c.Context(), userID)
		return
	}
	totpPendingMu.Lock()
	delete(totpPending, userID)
	totpPendingMu.Unlock()
}

func (h *Handler) twoFAAvailable(ctx context.Context) (bool, error) {
	cfg, err := security.Load(ctx, h.DB)
	if err != nil {
		return false, err
	}
	return cfg.TwoFA.AllowUserEnable, nil
}

// SetupTwoFA handles POST /api/v1/me/2fa/setup.
func (h *Handler) SetupTwoFA(c fiber.Ctx) error {
	keyInfo := auth.KeyInfoFromCtx(c)
	if keyInfo == nil || keyInfo.UserID == "" {
		return apierror.Send(c, fiber.StatusForbidden, "forbidden", "no user identity on this key")
	}

	available, err := h.twoFAAvailable(c.Context())
	if err != nil {
		h.Log.ErrorContext(c.Context(), "2fa setup: policy", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to start 2FA setup")
	}
	if !available {
		return apierror.Send(c, fiber.StatusForbidden, "forbidden", "two-factor authentication is not available")
	}

	enabled, err := h.DB.UserHasTOTPEnabled(c.Context(), keyInfo.UserID)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		h.Log.ErrorContext(c.Context(), "2fa setup: status", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to start 2FA setup")
	}
	if enabled {
		return apierror.BadRequest(c, "two-factor authentication is already enabled")
	}

	user, err := h.DB.GetUser(c.Context(), keyInfo.UserID)
	if err != nil {
		return apierror.InternalError(c, "failed to start 2FA setup")
	}

	setup, err := totplib.GenerateSetup(user.Email)
	if err != nil {
		h.Log.ErrorContext(c.Context(), "2fa setup: generate", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to start 2FA setup")
	}
	if err := h.storeTOTPPending(c, keyInfo.UserID, setup.Secret); err != nil {
		h.Log.ErrorContext(c.Context(), "2fa setup: store pending", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to start 2FA setup")
	}

	return c.JSON(twoFASetupResponse{
		Secret:     setup.Secret,
		OTPAuthURL: setup.OTPAuthURL,
	})
}

// VerifyTwoFA handles POST /api/v1/me/2fa/verify.
func (h *Handler) VerifyTwoFA(c fiber.Ctx) error {
	keyInfo := auth.KeyInfoFromCtx(c)
	if keyInfo == nil || keyInfo.UserID == "" {
		return apierror.Send(c, fiber.StatusForbidden, "forbidden", "no user identity on this key")
	}

	var req twoFAVerifyRequest
	if err := c.Bind().JSON(&req); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}
	if req.Code == "" {
		return apierror.BadRequest(c, "code is required")
	}

	secret, ok := h.loadTOTPPending(c, keyInfo.UserID)
	if !ok || secret == "" {
		return apierror.BadRequest(c, "no pending 2FA setup; start setup again")
	}
	if !totplib.Validate(secret, req.Code) {
		return apierror.BadRequest(c, "invalid verification code")
	}

	encrypted, err := totplib.EncryptSecret(secret, h.EncryptionKey, keyInfo.UserID)
	if err != nil {
		h.Log.ErrorContext(c.Context(), "2fa verify: encrypt", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to enable 2FA")
	}
	if err := h.DB.EnableUserTOTP(c.Context(), keyInfo.UserID, encrypted); err != nil {
		h.Log.ErrorContext(c.Context(), "2fa verify: enable", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to enable 2FA")
	}

	plain, hashes, err := totplib.GenerateBackupCodes()
	if err != nil {
		h.Log.ErrorContext(c.Context(), "2fa verify: backup codes", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to enable 2FA")
	}
	if err := h.DB.DeleteTOTPBackupCodes(c.Context(), keyInfo.UserID); err != nil {
		h.Log.ErrorContext(c.Context(), "2fa verify: clear backup", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to enable 2FA")
	}
	for _, hash := range hashes {
		if err := h.DB.InsertTOTPBackupCode(c.Context(), keyInfo.UserID, hash); err != nil {
			h.Log.ErrorContext(c.Context(), "2fa verify: insert backup", slog.String("error", err.Error()))
			return apierror.InternalError(c, "failed to enable 2FA")
		}
	}

	h.clearTOTPPending(c, keyInfo.UserID)
	h.auditTwoFA(c, keyInfo, "2fa.enabled")

	return c.JSON(twoFAVerifyResponse{BackupCodes: plain})
}

// DisableTwoFA handles DELETE /api/v1/me/2fa.
func (h *Handler) DisableTwoFA(c fiber.Ctx) error {
	keyInfo := auth.KeyInfoFromCtx(c)
	if keyInfo == nil || keyInfo.UserID == "" {
		return apierror.Send(c, fiber.StatusForbidden, "forbidden", "no user identity on this key")
	}

	var req twoFADisableRequest
	if err := c.Bind().JSON(&req); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}
	if req.Password == "" && req.Code == "" {
		return apierror.BadRequest(c, "password or verification code is required")
	}

	ctx := c.Context()
	authorized := false
	if req.Password != "" {
		if err := h.verifyUserPassword(ctx, keyInfo.UserID, req.Password); err == nil {
			authorized = true
		}
	}
	if !authorized && req.Code != "" {
		ok, err := h.verifyUserTOTPCode(ctx, keyInfo.UserID, req.Code)
		if err != nil {
			h.Log.ErrorContext(ctx, "2fa disable: verify", slog.String("error", err.Error()))
			return apierror.InternalError(c, "failed to disable 2FA")
		}
		authorized = ok
	}
	if !authorized {
		return apierror.Send(c, fiber.StatusBadRequest, "bad_request", "invalid credentials")
	}

	if err := h.DB.DisableUserTOTP(ctx, keyInfo.UserID); err != nil {
		h.Log.ErrorContext(ctx, "2fa disable", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to disable 2FA")
	}
	_ = h.DB.DeleteTOTPBackupCodes(ctx, keyInfo.UserID)
	h.clearTOTPPending(c, keyInfo.UserID)
	h.auditTwoFA(c, keyInfo, "2fa.disabled")
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) auditTwoFA(c fiber.Ctx, keyInfo *auth.KeyInfo, action string) {
	if h.AuditLogger == nil {
		return
	}
	h.AuditLogger.Log(audit.Event{
		Timestamp:  time.Now().UTC(),
		ActorID:    keyInfo.UserID,
		ActorType:  "user",
		ActorKeyID: keyInfo.ID,
		Action:     action,
		ResourceType: "user",
		ResourceID: keyInfo.UserID,
		IPAddress:  c.IP(),
		StatusCode: fiber.StatusOK,
		RequestID:  apierror.RequestIDFromCtx(c),
	})
}