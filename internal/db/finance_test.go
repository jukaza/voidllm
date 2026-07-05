package db

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestListTopupRequestsAdmin_DateFilterWithUserJoin(t *testing.T) {
	t.Parallel()
	d := openMigratedDB(t)
	ctx := context.Background()

	user := mustCreateUser(t, d, CreateUserParams{Email: "topup@example.com", DisplayName: "Topup User"})

	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	_, err := d.sql.ExecContext(ctx,
		`INSERT INTO topup_requests (id, user_id, amount, payment_ref, status, note, created_at, trade_no, pay_amount, credit_amount, bonus_amount, bonus_detail, expires_at)
		 VALUES (?, ?, ?, ?, 'pending', '', ?, 'VLTESTNO1', 100000, 100000, 0, '', ?)`,
		"019f0000-0000-7000-8000-000000000001", user.ID, 100000, "REF1", now, now,
	)
	if err != nil {
		t.Fatalf("insert topup: %v", err)
	}

	from := time.Now().UTC().Add(-24 * time.Hour)
	to := time.Now().UTC().Add(24 * time.Hour)

	reqs, err := d.ListTopupRequestsAdmin(ctx, TopupListFilter{
		From: from, To: to, Limit: 10,
	})
	if err != nil {
		t.Fatalf("ListTopupRequestsAdmin: %v", err)
	}
	if len(reqs) != 1 {
		t.Fatalf("got %d topups, want 1", len(reqs))
	}
	if reqs[0].UserEmail != user.Email {
		t.Fatalf("user email = %q, want %q", reqs[0].UserEmail, user.Email)
	}
}

func TestManualReviewTopup_ApproveCreditsWallet(t *testing.T) {
	t.Parallel()
	d := openMigratedDB(t)
	ctx := context.Background()

	user := mustCreateUser(t, d, CreateUserParams{Email: "approve@example.com", DisplayName: "Approve User"})
	admin := mustCreateUser(t, d, CreateUserParams{Email: "admin@example.com", DisplayName: "Admin"})
	if _, err := d.CreateWallet(ctx, user.ID, "VND"); err != nil {
		t.Fatalf("CreateWallet: %v", err)
	}

	topupID := "019f0000-0000-7000-8000-000000000002"
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	_, err := d.sql.ExecContext(ctx,
		`INSERT INTO topup_requests (id, user_id, amount, payment_ref, status, note, created_at, trade_no, pay_amount, credit_amount, bonus_amount, bonus_detail, expires_at)
		 VALUES (?, ?, ?, ?, 'pending', '', ?, 'VLTESTNO2', 50000, 55000, 5000, '', ?)`,
		topupID, user.ID, 50000, "REF2", now, now,
	)
	if err != nil {
		t.Fatalf("insert topup: %v", err)
	}

	balance, err := d.ManualReviewTopup(ctx, topupID, admin.ID, "approve", "manual fallback", 55000, 5000, "")
	if err != nil {
		t.Fatalf("ManualReviewTopup: %v", err)
	}
	if balance != 55000 {
		t.Fatalf("balance = %v, want 55000", balance)
	}

	tr, err := d.GetTopupRequest(ctx, topupID)
	if err != nil {
		t.Fatalf("GetTopupRequest: %v", err)
	}
	if tr.Status != "completed" {
		t.Fatalf("status = %q, want completed", tr.Status)
	}
	if tr.SepayTxID != "manual" {
		t.Fatalf("sepay_tx_id = %q, want manual", tr.SepayTxID)
	}
}

func TestManualReviewTopup_ApproveExpiredOrder(t *testing.T) {
	t.Parallel()
	d := openMigratedDB(t)
	ctx := context.Background()

	user := mustCreateUser(t, d, CreateUserParams{Email: "expired@example.com", DisplayName: "Expired User"})
	admin := mustCreateUser(t, d, CreateUserParams{Email: "admin2@example.com", DisplayName: "Admin"})
	if _, err := d.CreateWallet(ctx, user.ID, "VND"); err != nil {
		t.Fatalf("CreateWallet: %v", err)
	}

	topupID := "019f0000-0000-7000-8000-000000000003"
	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	_, err := d.sql.ExecContext(ctx,
		`INSERT INTO topup_requests (id, user_id, amount, payment_ref, status, note, created_at, trade_no, pay_amount, credit_amount, bonus_amount, bonus_detail, expires_at)
		 VALUES (?, ?, ?, ?, 'expired', '', ?, 'VLTESTNO3', 75000, 75000, 0, '', ?)`,
		topupID, user.ID, 75000, "REF3", now, now,
	)
	if err != nil {
		t.Fatalf("insert topup: %v", err)
	}

	balance, err := d.ManualReviewTopup(ctx, topupID, admin.ID, "approve", "late payment", 75000, 0, "")
	if err != nil {
		t.Fatalf("ManualReviewTopup expired: %v", err)
	}
	if balance != 75000 {
		t.Fatalf("balance = %v, want 75000", balance)
	}
}

func TestCompleteSepayTopup_AmountMismatch(t *testing.T) {
	t.Parallel()
	d := openMigratedDB(t)
	ctx := context.Background()

	user := mustCreateUser(t, d, CreateUserParams{Email: "mismatch@example.com", DisplayName: "Mismatch User"})
	if _, err := d.CreateWallet(ctx, user.ID, "VND"); err != nil {
		t.Fatalf("CreateWallet: %v", err)
	}

	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	_, err := d.sql.ExecContext(ctx,
		`INSERT INTO topup_requests (id, user_id, amount, payment_ref, status, note, created_at, trade_no, pay_amount, credit_amount, bonus_amount, bonus_detail, expires_at)
		 VALUES (?, ?, ?, ?, 'pending', '', ?, 'VLTESTNO4', 100000, 100000, 0, '', ?)`,
		"019f0000-0000-7000-8000-000000000004", user.ID, 100000, "REF4", now, now,
	)
	if err != nil {
		t.Fatalf("insert topup: %v", err)
	}

	_, err = d.CompleteSepayTopup(ctx, "VLTESTNO4", "tx-1", 50000, 100000, 0, "")
	if err == nil {
		t.Fatal("expected amount mismatch error")
	}
	if !errors.Is(err, ErrAmountMismatch) {
		t.Fatalf("err = %v, want ErrAmountMismatch", err)
	}
}

func TestCompleteSepayTopup_ExpiredOrder(t *testing.T) {
	t.Parallel()
	d := openMigratedDB(t)
	ctx := context.Background()

	user := mustCreateUser(t, d, CreateUserParams{Email: "expiredpay@example.com", DisplayName: "Expired Pay"})
	if _, err := d.CreateWallet(ctx, user.ID, "VND"); err != nil {
		t.Fatalf("CreateWallet: %v", err)
	}

	now := time.Now().UTC().Format("2006-01-02 15:04:05")
	_, err := d.sql.ExecContext(ctx,
		`INSERT INTO topup_requests (id, user_id, amount, payment_ref, status, note, created_at, trade_no, pay_amount, credit_amount, bonus_amount, bonus_detail, expires_at)
		 VALUES (?, ?, ?, ?, 'expired', '', ?, 'VLTESTNO5', 80000, 80000, 0, '', ?)`,
		"019f0000-0000-7000-8000-000000000005", user.ID, 80000, "REF5", now, now,
	)
	if err != nil {
		t.Fatalf("insert topup: %v", err)
	}

	balance, err := d.CompleteSepayTopup(ctx, "VLTESTNO5", "tx-2", 80000, 80000, 0, "")
	if err != nil {
		t.Fatalf("CompleteSepayTopup expired: %v", err)
	}
	if balance != 80000 {
		t.Fatalf("balance = %v, want 80000", balance)
	}
}

func TestGetFinanceSummary_UsageOutflowFromUsageEvents(t *testing.T) {
	t.Parallel()
	d := openMigratedDB(t)
	ctx := context.Background()

	now := time.Now().UTC()
	ts := now.Format("2006-01-02 15:04:05")
	_, err := d.sql.ExecContext(ctx,
		`INSERT INTO usage_events (id, user_id, key_id, key_type, request_id, model_name, prompt_tokens, completion_tokens, total_tokens, status_code, created_at, revenue)
		 VALUES (?, ?, ?, 'user_key', ?, 'gpt-test', 10, 5, 15, 200, ?, 2500),
		        (?, ?, ?, 'user_key', ?, 'gpt-test', 5, 2, 7, 200, ?, NULL)`,
		"019f1000-0000-7000-8000-000000000010", "user-1", "key-1", "req-1", ts,
		"019f1000-0000-7000-8000-000000000011", "user-1", "key-1", "req-2", ts,
	)
	if err != nil {
		t.Fatalf("insert usage_events: %v", err)
	}

	from := now.Add(-time.Hour)
	to := now.Add(time.Hour)
	summary, daily, err := d.GetFinanceSummary(ctx, from, to)
	if err != nil {
		t.Fatalf("GetFinanceSummary: %v", err)
	}
	if summary.UsageOutflow != 2500 {
		t.Fatalf("UsageOutflow = %v, want 2500", summary.UsageOutflow)
	}
	var usageDay float64
	for _, b := range daily {
		usageDay += b.UsageOutflow
	}
	if usageDay != 2500 {
		t.Fatalf("daily usage sum = %v, want 2500", usageDay)
	}
}