package admin

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"golang.org/x/crypto/bcrypt"

	"github.com/voidmind-io/voidllm/internal/apierror"
	"github.com/voidmind-io/voidllm/internal/db"
	totplib "github.com/voidmind-io/voidllm/internal/totp"
)

const login2FATTL = 5 * time.Minute

type login2FAChallengeResponse struct {
	Requires2FA bool   `json:"requires_2fa"`
	TempToken   string `json:"temp_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type login2FARequest struct {
	TempToken string `json:"temp_token"`
	Code      string `json:"code"`
}

type login2FAChallenge struct {
	UserID    string
	CreatedAt time.Time
}

var (
	login2FAMu        sync.Mutex
	login2FAChallenges = map[string]login2FAChallenge{}
)

func pruneLogin2FAChallenges() {
	now := time.Now().UTC()
	for token, ch := range login2FAChallenges {
		if now.Sub(ch.CreatedAt) > login2FATTL {
			delete(login2FAChallenges, token)
		}
	}
}

func (h *Handler) createLogin2FAChallenge(c fiber.Ctx, userID string) (login2FAChallengeResponse, error) {
	token, err := randomNonce()
	if err != nil {
		return login2FAChallengeResponse{}, err
	}
	payload, err := json.Marshal(login2FAChallenge{UserID: userID, CreatedAt: time.Now().UTC()})
	if err != nil {
		return login2FAChallengeResponse{}, err
	}

	if h.Redis != nil {
		if err := h.Redis.SetLoginChallenge(c.Context(), token, payload, login2FATTL); err != nil {
			return login2FAChallengeResponse{}, err
		}
	} else {
		login2FAMu.Lock()
		pruneLogin2FAChallenges()
		login2FAChallenges[token] = login2FAChallenge{UserID: userID, CreatedAt: time.Now().UTC()}
		login2FAMu.Unlock()
	}

	return login2FAChallengeResponse{
		Requires2FA: true,
		TempToken:   token,
		ExpiresIn:   int(login2FATTL.Seconds()),
	}, nil
}

func (h *Handler) consumeLogin2FAChallenge(c fiber.Ctx, token string) (login2FAChallenge, bool) {
	if h.Redis != nil {
		payload, ok, err := h.Redis.ConsumeLoginChallenge(c.Context(), token)
		if err != nil || !ok {
			return login2FAChallenge{}, false
		}
		var ch login2FAChallenge
		if err := json.Unmarshal(payload, &ch); err != nil {
			return login2FAChallenge{}, false
		}
		return ch, true
	}

	login2FAMu.Lock()
	defer login2FAMu.Unlock()
	pruneLogin2FAChallenges()
	ch, ok := login2FAChallenges[token]
	if !ok {
		return login2FAChallenge{}, false
	}
	delete(login2FAChallenges, token)
	return ch, true
}

// Login2FA handles POST /api/v1/auth/login/2fa.
func (h *Handler) Login2FA(c fiber.Ctx) error {
	var req login2FARequest
	if err := c.Bind().JSON(&req); err != nil {
		return apierror.BadRequest(c, "invalid request body")
	}
	if req.TempToken == "" || req.Code == "" {
		return apierror.BadRequest(c, "temp_token and code are required")
	}

	if h.LoginThrottle != nil {
		if err := h.LoginThrottle.Allow(c.IP(), "login-2fa:"+req.TempToken); err != nil {
			return apierror.Send(c, fiber.StatusTooManyRequests, "too_many_requests", "too many attempts, try again later")
		}
	}

	ch, ok := h.consumeLogin2FAChallenge(c, req.TempToken)
	if !ok {
		return apierror.Send(c, fiber.StatusUnauthorized, "unauthorized", "invalid or expired challenge")
	}

	ctx := c.Context()
	valid, err := h.verifyUserTOTPCode(ctx, ch.UserID, req.Code)
	if err != nil {
		h.Log.ErrorContext(ctx, "login 2fa: verify", slog.String("error", err.Error()))
		return apierror.InternalError(c, "authentication failed")
	}
	if !valid {
		if h.LoginThrottle != nil {
			h.LoginThrottle.RecordFailure("login-2fa:" + req.TempToken)
		}
		return apierror.Send(c, fiber.StatusUnauthorized, "unauthorized", "invalid verification code")
	}

	if h.LoginThrottle != nil {
		h.LoginThrottle.RecordSuccess("login-2fa:" + req.TempToken)
	}

	sess, err := h.issueUserSession(c, ch.UserID, "login_2fa")
	if err != nil {
		return apierror.InternalError(c, "authentication failed")
	}
	return c.Status(fiber.StatusOK).JSON(sess)
}

func (h *Handler) verifyUserTOTPCode(ctx context.Context, userID, code string) (bool, error) {
	if totplib.ValidateBackupCodeFormat(code) {
		hash := totplib.HashBackupCode(code)
		ok, err := h.DB.ConsumeTOTPBackupCode(ctx, userID, hash)
		if err != nil {
			return false, err
		}
		return ok, nil
	}

	encrypted, enabled, err := h.DB.GetUserTOTPEncrypted(ctx, userID)
	if err != nil {
		return false, err
	}
	if !enabled || encrypted == nil || *encrypted == "" {
		return false, nil
	}
	secret, err := totplib.DecryptSecret(*encrypted, h.EncryptionKey, userID)
	if err != nil {
		return false, err
	}
	return totplib.Validate(secret, code), nil
}

func (h *Handler) verifyUserPassword(ctx context.Context, userID, password string) error {
	_, hash, err := h.DB.GetUserPasswordHashByID(ctx, userID)
	if err != nil {
		if errors.Is(err, db.ErrNoPassword) {
			_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
			return err
		}
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return err
	}
	return nil
}