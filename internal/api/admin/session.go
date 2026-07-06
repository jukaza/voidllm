package admin

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/jukaza/tavo/internal/apierror"
	"github.com/jukaza/tavo/internal/audit"
	"github.com/jukaza/tavo/internal/auth"
	"github.com/jukaza/tavo/internal/db"
	"github.com/jukaza/tavo/pkg/keygen"
)

func (h *Handler) issueUserSession(c fiber.Ctx, userID, auditAction string) (loginResponse, error) {
	ctx := c.Context()

	role, _, err := h.DB.ResolveUserRole(ctx, userID)
	if err != nil {
		h.Log.ErrorContext(ctx, "session: resolve role", slog.String("error", err.Error()))
		return loginResponse{}, fmt.Errorf("authentication failed")
	}

	policy, err := h.loadSessionPolicy(c)
	if err != nil {
		h.Log.ErrorContext(ctx, "session: load policy", slog.String("error", err.Error()))
		return loginResponse{}, fmt.Errorf("authentication failed")
	}

	ip := clientIP(c)
	ua := clientUserAgent(c)

	if !policy.AllowMultiple {
		if err := h.DB.RevokeUserSessions(ctx, userID); err != nil {
			h.Log.ErrorContext(ctx, "session: revoke old sessions", slog.String("error", err.Error()))
		}
	} else {
		// Replace prior sessions from this browser instead of stacking a new row on
		// every re-login (common after dev server restarts).
		if ua != "" {
			revoked, err := h.DB.RevokeUserSessionsByMatchingClient(ctx, userID, ip, ua)
			if err != nil {
				h.Log.ErrorContext(ctx, "session: revoke same-client sessions", slog.String("error", err.Error()))
			} else {
				for _, sessionID := range revoked {
					h.evictSessionFromCache(userID, sessionID)
				}
			}
		}
	}

	if policy.AllowMultiple && policy.MaxConcurrent > 0 {
		// Keep room for the new session.
		if err := h.DB.TrimOldestUserSessions(ctx, userID, policy.MaxConcurrent-1); err != nil {
			h.Log.ErrorContext(ctx, "session: trim sessions", slog.String("error", err.Error()))
		}
	}

	key, err := keygen.Generate(keygen.KeyTypeSession)
	if err != nil {
		h.Log.ErrorContext(ctx, "session: generate key", slog.String("error", err.Error()))
		return loginResponse{}, fmt.Errorf("authentication failed")
	}

	keyHash := keygen.Hash(key, h.HMACSecret)
	ttl := time.Duration(policy.TTLHours) * time.Hour
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	expiresAt := time.Now().UTC().Add(ttl)
	expiresAtStr := expiresAt.Format(time.RFC3339)

	apiKey, err := h.DB.CreateAPIKey(ctx, db.CreateAPIKeyParams{
		KeyHash:   keyHash,
		KeyHint:   keygen.Hint(key),
		KeyType:   keygen.KeyTypeSession,
		Name:      "Login session",
		UserID:    &userID,
		ExpiresAt: &expiresAtStr,
		LoginIP:   &ip,
		UserAgent: &ua,
		CreatedBy: userID,
	})
	if err != nil {
		h.Log.ErrorContext(ctx, "session: create api key", slog.String("error", err.Error()))
		return loginResponse{}, fmt.Errorf("authentication failed")
	}

	h.KeyCache.Set(keyHash, auth.KeyInfo{
		ID:        apiKey.ID,
		KeyType:   keygen.KeyTypeSession,
		Role:      role,
		UserID:    userID,
		Name:      "Login session",
		ExpiresAt: &expiresAt,
	})

	user, err := h.DB.GetUser(ctx, userID)
	if err != nil {
		h.Log.ErrorContext(ctx, "session: get user", slog.String("error", err.Error()))
		return loginResponse{}, fmt.Errorf("authentication failed")
	}

	if h.AuditLogger != nil && auditAction != "" {
		h.AuditLogger.Log(audit.Event{
			Timestamp:    time.Now().UTC(),
			ActorID:      user.ID,
			ActorType:    "user",
			ActorKeyID:   apiKey.ID,
			Action:       auditAction,
			ResourceType: "session",
			ResourceID:   apiKey.ID,
			IPAddress:    c.IP(),
			StatusCode:   fiber.StatusOK,
			RequestID:    apierror.RequestIDFromCtx(c),
		})
	}

	me, err := h.buildMeResponse(c.Context(), user, role)
	if err != nil {
		h.Log.ErrorContext(ctx, "session: build me", slog.String("error", err.Error()))
		return loginResponse{}, fmt.Errorf("authentication failed")
	}

	return loginResponse{
		Token:     key,
		ExpiresAt: expiresAtStr,
		User:      me,
	}, nil
}