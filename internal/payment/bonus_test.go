package payment

import (
	"testing"
	"time"
)

func TestComputeQuoteStack(t *testing.T) {
	cfg := Config{
		BonusStackMode: "stack",
		TierBonuses: []TierBonus{
			{MinAmount: 100000, BonusPercent: 10},
		},
		Campaigns: []Campaign{
			{
				ID:           "tet",
				Name:         "Tet",
				Enabled:      true,
				StartAt:      "2020-01-01T00:00:00Z",
				EndAt:        "2099-01-01T00:00:00Z",
				BonusPercent: 15,
				MinAmount:    50000,
			},
		},
		FirstTopup: FirstTopupBonus{Enabled: true, BonusPercent: 5},
	}
	now := time.Date(2026, 1, 20, 12, 0, 0, 0, time.UTC)
	q := ComputeQuote(cfg, 100000, now, false)
	if q.BonusAmount != 30000 {
		t.Fatalf("bonus = %v, want 30000", q.BonusAmount)
	}
	if q.CreditAmount != 130000 {
		t.Fatalf("credit = %v, want 130000", q.CreditAmount)
	}
}

func TestComputeQuoteTierFixedOnly(t *testing.T) {
	cfg := Config{
		BonusStackMode: "stack",
		TierBonuses: []TierBonus{
			{MinAmount: 100000, BonusType: BonusTypeFixed, BonusFixed: 50000},
		},
	}
	q := ComputeQuote(cfg, 100000, time.Now(), true)
	if q.BonusAmount != 50000 {
		t.Fatalf("bonus = %v, want 50000", q.BonusAmount)
	}
	if q.CreditAmount != 150000 {
		t.Fatalf("credit = %v, want 150000", q.CreditAmount)
	}
}

func TestNormalizeTierBonusExclusive(t *testing.T) {
	tier := NormalizeTierBonus(TierBonus{
		MinAmount:    100000,
		BonusType:    BonusTypePercent,
		BonusPercent: 10,
		BonusFixed:   50000,
	})
	if tier.BonusFixed != 0 {
		t.Fatalf("fixed = %v, want 0 when type is percent", tier.BonusFixed)
	}
	if tier.BonusPercent != 10 {
		t.Fatalf("percent = %v, want 10", tier.BonusPercent)
	}

	tier = NormalizeTierBonus(TierBonus{
		MinAmount:    100000,
		BonusType:    BonusTypeFixed,
		BonusPercent: 10,
		BonusFixed:   50000,
	})
	if tier.BonusPercent != 0 {
		t.Fatalf("percent = %v, want 0 when type is fixed", tier.BonusPercent)
	}
}

func TestComputeQuoteStripsFirstTopupWhenAlreadyCompleted(t *testing.T) {
	cfg := Config{
		BonusStackMode: "stack",
		FirstTopup:     FirstTopupBonus{Enabled: true, BonusPercent: 10},
	}
	qNew := ComputeQuote(cfg, 100000, time.Now(), false)
	if qNew.BonusAmount != 10000 {
		t.Fatalf("new user bonus = %v, want 10000", qNew.BonusAmount)
	}
	qExisting := ComputeQuote(cfg, 100000, time.Now(), true)
	if qExisting.BonusAmount != 0 {
		t.Fatalf("existing user bonus = %v, want 0", qExisting.BonusAmount)
	}
	if qExisting.CreditAmount != 100000 {
		t.Fatalf("existing user credit = %v, want 100000", qExisting.CreditAmount)
	}
}

func TestComputeQuoteMax(t *testing.T) {
	cfg := Config{
		BonusStackMode: "max",
		TierBonuses: []TierBonus{
			{MinAmount: 100000, BonusPercent: 10},
		},
		Campaigns: []Campaign{
			{
				ID:           "sale",
				Name:         "Sale",
				Enabled:      true,
				StartAt:      "2020-01-01T00:00:00Z",
				EndAt:        "2099-01-01T00:00:00Z",
				BonusPercent: 20,
				MinAmount:    50000,
			},
		},
	}
	now := time.Date(2026, 1, 20, 12, 0, 0, 0, time.UTC)
	q := ComputeQuote(cfg, 100000, now, true)
	if q.BonusAmount != 20000 {
		t.Fatalf("bonus = %v, want 20000", q.BonusAmount)
	}
}