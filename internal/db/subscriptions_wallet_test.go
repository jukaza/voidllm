package db

import (
	"context"
	"testing"
)

func TestPurchaseSubscriptionWithWallet_DebitsAndGrants(t *testing.T) {
	t.Parallel()
	d := openMigratedDB(t)
	ctx := context.Background()

	user := mustCreateUser(t, d, CreateUserParams{Email: "subbuy@example.com", DisplayName: "Buyer"})
	if _, err := d.CreateWallet(ctx, user.ID, "VND"); err != nil {
		t.Fatalf("CreateWallet: %v", err)
	}
	if _, err := d.ApplyTransaction(ctx, ApplyTransactionParams{
		UserID: user.ID, Type: "topup", Amount: 1_000_000, Description: "seed",
	}); err != nil {
		t.Fatalf("seed wallet: %v", err)
	}

	pkg, err := d.CreateSubscriptionPackage(ctx, CreateSubscriptionPackageParams{
		Name: "Test Pack", Enabled: true,
	})
	if err != nil {
		t.Fatalf("CreateSubscriptionPackage: %v", err)
	}
	plan, err := d.CreateSubscriptionPlan(ctx, CreateSubscriptionPlanParams{
		PackageID: pkg.ID, Name: "Basic", Price: 10_000, ValidityDays: 30, ForSale: true,
	})
	if err != nil {
		t.Fatalf("CreateSubscriptionPlan: %v", err)
	}

	us, balance, err := d.PurchaseSubscriptionWithWallet(ctx, user.ID, plan.ID)
	if err != nil {
		t.Fatalf("PurchaseSubscriptionWithWallet: %v", err)
	}
	if us == nil || us.Status != "active" {
		t.Fatalf("subscription = %+v, want active", us)
	}
	if balance != 990_000 {
		t.Fatalf("balance = %v, want 990000", balance)
	}

	w, err := d.GetWalletByUser(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetWalletByUser: %v", err)
	}
	if w.Balance != 990_000 {
		t.Fatalf("wallet balance = %v, want 990000", w.Balance)
	}
}