package admin

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
)

func TestOAuthExchangeInMemorySingleUse(t *testing.T) {
	t.Parallel()

	h := &Handler{}
	app := fiber.New()
	app.Post("/exchange", func(c fiber.Ctx) error {
		return h.OAuthExchange(c)
	})
	app.Get("/store", func(c fiber.Ctx) error {
		sess := loginResponse{
			Token:     "session-token",
			ExpiresAt: time.Now().UTC().Add(time.Hour).Format(time.RFC3339),
			User: meResponse{
				ID:    "user-1",
				Email: "user@example.com",
			},
		}
		code, err := h.storeOAuthExchange(c, sess)
		if err != nil {
			return err
		}
		return c.SendString(code)
	})

	storeReq := httptest.NewRequest(fiber.MethodGet, "/store", nil)
	storeResp, err := app.Test(storeReq, fiber.TestConfig{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer storeResp.Body.Close()
	storeBody, _ := io.ReadAll(storeResp.Body)
	code := string(bytes.TrimSpace(storeBody))
	if code == "" {
		t.Fatal("expected non-empty exchange code")
	}

	payload, _ := json.Marshal(map[string]string{"code": code})
	exchangeReq := httptest.NewRequest(fiber.MethodPost, "/exchange", bytes.NewReader(payload))
	exchangeReq.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(exchangeReq, fiber.TestConfig{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("first exchange: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("first exchange status = %d, want 200", resp.StatusCode)
	}
	_ = resp.Body.Close()

	resp2, err := app.Test(exchangeReq, fiber.TestConfig{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("second exchange: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("second exchange status = %d, want 401", resp2.StatusCode)
	}
}

func TestPruneOAuthExchanges(t *testing.T) {
	oauthExchangeMu.Lock()
	oauthExchanges["expired"] = oauthExchangeEntry{
		Token:     "old",
		CreatedAt: time.Now().UTC().Add(-3 * time.Minute),
	}
	oauthExchangeMu.Unlock()

	pruneOAuthExchanges()

	oauthExchangeMu.Lock()
	_, ok := oauthExchanges["expired"]
	oauthExchangeMu.Unlock()
	if ok {
		t.Fatal("expected expired exchange code to be pruned")
	}
}