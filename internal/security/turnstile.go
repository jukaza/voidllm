package security

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const turnstileVerifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

// VerifyTurnstile validates a Turnstile token when enabled.
func VerifyTurnstile(ctx context.Context, secret, token, remoteIP string) error {
	if strings.TrimSpace(secret) == "" {
		return fmt.Errorf("turnstile is not configured")
	}
	if strings.TrimSpace(token) == "" {
		return fmt.Errorf("turnstile token is required")
	}

	form := url.Values{}
	form.Set("secret", secret)
	form.Set("response", token)
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, turnstileVerifyURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("turnstile verify request failed")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("turnstile verify read failed")
	}

	var result struct {
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("turnstile verify parse failed")
	}
	if !result.Success {
		return fmt.Errorf("turnstile verification failed")
	}
	return nil
}