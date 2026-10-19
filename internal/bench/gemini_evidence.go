package bench

// GeminiTokens preserves the CLI's native token fields without filling absent values.
// Prompt includes cached input; candidates and thoughts are separate output counts.
type GeminiTokens struct {
	Prompt     *int64 `json:"prompt,omitempty"`
	Candidates *int64 `json:"candidates,omitempty"`
	Total      *int64 `json:"total,omitempty"`
	Cached     *int64 `json:"cached,omitempty"`
	Thoughts   *int64 `json:"thoughts,omitempty"`
	Tool       *int64 `json:"tool,omitempty"`
	Input      *int64 `json:"input,omitempty"`
}

// GeminiAPIUsage preserves request/error counters needed to detect unaccounted failed calls.
type GeminiAPIUsage struct {
	TotalRequests *int64 `json:"totalRequests,omitempty"`
	TotalErrors   *int64 `json:"totalErrors,omitempty"`
}

// GeminiReportedUsage keeps per-model native counts instead of summing nested role totals twice.
type GeminiReportedUsage struct {
	Models map[string]*GeminiTokens   `json:"models"`
	API    map[string]*GeminiAPIUsage `json:"api"`
}

// GeminiEvidence describes the CLI's final report; it cannot establish a backend model version.
type GeminiEvidence struct {
	SessionID          string               `json:"session_id"`
	Terminal           string               `json:"terminal"`
	UsageStatus        string               `json:"usage_status"`
	ReportedUsage      *GeminiReportedUsage `json:"reported_usage"`
	ReportedModels     []string             `json:"reported_models"`
	ObservedItemTypes  []string             `json:"observed_item_types"`
	CompletedItemTypes []string             `json:"completed_item_types"`
	EventCount         int                  `json:"event_count"`
	ResolvedModel      *string              `json:"resolved_model"`
	ResolvedReasoning  *string              `json:"resolved_reasoning"`
	Warnings           []string             `json:"warnings,omitempty"`
	OutputTruncated    bool                 `json:"output_truncated,omitempty"`
}
