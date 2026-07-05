package payment

import (
	"encoding/json"
	"math"
	"time"
)

const (
	BonusTypePercent = "percent"
	BonusTypeFixed   = "fixed"
)

// BonusDetail records which promotions were applied to a top-up.
type BonusDetail struct {
	StackMode         string  `json:"stack_mode"`
	TierMinAmount     float64 `json:"tier_min_amount,omitempty"`
	TierPercent       float64 `json:"tier_percent,omitempty"`
	TierFixed         float64 `json:"tier_fixed,omitempty"`
	TierBonus         float64 `json:"tier_bonus,omitempty"`
	CampaignID        string  `json:"campaign_id,omitempty"`
	CampaignName      string  `json:"campaign_name,omitempty"`
	CampaignPercent   float64 `json:"campaign_percent,omitempty"`
	CampaignFixed     float64 `json:"campaign_fixed,omitempty"`
	CampaignBonus     float64 `json:"campaign_bonus,omitempty"`
	FirstTopupPercent float64 `json:"first_topup_percent,omitempty"`
	FirstTopupFixed   float64 `json:"first_topup_fixed,omitempty"`
	FirstTopupBonus   float64 `json:"first_topup_bonus,omitempty"`
}

// Quote is the computed top-up breakdown for a pay amount.
type Quote struct {
	PayAmount    float64     `json:"pay_amount"`
	CreditAmount float64     `json:"credit_amount"`
	BonusAmount  float64     `json:"bonus_amount"`
	BonusDetail  BonusDetail `json:"bonus_detail"`
}

// ComputeQuote calculates credit and bonus for a pay amount.
func ComputeQuote(cfg Config, payAmount float64, now time.Time, hasCompletedTopup bool) Quote {
	payAmount = roundMoney(payAmount)
	detail := BonusDetail{StackMode: cfg.BonusStackMode}

	tier := bestTier(cfg.TierBonuses, payAmount)
	var tierBonus float64
	if tier != nil {
		normalized := NormalizeTierBonus(*tier)
		detail.TierMinAmount = normalized.MinAmount
		if normalized.BonusType == BonusTypeFixed {
			detail.TierFixed = normalized.BonusFixed
		} else {
			detail.TierPercent = normalized.BonusPercent
		}
		tierBonus = bonusAmount(payAmount, normalized.BonusType, normalized.BonusPercent, normalized.BonusFixed, 0)
		detail.TierBonus = tierBonus
	}

	var campaignBonus float64
	if campaign := bestCampaign(cfg.Campaigns, payAmount, now, hasCompletedTopup); campaign != nil {
		normalized := NormalizeCampaign(*campaign)
		raw := bonusAmount(payAmount, normalized.BonusType, normalized.BonusPercent, normalized.BonusFixed, normalized.MaxBonus)
		campaignBonus = raw
		detail.CampaignID = normalized.ID
		detail.CampaignName = normalized.Name
		if normalized.BonusType == BonusTypeFixed {
			detail.CampaignFixed = normalized.BonusFixed
		} else {
			detail.CampaignPercent = normalized.BonusPercent
		}
		detail.CampaignBonus = campaignBonus
	}

	var firstTopupBonus float64
	if !hasCompletedTopup && cfg.FirstTopup.Enabled {
		normalized := NormalizeFirstTopup(cfg.FirstTopup)
		if normalized.BonusType == BonusTypeFixed {
			detail.FirstTopupFixed = normalized.BonusFixed
		} else {
			detail.FirstTopupPercent = normalized.BonusPercent
		}
		firstTopupBonus = bonusAmount(payAmount, normalized.BonusType, normalized.BonusPercent, normalized.BonusFixed, 0)
		detail.FirstTopupBonus = firstTopupBonus
	}

	var totalBonus float64
	switch cfg.BonusStackMode {
	case "max":
		totalBonus = math.Max(tierBonus, math.Max(campaignBonus, firstTopupBonus))
	default:
		totalBonus = tierBonus + campaignBonus + firstTopupBonus
	}

	return Quote{
		PayAmount:    payAmount,
		CreditAmount: roundMoney(payAmount + totalBonus),
		BonusAmount:  roundMoney(totalBonus),
		BonusDetail:  detail,
	}
}

// MarshalBonusDetail encodes bonus detail for persistence.
func MarshalBonusDetail(d BonusDetail) string {
	b, err := json.Marshal(d)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func bestTier(tiers []TierBonus, payAmount float64) *TierBonus {
	var best *TierBonus
	for i := range tiers {
		t := NormalizeTierBonus(tiers[i])
		if payAmount+0.01 < t.MinAmount {
			continue
		}
		if !hasPositiveBonus(t.BonusType, t.BonusPercent, t.BonusFixed) {
			continue
		}
		if best == nil || t.MinAmount > best.MinAmount {
			copy := t
			best = &copy
		}
	}
	return best
}

func bestCampaign(campaigns []Campaign, payAmount float64, now time.Time, hasCompletedTopup bool) *Campaign {
	var best *Campaign
	var bestBonus float64
	for i := range campaigns {
		c := campaigns[i]
		if !campaignActive(c, now, hasCompletedTopup) {
			continue
		}
		if payAmount+0.01 < c.MinAmount {
			continue
		}
		normalized := NormalizeCampaign(c)
		if !hasPositiveBonus(normalized.BonusType, normalized.BonusPercent, normalized.BonusFixed) {
			continue
		}
		value := bonusAmount(payAmount, normalized.BonusType, normalized.BonusPercent, normalized.BonusFixed, normalized.MaxBonus)
		if best == nil || value > bestBonus {
			copy := normalized
			best = &copy
			bestBonus = value
		}
	}
	return best
}

func bonusAmount(payAmount float64, bonusType string, percent, fixed, maxBonus float64) float64 {
	var raw float64
	switch bonusType {
	case BonusTypeFixed:
		raw = fixed
	default:
		raw = payAmount * percent / 100
	}
	if bonusType != BonusTypeFixed && maxBonus > 0 {
		raw = math.Min(raw, maxBonus)
	}
	return roundMoney(raw)
}

func hasPositiveBonus(bonusType string, percent, fixed float64) bool {
	switch bonusType {
	case BonusTypeFixed:
		return fixed > 0
	default:
		return percent > 0
	}
}

func inferBonusType(explicit string, percent, fixed float64) string {
	switch explicit {
	case BonusTypeFixed, BonusTypePercent:
		return explicit
	}
	if fixed > 0 && percent <= 0 {
		return BonusTypeFixed
	}
	return BonusTypePercent
}

// NormalizeTierBonus keeps exactly one bonus mode: percent or fixed amount.
func NormalizeTierBonus(t TierBonus) TierBonus {
	t.BonusType = inferBonusType(t.BonusType, t.BonusPercent, t.BonusFixed)
	if t.BonusType == BonusTypeFixed {
		t.BonusPercent = 0
	} else {
		t.BonusType = BonusTypePercent
		t.BonusFixed = 0
	}
	return t
}

// NormalizeCampaign keeps exactly one bonus mode: percent or fixed amount.
func NormalizeCampaign(c Campaign) Campaign {
	c.BonusType = inferBonusType(c.BonusType, c.BonusPercent, c.BonusFixed)
	if c.BonusType == BonusTypeFixed {
		c.BonusPercent = 0
		c.MaxBonus = 0
	} else {
		c.BonusType = BonusTypePercent
		c.BonusFixed = 0
	}
	return c
}

// NormalizeFirstTopup keeps exactly one bonus mode: percent or fixed amount.
func NormalizeFirstTopup(f FirstTopupBonus) FirstTopupBonus {
	f.BonusType = inferBonusType(f.BonusType, f.BonusPercent, f.BonusFixed)
	if f.BonusType == BonusTypeFixed {
		f.BonusPercent = 0
	} else {
		f.BonusType = BonusTypePercent
		f.BonusFixed = 0
	}
	return f
}

func roundMoney(v float64) float64 {
	return math.Round(v)
}