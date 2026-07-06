package admin

import (
	"fmt"
	"strings"

	"github.com/jukaza/tavo/internal/db"
	"github.com/jukaza/tavo/pkg/crypto"
	"github.com/jukaza/tavo/pkg/keygen"
)

func apiKeyAAD(keyID string) []byte {
	return []byte("api_key:" + keyID)
}

func (h *Handler) encryptStoredAPIKey(keyID, plaintext string) (*string, error) {
	plaintext = strings.TrimSpace(plaintext)
	if plaintext == "" {
		return nil, nil
	}
	enc, err := crypto.EncryptString(plaintext, h.EncryptionKey, apiKeyAAD(keyID))
	if err != nil {
		return nil, fmt.Errorf("encrypt api key: %w", err)
	}
	return &enc, nil
}

func (h *Handler) decryptStoredAPIKey(k *db.APIKey) (string, error) {
	if k.KeyEncrypted == nil || strings.TrimSpace(*k.KeyEncrypted) == "" {
		return "", errAPIKeyNotRetrievable
	}
	plaintext, err := crypto.DecryptString(*k.KeyEncrypted, h.EncryptionKey, apiKeyAAD(k.ID))
	if err != nil {
		return "", fmt.Errorf("decrypt api key: %w", err)
	}
	return plaintext, nil
}

func (h *Handler) storedAPIKeyParams(keyID, keyType, plaintext string) (db.CreateAPIKeyParams, error) {
	var enc *string
	if keyType == keygen.KeyTypeUser {
		var err error
		enc, err = h.encryptStoredAPIKey(keyID, plaintext)
		if err != nil {
			return db.CreateAPIKeyParams{}, err
		}
	}
	return db.CreateAPIKeyParams{ID: keyID, KeyEncrypted: enc}, nil
}