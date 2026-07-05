package totp

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/voidmind-io/voidllm/pkg/crypto"
)

const (
	backupCodeCount = 10
	issuer          = "VoidLLM"
)

// SetupResult holds a new TOTP enrollment payload.
type SetupResult struct {
	Secret     string
	OTPAuthURL string
}

// GenerateSetup creates a new TOTP secret and otpauth URL for enrollment.
func GenerateSetup(accountName string) (SetupResult, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      issuer,
		AccountName: accountName,
		SecretSize:  20,
	})
	if err != nil {
		return SetupResult{}, fmt.Errorf("generate totp: %w", err)
	}
	return SetupResult{
		Secret:     key.Secret(),
		OTPAuthURL: key.URL(),
	}, nil
}

// Validate checks a TOTP code against the plaintext secret (window ±1 step).
func Validate(secret, code string) bool {
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return false
	}
	return totp.Validate(code, secret)
}

// EncryptSecret encrypts a TOTP secret for storage.
func EncryptSecret(secret string, encryptionKey []byte, userID string) (string, error) {
	return crypto.EncryptString(secret, encryptionKey, []byte("totp:"+userID))
}

// DecryptSecret decrypts a stored TOTP secret.
func DecryptSecret(ciphertext string, encryptionKey []byte, userID string) (string, error) {
	return crypto.DecryptString(ciphertext, encryptionKey, []byte("totp:"+userID))
}

// GenerateBackupCodes returns plaintext codes and their SHA-256 hex hashes.
func GenerateBackupCodes() (plain []string, hashes []string, err error) {
	plain = make([]string, backupCodeCount)
	hashes = make([]string, backupCodeCount)
	for i := range plain {
		var b [4]byte
		if _, err := rand.Read(b[:]); err != nil {
			return nil, nil, fmt.Errorf("backup code entropy: %w", err)
		}
		plain[i] = fmt.Sprintf("%04x-%04x", uint16(b[0])<<8|uint16(b[1]), uint16(b[2])<<8|uint16(b[3]))
		hashes[i] = hashBackupCode(plain[i])
	}
	return plain, hashes, nil
}

// HashBackupCode returns SHA-256 hex of a normalized backup code.
func HashBackupCode(code string) string {
	return hashBackupCode(code)
}

func hashBackupCode(code string) string {
	normalized := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(code), " ", ""))
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}

// ValidateBackupCodeFormat accepts xxxx-xxxx style codes.
func ValidateBackupCodeFormat(code string) bool {
	code = strings.TrimSpace(code)
	if len(code) != 9 {
		return false
	}
	parts := strings.Split(code, "-")
	return len(parts) == 2 && len(parts[0]) == 4 && len(parts[1]) == 4
}

// ParseKeyURL validates otpauth URL generation (test helper).
func ParseKeyURL(url string) (*otp.Key, error) {
	return otp.NewKeyFromURL(url)
}