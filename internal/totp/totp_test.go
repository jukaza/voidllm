package totp_test

import (
	"testing"

	"github.com/voidmind-io/voidllm/internal/totp"
	"github.com/voidmind-io/voidllm/pkg/crypto"
)

func TestGenerateSetupAndValidate(t *testing.T) {
	setup, err := totp.GenerateSetup("user@example.com")
	if err != nil {
		t.Fatalf("GenerateSetup: %v", err)
	}
	if setup.Secret == "" || setup.OTPAuthURL == "" {
		t.Fatal("expected secret and otpauth url")
	}
	if _, err := totp.ParseKeyURL(setup.OTPAuthURL); err != nil {
		t.Fatalf("ParseKeyURL: %v", err)
	}
}

func TestBackupCodesHashAndFormat(t *testing.T) {
	plain, hashes, err := totp.GenerateBackupCodes()
	if err != nil {
		t.Fatalf("GenerateBackupCodes: %v", err)
	}
	if len(plain) != 10 || len(hashes) != 10 {
		t.Fatalf("expected 10 codes, got %d", len(plain))
	}
	for _, c := range plain {
		if !totp.ValidateBackupCodeFormat(c) {
			t.Fatalf("invalid format: %q", c)
		}
		if totp.HashBackupCode(c) != totp.HashBackupCode(c) {
			t.Fatal("hash not stable")
		}
	}
}

func TestEncryptRoundTrip(t *testing.T) {
	key, err := crypto.ParseKey("dev-encryption-key-32bytes-long!!")
	if err != nil {
		t.Fatalf("ParseKey: %v", err)
	}
	defer crypto.ZeroKey(key)

	secret := "JBSWY3DPEHPK3PXP"
	userID := "user-123"
	enc, err := totp.EncryptSecret(secret, key, userID)
	if err != nil {
		t.Fatalf("EncryptSecret: %v", err)
	}
	got, err := totp.DecryptSecret(enc, key, userID)
	if err != nil {
		t.Fatalf("DecryptSecret: %v", err)
	}
	if got != secret {
		t.Fatalf("got %q want %q", got, secret)
	}
}