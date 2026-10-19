package bench

// CodexTokens retains native cumulative or last-request counters without filling
// missing values. Cache and reasoning counts are subsets, never extra tokens.
type CodexTokens struct {
	InputTokens           *int64 `json:"inputTokens,omitempty"`
	CachedInputTokens     *int64 `json:"cachedInputTokens,omitempty"`
	CacheWriteInputTokens *int64 `json:"cacheWriteInputTokens,omitempty"`
	OutputTokens          *int64 `json:"outputTokens,omitempty"`
	ReasoningOutputTokens *int64 `json:"reasoningOutputTokens,omitempty"`
	TotalTokens           *int64 `json:"totalTokens,omitempty"`
}

// CodexEvidence records CLI-resolved thread configuration, not backend model
// attestation. Total is the latest cumulative snapshot, not a sum of snapshots.
type CodexEvidence struct {
	IdentitySource  string       `json:"identity_source"`
	ThreadID        string       `json:"thread_id"`
	TurnID          string       `json:"turn_id"`
	Model           *string      `json:"model"`
	ModelProvider   *string      `json:"model_provider"`
	ReasoningEffort *string      `json:"reasoning_effort"`
	Total           *CodexTokens `json:"total"`
	Last            *CodexTokens `json:"last"`
}
