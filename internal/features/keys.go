package features

// Settings keys stored in the settings table.
const (
	KeyWalletEnforceBalance = "features.wallet.enforce_balance"
	KeyWalletInitialBalance = "features.wallet.initial_balance_vnd"
	KeyModulesPublicCatalog = "features.modules.public_catalog"
	KeyModulesPlayground    = "features.modules.playground"
)

// MaxInitialBalanceVND caps signup credit to prevent operator typos.
const MaxInitialBalanceVND = 10_000_000