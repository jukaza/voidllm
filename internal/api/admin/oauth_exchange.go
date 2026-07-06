package admin

import (
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/jukaza/tavo/internal/apierror"
)

const oauthExchangeTTL = 2 * time.Minute

type oauthExchangeEntry struct {
	Token     string
	ExpiresAt string
	User      meResponse
	CreatedAt time.Time
}

var (
	oauthExchangeMu sync.Mutex
	oauthExchanges  = map[string]oauthExchangeEntry{}
)

func pruneOAuthExchanges() {
	now := time.Now().UTC()
	for code, entry := range oauthExchanges {
		if now.Sub(entry.CreatedAt) > oauthExchangeTTL {
			delete(oauthExchanges, code)
		}
	}
}

func (h *Handler) storeOAuthExchange(c fiber.Ctx, sess loginResponse) (string, error) {
	code, err := randomNonce()
	if err != nil {
		return "", err
	}
	entry := oauthExchangeEntry{
		Token:     sess.Token,
		ExpiresAt: sess.ExpiresAt,
		User:      sess.User,
		CreatedAt: time.Now().UTC(),
	}

	if h.Redis != nil {
		payload, err := json.Marshal(entry)
		if err != nil {
			return "", err
		}
		if err := h.Redis.SetOAuthExchange(c.Context(), code, payload, oauthExchangeTTL); err != nil {
			return "", err
		}
		return code, nil
	}

	oauthExchangeMu.Lock()
	pruneOAuthExchanges()
	oauthExchanges[code] = entry
	oauthExchangeMu.Unlock()
	return code, nil
}

func (h *Handler) consumeOAuthExchange(c fiber.Ctx, code string) (oauthExchangeEntry, bool) {
	if h.Redis != nil {
		payload, ok, err := h.Redis.ConsumeOAuthExchange(c.Context(), code)
		if err != nil || !ok {
			return oauthExchangeEntry{}, false
		}
		var entry oauthExchangeEntry
		if err := json.Unmarshal(payload, &entry); err != nil {
			return oauthExchangeEntry{}, false
		}
		return entry, true
	}

	oauthExchangeMu.Lock()
	defer oauthExchangeMu.Unlock()
	entry, ok := oauthExchanges[code]
	if !ok {
		return oauthExchangeEntry{}, false
	}
	delete(oauthExchanges, code)
	if time.Since(entry.CreatedAt) > oauthExchangeTTL {
		return oauthExchangeEntry{}, false
	}
	return entry, true
}

// OAuthExchange handles POST /api/v1/auth/oauth/exchange.
func (h *Handler) OAuthExchange(c fiber.Ctx) error {
	var req struct {
		Code string `json:"code"`
	}
	if err := c.Bind().JSON(&req); err != nil || strings.TrimSpace(req.Code) == "" {
		return apierror.BadRequest(c, "code is required")
	}
	entry, ok := h.consumeOAuthExchange(c, strings.TrimSpace(req.Code))
	if !ok {
		return apierror.Send(c, fiber.StatusUnauthorized, "unauthorized", "invalid or expired code")
	}
	return c.JSON(loginResponse{
		Token:     entry.Token,
		ExpiresAt: entry.ExpiresAt,
		User:      entry.User,
	})
}