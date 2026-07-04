package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// Wallet represents a customer's prepaid wallet. Balance is a cached
// aggregate; the append-only transactions table is the source of truth.
type Wallet struct {
	ID        string
	UserID    string
	Balance   float64
	Currency  string
	CreatedAt string
	UpdatedAt string
}

// Transaction is one append-only ledger entry. Amount is positive for credits
// (topup/adjustment/refund) and negative for debits (usage).
type Transaction struct {
	ID           string
	UserID       string
	Type         string
	Amount       float64
	BalanceAfter float64
	RefID        string
	Description  string
	CreatedAt    string
}

// TopupRequest is a manual top-up awaiting admin review.
type TopupRequest struct {
	ID         string
	UserID     string
	Amount     float64
	PaymentRef string
	Status     string
	ReviewedBy *string
	ReviewedAt *string
	Note       string
	CreatedAt  string
}

const walletSelectColumns = "id, user_id, balance, currency, created_at, updated_at"
const transactionSelectColumns = "id, user_id, type, amount, balance_after, ref_id, description, created_at"
const topupSelectColumns = "id, user_id, amount, payment_ref, status, reviewed_by, reviewed_at, note, created_at"

// CreateWallet creates an empty wallet for a user. It returns ErrConflict if
// the user already has a wallet.
func (d *DB) CreateWallet(ctx context.Context, userID, currency string) (*Wallet, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("create wallet: generate id: %w", err)
	}
	if currency == "" {
		currency = "VND"
	}

	p := d.dialect.Placeholder
	insertQuery := "INSERT INTO wallets (id, user_id, balance, currency, created_at, updated_at) " +
		"VALUES (" + p(1) + ", " + p(2) + ", 0, " + p(3) + ", CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)"
	selectQuery := "SELECT " + walletSelectColumns + " FROM wallets WHERE id = " + p(1)

	var w *Wallet
	err = d.WithTx(ctx, func(q Querier) error {
		if _, execErr := q.ExecContext(ctx, insertQuery, id.String(), userID, currency); execErr != nil {
			return translateError(execErr)
		}
		row := q.QueryRowContext(ctx, selectQuery, id.String())
		var scanErr error
		w, scanErr = scanWallet(row)
		return scanErr
	})
	if err != nil {
		return nil, fmt.Errorf("create wallet: %w", err)
	}
	return w, nil
}

// GetWalletByUser retrieves a user's wallet. Returns ErrNotFound when the
// user has no wallet.
func (d *DB) GetWalletByUser(ctx context.Context, userID string) (*Wallet, error) {
	query := "SELECT " + walletSelectColumns + " FROM wallets WHERE user_id = " + d.dialect.Placeholder(1)
	row := d.sql.QueryRowContext(ctx, query, userID)
	w, err := scanWallet(row)
	if err != nil {
		return nil, fmt.Errorf("GetWalletByUser %s: %w", userID, translateError(err))
	}
	return w, nil
}

// ApplyTransactionParams describes one ledger entry to append.
type ApplyTransactionParams struct {
	UserID      string
	Type        string // 'topup' | 'usage' | 'adjustment' | 'refund'
	Amount      float64
	RefID       string
	Description string
}

// ApplyTransaction atomically appends a ledger entry and updates the cached
// wallet balance. It returns the new balance. The wallet row is locked for
// the duration of the transaction (UPDATE first) so concurrent applies
// serialize per user. Returns ErrNotFound when the user has no wallet.
func (d *DB) ApplyTransaction(ctx context.Context, params ApplyTransactionParams) (float64, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return 0, fmt.Errorf("apply transaction: generate id: %w", err)
	}

	p := d.dialect.Placeholder
	updateQuery := "UPDATE wallets SET balance = balance + " + p(1) + ", updated_at = CURRENT_TIMESTAMP" +
		" WHERE user_id = " + p(2)
	balanceQuery := "SELECT balance FROM wallets WHERE user_id = " + p(1)
	insertQuery := "INSERT INTO transactions (id, user_id, type, amount, balance_after, ref_id, description, created_at) " +
		"VALUES (" + p(1) + ", " + p(2) + ", " + p(3) + ", " + p(4) + ", " + p(5) + ", " + p(6) + ", " + p(7) + ", CURRENT_TIMESTAMP)"

	var newBalance float64
	err = d.WithTx(ctx, func(q Querier) error {
		res, execErr := q.ExecContext(ctx, updateQuery, params.Amount, params.UserID)
		if execErr != nil {
			return translateError(execErr)
		}
		affected, raErr := res.RowsAffected()
		if raErr != nil {
			return raErr
		}
		if affected == 0 {
			return ErrNotFound
		}
		if scanErr := q.QueryRowContext(ctx, balanceQuery, params.UserID).Scan(&newBalance); scanErr != nil {
			return scanErr
		}
		_, execErr = q.ExecContext(ctx, insertQuery,
			id.String(), params.UserID, params.Type, params.Amount, newBalance, params.RefID, params.Description)
		return translateError(execErr)
	})
	if err != nil {
		return 0, fmt.Errorf("ApplyTransaction user=%s: %w", params.UserID, err)
	}
	return newBalance, nil
}

// ListTransactions returns a page of a user's ledger entries, newest first.
// cursor is an exclusive upper bound on ID for keyset pagination (IDs are
// UUIDv7, so descending ID order is descending time order).
func (d *DB) ListTransactions(ctx context.Context, userID, cursor string, limit int) ([]Transaction, error) {
	p := d.dialect.Placeholder
	argN := 1
	conditions := []string{"user_id = " + p(argN)}
	args := []any{userID}
	argN++

	if cursor != "" {
		conditions = append(conditions, "id < "+p(argN))
		args = append(args, cursor)
		argN++
	}

	query := "SELECT " + transactionSelectColumns + " FROM transactions WHERE " +
		strings.Join(conditions, " AND ") + " ORDER BY id DESC LIMIT " + p(argN)
	args = append(args, limit)

	rows, err := d.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ListTransactions query: %w", err)
	}
	defer rows.Close()

	var txs []Transaction
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ID, &t.UserID, &t.Type, &t.Amount, &t.BalanceAfter, &t.RefID, &t.Description, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("ListTransactions scan: %w", err)
		}
		txs = append(txs, t)
	}
	return txs, rows.Err()
}

// LoadAllWalletBalances returns (user_id, balance) for every wallet. Used to
// seed the in-memory wallet service at startup.
func (d *DB) LoadAllWalletBalances(ctx context.Context) (map[string]float64, error) {
	rows, err := d.sql.QueryContext(ctx, "SELECT user_id, balance FROM wallets")
	if err != nil {
		return nil, fmt.Errorf("LoadAllWalletBalances query: %w", err)
	}
	defer rows.Close()

	balances := make(map[string]float64)
	for rows.Next() {
		var userID string
		var balance float64
		if err := rows.Scan(&userID, &balance); err != nil {
			return nil, fmt.Errorf("LoadAllWalletBalances scan: %w", err)
		}
		balances[userID] = balance
	}
	return balances, rows.Err()
}

// CreateTopupRequest records a customer's manual top-up request in status
// 'pending'.
func (d *DB) CreateTopupRequest(ctx context.Context, userID string, amount float64, paymentRef string) (*TopupRequest, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("create topup request: generate id: %w", err)
	}

	p := d.dialect.Placeholder
	insertQuery := "INSERT INTO topup_requests (id, user_id, amount, payment_ref, status, created_at) " +
		"VALUES (" + p(1) + ", " + p(2) + ", " + p(3) + ", " + p(4) + ", 'pending', CURRENT_TIMESTAMP)"
	selectQuery := "SELECT " + topupSelectColumns + " FROM topup_requests WHERE id = " + p(1)

	var tr *TopupRequest
	err = d.WithTx(ctx, func(q Querier) error {
		if _, execErr := q.ExecContext(ctx, insertQuery, id.String(), userID, amount, paymentRef); execErr != nil {
			return translateError(execErr)
		}
		row := q.QueryRowContext(ctx, selectQuery, id.String())
		var scanErr error
		tr, scanErr = scanTopupRequest(row)
		return scanErr
	})
	if err != nil {
		return nil, fmt.Errorf("create topup request: %w", err)
	}
	return tr, nil
}

// GetTopupRequest retrieves a top-up request by ID.
func (d *DB) GetTopupRequest(ctx context.Context, id string) (*TopupRequest, error) {
	query := "SELECT " + topupSelectColumns + " FROM topup_requests WHERE id = " + d.dialect.Placeholder(1)
	row := d.sql.QueryRowContext(ctx, query, id)
	tr, err := scanTopupRequest(row)
	if err != nil {
		return nil, fmt.Errorf("GetTopupRequest %s: %w", id, translateError(err))
	}
	return tr, nil
}

// ListTopupRequests returns a page of top-up requests, newest first,
// optionally filtered by status and/or user.
func (d *DB) ListTopupRequests(ctx context.Context, status, userID, cursor string, limit int) ([]TopupRequest, error) {
	p := d.dialect.Placeholder
	argN := 1
	var conditions []string
	var args []any

	if status != "" {
		conditions = append(conditions, "status = "+p(argN))
		args = append(args, status)
		argN++
	}
	if userID != "" {
		conditions = append(conditions, "user_id = "+p(argN))
		args = append(args, userID)
		argN++
	}
	if cursor != "" {
		conditions = append(conditions, "id < "+p(argN))
		args = append(args, cursor)
		argN++
	}

	query := "SELECT " + topupSelectColumns + " FROM topup_requests"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY id DESC LIMIT " + p(argN)
	args = append(args, limit)

	rows, err := d.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ListTopupRequests query: %w", err)
	}
	defer rows.Close()

	var reqs []TopupRequest
	for rows.Next() {
		var t TopupRequest
		if err := rows.Scan(&t.ID, &t.UserID, &t.Amount, &t.PaymentRef, &t.Status, &t.ReviewedBy, &t.ReviewedAt, &t.Note, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("ListTopupRequests scan: %w", err)
		}
		reqs = append(reqs, t)
	}
	return reqs, rows.Err()
}

// ReviewTopupRequest transitions a pending request to 'approved' or
// 'rejected'. On approval it atomically appends a 'topup' transaction and
// credits the wallet, returning the new balance (0 on rejection).
// Returns ErrNotFound if the request does not exist or is not pending.
func (d *DB) ReviewTopupRequest(ctx context.Context, requestID, reviewerID, newStatus, note string) (float64, error) {
	if newStatus != "approved" && newStatus != "rejected" {
		return 0, fmt.Errorf("ReviewTopupRequest: invalid status %q", newStatus)
	}

	p := d.dialect.Placeholder
	// Guard on status='pending' so double-review is impossible.
	updateQuery := "UPDATE topup_requests SET status = " + p(1) + ", reviewed_by = " + p(2) +
		", reviewed_at = CURRENT_TIMESTAMP, note = " + p(3) +
		" WHERE id = " + p(4) + " AND status = 'pending'"
	selectQuery := "SELECT user_id, amount FROM topup_requests WHERE id = " + p(1)

	var newBalance float64
	err := d.WithTx(ctx, func(q Querier) error {
		res, execErr := q.ExecContext(ctx, updateQuery, newStatus, reviewerID, note, requestID)
		if execErr != nil {
			return translateError(execErr)
		}
		affected, raErr := res.RowsAffected()
		if raErr != nil {
			return raErr
		}
		if affected == 0 {
			return ErrNotFound
		}
		if newStatus != "approved" {
			return nil
		}

		var userID string
		var amount float64
		if scanErr := q.QueryRowContext(ctx, selectQuery, requestID).Scan(&userID, &amount); scanErr != nil {
			return scanErr
		}

		txID, idErr := uuid.NewV7()
		if idErr != nil {
			return idErr
		}
		creditQuery := "UPDATE wallets SET balance = balance + " + p(1) + ", updated_at = CURRENT_TIMESTAMP WHERE user_id = " + p(2)
		res, execErr = q.ExecContext(ctx, creditQuery, amount, userID)
		if execErr != nil {
			return translateError(execErr)
		}
		affected, raErr = res.RowsAffected()
		if raErr != nil {
			return raErr
		}
		if affected == 0 {
			return fmt.Errorf("user %s has no wallet: %w", userID, ErrNotFound)
		}
		if scanErr := q.QueryRowContext(ctx, "SELECT balance FROM wallets WHERE user_id = "+p(1), userID).Scan(&newBalance); scanErr != nil {
			return scanErr
		}
		insertTx := "INSERT INTO transactions (id, user_id, type, amount, balance_after, ref_id, description, created_at) " +
			"VALUES (" + p(1) + ", " + p(2) + ", 'topup', " + p(3) + ", " + p(4) + ", " + p(5) + ", 'Top-up approved', CURRENT_TIMESTAMP)"
		_, execErr = q.ExecContext(ctx, insertTx, txID.String(), userID, amount, newBalance, requestID)
		return translateError(execErr)
	})
	if err != nil {
		return 0, fmt.Errorf("ReviewTopupRequest %s: %w", requestID, err)
	}
	return newBalance, nil
}

func scanWallet(row *sql.Row) (*Wallet, error) {
	var w Wallet
	err := row.Scan(&w.ID, &w.UserID, &w.Balance, &w.Currency, &w.CreatedAt, &w.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func scanTopupRequest(row *sql.Row) (*TopupRequest, error) {
	var t TopupRequest
	err := row.Scan(&t.ID, &t.UserID, &t.Amount, &t.PaymentRef, &t.Status, &t.ReviewedBy, &t.ReviewedAt, &t.Note, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
