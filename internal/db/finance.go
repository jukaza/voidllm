package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const topupDateExpr = "COALESCE(tr.completed_at, tr.created_at)"

// FinanceSummary holds aggregate wallet cash-flow metrics.
type FinanceSummary struct {
	WalletLiability     float64
	TopupInflow         float64
	TopupPayAmount      float64
	TopupBonus          float64
	UsageOutflow        float64
	AdjustmentNet       float64
	RefundTotal         float64
	PendingTopupCount   int64
	CompletedTopupCount int64
}

// FinanceDailyBucket is one day of cash-flow aggregates (Asia/Ho_Chi_Minh).
type FinanceDailyBucket struct {
	Day           string
	TopupInflow   float64
	UsageOutflow  float64
	AdjustmentNet float64
	RefundTotal   float64
	NetFlow       float64
}

// TopupListFilter describes an admin top-up list query.
type TopupListFilter struct {
	From, To time.Time
	Status   string
	UserID   string
	Cursor   string
	Limit    int
	SkipDate bool // true when status=pending — return all pending regardless of age
}

// TopupWithUser is a top-up order with customer identity for admin views.
type TopupWithUser struct {
	TopupRequest
	UserEmail       string
	UserDisplayName string
}

// TransactionListFilter describes an admin ledger list query.
type TransactionListFilter struct {
	From, To time.Time
	Type   string
	UserID string
	Cursor string
	Limit  int
}

// TransactionWithUser is a ledger row with customer identity for admin views.
type TransactionWithUser struct {
	Transaction
	UserEmail       string
	UserDisplayName string
}

// GetFinanceSummary returns aggregate totals and daily buckets for a time range.
func (d *DB) GetFinanceSummary(ctx context.Context, from, to time.Time) (*FinanceSummary, []FinanceDailyBucket, error) {
	p := d.dialect.Placeholder
	fromStr := from.UTC().Format(time.RFC3339)
	toStr := to.UTC().Format(time.RFC3339)

	summary := &FinanceSummary{}

	if err := d.sql.QueryRowContext(ctx,
		"SELECT COALESCE(SUM(balance), 0) FROM wallets",
	).Scan(&summary.WalletLiability); err != nil {
		return nil, nil, fmt.Errorf("finance summary wallet liability: %w", err)
	}

	txRange := d.dialect.TimestampGTE("created_at", p(1)) + " AND " + d.dialect.TimestampLTE("created_at", p(2))

	typeAgg := func(txType string, dest *float64, abs bool) error {
		expr := "COALESCE(SUM(amount), 0)"
		if abs {
			expr = "COALESCE(ABS(SUM(amount)), 0)"
		}
		q := "SELECT " + expr + " FROM transactions WHERE type = " + p(3) + " AND " + txRange
		return d.sql.QueryRowContext(ctx, q, fromStr, toStr, txType).Scan(dest)
	}

	if err := typeAgg("topup", &summary.TopupInflow, false); err != nil {
		return nil, nil, fmt.Errorf("finance summary topup inflow: %w", err)
	}
	// Billed API usage lives in usage_events.revenue. Wallet ledger debits (transactions
	// type=usage) are only written when enforce_balance is on, so events are authoritative.
	usageRange := d.dialect.TimestampGTE("created_at", p(1)) + " AND " + d.dialect.TimestampLTE("created_at", p(2))
	if err := d.sql.QueryRowContext(ctx,
		"SELECT COALESCE(SUM(revenue), 0) FROM usage_events WHERE revenue IS NOT NULL AND revenue > 0 AND "+usageRange,
		fromStr, toStr,
	).Scan(&summary.UsageOutflow); err != nil {
		return nil, nil, fmt.Errorf("finance summary usage outflow: %w", err)
	}
	if err := typeAgg("adjustment", &summary.AdjustmentNet, false); err != nil {
		return nil, nil, fmt.Errorf("finance summary adjustment: %w", err)
	}
	if err := typeAgg("refund", &summary.RefundTotal, false); err != nil {
		return nil, nil, fmt.Errorf("finance summary refund: %w", err)
	}

	if err := d.sql.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM topup_requests WHERE status = 'pending'",
	).Scan(&summary.PendingTopupCount); err != nil {
		return nil, nil, fmt.Errorf("finance summary pending count: %w", err)
	}

	topupRange := d.dialect.TimestampGTE("completed_at", p(1)) + " AND " + d.dialect.TimestampLTE("completed_at", p(2))
	if err := d.sql.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM topup_requests WHERE status = 'completed' AND "+topupRange,
		fromStr, toStr,
	).Scan(&summary.CompletedTopupCount); err != nil {
		return nil, nil, fmt.Errorf("finance summary completed count: %w", err)
	}
	if err := d.sql.QueryRowContext(ctx,
		"SELECT COALESCE(SUM(pay_amount), 0), COALESCE(SUM(bonus_amount), 0) FROM topup_requests WHERE status = 'completed' AND "+topupRange,
		fromStr, toStr,
	).Scan(&summary.TopupPayAmount, &summary.TopupBonus); err != nil {
		return nil, nil, fmt.Errorf("finance summary pay/bonus: %w", err)
	}

	daily, err := d.financeDailyBuckets(ctx, fromStr, toStr)
	if err != nil {
		return nil, nil, err
	}
	return summary, daily, nil
}

func (d *DB) financeDailyBuckets(ctx context.Context, fromStr, toStr string) ([]FinanceDailyBucket, error) {
	p := d.dialect.Placeholder
	txRange := d.dialect.TimestampGTE("created_at", p(1)) + " AND " + d.dialect.TimestampLTE("created_at", p(2))
	dayExpr := d.dialect.DayTruncVN("created_at")

	txQuery := `SELECT ` + dayExpr + ` AS day, type, SUM(amount) AS agg
  FROM transactions
  WHERE type != 'usage' AND ` + txRange + `
  GROUP BY ` + dayExpr + `, type
  ORDER BY day ASC`

	rows, err := d.sql.QueryContext(ctx, txQuery, fromStr, toStr)
	if err != nil {
		return nil, fmt.Errorf("finance daily buckets query: %w", err)
	}
	defer rows.Close()

	byDay := make(map[string]*FinanceDailyBucket)
	var order []string
	ensureDay := func(day string) *FinanceDailyBucket {
		b, ok := byDay[day]
		if !ok {
			b = &FinanceDailyBucket{Day: day}
			byDay[day] = b
			order = append(order, day)
		}
		return b
	}
	for rows.Next() {
		var day, txType string
		var agg float64
		if err := rows.Scan(&day, &txType, &agg); err != nil {
			return nil, fmt.Errorf("finance daily buckets scan: %w", err)
		}
		b := ensureDay(day)
		switch txType {
		case "topup":
			b.TopupInflow = agg
		case "adjustment":
			b.AdjustmentNet = agg
		case "refund":
			b.RefundTotal = agg
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	_ = rows.Close()

	usageDayExpr := d.dialect.DayTruncVN("created_at")
	usageQuery := `SELECT ` + usageDayExpr + ` AS day, COALESCE(SUM(revenue), 0) AS agg
  FROM usage_events
  WHERE revenue IS NOT NULL AND revenue > 0 AND ` + txRange + `
  GROUP BY ` + usageDayExpr + `
  ORDER BY day ASC`
	usageRows, err := d.sql.QueryContext(ctx, usageQuery, fromStr, toStr)
	if err != nil {
		return nil, fmt.Errorf("finance daily usage buckets query: %w", err)
	}
	defer usageRows.Close()
	for usageRows.Next() {
		var day string
		var agg float64
		if err := usageRows.Scan(&day, &agg); err != nil {
			return nil, fmt.Errorf("finance daily usage buckets scan: %w", err)
		}
		ensureDay(day).UsageOutflow = agg
	}
	if err := usageRows.Err(); err != nil {
		return nil, err
	}

	out := make([]FinanceDailyBucket, 0, len(order))
	for _, day := range order {
		b := byDay[day]
		b.NetFlow = b.TopupInflow - b.UsageOutflow + b.AdjustmentNet + b.RefundTotal
		out = append(out, *b)
	}
	return out, nil
}

// ListTopupRequestsAdmin returns top-up orders with user identity for admin finance.
func (d *DB) ListTopupRequestsAdmin(ctx context.Context, filter TopupListFilter) ([]TopupWithUser, error) {
	p := d.dialect.Placeholder
	argN := 1
	var conditions []string
	var args []any

	if filter.Status != "" {
		conditions = append(conditions, "tr.status = "+p(argN))
		args = append(args, filter.Status)
		argN++
	}
	if filter.UserID != "" {
		conditions = append(conditions, "tr.user_id = "+p(argN))
		args = append(args, filter.UserID)
		argN++
	}
	if !filter.SkipDate {
		dateCol := topupDateExpr
		conditions = append(conditions, d.dialect.TimestampGTE(dateCol, p(argN)))
		args = append(args, filter.From.UTC().Format(time.RFC3339))
		argN++
		conditions = append(conditions, d.dialect.TimestampLTE(dateCol, p(argN)))
		args = append(args, filter.To.UTC().Format(time.RFC3339))
		argN++
	}
	if filter.Cursor != "" {
		conditions = append(conditions, "tr.id < "+p(argN))
		args = append(args, filter.Cursor)
		argN++
	}

	const adminTopupCols = "tr.id, tr.user_id, tr.amount, tr.payment_ref, tr.status, tr.reviewed_by, tr.reviewed_at, tr.note, tr.created_at, " +
		"tr.trade_no, tr.pay_amount, tr.credit_amount, tr.bonus_amount, tr.bonus_detail, tr.expires_at, tr.sepay_tx_id, tr.completed_at"
	query := "SELECT " + adminTopupCols + ", COALESCE(u.email, ''), COALESCE(u.display_name, '') " +
		"FROM topup_requests tr LEFT JOIN users u ON u.id = tr.user_id AND u.deleted_at IS NULL"
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY tr.id DESC LIMIT " + p(argN)
	args = append(args, filter.Limit)

	rows, err := d.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ListTopupRequestsAdmin query: %w", err)
	}
	defer rows.Close()

	var out []TopupWithUser
	for rows.Next() {
		item, scanErr := scanTopupWithUserFromRows(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("ListTopupRequestsAdmin scan: %w", scanErr)
		}
		out = append(out, *item)
	}
	return out, rows.Err()
}

// ListTransactionsAdmin returns ledger rows with user identity for admin finance.
func (d *DB) ListTransactionsAdmin(ctx context.Context, filter TransactionListFilter) ([]TransactionWithUser, error) {
	p := d.dialect.Placeholder
	argN := 1
	var conditions []string
	var args []any

	conditions = append(conditions, d.dialect.TimestampGTE("t.created_at", p(argN)))
	args = append(args, filter.From.UTC().Format(time.RFC3339))
	argN++
	conditions = append(conditions, d.dialect.TimestampLTE("t.created_at", p(argN)))
	args = append(args, filter.To.UTC().Format(time.RFC3339))
	argN++

	if filter.Type != "" {
		conditions = append(conditions, "t.type = "+p(argN))
		args = append(args, filter.Type)
		argN++
	}
	if filter.UserID != "" {
		conditions = append(conditions, "t.user_id = "+p(argN))
		args = append(args, filter.UserID)
		argN++
	}
	if filter.Cursor != "" {
		conditions = append(conditions, "t.id < "+p(argN))
		args = append(args, filter.Cursor)
		argN++
	}

	query := "SELECT t.id, t.user_id, t.type, t.amount, t.balance_after, t.ref_id, t.description, t.created_at, " +
		"COALESCE(u.email, ''), COALESCE(u.display_name, '') " +
		"FROM transactions t LEFT JOIN users u ON u.id = t.user_id AND u.deleted_at IS NULL " +
		"WHERE " + strings.Join(conditions, " AND ") +
		" ORDER BY t.id DESC LIMIT " + p(argN)
	args = append(args, filter.Limit)

	rows, err := d.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ListTransactionsAdmin query: %w", err)
	}
	defer rows.Close()

	var out []TransactionWithUser
	for rows.Next() {
		var item TransactionWithUser
		if err := rows.Scan(
			&item.ID, &item.UserID, &item.Type, &item.Amount, &item.BalanceAfter,
			&item.RefID, &item.Description, &item.CreatedAt,
			&item.UserEmail, &item.UserDisplayName,
		); err != nil {
			return nil, fmt.Errorf("ListTransactionsAdmin scan: %w", err)
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func scanTopupWithUserFromRows(rows *sql.Rows) (*TopupWithUser, error) {
	var item TopupWithUser
	var payAmount, creditAmount sql.NullFloat64
	err := rows.Scan(
		&item.ID, &item.UserID, &item.Amount, &item.PaymentRef, &item.Status,
		&item.ReviewedBy, &item.ReviewedAt, &item.Note, &item.CreatedAt,
		&item.TradeNo, &payAmount, &creditAmount, &item.BonusAmount, &item.BonusDetail,
		&item.ExpiresAt, &item.SepayTxID, &item.CompletedAt,
		&item.UserEmail, &item.UserDisplayName,
	)
	if err != nil {
		return nil, err
	}
	if payAmount.Valid {
		item.PayAmount = payAmount.Float64
	}
	if creditAmount.Valid {
		item.CreditAmount = creditAmount.Float64
	}
	return &item, nil
}