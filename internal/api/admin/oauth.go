package admin

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v3"
	"golang.org/x/oauth2"
	githuboauth "golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"

	"github.com/jukaza/tavo/internal/apierror"
	"github.com/jukaza/tavo/internal/auth"
	"github.com/jukaza/tavo/internal/db"
	"github.com/jukaza/tavo/internal/security"
	"github.com/jukaza/tavo/internal/site"
	"github.com/jukaza/tavo/pkg/keygen"
)

const (
	oauthStateCookie = "tavo_oauth_state"
	oauthBindCookie  = "tavo_oauth_bind"
)

var validOAuthProviders = map[string]struct{}{
	"google": {},
	"github": {},
}

func normalizeOAuthProvider(provider string) (string, error) {
	p := strings.ToLower(strings.TrimSpace(provider))
	if _, ok := validOAuthProviders[p]; !ok {
		return "", fmt.Errorf("unknown provider")
	}
	return p, nil
}

func oauthCookieSecure(c fiber.Ctx) bool {
	if c.Protocol() == "https" {
		return true
	}
	return strings.EqualFold(c.Get("X-Forwarded-Proto"), "https")
}

type oauthState struct {
	Provider    string `json:"provider"`
	Mode        string `json:"mode"`
	Nonce       string `json:"nonce"`
	UserID      string `json:"user_id,omitempty"`
	AcceptTerms bool   `json:"accept_terms,omitempty"`
}

type oauthProfile struct {
	Provider      string
	ExternalID    string
	Email         string
	EmailVerified bool
	Label         string
}

func (h *Handler) apiBaseURL(c fiber.Ctx) string {
	proto := c.Get("X-Forwarded-Proto")
	if proto == "" {
		if c.Protocol() == "https" {
			proto = "https"
		} else {
			proto = "http"
		}
	}
	host := c.Get("X-Forwarded-Host")
	if host == "" {
		host = c.Hostname()
	}
	return proto + "://" + host
}

func (h *Handler) publicUIOrigin(c fiber.Ctx) string {
	if strings.TrimSpace(h.PublicUIOrigin) != "" {
		return strings.TrimRight(h.PublicUIOrigin, "/")
	}
	return h.apiBaseURL(c)
}

func (h *Handler) oauthCallbackURL(c fiber.Ctx, provider string) string {
	return h.apiBaseURL(c) + "/api/v1/auth/oauth/" + provider + "/callback"
}

func (h *Handler) keyInfoFromBearer(c fiber.Ctx) *auth.KeyInfo {
	header := c.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return nil
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	if token == "" {
		return nil
	}
	hash := keygen.Hash(token, h.HMACSecret)
	info, ok := h.KeyCache.Get(hash)
	if !ok {
		return nil
	}
	return &info
}

// StartOAuth handles GET /api/v1/auth/oauth/:provider
func (h *Handler) StartOAuth(c fiber.Ctx) error {
	provider, err := normalizeOAuthProvider(c.Params("provider"))
	if err != nil {
		return apierror.BadRequest(c, "unknown oauth provider")
	}
	mode := strings.ToLower(strings.TrimSpace(c.Query("mode", "login")))
	if mode != "login" && mode != "signup" && mode != "bind" {
		return apierror.BadRequest(c, "invalid oauth mode")
	}

	sec, err := security.LoadInternal(c.Context(), h.DB)
	if err != nil {
		return apierror.InternalError(c, "oauth unavailable")
	}

	bindUserID := ""
	if mode == "bind" {
		keyInfo := h.keyInfoFromBearer(c)
		if keyInfo != nil && keyInfo.UserID != "" {
			bindUserID = keyInfo.UserID
		}
		if bindUserID == "" {
			bindUserID = strings.TrimSpace(c.Cookies(oauthBindCookie))
		}
		if bindUserID == "" {
			return apierror.Send(c, fiber.StatusUnauthorized, "unauthorized", "login required to link account")
		}
		c.Cookie(&fiber.Cookie{Name: oauthBindCookie, Value: "", MaxAge: -1, Path: "/"})
	}

	cfg, secret, err := oauthProviderConfig(sec, provider)
	if err != nil || !cfg.Enabled || strings.TrimSpace(cfg.ClientID) == "" || strings.TrimSpace(secret) == "" {
		return apierror.BadRequest(c, "oauth provider is not configured")
	}
	if mode == "login" && !cfg.AllowLogin {
		return apierror.BadRequest(c, "oauth login is disabled for this provider")
	}
	if mode == "signup" && !cfg.AllowSignup {
		return apierror.BadRequest(c, "oauth signup is disabled for this provider")
	}

	oauthCfg, err := h.buildOAuth2Config(c, provider, cfg.ClientID, secret)
	if err != nil {
		return apierror.InternalError(c, "oauth unavailable")
	}
	acceptTerms := false
	if mode == "signup" {
		q := strings.ToLower(strings.TrimSpace(c.Query("accept_terms")))
		acceptTerms = q == "1" || q == "true" || q == "yes"
	}

	nonce, err := randomNonce()
	if err != nil {
		return apierror.InternalError(c, "oauth unavailable")
	}
	state := oauthState{Provider: provider, Mode: mode, Nonce: nonce, UserID: bindUserID, AcceptTerms: acceptTerms}
	stateRaw, _ := json.Marshal(state)
	stateB64 := base64.RawURLEncoding.EncodeToString(stateRaw)

	sameSite := "Lax"
	if mode == "bind" {
		sameSite = "Strict"
	}
	c.Cookie(&fiber.Cookie{
		Name:     oauthStateCookie,
		Value:    stateB64,
		HTTPOnly: true,
		Secure:   oauthCookieSecure(c),
		SameSite: sameSite,
		MaxAge:   600,
		Path:     "/",
	})

	authURL := oauthCfg.AuthCodeURL(stateB64, oauth2.AccessTypeOffline)
	return c.Redirect().To(authURL)
}

// OAuthCallback handles GET /api/v1/auth/oauth/:provider/callback
func (h *Handler) OAuthCallback(c fiber.Ctx) error {
	provider, err := normalizeOAuthProvider(c.Params("provider"))
	if err != nil {
		return h.redirectOAuthError(c, "unknown oauth provider")
	}
	if errMsg := c.Query("error"); errMsg != "" {
		return h.redirectOAuthError(c, errMsg)
	}

	stateCookie := c.Cookies(oauthStateCookie)
	if stateCookie == "" {
		return h.redirectOAuthError(c, "missing oauth state")
	}
	c.Cookie(&fiber.Cookie{Name: oauthStateCookie, Value: "", MaxAge: -1, Path: "/"})

	stateBytes, err := base64.RawURLEncoding.DecodeString(stateCookie)
	if err != nil {
		return h.redirectOAuthError(c, "invalid oauth state")
	}
	var state oauthState
	if err := json.Unmarshal(stateBytes, &state); err != nil || state.Provider != provider {
		return h.redirectOAuthError(c, "invalid oauth state")
	}
	if strings.TrimSpace(state.Nonce) == "" {
		return h.redirectOAuthError(c, "invalid oauth state")
	}
	if c.Query("state") != stateCookie {
		return h.redirectOAuthError(c, "oauth state mismatch")
	}

	sec, err := security.LoadInternal(c.Context(), h.DB)
	if err != nil {
		return h.redirectOAuthError(c, "oauth unavailable")
	}
	cfg, secret, err := oauthProviderConfig(sec, provider)
	if err != nil {
		return h.redirectOAuthError(c, "oauth unavailable")
	}
	oauthCfg, err := h.buildOAuth2Config(c, provider, cfg.ClientID, secret)
	if err != nil {
		return apierror.InternalError(c, "oauth unavailable")
	}

	token, err := oauthCfg.Exchange(c.Context(), c.Query("code"))
	if err != nil {
		h.Log.ErrorContext(c.Context(), "oauth exchange", slog.String("error", err.Error()))
		return h.redirectOAuthError(c, "oauth token exchange failed")
	}

	profile, err := h.fetchOAuthProfile(c.Context(), provider, oauthCfg, token)
	if err != nil {
		h.Log.ErrorContext(c.Context(), "oauth profile", slog.String("error", err.Error()))
		return h.redirectOAuthError(c, err.Error())
	}

	switch state.Mode {
	case "bind":
		if state.UserID == "" {
			return h.redirectOAuthError(c, "bind session expired")
		}
		if err := h.oauthBind(c, state.UserID, profile); err != nil {
			return h.redirectOAuthError(c, err.Error())
		}
		return c.Redirect().To(h.publicUIOrigin(c) + "/account?tab=connections")
	case "signup":
		siteCfg, err := site.Load(c.Context(), h.DB)
		if err != nil || !siteCfg.RegisterEnabled {
			return h.redirectOAuthError(c, "registration is disabled")
		}
		if siteCfg.UserAgreementEnabled && !state.AcceptTerms {
			return h.redirectOAuthError(c, "you must accept the terms of service")
		}
		fallthrough
	case "login":
		sess, err := h.oauthLoginOrSignup(c, profile, state.Mode == "signup")
		if err != nil {
			return h.redirectOAuthError(c, err.Error())
		}
		code, err := h.storeOAuthExchange(c, sess)
		if err != nil {
			return h.redirectOAuthError(c, "oauth unavailable")
		}
		q := url.Values{}
		q.Set("code", code)
		return c.Redirect().To(h.publicUIOrigin(c) + "/auth/callback?" + q.Encode())
	default:
		return h.redirectOAuthError(c, "invalid oauth mode")
	}
}

func (h *Handler) redirectOAuthError(c fiber.Ctx, msg string) error {
	q := url.Values{}
	q.Set("error", msg)
	return c.Redirect().To(h.publicUIOrigin(c) + "/auth/callback?" + q.Encode())
}

func oauthProviderConfig(sec security.InternalConfig, provider string) (security.OAuthProviderConfig, string, error) {
	switch provider {
	case "google":
		return sec.OAuth.Google, sec.GoogleSecret, nil
	case "github":
		return sec.OAuth.GitHub, sec.GitHubSecret, nil
	default:
		return security.OAuthProviderConfig{}, "", fmt.Errorf("unknown provider")
	}
}

func (h *Handler) buildOAuth2Config(c fiber.Ctx, provider, clientID, clientSecret string) (*oauth2.Config, error) {
	redirectURL := h.oauthCallbackURL(c, provider)
	switch provider {
	case "google":
		return &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"openid", "email", "profile"},
			Endpoint:     google.Endpoint,
		}, nil
	case "github":
		return &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"read:user", "user:email"},
			Endpoint:     githuboauth.Endpoint,
		}, nil
	default:
		return nil, fmt.Errorf("unknown provider")
	}
}

func (h *Handler) fetchOAuthProfile(ctx context.Context, provider string, cfg *oauth2.Config, token *oauth2.Token) (oauthProfile, error) {
	switch provider {
	case "google":
		return fetchGoogleProfile(ctx, cfg, token)
	case "github":
		return fetchGitHubProfile(ctx, cfg, token)
	default:
		return oauthProfile{}, fmt.Errorf("unknown provider")
	}
}

func fetchGoogleProfile(ctx context.Context, cfg *oauth2.Config, token *oauth2.Token) (oauthProfile, error) {
	client := cfg.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return oauthProfile{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var data struct {
		ID            string `json:"id"`
		Email         string `json:"email"`
		VerifiedEmail bool   `json:"verified_email"`
		Name          string `json:"name"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return oauthProfile{}, err
	}
	if strings.TrimSpace(data.Email) == "" {
		return oauthProfile{}, fmt.Errorf("google account has no email")
	}
	if !data.VerifiedEmail {
		return oauthProfile{}, fmt.Errorf("google email is not verified")
	}
	label := data.Name
	if label == "" {
		label = data.Email
	}
	return oauthProfile{
		Provider: "google", ExternalID: data.ID, Email: strings.ToLower(data.Email),
		EmailVerified: true, Label: label,
	}, nil
}

type githubEmailEntry struct {
	Email    string
	Primary  bool
	Verified bool
}

func pickVerifiedGitHubEmail(emails []githubEmailEntry) string {
	for _, e := range emails {
		if e.Primary && e.Verified && strings.TrimSpace(e.Email) != "" {
			return strings.ToLower(strings.TrimSpace(e.Email))
		}
	}
	for _, e := range emails {
		if e.Verified && strings.TrimSpace(e.Email) != "" {
			return strings.ToLower(strings.TrimSpace(e.Email))
		}
	}
	return ""
}

func fetchGitHubProfile(ctx context.Context, cfg *oauth2.Config, token *oauth2.Token) (oauthProfile, error) {
	client := cfg.Client(ctx, token)
	userResp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return oauthProfile{}, err
	}
	defer userResp.Body.Close()
	userBody, _ := io.ReadAll(io.LimitReader(userResp.Body, 1<<20))
	var user struct {
		ID    int64  `json:"id"`
		Login string `json:"login"`
		Name  string `json:"name"`
	}
	if err := json.Unmarshal(userBody, &user); err != nil {
		return oauthProfile{}, err
	}

	emResp, err := client.Get("https://api.github.com/user/emails")
	if err != nil {
		return oauthProfile{}, err
	}
	defer emResp.Body.Close()
	emBody, _ := io.ReadAll(io.LimitReader(emResp.Body, 1<<20))
	var rawEmails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.Unmarshal(emBody, &rawEmails); err != nil {
		return oauthProfile{}, err
	}
	emails := make([]githubEmailEntry, len(rawEmails))
	for i, e := range rawEmails {
		emails[i] = githubEmailEntry{Email: e.Email, Primary: e.Primary, Verified: e.Verified}
	}
	email := pickVerifiedGitHubEmail(emails)
	if email == "" {
		return oauthProfile{}, fmt.Errorf("github account has no verified email")
	}
	label := user.Login
	if user.Name != "" {
		label = user.Name
	}
	return oauthProfile{
		Provider: "github", ExternalID: fmt.Sprintf("%d", user.ID), Email: email,
		EmailVerified: true, Label: label,
	}, nil
}

func (h *Handler) oauthLoginOrSignup(c fiber.Ctx, profile oauthProfile, isSignup bool) (loginResponse, error) {
	ctx := c.Context()
	if conn, err := h.DB.GetOAuthConnectionByExternal(ctx, profile.Provider, profile.ExternalID); err == nil {
		return h.issueUserSession(c, conn.UserID, "oauth_login")
	} else if !errors.Is(err, db.ErrNotFound) {
		return loginResponse{}, fmt.Errorf("authentication failed")
	}

	user, err := h.DB.GetUserByEmail(ctx, profile.Email)
	if err == nil {
		if _, _, hashErr := h.DB.GetUserPasswordHashByID(ctx, user.ID); hashErr == nil {
			return loginResponse{}, fmt.Errorf("account exists — sign in with password and link this provider from Account settings")
		} else if !errors.Is(hashErr, db.ErrNoPassword) {
			return loginResponse{}, fmt.Errorf("authentication failed")
		}
		if _, linkErr := h.DB.UpsertOAuthConnection(ctx, user.ID, profile.Provider, profile.ExternalID, profile.Label); linkErr != nil {
			return loginResponse{}, fmt.Errorf("failed to link account")
		}
		return h.issueUserSession(c, user.ID, "oauth_login")
	}
	if !errors.Is(err, db.ErrNotFound) {
		return loginResponse{}, fmt.Errorf("authentication failed")
	}
	if !isSignup {
		return loginResponse{}, fmt.Errorf("no account found — sign up first")
	}

	newUser, err := h.DB.CreateUser(ctx, db.CreateUserParams{
		Email:        profile.Email,
		DisplayName:  profile.Label,
		AuthProvider: profile.Provider,
		ExternalID:   &profile.ExternalID,
	})
	if err != nil {
		if errors.Is(err, db.ErrConflict) {
			return loginResponse{}, fmt.Errorf("email already registered")
		}
		return loginResponse{}, fmt.Errorf("signup failed")
	}
	if _, err := h.DB.CreateWallet(ctx, newUser.ID, ""); err != nil {
		return loginResponse{}, fmt.Errorf("signup failed")
	}
	if h.Wallet != nil {
		h.Wallet.Register(newUser.ID)
	}
	if err := h.creditSignupWallet(ctx, newUser.ID); err != nil {
		return loginResponse{}, fmt.Errorf("signup failed")
	}
	if _, err := h.DB.UpsertOAuthConnection(ctx, newUser.ID, profile.Provider, profile.ExternalID, profile.Label); err != nil {
		return loginResponse{}, fmt.Errorf("signup failed")
	}
	return h.issueUserSession(c, newUser.ID, "oauth_signup")
}

func (h *Handler) oauthBind(c fiber.Ctx, userID string, profile oauthProfile) error {
	ctx := c.Context()
	if existing, err := h.DB.GetOAuthConnectionByExternal(ctx, profile.Provider, profile.ExternalID); err == nil {
		if existing.UserID != userID {
			return fmt.Errorf("this external account is already linked to another user")
		}
		return nil
	} else if !errors.Is(err, db.ErrNotFound) {
		return fmt.Errorf("failed to link account")
	}
	if _, err := h.DB.UpsertOAuthConnection(ctx, userID, profile.Provider, profile.ExternalID, profile.Label); err != nil {
		return fmt.Errorf("failed to link account")
	}
	return nil
}

func randomNonce() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}