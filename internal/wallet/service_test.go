package wallet

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/voidmind-io/voidllm/internal/config"
	"github.com/voidmind-io/voidllm/internal/db"
)

func openTestDB(t *testing.T, name string) *db.DB {
	t.Helper()
	ctx := context.Background()
	database, err := db.Open(ctx, config.DatabaseConfig{
		Driver:          "sqlite",
		DSN:             "file:" + name + "?mode=memory&cache=private",
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxLifetime: time.Minute,
	})
	if err != nil {
		t.Fatalf("open test DB: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := db.RunMigrations(ctx, database.SQL(), db.SQLiteDialect{}, slog.Default()); err != nil {
		t.Fatalf("run migrations: %v", err)
	}
	return database
}

func createTestUser(t *testing.T, database *db.DB, email string) string {
	t.Helper()
	u, err := database.CreateUser(context.Background(), db.CreateUserParams{
		Email:       email,
		DisplayName: "Wallet Test",
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return u.ID
}

func newTestService(t *testing.T, database *db.DB) *Service {
	t.Helper()
	svc, err := NewService(context.Background(), database, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	return svc
}

func TestCheck_NoWallet(t *testing.T) {
	t.Parallel()
	database := openTestDB(t, "TestWalletCheckNoWallet")
	svc := newTestService(t, database)

	if err := svc.Check("nonexistent-user"); err != ErrNoWallet {
		t.Errorf("Check = %v, want ErrNoWallet", err)
	}
}

func TestCheck_ZeroBalance(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	database := openTestDB(t, "TestWalletCheckZero")
	userID := createTestUser(t, database, "zero@test.io")
	if _, err := database.CreateWallet(ctx, userID, "VND"); err != nil {
		t.Fatalf("create wallet: %v", err)
	}
	svc := newTestService(t, database)

	if err := svc.Check(userID); err != ErrInsufficientBalance {
		t.Errorf("Check = %v, want ErrInsufficientBalance", err)
	}
}

func TestCreditThenCheckAndDebit(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	database := openTestDB(t, "TestWalletCreditDebit")
	userID := createTestUser(t, database, "cd@test.io")
	if _, err := database.CreateWallet(ctx, userID, "VND"); err != nil {
		t.Fatalf("create wallet: %v", err)
	}
	svc := newTestService(t, database)

	newBalance, err := svc.Credit(ctx, userID, 10.0, "topup", "ref-1", "test topup")
	if err != nil {
		t.Fatalf("Credit: %v", err)
	}
	if newBalance != 10.0 {
		t.Errorf("balance after credit = %v, want 10", newBalance)
	}
	if err := svc.Check(userID); err != nil {
		t.Errorf("Check after credit = %v, want nil", err)
	}

	svc.Debit(ctx, userID, 4.0, "req-1", "usage")
	balance, ok := svc.Balance(userID)
	if !ok || balance != 6.0 {
		t.Errorf("balance after debit = %v (ok=%v), want 6", balance, ok)
	}

	// Ledger must have two entries and balance_after must reconcile.
	txs, err := database.ListTransactions(ctx, userID, "", 10)
	if err != nil {
		t.Fatalf("ListTransactions: %v", err)
	}
	if len(txs) != 2 {
		t.Fatalf("transactions = %d, want 2", len(txs))
	}
	if txs[0].Type != "usage" || txs[0].Amount != -4.0 || txs[0].BalanceAfter != 6.0 {
		t.Errorf("latest tx = %+v, want usage/-4/6", txs[0])
	}

	// DB wallet row must match the cache.
	w, err := database.GetWalletByUser(ctx, userID)
	if err != nil {
		t.Fatalf("GetWalletByUser: %v", err)
	}
	if w.Balance != 6.0 {
		t.Errorf("DB balance = %v, want 6", w.Balance)
	}
}

func TestDebit_AllowsNegativeBalance(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	database := openTestDB(t, "TestWalletNegative")
	userID := createTestUser(t, database, "neg@test.io")
	if _, err := database.CreateWallet(ctx, userID, "VND"); err != nil {
		t.Fatalf("create wallet: %v", err)
	}
	svc := newTestService(t, database)
	if _, err := svc.Credit(ctx, userID, 1.0, "topup", "", ""); err != nil {
		t.Fatalf("Credit: %v", err)
	}

	// Streaming responses can overshoot the remaining balance; the debit must
	// still be recorded and the balance go negative (prepaid overdraft).
	svc.Debit(ctx, userID, 2.5, "req-x", "usage overshoot")
	balance, _ := svc.Balance(userID)
	if balance != -1.5 {
		t.Errorf("balance = %v, want -1.5", balance)
	}
	if err := svc.Check(userID); err != ErrInsufficientBalance {
		t.Errorf("Check on negative balance = %v, want ErrInsufficientBalance", err)
	}
}

func TestSeedFromDB(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	database := openTestDB(t, "TestWalletSeed")
	userID := createTestUser(t, database, "seed@test.io")
	if _, err := database.CreateWallet(ctx, userID, "VND"); err != nil {
		t.Fatalf("create wallet: %v", err)
	}
	if _, err := database.ApplyTransaction(ctx, db.ApplyTransactionParams{
		UserID: userID, Type: "topup", Amount: 42.0,
	}); err != nil {
		t.Fatalf("ApplyTransaction: %v", err)
	}

	// A fresh service must pick up the persisted balance.
	svc := newTestService(t, database)
	balance, ok := svc.Balance(userID)
	if !ok || balance != 42.0 {
		t.Errorf("seeded balance = %v (ok=%v), want 42", balance, ok)
	}
}

func TestRegister(t *testing.T) {
	t.Parallel()
	database := openTestDB(t, "TestWalletRegister")
	svc := newTestService(t, database)

	svc.Register("new-user")
	if err := svc.Check("new-user"); err != ErrInsufficientBalance {
		t.Errorf("Check after Register = %v, want ErrInsufficientBalance (zero balance)", err)
	}
}
