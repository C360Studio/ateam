package bench

// ClaudeTokens preserves main-loop counters. Input excludes both cache categories;
// output already includes thinking. Missing native fields remain nil.
type ClaudeTokens struct {
	Input         *int64 `json:"input_tokens,omitempty"`
	Output        *int64 `json:"output_tokens,omitempty"`
	CacheCreation *int64 `json:"cache_creation_input_tokens,omitempty"`
	CacheRead     *int64 `json:"cache_read_input_tokens,omitempty"`
}

// ClaudeModelUsage preserves whole-call native counters, including auxiliary calls.
// CostUSD is the CLI's client-side estimate, not authoritative provider billing.
type ClaudeModelUsage struct {
	Input          *int64   `json:"inputTokens,omitempty"`
	Output         *int64   `json:"outputTokens,omitempty"`
	CacheCreation  *int64   `json:"cacheCreationInputTokens,omitempty"`
	CacheRead      *int64   `json:"cacheReadInputTokens,omitempty"`
	Thinking       *int64   `json:"thinkingTokens,omitempty"`
	WebSearch      *int64   `json:"webSearchRequests,omitempty"`
	CostUSD        *float64 `json:"costUSD,omitempty"`
	CanonicalModel *string  `json:"canonicalModel,omitempty"`
	Provider       *string  `json:"provider,omitempty"`
	CostBasis      string   `json:"costBasis,omitempty"`
}

// ClaudeReportedUsage retains the two overlapping accounting scopes separately.
// Never add main-loop Usage to Models or interpret TotalCostUSD as a hard cap.
type ClaudeReportedUsage struct {
	Usage        *ClaudeTokens                `json:"usage"`
	Models       map[string]*ClaudeModelUsage `json:"modelUsage"`
	TotalCostUSD *float64                     `json:"total_cost_usd,omitempty"`
}
