package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/jukaza/tavo/internal/config"
	"github.com/jukaza/tavo/internal/db"
	"github.com/jukaza/tavo/internal/upstream"
	"github.com/jukaza/tavo/pkg/crypto"
)

func main() {
	keyRaw := os.Getenv("TAVO_ENCRYPTION_KEY")
	if keyRaw == "" {
		keyRaw = "dev-encryption-key-32bytes-long!!"
	}
	encKey, err := crypto.ParseKey(keyRaw)
	if err != nil {
		panic(err)
	}
	defer crypto.ZeroKey(encKey)

	dsn := os.Getenv("TAVO_DATABASE_DSN")
	if dsn == "" {
		dsn = "./voidllm.db"
	}

	database, err := db.OpenAndMigrate(context.Background(), config.DatabaseConfig{
		Driver: "sqlite",
		DSN:    dsn,
	}, slog.Default())
	if err != nil {
		panic(err)
	}
	defer database.Close()

	store := &upstream.Store{DB: database, EncryptionKey: encKey}
	conns, err := database.ListProviderConnections(context.Background(), "019f2cfc-1fa6-755f-ab4b-5ed3a0f977e8", true)
	if err != nil {
		panic(err)
	}
	fmt.Printf("active connections: %d\n", len(conns))
	for _, conn := range conns {
		plain, decErr := store.DecryptKey(&conn)
		if decErr != nil {
			fmt.Printf("  %s decrypt FAILED: %v\n", conn.ID, decErr)
			continue
		}
		prefix := plain
		if len(prefix) > 16 {
			prefix = prefix[:16]
		}
		fmt.Printf("  %s decrypt OK prefix=%q len=%d\n", conn.ID, prefix, len(plain))
	}

	// Also verify config encryption key parsing matches server startup.
	cfg, _, err := config.Load("tavo.dev.yaml")
	if err != nil {
		fmt.Println("config load err:", err)
		return
	}
	cfgKey, err := crypto.ParseKey(cfg.Settings.EncryptionKey)
	if err != nil {
		fmt.Println("config key parse err:", err)
		return
	}
	defer crypto.ZeroKey(cfgKey)
	fmt.Printf("config encryption key matches env: %v\n", string(encKey) == string(cfgKey))
}