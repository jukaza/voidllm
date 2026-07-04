package db

// OverlayProviderConnection fills empty deployment connection fields from a
// provider record. provKey is the decrypted provider API key.
func OverlayProviderConnection(provider, baseURL, apiKey string, prov *Provider, provKey string) (string, string, string) {
	if prov == nil {
		return provider, baseURL, apiKey
	}
	if provider == "" {
		provider = prov.Protocol
	}
	if baseURL == "" {
		baseURL = prov.BaseURL
	}
	if apiKey == "" {
		apiKey = provKey
	}
	return provider, baseURL, apiKey
}