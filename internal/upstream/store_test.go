package upstream

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/voidmind-io/voidllm/internal/config"
	"github.com/voidmind-io/voidllm/internal/db"
	"github.com/voidmind-io/voidllm/pkg/crypto"
)

func testEncryptionKey() []byte {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	return key
}

func openTestDB(t *testing.T) *db.DB {
	t.Helper()
	safeName := strings.NewReplacer("/", "_", " ", "_", "#", "_").Replace(t.Name())
	cfg := config.DatabaseConfig{
		Driver:          "sqlite",
		DSN:             "file:" + safeName + "?mode=memory&cache=private",
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxLifetime: time.Minute,
	}
	ctx := context.Background()
	d, err := db.Open(ctx, cfg)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = d.Close() })
	if err := db.RunMigrations(ctx, d.SQL(), d.Dialect(), slog.Default()); err != nil {
		t.Fatalf("RunMigrations() error = %v", err)
	}
	return d
}

func TestStore_Select_PausedProvider(t *testing.T) {
	d := openTestDB(t)
	ctx := context.Background()
	prov, err := d.CreateProvider(ctx, db.CreateProviderParams{Name: "paused-prov", Status: "paused"})
	if err != nil {
		t.Fatalf("CreateProvider: %v", err)
	}
	if _, err := d.CreateProviderConnection(ctx, db.CreateProviderConnectionParams{
		ProviderID: prov.ID, Name: "key-1",
	}); err != nil {
		t.Fatalf("CreateProviderConnection: %v", err)
	}

	store := &Store{DB: d, EncryptionKey: testEncryptionKey()}
	_, selErr := store.Select(ctx, prov.ID, "gpt-4o", "fill-first", 1, nil)
	if !errors.Is(selErr, ErrProviderPaused) {
		t.Fatalf("Select() error = %v, want ErrProviderPaused", selErr)
	}
}

func TestStore_DecryptKey(t *testing.T) {
	d := openTestDB(t)
	ctx := context.Background()
	prov, err := d.CreateProvider(ctx, db.CreateProviderParams{Name: "decrypt-prov", Status: "active"})
	if err != nil {
		t.Fatalf("CreateProvider: %v", err)
	}
	conn, err := d.CreateProviderConnection(ctx, db.CreateProviderConnectionParams{
		ProviderID: prov.ID, Name: "key-1",
	})
	if err != nil {
		t.Fatalf("CreateProviderConnection: %v", err)
	}

	encKey := testEncryptionKey()
	enc, err := crypto.EncryptString("sk-secret", encKey, connectionAAD(conn.ID))
	if err != nil {
		t.Fatalf("EncryptString: %v", err)
	}
	updated, err := d.UpdateProviderConnection(ctx, conn.ID, db.UpdateProviderConnectionParams{
		APIKeyEncrypted: &enc,
	})
	if err != nil {
		t.Fatalf("UpdateProviderConnection: %v", err)
	}

	store := &Store{DB: d, EncryptionKey: encKey}
	got, decErr := store.DecryptKey(updated)
	if decErr != nil {
		t.Fatalf("DecryptKey() error = %v", decErr)
	}
	if got != "sk-secret" {
		t.Fatalf("DecryptKey() = %q, want sk-secret", got)
	}

	got, decErr = store.DecryptKey(&db.ProviderConnection{ID: conn.ID})
	if decErr != nil {
		t.Fatalf("DecryptKey(nil ciphertext) error = %v", decErr)
	}
	if got != "" {
		t.Fatalf("DecryptKey(nil ciphertext) = %q, want empty", got)
	}

	badEnc, err := crypto.EncryptString("sk-wrong-aad", encKey, []byte("wrong-aad"))
	if err != nil {
		t.Fatalf("EncryptString wrong aad: %v", err)
	}
	badConn := &db.ProviderConnection{ID: conn.ID, APIKeyEncrypted: &badEnc}
	_, decErr = store.DecryptKey(badConn)
	if decErr == nil {
		t.Fatal("DecryptKey(wrong AAD) expected error")
	}
}