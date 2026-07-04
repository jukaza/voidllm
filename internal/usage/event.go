// Package usage provides async usage event logging for the proxy hot path.
package usage

// Event represents a single proxy request for usage tracking.
type Event struct {
	KeyID              string
	KeyType            string
	UserID             string
	ModelName          string
	RequestedModelName string
	PromptTokens       int
	CompletionTokens   int
	TotalTokens        int
	CostEstimate       *float64
	DurationMS         int
	TTFT_MS            *int
	TokensPerSecond    *float64
	StatusCode         int
	RequestID          string
	CachedTokens       int
	Revenue            *float64
	DeploymentID       string
}