// Package wallet provides the prepaid-wallet service for the API-resell
// marketplace: an in-memory balance cache for hot-path checks plus async
// debit recording through the DB transaction ledger.
package wallet

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/voidmind-io/voidllm/internal/db"
)

// ErrInsufficientBalance is returned by Check when the user's balance is
// zero or negative.
var ErrInsufficientBalance = errors.New("insufficient wallet balance")

// ErrNoWallet is returned by Check when the user has no wallet at all.
var ErrNoWallet = errors.New("no wallet for user")

// Service tracks per-user wallet balances in memory for lock-cheap hot-path
// checks. The DB transactions table is the source of truth; the in-memory
// map is a cache seeded at startup and updated on every credit/debit.
//
// Debits are applied to the in-memory balance immediately (so subsequent
// checks see the spend) and persisted asynchronously by the caller's
// pipeline (usage logger). Small negative balances between flushes are
// acceptable for a prepaid model.
type Service struct {
	mu       sync.RWMutex
	balances map[string]float64 // userID → balance
	db       *db.DB
	log      *slog.Logger
}

// NewService constructs a Service and seeds balances from the DB.
func NewService(ctx context.Context, database *db.DB, log *slog.Logger) (*Service, error) {
	balances, err := database.LoadAllWalletBalances(ctx)
	if err != nil {
		return nil, fmt.Errorf("seed wallet balances: %w", err)
	}
	return &Service{
		balances: balances,
		db:       database,
		log:      log,
	}, nil
}

// Check returns nil when the user has a wallet with positive balance.
// It returns ErrNoWallet when no wallet exists and ErrInsufficientBalance
// when the balance is zero or negative.
func (s *Service) Check(userID string) error {
	s.mu.RLock()
	balance, ok := s.balances[userID]
	s.mu.RUnlock()
	if !ok {
		return ErrNoWallet
	}
	if balance <= 0 {
		return ErrInsufficientBalance
	}
	return nil
}

// Balance returns the cached balance and whether a wallet exists.
func (s *Service) Balance(userID string) (float64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	b, ok := s.balances[userID]
	return b, ok
}

// Debit charges amount (positive) against the user's wallet: the in-memory
// balance is reduced immediately and the ledger entry is written to the DB
// synchronously. Callers on the hot path must invoke this from an async
// worker (the usage logger pipeline), never inline with the request.
func (s *Service) Debit(ctx context.Context, userID string, amount float64, refID, description string) {
	if amount <= 0 {
		return
	}
	s.DebitMemory(userID, amount)
	s.PersistDebit(ctx, userID, amount, refID, description)
}

// DebitMemory reduces only the in-memory balance. It is cheap (one mutex
// acquisition) and safe to call on the hot path so that subsequent Check
// calls see the spend before the ledger write lands. Callers must follow up
// with PersistDebit from a background worker.
func (s *Service) DebitMemory(userID string, amount float64) {
	if amount <= 0 {
		return
	}
	s.mu.Lock()
	if _, ok := s.balances[userID]; ok {
		s.balances[userID] -= amount
	}
	s.mu.Unlock()
}

// PersistDebit writes the usage ledger entry to the DB and re-syncs the
// cached balance to the authoritative DB value. It must be paired with a
// prior DebitMemory call; the re-sync corrects any drift between the two.
func (s *Service) PersistDebit(ctx context.Context, userID string, amount float64, refID, description string) {
	if amount <= 0 {
		return
	}

	newBalance, err := s.db.ApplyTransaction(ctx, db.ApplyTransactionParams{
		UserID:      userID,
		Type:        "usage",
		Amount:      -amount,
		RefID:       refID,
		Description: description,
	})
	if err != nil {
		s.log.ErrorContext(ctx, "wallet debit persist failed",
			slog.String("user_id", userID),
			slog.Float64("amount", amount),
			slog.String("error", err.Error()))
		return
	}

	// Re-sync the cache to the authoritative DB balance to correct any drift.
	s.mu.Lock()
	s.balances[userID] = newBalance
	s.mu.Unlock()
}

// Credit adds amount (positive) to the user's wallet, persisting a ledger
// entry of the given type ('topup' | 'adjustment' | 'refund') and updating
// the cache. Used by top-up approval and admin adjustments.
func (s *Service) Credit(ctx context.Context, userID string, amount float64, txType, refID, description string) (float64, error) {
	newBalance, err := s.db.ApplyTransaction(ctx, db.ApplyTransactionParams{
		UserID:      userID,
		Type:        txType,
		Amount:      amount,
		RefID:       refID,
		Description: description,
	})
	if err != nil {
		return 0, err
	}
	s.mu.Lock()
	s.balances[userID] = newBalance
	s.mu.Unlock()
	return newBalance, nil
}

// SetBalance overwrites the cached balance for a user. Used after flows that
// update the DB outside this service (e.g. ReviewTopupRequest).
func (s *Service) SetBalance(userID string, balance float64) {
	s.mu.Lock()
	s.balances[userID] = balance
	s.mu.Unlock()
}

// Register adds a zero-balance cache entry for a newly created wallet so
// that hot-path checks stop returning ErrNoWallet.
func (s *Service) Register(userID string) {
	s.mu.Lock()
	if _, ok := s.balances[userID]; !ok {
		s.balances[userID] = 0
	}
	s.mu.Unlock()
}
