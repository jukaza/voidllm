package payment

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"
)

func TestVerifyWebhookAuthAPIKey(t *testing.T) {
	cfg := SepayConfig{
		WebhookAuthMode: WebhookAuthAPIKey,
		WebhookToken:    "secret-token",
	}
	err := VerifyWebhookAuth(cfg, SepayWebhookAuthHeaders{
		Authorization: "Apikey secret-token",
	}, []byte(`{"id":1}`), time.Now())
	if err != nil {
		t.Fatalf("VerifyWebhookAuth: %v", err)
	}
}

func TestVerifyWebhookAuthHMAC(t *testing.T) {
	secret := "hmac-secret"
	body := []byte(`{"id":92704,"transferType":"in"}`)
	ts := time.Now().Unix()
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(fmt.Sprintf("%d.%s", ts, string(body))))
	sig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	cfg := SepayConfig{
		WebhookAuthMode: WebhookAuthHMAC,
		WebhookSecret:   secret,
	}
	err := VerifyWebhookAuth(cfg, SepayWebhookAuthHeaders{
		Signature: sig,
		Timestamp: fmt.Sprintf("%d", ts),
	}, body, time.Now())
	if err != nil {
		t.Fatalf("VerifyWebhookAuth HMAC: %v", err)
	}
}

func TestExtractTradeNo(t *testing.T) {
	got := ExtractTradeNo("VLABCNO123 chuyen tien", "")
	if got != "VLABCNO123" {
		t.Fatalf("ExtractTradeNo content = %q", got)
	}
	got = ExtractTradeNo("", "SEVN63DC8E5C")
	if got != "" {
		t.Fatalf("ExtractTradeNo code without VL pattern = %q, want empty", got)
	}
	got = ExtractTradeNo("SEVN63DC8E5C chuyen tien", "SEVN63DC8E5C")
	if got != "" {
		t.Fatalf("ExtractTradeNo arbitrary memo = %q, want empty", got)
	}
}

func TestAccountNumberMatches(t *testing.T) {
	if !AccountNumberMatches("01017588888", "1017588888") {
		t.Fatal("expected account numbers to match after normalization")
	}
}