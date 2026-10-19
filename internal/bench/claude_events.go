package bench

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
)

type claudeOutput struct {
	Type             string                       `json:"type"`
	Subtype          string                       `json:"subtype"`
	IsError          *bool                        `json:"is_error"`
	SessionID        string                       `json:"session_id"`
	Result           *string                      `json:"result"`
	StructuredOutput json.RawMessage              `json:"structured_output"`
	Usage            *ClaudeTokens                `json:"usage"`
	Models           map[string]*ClaudeModelUsage `json:"modelUsage"`
	TotalCostUSD     *float64                     `json:"total_cost_usd"`
	NumTurns         *int64                       `json:"num_turns"`
	StopReason       *string                      `json:"stop_reason"`
	TerminalReason   *string                      `json:"terminal_reason"`
	Errors           []string                     `json:"errors"`
	Error            json.RawMessage              `json:"error"`
	APIError         json.RawMessage              `json:"api_error"`
	APIErrorStatus   *int64                       `json:"api_error_status"`
	DeferredTool     json.RawMessage              `json:"deferred_tool_use"`
}

func parseClaudeStructuredOutput(stdout string) (AdapterResponse, error) {
	return parseClaudeResult(stdout, true)
}

// parseClaudeOutput consumes one --print --output-format json result, not a stream.
// Native schema: Claude Code 2.1.292 and the official Agent SDK ResultMessage.
// Accounting scopes: https://code.claude.com/docs/en/agent-sdk/cost-tracking
func parseClaudeOutput(stdout string) (AdapterResponse, error) {
	return parseClaudeResult(stdout, false)
}

func parseClaudeResult(stdout string, structured bool) (AdapterResponse, error) {
	r := AdapterResponse{Version: 1, Native: &NativeEvidence{Terminal: "incomplete", UsageStatus: "missing"}}
	if len(stdout) > maxAdapterBytes {
		return r, fmt.Errorf("claude output exceeds 1 MiB")
	}
	if strings.TrimSpace(stdout) == "" {
		return r, fmt.Errorf("claude output is empty")
	}
	if err := uniqueJSON([]byte(stdout)); err != nil {
		r.Native.UsageStatus = "invalid"
		return r, fmt.Errorf("claude JSON: %w", err)
	}
	var output claudeOutput
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		r.Native.UsageStatus = "invalid"
		return r, fmt.Errorf("claude JSON: %w", err)
	}
	n := r.Native
	n.SessionID, n.EventCount = output.SessionID, 1
	if err := consumeClaudeUsage(&r, output); err != nil {
		r.Usage, n.UsageStatus = nil, "invalid"
		return r, err
	}
	if output.Type != "result" || output.IsError == nil || strings.TrimSpace(output.SessionID) == "" {
		return r, fmt.Errorf("claude output requires result type, is_error and session")
	}
	if err := validClaudeCount(output.NumTurns); err != nil {
		return r, fmt.Errorf("claude turns: %w", err)
	}
	apiFailure := claudePresent(output.Error) || claudePresent(output.APIError) || output.APIErrorStatus != nil
	failure := *output.IsError || len(output.Errors) > 0 || apiFailure
	if output.Subtype != "success" || failure {
		n.Terminal = "failed"
		// A crash can zero all native counters. API failures may likewise leave
		// unaccounted requests. Preserve the report without calling it complete.
		if apiFailure || output.Subtype == "error_during_execution" || output.Subtype == "success" || !knownClaudeFailure(output.Subtype) {
			r.Usage = nil
			if n.UsageStatus == "reported" {
				n.UsageStatus = "partial"
			}
		}
		return r, fmt.Errorf("claude did not complete successfully; see retained native output")
	}
	if (output.TerminalReason != nil && *output.TerminalReason != "completed") || claudePresent(output.DeferredTool) {
		return r, fmt.Errorf("claude result ended without completing the task")
	}
	// Claude's StructuredOutput tool ends the turn with stop_reason=tool_use.
	// Only schema mode can accept that; its validated native object is required below.
	structuredTool := structured && output.StopReason != nil && *output.StopReason == "tool_use"
	if output.StopReason != nil && *output.StopReason != "end_turn" && *output.StopReason != "stop_sequence" && !structuredTool {
		return r, fmt.Errorf("claude result has an incomplete stop reason")
	}
	var finalJSON []byte
	if structured {
		if !claudePresent(output.StructuredOutput) {
			return r, fmt.Errorf("claude result requires structured_output")
		}
		finalJSON = output.StructuredOutput
		// JSON Schema property names are case-sensitive; decoding straight into a
		// Go struct would also accept Source or SOURCE and overwrite source aliases.
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(finalJSON, &fields); err != nil || len(fields) != 1 || fields["source"] == nil {
			return r, fmt.Errorf("claude structured_output requires only the source property")
		}
	} else {
		if output.Result == nil {
			return r, fmt.Errorf("claude result requires final response text")
		}
		finalJSON = []byte(*output.Result)
	}
	n.Terminal = "completed"
	var final struct {
		Source string `json:"source"`
	}
	if err := uniqueJSON(finalJSON); err != nil {
		return r, fmt.Errorf("claude final message: %w", err)
	}
	if err := strictJSON(finalJSON, &final); err != nil {
		return r, fmt.Errorf("claude final message: %w", err)
	}
	if strings.TrimSpace(final.Source) == "" {
		return r, fmt.Errorf("claude final message requires nonempty source")
	}
	r.Source = final.Source
	return r, nil
}

func claudePresent(value json.RawMessage) bool {
	return len(value) > 0 && !bytes.Equal(bytes.TrimSpace(value), []byte("null"))
}

func knownClaudeFailure(subtype string) bool {
	return subtype == "error_max_turns" || subtype == "error_max_budget_usd" || subtype == "error_max_structured_output_retries"
}

func consumeClaudeUsage(r *AdapterResponse, output claudeOutput) error {
	n := r.Native
	if output.Usage == nil && output.Models == nil && output.TotalCostUSD == nil {
		return nil
	}
	n.ClaudeReportedUsage = &ClaudeReportedUsage{Usage: output.Usage, Models: output.Models, TotalCostUSD: output.TotalCostUSD}
	n.UsageStatus = "partial"
	if err := validClaudeCost(output.TotalCostUSD); err != nil {
		return err
	}
	if output.Usage != nil {
		for _, count := range []*int64{output.Usage.Input, output.Usage.Output, output.Usage.CacheCreation, output.Usage.CacheRead} {
			if err := validClaudeCount(count); err != nil {
				return fmt.Errorf("claude main-loop usage: %w", err)
			}
		}
	}
	complete, thinkingComplete := len(output.Models) > 0, len(output.Models) > 0
	var input, outputTokens, creation, read, thinking int64
	for model := range output.Models {
		n.ReportedModels = append(n.ReportedModels, model)
	}
	sort.Strings(n.ReportedModels)
	for _, model := range n.ReportedModels {
		if strings.TrimSpace(model) == "" {
			return fmt.Errorf("claude model identifier is empty")
		}
		u := output.Models[model]
		if u == nil {
			complete, thinkingComplete = false, false
			continue
		}
		// The MVP pins an undated first-party model ID. Other canonical aliases
		// require an explicit telemetry-contract review before being accepted.
		if u.CanonicalModel != nil && *u.CanonicalModel != model {
			return fmt.Errorf("claude canonical pricing model contradicts reported model")
		}
		if u.Provider != nil && *u.Provider != "firstParty" {
			return fmt.Errorf("claude reported a provider other than direct Anthropic")
		}
		for _, count := range []*int64{u.Input, u.Output, u.CacheCreation, u.CacheRead, u.Thinking, u.WebSearch} {
			if err := validClaudeCount(count); err != nil {
				return fmt.Errorf("claude model usage for %q: %w", model, err)
			}
		}
		if err := validClaudeCost(u.CostUSD); err != nil {
			return err
		}
		if u.Thinking != nil && u.Output != nil && *u.Thinking > *u.Output {
			return fmt.Errorf("claude thinking tokens exceed output")
		}
		if u.Input == nil || u.Output == nil || u.CacheCreation == nil || u.CacheRead == nil {
			complete = false
			continue
		}
		input += *u.Input
		outputTokens += *u.Output
		creation += *u.CacheCreation
		read += *u.CacheRead
		if u.Thinking == nil {
			thinkingComplete = false
		} else {
			thinking += *u.Thinking
		}
	}
	totalInput := input + creation + read
	for _, count := range []*int64{&totalInput, &outputTokens, &thinking} {
		if err := validClaudeCount(count); err != nil {
			return fmt.Errorf("claude aggregate usage: %w", err)
		}
	}
	if !complete {
		return nil
	}
	// Main-loop usage overlaps modelUsage. It is not another billable bucket,
	// and may be smaller because helpers and the budget-crossing turn are omitted.
	if u := output.Usage; u != nil {
		for _, pair := range []struct {
			main *int64
			all  int64
		}{{u.Input, input}, {u.Output, outputTokens}, {u.CacheCreation, creation}, {u.CacheRead, read}} {
			if pair.main != nil && *pair.main > pair.all {
				return fmt.Errorf("claude main-loop tokens exceed whole-call model usage")
			}
		}
	}
	r.Usage = &Usage{InputTokens: totalInput, OutputTokens: outputTokens, CachedInputTokens: &read}
	if thinkingComplete {
		r.Usage.ReasoningOutputTokens = &thinking
	}
	n.UsageStatus = "reported"
	return nil
}

func validClaudeCount(count *int64) error {
	if count != nil && (*count < 0 || *count > 1e12) {
		return fmt.Errorf("invalid native token or call count")
	}
	return nil
}

func validClaudeCost(cost *float64) error {
	if cost != nil && (math.IsNaN(*cost) || math.IsInf(*cost, 0) || *cost < 0 || *cost > 1e12) {
		return fmt.Errorf("invalid claude estimated cost")
	}
	return nil
}
