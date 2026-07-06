package payment

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	WebhookAuthAPIKey = "api_key"
	WebhookAuthHMAC   = "hmac"
)

// SepayWebhookPayload is the JSON body SePay POSTs to the webhook endpoint.
// See https://docs.sepay.vn/tich-hop-webhooks.html
type SepayWebhookPayload struct {
	ID              int64   `json:"id"`
	Gateway         string  `json:"gateway"`
	TransactionDate string  `json:"transactionDate"`
	AccountNumber   string  `json:"accountNumber"`
	SubAccount      string  `json:"subAccount"`
	Code            string  `json:"code"`
	Content         string  `json:"content"`
	TransferType    string  `json:"transferType"`
	Description     string  `json:"description"`
	TransferAmount  float64 `json:"transferAmount"`
	Accumulated     float64 `json:"accumulated"`
	ReferenceCode   string  `json:"referenceCode"`
}

// SepayWebhookAuthHeaders carries auth-related HTTP headers from SePay.
type SepayWebhookAuthHeaders struct {
	Authorization      string
	Signature          string
	Timestamp          string
	BearerToken        string
}

// SepayWebhookIPs are the published SePay webhook sender addresses.
// See https://docs.sepay.vn/tich-hop-webhooks.html
var SepayWebhookIPs = []string{
	"172.236.138.20",
	"172.233.83.68",
	"171.244.35.2",
	"151.158.108.68",
	"151.158.109.79",
	"103.255.238.139",
	"2400:8905::2000:8cff:fe98:45cd",
	"2600:3c15::2000:8aff:fedd:874b",
}

var tradeNoPattern = regexp.MustCompile(`VL[A-Za-z0-9]+NO[A-Za-z0-9]+`)

// WebhookAuthConfigured reports whether webhook credentials match the selected auth mode.
func (s SepayConfig) WebhookAuthConfigured() bool {
	switch strings.TrimSpace(s.WebhookAuthMode) {
	case WebhookAuthHMAC:
		return strings.TrimSpace(s.WebhookSecret) != ""
	default:
		return strings.TrimSpace(s.WebhookToken) != ""
	}
}

// IsConfigured reports whether SePay bank details and webhook auth are present.
func (s SepayConfig) IsConfigured() bool {
	return strings.TrimSpace(s.BankCode) != "" &&
		strings.TrimSpace(s.AccountNumber) != "" &&
		strings.TrimSpace(s.AccountName) != "" &&
		s.WebhookAuthConfigured()
}

// IsSepayWebhookIP reports whether clientIP is a known SePay webhook sender.
func IsSepayWebhookIP(clientIP string) bool {
	ip := strings.TrimSpace(clientIP)
	if ip == "" {
		return false
	}
	for _, allowed := range SepayWebhookIPs {
		if ip == allowed {
			return true
		}
	}
	return false
}

// ExtractTradeNo finds the Tavo order reference (VL…NO…) in SePay transfer content/code.
// Arbitrary memo text is not accepted to avoid matching unrelated pending orders.
func ExtractTradeNo(content, code string) string {
	for _, raw := range []string{content, code} {
		if tradeNo := tradeNoPattern.FindString(raw); tradeNo != "" {
			return tradeNo
		}
	}
	return ""
}

// AccountNumberMatches compares webhook account number with configured account.
func AccountNumberMatches(configured, received string) bool {
	return normalizeAccountNumber(configured) == normalizeAccountNumber(received)
}

func normalizeAccountNumber(v string) string {
	return strings.TrimLeft(strings.TrimSpace(v), "0")
}

// VerifyWebhookAuth validates SePay webhook credentials per configured auth mode.
func VerifyWebhookAuth(cfg SepayConfig, headers SepayWebhookAuthHeaders, rawBody []byte, now time.Time) error {
	mode := strings.TrimSpace(cfg.WebhookAuthMode)
	if mode == "" {
		mode = WebhookAuthAPIKey
	}

	switch mode {
	case WebhookAuthHMAC:
		secret := strings.TrimSpace(cfg.WebhookSecret)
		if secret == "" {
			return fmt.Errorf("webhook secret not configured")
		}
		return verifyHMACSignature(secret, headers.Signature, headers.Timestamp, rawBody, now)
	case WebhookAuthAPIKey:
		expected := strings.TrimSpace(cfg.WebhookToken)
		if expected == "" {
			return fmt.Errorf("webhook token not configured")
		}
		token := extractAPIKey(headers.Authorization)
		if token == "" || subtle.ConstantTimeCompare([]byte(token), []byte(expected)) != 1 {
			return fmt.Errorf("invalid api key")
		}
		return nil
	default:
		return fmt.Errorf("unsupported webhook auth mode %q", mode)
	}
}

func extractAPIKey(authHeader string) string {
	authHeader = strings.TrimSpace(authHeader)
	for _, prefix := range []string{"Apikey ", "ApiKey "} {
		if strings.HasPrefix(authHeader, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(authHeader, prefix))
		}
	}
	return ""
}

func verifyHMACSignature(secret, signature, timestamp string, rawBody []byte, now time.Time) error {
	signature = strings.TrimSpace(signature)
	timestamp = strings.TrimSpace(timestamp)
	if signature == "" || timestamp == "" {
		return fmt.Errorf("missing hmac headers")
	}

	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp")
	}
	if absDuration(now.Unix()-ts) > 5*time.Minute {
		return fmt.Errorf("request expired")
	}

	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(fmt.Sprintf("%d.%s", ts, string(rawBody))))
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	if subtle.ConstantTimeCompare([]byte(expected), []byte(signature)) != 1 {
		return fmt.Errorf("invalid signature")
	}
	return nil
}

func absDuration(seconds int64) time.Duration {
	if seconds < 0 {
		seconds = -seconds
	}
	return time.Duration(seconds) * time.Second
}