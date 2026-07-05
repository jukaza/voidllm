package admin

import (
	"errors"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/voidmind-io/voidllm/internal/apierror"
	"github.com/voidmind-io/voidllm/internal/audit"
	"github.com/voidmind-io/voidllm/internal/auth"
	"github.com/voidmind-io/voidllm/internal/db"
	"github.com/voidmind-io/voidllm/pkg/keygen"
)

type sessionItem struct {
	ID          string `json:"id"`
	IP          string `json:"ip"`
	UserAgent   string `json:"user_agent,omitempty"`
	DeviceLabel string `json:"device_label"`
	CreatedAt   string `json:"created_at"`
	LastSeenAt  string `json:"last_seen_at,omitempty"`
	Current     bool   `json:"current"`
}

type sessionsResponse struct {
	Sessions []sessionItem `json:"sessions"`
}

// MeSessions handles GET /api/v1/me/sessions.
func (h *Handler) MeSessions(c fiber.Ctx) error {
	keyInfo := auth.KeyInfoFromCtx(c)
	if keyInfo == nil || keyInfo.UserID == "" {
		return apierror.Send(c, fiber.StatusForbidden, "forbidden", "no user identity on this key")
	}

	rows, err := h.DB.ListUserSessions(c.Context(), keyInfo.UserID)
	if err != nil {
		h.Log.ErrorContext(c.Context(), "me sessions", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to load sessions")
	}

	items := make([]sessionItem, 0, len(rows))
	for _, row := range rows {
		item := sessionItem{
			ID:        row.ID,
			DeviceLabel: "Unknown device",
			CreatedAt: row.CreatedAt,
			Current:   row.ID == keyInfo.ID,
		}
		if row.LoginIP != nil {
			item.IP = *row.LoginIP
		}
		if row.UserAgent != nil {
			item.UserAgent = *row.UserAgent
			item.DeviceLabel = deviceLabelFromUserAgent(*row.UserAgent)
		}
		if row.LastUsedAt != nil {
			item.LastSeenAt = *row.LastUsedAt
		}
		items = append(items, item)
	}
	return c.JSON(sessionsResponse{Sessions: items})
}

// RevokeMeSession handles DELETE /api/v1/me/sessions/:id.
func (h *Handler) RevokeMeSession(c fiber.Ctx) error {
	keyInfo := auth.KeyInfoFromCtx(c)
	if keyInfo == nil || keyInfo.UserID == "" {
		return apierror.Send(c, fiber.StatusForbidden, "forbidden", "no user identity on this key")
	}
	sessionID := c.Params("id")
	if sessionID == "" {
		return apierror.BadRequest(c, "session id is required")
	}
	if sessionID == keyInfo.ID {
		return apierror.BadRequest(c, "cannot revoke the current session")
	}

	if err := h.DB.RevokeUserSessionByID(c.Context(), keyInfo.UserID, sessionID); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return apierror.NotFound(c, "session not found")
		}
		h.Log.ErrorContext(c.Context(), "revoke session", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to revoke session")
	}

	h.evictSessionFromCache(keyInfo.UserID, sessionID)
	h.auditSessionRevoke(c, keyInfo, sessionID, "session.revoked")
	return c.SendStatus(fiber.StatusNoContent)
}

// RevokeOtherMeSessions handles DELETE /api/v1/me/sessions.
func (h *Handler) RevokeOtherMeSessions(c fiber.Ctx) error {
	keyInfo := auth.KeyInfoFromCtx(c)
	if keyInfo == nil || keyInfo.UserID == "" {
		return apierror.Send(c, fiber.StatusForbidden, "forbidden", "no user identity on this key")
	}

	n, err := h.DB.RevokeOtherUserSessions(c.Context(), keyInfo.UserID, keyInfo.ID)
	if err != nil {
		h.Log.ErrorContext(c.Context(), "revoke other sessions", slog.String("error", err.Error()))
		return apierror.InternalError(c, "failed to revoke sessions")
	}

	h.evictOtherSessionsFromCache(keyInfo.UserID, keyInfo.ID)
	if n > 0 {
		h.auditSessionRevoke(c, keyInfo, "", "session.revoked_all")
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) evictSessionFromCache(userID, sessionID string) {
	var toEvict []string
	h.KeyCache.Range(func(keyHash string, ki auth.KeyInfo) bool {
		if ki.UserID == userID && ki.ID == sessionID {
			toEvict = append(toEvict, keyHash)
		}
		return true
	})
	for _, keyHash := range toEvict {
		h.KeyCache.Delete(keyHash)
	}
}

func (h *Handler) evictOtherSessionsFromCache(userID, exceptKeyID string) {
	var toEvict []string
	h.KeyCache.Range(func(keyHash string, ki auth.KeyInfo) bool {
		if ki.UserID == userID && ki.ID != exceptKeyID {
			toEvict = append(toEvict, keyHash)
		}
		return true
	})
	for _, keyHash := range toEvict {
		h.KeyCache.Delete(keyHash)
	}
}

func (h *Handler) auditSessionRevoke(c fiber.Ctx, keyInfo *auth.KeyInfo, sessionID, action string) {
	if h.AuditLogger == nil {
		return
	}
	h.AuditLogger.Log(audit.Event{
		Timestamp:    time.Now().UTC(),
		ActorID:      keyInfo.UserID,
		ActorType:    "user",
		ActorKeyID:   keyInfo.ID,
		Action:       action,
		ResourceType: "session",
		ResourceID:   sessionID,
		IPAddress:    c.IP(),
		StatusCode:   fiber.StatusNoContent,
		RequestID:    apierror.RequestIDFromCtx(c),
	})
}

func (h *Handler) touchSessionIfNeeded(c fiber.Ctx, keyInfo *auth.KeyInfo) {
	if keyInfo == nil || keyInfo.KeyType != keygen.KeyTypeSession {
		return
	}
	_ = h.DB.TouchSessionLastUsed(c.Context(), keyInfo.ID, 5*time.Minute)
}