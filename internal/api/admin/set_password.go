package admin

import (
	"errors"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
	"golang.org/x/crypto/bcrypt"

	"github.com/voidmind-io/voidllm/internal/apierror"
	"github.com/voidmind-io/voidllm/internal/audit"
	"github.com/voidmind-io/voidllm/internal/auth"
	"github.com/voidmind-io/voidllm/internal/db"
	"github.com/voidmind-io/voidllm/internal/security"
)

type setPasswordRequest struct {
	NewPassword string `json:"new_password"`
}

// SetOwnPassword handles POST /api/v1/me/password/set for OAuth-only accounts.
func (h *Handler) SetOwnPassword(c fiber.Ctx) error {
	keyInfo := auth.KeyInfoFromCtx(c)
	if keyInfo == nil || keyInfo.UserID == "" {
		return apierror.Send(c, fiber.StatusForbidden, "forbidden", "no user identity on this key")
	}

	sec, err := security.Load(c.Context(), h.DB)
	if err != nil {
		h.Log.ErrorContext(c.Context(), "set password: policy", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to set password")
	}
	if !sec.Password.AllowOAuthSetPassword {
		return apierror.Send(c, fiber.StatusForbidden, "forbidden", "setting a password is not allowed")
	}

	var req setPasswordRequest
	if err := c.Bind().JSON(&req); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}
	minLen := sec.Password.MinLength
	if minLen < 8 {
		minLen = 8
	}
	if len(req.NewPassword) < minLen {
		return apierror.BadRequest(c, "new_password is too short")
	}
	if len(req.NewPassword) > 72 {
		return apierror.BadRequest(c, "new_password must be at most 72 bytes")
	}

	profile, err := h.DB.GetUserSecurityProfile(c.Context(), keyInfo.UserID)
	if err != nil {
		h.Log.ErrorContext(c.Context(), "set password: profile", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to set password")
	}
	if profile.HasPassword {
		return apierror.BadRequest(c, "password is already set; use change password instead")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		h.Log.ErrorContext(c.Context(), "set password: bcrypt", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to set password")
	}

	setLocal := profile.AuthProvider != "local"
	if err := h.DB.SetUserPasswordHash(c.Context(), keyInfo.UserID, string(hash), setLocal); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "user not found")
		}
		h.Log.ErrorContext(c.Context(), "set password: update", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to set password")
	}

	if h.AuditLogger != nil {
		h.AuditLogger.Log(audit.Event{
			Timestamp:    time.Now().UTC(),
			ActorID:      keyInfo.UserID,
			ActorType:    "user",
			ActorKeyID:   keyInfo.ID,
			Action:       "password.set",
			ResourceType: "user",
			ResourceID:   keyInfo.UserID,
			IPAddress:    c.IP(),
			StatusCode:   fiber.StatusOK,
			RequestID:    apierror.RequestIDFromCtx(c),
		})
	}

	return c.SendStatus(fiber.StatusOK)
}