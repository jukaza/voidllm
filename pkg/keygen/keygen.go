// Package keygen provides API key generation, HMAC-SHA256 hashing, and
// prefix validation for Tavo's authentication system.
package keygen

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
)

// Key type constants identify the category of an API key.
const (
	KeyTypeUser    = "user_key"
	KeyTypeSession = "session_key"
)

// PrefixAPI is the standard prefix for newly generated keys (OpenAI-style).
const PrefixAPI = "sk-"

// Legacy prefixes remain valid for keys created before the format change.
const (
	PrefixLegacyUser    = "vl_uk_"
	PrefixLegacySession = "vl_sk_"
)

// apiKeyRandomBytes is the entropy behind the hex suffix (sk- + 48 hex chars).
const apiKeyRandomBytes = 24

// Generate creates a new random API key for the given keyType. All new keys use
// the sk- prefix followed by 48 hex characters (51 characters total).
func Generate(keyType string) (string, error) {
	if _, err := keyTypeFor(keyType); err != nil {
		return "", err
	}

	raw := make([]byte, apiKeyRandomBytes)
	if _, err := io.ReadFull(rand.Reader, raw); err != nil {
		return "", fmt.Errorf("generate key: read random bytes: %w", err)
	}

	return PrefixAPI + hex.EncodeToString(raw), nil
}

// Hash returns the hex-encoded HMAC-SHA256 of plaintextKey using hmacSecret.
// The result is used as the stored representation of an API key — the
// plaintext is never persisted. HMAC-SHA256 never returns an error for a
// valid (non-nil) key, so no error is returned.
func Hash(plaintextKey string, hmacSecret []byte) string {
	mac := hmac.New(sha256.New, hmacSecret)
	// Write never returns an error for hash.Hash implementations.
	_, _ = io.WriteString(mac, plaintextKey)
	return hex.EncodeToString(mac.Sum(nil))
}

// Hint returns a short, non-secret representation of the key suitable for
// display in logs and UIs. If the key is 10 characters or shorter it is
// returned as-is. Otherwise the format is "<first6>...<last4>", e.g.
// "sk-a3f...e8b1".
func Hint(plaintextKey string) string {
	if len(plaintextKey) <= 10 {
		return plaintextKey
	}
	return plaintextKey[:6] + "..." + plaintextKey[len(plaintextKey)-4:]
}

// Verify reports whether plaintextKey, when HMAC'd with hmacSecret, matches
// storedHash. The comparison is constant-time to prevent timing attacks.
// storedHash must be the hex-encoded output of Hash. Returns false if either
// hex string is malformed.
func Verify(plaintextKey string, hmacSecret []byte, storedHash string) bool {
	computed := Hash(plaintextKey, hmacSecret)
	computedBytes, err := hex.DecodeString(computed)
	if err != nil {
		return false
	}
	expectedBytes, err := hex.DecodeString(storedHash)
	if err != nil {
		return false
	}
	return hmac.Equal(computedBytes, expectedBytes)
}

// ValidatePrefix inspects the key's prefix and returns the corresponding
// KeyType constant when known. New sk- keys are typed as user_key here;
// session vs user is resolved from the database after hash lookup.
// Legacy vl_uk_ / vl_sk_ prefixes remain accepted.
func ValidatePrefix(key string) (string, error) {
	switch {
	case strings.HasPrefix(key, PrefixLegacyUser):
		return KeyTypeUser, nil
	case strings.HasPrefix(key, PrefixLegacySession):
		return KeyTypeSession, nil
	case strings.HasPrefix(key, PrefixAPI):
		body := key[len(PrefixAPI):]
		if len(body) < 32 {
			return "", fmt.Errorf("validate prefix: API key body too short")
		}
		return KeyTypeUser, nil
	default:
		return "", fmt.Errorf("validate prefix: unrecognized key prefix")
	}
}

func keyTypeFor(keyType string) (string, error) {
	switch keyType {
	case KeyTypeUser, KeyTypeSession:
		return keyType, nil
	default:
		return "", fmt.Errorf("prefix for: unknown key type %q", keyType)
	}
}