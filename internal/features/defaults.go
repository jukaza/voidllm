package features

// DefaultConfig is used when a settings key is missing or invalid.
func DefaultConfig() Config {
	return Config{
		Wallet: WalletConfig{
			EnforceBalance:    false,
			InitialBalanceVND: 0,
		},
		Modules: ModulesConfig{
			PublicCatalog: true,
			Playground:    true,
		},
	}
}