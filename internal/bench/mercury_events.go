package bench

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"
)

// mercuryOutput retains the unmodified native result and provider capture together.
type mercuryOutput struct {
	Claude          string `json:"claude"`
	Capture         string `json:"capture"`
	Error           string `json:"error,omitempty"`
	OutputTruncated bool   `json:"output_truncated,omitempty"`
}

type mercuryRecord struct {
	Type         string `json:"type"`
	ID           int    `json:"id"`
	Model        string `json:"model,omitempty"`
	Effort       string `json:"reasoning_effort,omitempty"`
	MaxTokens    int    `json:"max_tokens,omitempty"`
	Stream       bool   `json:"stream,omitempty"`
	IncludeUsage bool   `json:"include_usage,omitempty"`
	Retries      *int   `json:"retry_count,omitempty"`
	Status       int    `json:"status,omitempty"`
	EOF          bool   `json:"eof,omitempty"`
	Body         string `json:"body,omitempty"`
	Error        string `json:"error,omitempty"`
}

func parseMercuryOutput(stdout string) (r AdapterResponse, err error) {
	r = AdapterResponse{Version: 1, Native: &NativeEvidence{Terminal: "failed", UsageStatus: "invalid"}}
	var envelope mercuryOutput
	if len(stdout) > 4*maxAdapterBytes {
		return r, fmt.Errorf("mercury output exceeds capture envelope limit")
	}
	if err := uniqueJSON([]byte(stdout)); err != nil {
		return r, fmt.Errorf("mercury envelope: %w", err)
	}
	if err := strictJSON([]byte(stdout), &envelope); err != nil {
		return r, fmt.Errorf("mercury envelope: %w", err)
	}
	defer func() {
		if envelope.OutputTruncated {
			markGeminiTruncated(&r)
			err = fmt.Errorf("claude output exceeded 1 MiB per stream")
		}
	}()
	// LiteLLM can synthesize usage and omit provider reasoning details. Never
	// expose its translated accounting or pricing as authoritative Mercury usage.
	if err := uniqueJSON([]byte(envelope.Claude)); err != nil {
		return r, fmt.Errorf("mercury native JSON: %w", err)
	}
	var native map[string]json.RawMessage
	if err := json.Unmarshal([]byte(envelope.Claude), &native); err != nil || native == nil {
		return r, fmt.Errorf("mercury requires a native result object")
	}
	for key := range native {
		if strings.EqualFold(key, "usage") || strings.EqualFold(key, "modelUsage") || strings.EqualFold(key, "total_cost_usd") {
			delete(native, key)
		}
	}
	clean, err := json.Marshal(native)
	if err != nil {
		return r, err
	}
	r, err = parseClaudeStructuredOutput(string(clean))
	if err != nil {
		return r, err
	}
	r.Native.Terminal, r.Native.UsageStatus = "failed", "invalid"
	if envelope.Error != "" {
		return r, fmt.Errorf("mercury runner: %s", envelope.Error)
	}
	usage, count, err := parseMercuryCapture(envelope.Capture)
	if err != nil {
		return r, err
	}
	r.Usage = usage
	r.Native.Terminal, r.Native.UsageStatus = "completed", "reported"
	r.Native.ReportedModels = []string{"mercury-3"}
	r.Native.EventCount = count
	return r, nil
}

func parseMercuryCapture(capture string) (*Usage, int, error) {
	if len(capture) > maxAdapterBytes || !strings.HasSuffix(capture, "\n") {
		return nil, 0, fmt.Errorf("mercury capture missing, oversized or truncated")
	}
	intents, responses := map[int]bool{}, map[int]bool{}
	total := &Usage{}
	cache, reasoning := int64(0), int64(0)
	cacheKnown, reasoningKnown, bodyBytes := true, true, 0
	lines := strings.Split(strings.TrimSuffix(capture, "\n"), "\n")
	if len(lines) > 8 {
		return nil, 0, fmt.Errorf("mercury capture exceeds four exchanges")
	}
	for _, line := range lines {
		var record mercuryRecord
		if err := uniqueJSON([]byte(line)); err != nil {
			return nil, 0, fmt.Errorf("mercury capture JSON: %w", err)
		}
		if err := strictJSON([]byte(line), &record); err != nil {
			return nil, 0, fmt.Errorf("mercury capture record: %w", err)
		}
		if record.ID < 1 || record.ID > 4 || record.Error != "" {
			return nil, 0, fmt.Errorf("mercury capture failed or has invalid exchange ID")
		}
		switch record.Type {
		case "intent":
			if intents[record.ID] || record.Model != "mercury-3" || record.Effort != "low" || record.MaxTokens != 8192 || !record.Stream || !record.IncludeUsage || record.Retries == nil || *record.Retries != 0 {
				return nil, 0, fmt.Errorf("mercury request controls or inventory mismatch")
			}
			intents[record.ID] = true
		case "response":
			if !intents[record.ID] || responses[record.ID] || record.Status != 200 || !record.EOF {
				return nil, 0, fmt.Errorf("mercury response incomplete or uncorrelated")
			}
			bodyBytes += len(record.Body)
			if bodyBytes > maxAdapterBytes {
				return nil, 0, fmt.Errorf("mercury provider bodies exceed 1 MiB")
			}
			u, err := parseMercurySSE(record.Body)
			if err != nil {
				return nil, 0, err
			}
			responses[record.ID] = true
			total.InputTokens += u.InputTokens
			total.OutputTokens += u.OutputTokens
			if u.CachedInputTokens == nil {
				cacheKnown = false
			} else {
				cache += *u.CachedInputTokens
			}
			if u.ReasoningOutputTokens == nil {
				reasoningKnown = false
			} else {
				reasoning += *u.ReasoningOutputTokens
			}
		default:
			return nil, 0, fmt.Errorf("mercury capture contains failure or unknown record")
		}
	}
	if len(intents) == 0 || len(intents) != len(responses) {
		return nil, 0, fmt.Errorf("mercury capture has unaccounted requests")
	}
	if cacheKnown {
		total.CachedInputTokens = &cache
	}
	if reasoningKnown {
		total.ReasoningOutputTokens = &reasoning
	}
	if err := validateUsage(total); err != nil {
		return nil, 0, err
	}
	return total, len(responses), nil
}

func parseMercurySSE(body string) (*Usage, error) {
	if !utf8.ValidString(body) {
		return nil, fmt.Errorf("mercury response is not UTF-8")
	}
	body = strings.ReplaceAll(body, "\r\n", "\n")
	if strings.Contains(body, "\r") || !strings.HasSuffix(body, "\n\n") {
		return nil, fmt.Errorf("mercury SSE has incomplete framing")
	}
	var usage *Usage
	done, modelSeen := false, false
	for _, event := range strings.Split(strings.TrimSuffix(body, "\n\n"), "\n\n") {
		var data []string
		for _, line := range strings.Split(event, "\n") {
			if strings.HasPrefix(line, "data:") {
				data = append(data, strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
			} else if line != "" && !strings.HasPrefix(line, ":") {
				return nil, fmt.Errorf("mercury SSE unsupported field")
			}
		}
		if len(data) == 0 {
			continue
		}
		if done {
			return nil, fmt.Errorf("mercury SSE data after DONE")
		}
		payload := strings.Join(data, "\n")
		if payload == "[DONE]" {
			done = true
			continue
		}
		if err := uniqueJSON([]byte(payload)); err != nil {
			return nil, fmt.Errorf("mercury SSE JSON: %w", err)
		}
		var chunk struct {
			Model string          `json:"model"`
			Error json.RawMessage `json:"error"`
			Usage *struct {
				Prompt        *int64 `json:"prompt_tokens"`
				Completion    *int64 `json:"completion_tokens"`
				Total         *int64 `json:"total_tokens"`
				PromptDetails *struct {
					Cached *int64 `json:"cached_tokens"`
				} `json:"prompt_tokens_details"`
				CompletionDetails *struct {
					Reasoning *int64 `json:"reasoning_tokens"`
				} `json:"completion_tokens_details"`
			} `json:"usage"`
		}
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil || claudePresent(chunk.Error) {
			return nil, fmt.Errorf("mercury invalid provider chunk")
		}
		if chunk.Model != "" {
			if chunk.Model != "mercury-3" {
				return nil, fmt.Errorf("mercury provider model mismatch")
			}
			modelSeen = true
		}
		if u := chunk.Usage; u != nil {
			if usage != nil || u.Prompt == nil || u.Completion == nil {
				return nil, fmt.Errorf("mercury primary usage missing or repeated")
			}
			for _, count := range []*int64{u.Prompt, u.Completion, u.Total} {
				if err := validClaudeCount(count); err != nil {
					return nil, err
				}
			}
			if u.Total != nil && *u.Total != *u.Prompt+*u.Completion {
				return nil, fmt.Errorf("mercury total usage contradicts primary counts")
			}
			usage = &Usage{InputTokens: *u.Prompt, OutputTokens: *u.Completion}
			if u.PromptDetails != nil {
				usage.CachedInputTokens = u.PromptDetails.Cached
			}
			if u.CompletionDetails != nil {
				usage.ReasoningOutputTokens = u.CompletionDetails.Reasoning
			}
			if err := validateUsage(usage); err != nil {
				return nil, err
			}
		}
	}
	if !done || !modelSeen || usage == nil {
		return nil, fmt.Errorf("mercury provider usage, model or DONE missing")
	}
	return usage, nil
}
