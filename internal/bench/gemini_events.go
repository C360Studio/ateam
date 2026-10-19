package bench

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

type geminiOutput struct {
	SessionID string          `json:"session_id"`
	Response  *string         `json:"response"`
	Stats     *geminiStats    `json:"stats"`
	Error     json.RawMessage `json:"error"`
	Warnings  []string        `json:"warnings"`
}

type geminiStats struct {
	Models map[string]*geminiModelStats `json:"models"`
	Tools  *struct {
		ByName map[string]struct {
			Count   *int64 `json:"count"`
			Success *int64 `json:"success"`
			Fail    *int64 `json:"fail"`
		} `json:"byName"`
	} `json:"tools"`
}

type geminiModelStats struct {
	API    *GeminiAPIUsage `json:"api"`
	Tokens *GeminiTokens   `json:"tokens"`
}

// parseGeminiOutput retains native evidence even when the CLI or candidate fails.
// JSON output exposes thoughts, which Gemini CLI 0.43.0 omits from stream-json.
func parseGeminiOutput(stdout string) (AdapterResponse, error) {
	r := AdapterResponse{Version: 1, Native: &GeminiEvidence{Terminal: "incomplete", UsageStatus: "missing"}}
	if len(stdout) > maxAdapterBytes {
		return r, fmt.Errorf("gemini output exceeds 1 MiB")
	}
	if strings.TrimSpace(stdout) == "" {
		return r, fmt.Errorf("gemini output is empty")
	}
	if err := uniqueJSON([]byte(stdout)); err != nil {
		r.Native.UsageStatus = "invalid"
		return r, fmt.Errorf("gemini JSON: %w", err)
	}
	var output geminiOutput
	if err := json.Unmarshal([]byte(stdout), &output); err != nil {
		r.Native.UsageStatus = "invalid"
		return r, fmt.Errorf("gemini JSON: %w", err)
	}
	n := r.Native
	n.SessionID = output.SessionID
	n.Warnings = output.Warnings
	n.EventCount = 1
	hasError := len(output.Error) > 0 && !bytes.Equal(output.Error, []byte("null"))
	if hasError {
		n.Terminal = "failed"
	} else if output.Response != nil {
		n.Terminal = "completed"
	}
	if err := consumeGeminiStats(&r, output.Stats); err != nil {
		r.Usage = nil
		n.UsageStatus = "invalid"
		return r, err
	}
	if hasError {
		return r, fmt.Errorf("gemini reported an error; see retained native output")
	}
	if n.Terminal != "completed" || strings.TrimSpace(n.SessionID) == "" {
		return r, fmt.Errorf("gemini output requires a session and completed response")
	}
	var final struct {
		Source string `json:"source"`
	}
	if err := uniqueJSON([]byte(*output.Response)); err != nil {
		return r, fmt.Errorf("gemini final message: %w", err)
	}
	if err := strictJSON([]byte(*output.Response), &final); err != nil {
		return r, fmt.Errorf("gemini final message: %w", err)
	}
	if strings.TrimSpace(final.Source) == "" {
		return r, fmt.Errorf("gemini final message requires nonempty source")
	}
	r.Source = final.Source
	return r, nil
}

func consumeGeminiStats(r *AdapterResponse, stats *geminiStats) error {
	if stats == nil {
		return nil
	}
	n := r.Native
	if err := consumeGeminiTools(n, stats); err != nil {
		return err
	}
	if len(stats.Models) == 0 {
		return nil
	}
	n.ReportedUsage = &GeminiReportedUsage{Models: map[string]*GeminiTokens{}, API: map[string]*GeminiAPIUsage{}}
	n.UsageStatus = "partial"
	complete, cacheComplete := true, true
	var input, output, cache, thoughts int64
	for model := range stats.Models {
		n.ReportedModels = append(n.ReportedModels, model)
	}
	sort.Strings(n.ReportedModels)
	for _, model := range n.ReportedModels {
		metrics := stats.Models[model]
		if strings.TrimSpace(model) == "" {
			return fmt.Errorf("gemini model identifier is empty")
		}
		if metrics == nil {
			complete = false
			n.ReportedUsage.Models[model] = nil
			n.ReportedUsage.API[model] = nil
			continue
		}
		tokens := metrics.Tokens
		n.ReportedUsage.Models[model] = tokens
		n.ReportedUsage.API[model] = metrics.API
		accounted, err := geminiUsageComplete(metrics)
		if err != nil {
			return fmt.Errorf("gemini native usage for %q: %w", model, err)
		}
		if !accounted {
			complete = false
			continue
		}
		input += *tokens.Prompt + *tokens.Tool
		output += *tokens.Candidates + *tokens.Thoughts
		thoughts += *tokens.Thoughts
		if tokens.Cached == nil {
			cacheComplete = false
		} else {
			cache += *tokens.Cached
		}
	}
	for _, count := range []*int64{&input, &output, &cache, &thoughts} {
		if err := validGeminiCount(count); err != nil {
			return fmt.Errorf("gemini aggregate usage: %w", err)
		}
	}
	if complete {
		r.Usage = &Usage{InputTokens: input, OutputTokens: output, ReasoningOutputTokens: &thoughts}
		if cacheComplete {
			r.Usage.CachedInputTokens = &cache
		}
		n.UsageStatus = "reported"
	}
	return nil
}

// Failed API requests have no token contribution in CLI telemetry, even after a retry succeeds.
func geminiUsageComplete(metrics *geminiModelStats) (bool, error) {
	t, api := metrics.Tokens, metrics.API
	if t != nil {
		if err := validateGeminiTokens(t); err != nil {
			return false, err
		}
	}
	if api == nil {
		return false, nil
	}
	for _, count := range []*int64{api.TotalRequests, api.TotalErrors} {
		if err := validGeminiCount(count); err != nil {
			return false, err
		}
	}
	if api.TotalRequests != nil && api.TotalErrors != nil && *api.TotalErrors > *api.TotalRequests {
		return false, fmt.Errorf("API errors exceed requests")
	}
	if api.TotalRequests != nil && *api.TotalRequests == 0 && t != nil {
		for _, count := range []*int64{t.Prompt, t.Candidates, t.Total, t.Thoughts, t.Tool, t.Cached, t.Input} {
			if count != nil && *count > 0 {
				return false, fmt.Errorf("positive tokens without an API request")
			}
		}
	}
	if api.TotalRequests == nil || api.TotalErrors == nil || *api.TotalRequests == 0 || *api.TotalErrors > 0 {
		return false, nil
	}
	return t != nil && t.Prompt != nil && t.Candidates != nil && t.Thoughts != nil && t.Total != nil && t.Tool != nil, nil
}

func consumeGeminiTools(n *GeminiEvidence, stats *geminiStats) error {
	if stats.Tools != nil && stats.Tools.ByName != nil {
		n.ObservedItemTypes = []string{}
		n.CompletedItemTypes = []string{}
		names := make([]string, 0, len(stats.Tools.ByName))
		for name := range stats.Tools.ByName {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			calls := stats.Tools.ByName[name]
			for _, count := range []*int64{calls.Count, calls.Success, calls.Fail} {
				if err := validGeminiCount(count); err != nil {
					return fmt.Errorf("gemini tool stats: %w", err)
				}
			}
			if strings.TrimSpace(name) == "" {
				return fmt.Errorf("gemini tool name is empty")
			}
			if calls.Count != nil && *calls.Count > 0 {
				n.ObservedItemTypes = append(n.ObservedItemTypes, name)
			}
			if calls.Success != nil && calls.Fail != nil && calls.Count != nil && *calls.Success+*calls.Fail != *calls.Count {
				return fmt.Errorf("gemini tool outcomes disagree with count")
			}
			if (calls.Success != nil && *calls.Success > 0) || (calls.Fail != nil && *calls.Fail > 0) {
				n.CompletedItemTypes = append(n.CompletedItemTypes, name)
			}
		}
		sort.Strings(n.ObservedItemTypes)
		sort.Strings(n.CompletedItemTypes)
	}
	return nil
}

func validGeminiCount(count *int64) error {
	if count != nil && (*count < 0 || *count > 1e12) {
		return fmt.Errorf("invalid native token or call count")
	}
	return nil
}

func validateGeminiTokens(t *GeminiTokens) error {
	for _, count := range []*int64{t.Prompt, t.Candidates, t.Total, t.Cached, t.Thoughts, t.Tool, t.Input} {
		if err := validGeminiCount(count); err != nil {
			return err
		}
	}
	if t.Prompt != nil && t.Cached != nil && *t.Cached > *t.Prompt {
		return fmt.Errorf("cached input exceeds prompt")
	}
	if t.Prompt != nil && t.Cached != nil && t.Input != nil && *t.Input != *t.Prompt-*t.Cached {
		return fmt.Errorf("uncached input disagrees with prompt minus cache")
	}
	// Gemini defines total as prompt + tool-use prompt + candidates + thoughts.
	// Never infer an absent count by subtracting the other fields from total.
	if t.Prompt != nil && t.Candidates != nil && t.Thoughts != nil && t.Total != nil && t.Tool != nil && *t.Total != *t.Prompt+*t.Tool+*t.Candidates+*t.Thoughts {
		return fmt.Errorf("total disagrees with prompt plus tool plus candidates plus thoughts")
	}
	return nil
}

// uniqueJSON refuses duplicate object keys, trailing values and excessive nesting.
// Decoder's normal duplicate-key overwrite would hide contradictory usage or outcomes.
func uniqueJSON(data []byte) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	if err := uniqueJSONValue(d, 0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return fmt.Errorf("expected one JSON value")
	}
	return nil
}

func uniqueJSONValue(d *json.Decoder, depth int) error {
	if depth > 64 {
		return fmt.Errorf("JSON nesting exceeds 64")
	}
	token, err := d.Token()
	if err != nil {
		return err
	}
	open, compound := token.(json.Delim)
	if !compound {
		return nil
	}
	seen := map[string]bool{}
	for d.More() {
		if open == '{' {
			key, err := d.Token()
			if err != nil {
				return err
			}
			name, ok := key.(string)
			if !ok || seen[name] {
				return fmt.Errorf("duplicate or invalid object key")
			}
			seen[name] = true
		}
		if err := uniqueJSONValue(d, depth+1); err != nil {
			return err
		}
	}
	_, err = d.Token()
	return err
}
