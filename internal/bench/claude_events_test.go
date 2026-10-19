package bench

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

const claudeGoodModel = `{"inputTokens":100,"outputTokens":40,"cacheCreationInputTokens":200,"cacheReadInputTokens":300,"thinkingTokens":10,"costUSD":0.002}`

func claudeTestOutput(models string) string {
	return `{"type":"result","subtype":"success","is_error":false,"session_id":"test-session","result":"{\"source\":\"package example\\n\"}","usage":{"input_tokens":100,"output_tokens":40,"cache_creation_input_tokens":200,"cache_read_input_tokens":300},"modelUsage":` + models + `,"num_turns":1,"total_cost_usd":0.002,"stop_reason":"end_turn","terminal_reason":"completed"}`
}

func claudeGoodOutput() string {
	return claudeTestOutput(`{"claude-haiku-4-5-20251001":` + claudeGoodModel + `}`)
}

// The oracle is the native accounting definition: 100 uncached + 200 cache
// creation + 300 cache read = 600 input; output 40 already includes thinking 10.
// Adding main-loop usage or thinking twice must fail this assertion.
func TestClaudeOutputAccounting(t *testing.T) {
	r, err := parseClaudeOutput(claudeGoodOutput())
	if err != nil || r.Source != "package example\n" || r.Usage == nil {
		t.Fatalf("success lost: %+v %v", r, err)
	}
	if r.Usage.InputTokens != 600 || r.Usage.OutputTokens != 40 || *r.Usage.CachedInputTokens != 300 || *r.Usage.ReasoningOutputTokens != 10 {
		t.Fatalf("native accounting double-counted or dropped tokens: %+v", r.Usage)
	}
	n := r.Native
	if n.Terminal != "completed" || n.UsageStatus != "reported" || n.SessionID != "test-session" || n.EventCount != 1 || !reflect.DeepEqual(n.ReportedModels, []string{"claude-haiku-4-5-20251001"}) {
		t.Fatalf("lost native envelope: %+v", n)
	}
	if n.ResolvedModel != nil || n.ResolvedReasoning != nil || n.ObservedItemTypes != nil || n.CompletedItemTypes != nil {
		t.Fatal("invented backend identity, reasoning or tool observations")
	}
	if n.ClaudeReportedUsage == nil || *n.ClaudeReportedUsage.TotalCostUSD != 0.002 || *n.ClaudeReportedUsage.Usage.Input != 100 || *n.ClaudeReportedUsage.Models["claude-haiku-4-5-20251001"].CacheCreation != 200 {
		t.Fatalf("lost raw accounting: %+v", n.ClaudeReportedUsage)
	}
}

func TestClaudeOutputWholeCallAndOptionalUsage(t *testing.T) {
	helper := `{"inputTokens":50,"outputTokens":10,"cacheCreationInputTokens":25,"cacheReadInputTokens":25}`
	r, err := parseClaudeOutput(claudeTestOutput(`{"z-main":` + claudeGoodModel + `,"a-helper":` + helper + `}`))
	if err != nil || r.Usage == nil || r.Usage.InputTokens != 700 || r.Usage.OutputTokens != 50 || *r.Usage.CachedInputTokens != 325 || r.Usage.ReasoningOutputTokens != nil {
		t.Fatalf("whole-call helpers were omitted or missing thinking invented: %+v %v", r, err)
	}
	if !reflect.DeepEqual(r.Native.ReportedModels, []string{"a-helper", "z-main"}) {
		t.Fatalf("lost deterministic model identity: %+v", r.Native.ReportedModels)
	}
	for _, field := range []string{"usage", "total_cost_usd"} {
		var obj map[string]any
		if err := json.Unmarshal([]byte(claudeGoodOutput()), &obj); err != nil {
			t.Fatal(err)
		}
		delete(obj, field)
		data, _ := json.Marshal(obj)
		r, err := parseClaudeOutput(string(data))
		if err != nil || r.Usage == nil || r.Usage.InputTokens != 600 {
			t.Fatalf("optional %s discarded whole-call counters: %+v %v", field, r, err)
		}
	}
	// Optional metadata is retained when consistent; unknown additions are harmless.
	withMetadata := strings.Replace(claudeGoodOutput(), `"costUSD":0.002`, `"costUSD":0.002,"canonicalModel":"claude-haiku-4-5-20251001","provider":"firstParty","costBasis":"list","future_metric":12`, 1)
	r, err = parseClaudeOutput(withMetadata)
	if err != nil || !reflect.DeepEqual(r.Native.ReportedModels, []string{"claude-haiku-4-5-20251001"}) || *r.Native.ClaudeReportedUsage.Models["claude-haiku-4-5-20251001"].Provider != "firstParty" {
		t.Fatalf("lost native model metadata: %+v %v", r, err)
	}
}

// Exhaustively generate all presence combinations of four required whole-call
// counters. Zero is present; absent and null are both unknown, never inferred.
func TestClaudeUsagePresenceMatrix(t *testing.T) {
	fields := []string{"inputTokens", "outputTokens", "cacheCreationInputTokens", "cacheReadInputTokens"}
	checked := 0
	for _, absent := range []string{"missing", "null"} {
		for mask := 0; mask < 16; mask++ {
			t.Run(fmt.Sprintf("%s/%04b", absent, mask), func(t *testing.T) {
				model := map[string]any{}
				for i, name := range fields {
					if mask&(1<<i) != 0 {
						model[name] = 0
					} else if absent == "null" {
						model[name] = nil
					}
				}
				data, _ := json.Marshal(map[string]any{"model": model})
				text := claudeTestOutput(string(data))
				text = strings.Replace(text, `"usage":{"input_tokens":100,"output_tokens":40,"cache_creation_input_tokens":200,"cache_read_input_tokens":300}`, `"usage":null`, 1)
				r, err := parseClaudeOutput(text)
				if err != nil || (r.Usage != nil) != (mask == 15) {
					t.Fatalf("presence matrix invented or lost known zero counts: mask=%04b response=%+v error=%v", mask, r, err)
				}
				if mask != 15 && r.Native.UsageStatus != "partial" {
					t.Fatalf("missing native count incorrectly marked %s", r.Native.UsageStatus)
				}
				checked++
			})
		}
	}
	if checked != 32 {
		t.Fatalf("presence assertions did not execute: %d", checked)
	}
	for _, models := range []string{"null", "{}", `{"model":null}`, `{"model":{}}`} {
		r, err := parseClaudeOutput(claudeTestOutput(models))
		if err != nil || r.Usage != nil || r.Native.UsageStatus != "partial" {
			t.Fatalf("main-loop counts substituted for unknown whole-call usage: %+v %v", r, err)
		}
	}
	// A complete model cannot make an incomplete sibling accountable.
	for _, field := range fields {
		var partial map[string]any
		if err := json.Unmarshal([]byte(claudeGoodModel), &partial); err != nil {
			t.Fatal(err)
		}
		delete(partial, field)
		data, _ := json.Marshal(partial)
		r, err := parseClaudeOutput(claudeTestOutput(`{"complete":` + claudeGoodModel + `,"partial":` + string(data) + `}`))
		if err != nil || r.Usage != nil || r.Native.UsageStatus != "partial" {
			t.Fatalf("missing %s in one model hidden by sibling: %+v %v", field, r, err)
		}
	}
}

func TestClaudeOutputFailuresRetainEvidence(t *testing.T) {
	for _, subtype := range []string{"error_max_turns", "error_max_budget_usd", "error_max_structured_output_retries", "error_during_execution", "future_error", "success"} {
		t.Run(subtype, func(t *testing.T) {
			text := strings.Replace(claudeGoodOutput(), `"subtype":"success"`, `"subtype":"`+subtype+`"`, 1)
			text = strings.Replace(text, `"is_error":false`, `"is_error":true`, 1)
			r, err := parseClaudeOutput(text)
			wantUsage := subtype == "error_max_turns" || subtype == "error_max_budget_usd" || subtype == "error_max_structured_output_retries"
			if err == nil || r.Source != "" || r.Native.Terminal != "failed" || r.Native.ClaudeReportedUsage == nil || (r.Usage != nil) != wantUsage {
				t.Fatalf("failure lost accounting or supplied quality source: %+v %v", r, err)
			}
		})
	}
	for name, replace := range map[string]string{
		"errors": `"errors":["failed"]`, "error": `"error":{"message":"failed"}`,
		"api error": `"api_error":"overloaded"`, "api status": `"api_error_status":529`,
	} {
		t.Run(name, func(t *testing.T) {
			text := strings.Replace(claudeGoodOutput(), `"type":`, replace+`,"type":`, 1)
			r, err := parseClaudeOutput(text)
			if err == nil || r.Source != "" || r.Usage != nil || r.Native.ClaudeReportedUsage == nil {
				t.Fatalf("API failure incorrectly graded or claimed complete usage: %+v %v", r, err)
			}
		})
	}
	text := strings.Replace(claudeGoodOutput(), `"subtype":"success"`, `"subtype":"error_max_turns","api_error_status":529`, 1)
	r, err := parseClaudeOutput(text)
	if err == nil || r.Usage != nil || r.Source != "" || r.Native.UsageStatus != "partial" {
		t.Fatalf("turn limit hid unaccounted API failure: %+v %v", r, err)
	}
}

func TestClaudeOutputRejectsMalformedAccounting(t *testing.T) {
	good := claudeGoodOutput()
	for name, text := range map[string]string{
		"duplicate envelope":      good + good,
		"duplicate key":           strings.Replace(good, `"type":`, `"type":"result","type":`, 1),
		"duplicate count":         strings.Replace(good, `"inputTokens":100`, `"inputTokens":100,"inputTokens":101`, 1),
		"negative":                strings.Replace(good, `"inputTokens":100`, `"inputTokens":-1`, 1),
		"fractional":              strings.Replace(good, `"inputTokens":100`, `"inputTokens":0.5`, 1),
		"huge":                    strings.Replace(good, `"inputTokens":100`, `"inputTokens":1000000000001`, 1),
		"overflow":                strings.Replace(good, `"inputTokens":100`, `"inputTokens":9223372036854775808`, 1),
		"string":                  strings.Replace(good, `"inputTokens":100`, `"inputTokens":"100"`, 1),
		"bad main usage":          strings.Replace(good, `"input_tokens":100`, `"input_tokens":-1`, 1),
		"main exceeds all":        strings.Replace(good, `"input_tokens":100`, `"input_tokens":101`, 1),
		"thinking exceeds output": strings.Replace(good, `"thinkingTokens":10`, `"thinkingTokens":41`, 1),
		"negative cost":           strings.Replace(good, `"total_cost_usd":0.002`, `"total_cost_usd":-1`, 1),
		"infinite cost":           strings.Replace(good, `"costUSD":0.002`, `"costUSD":1e309`, 1),
		"empty model":             strings.Replace(good, `"claude-haiku-4-5-20251001":`, `" ":`, 1),
		"canonical mismatch":      strings.Replace(good, `"costUSD":0.002`, `"costUSD":0.002,"canonicalModel":"claude-opus"`, 1),
		"provider mismatch":       strings.Replace(good, `"costUSD":0.002`, `"costUSD":0.002,"provider":"bedrock"`, 1),
		"empty canonical model":   strings.Replace(good, `"costUSD":0.002`, `"costUSD":0.002,"canonicalModel":""`, 1),
		"empty provider":          strings.Replace(good, `"costUSD":0.002`, `"costUSD":0.002,"provider":""`, 1),
		"malformed":               "{",
	} {
		t.Run(name, func(t *testing.T) {
			r, err := parseClaudeOutput(text)
			if err == nil || r.Usage != nil || r.Source != "" || r.Native.UsageStatus != "invalid" {
				t.Fatalf("invalid accounting accepted: %+v %v", r, err)
			}
		})
	}
	// Per-field bounds and aggregate bounds are separate: each bucket fits but
	// 1e12 uncached plus even one cached token exceeds the whole-call limit.
	model := `{"inputTokens":1000000000000,"outputTokens":0,"cacheCreationInputTokens":0,"cacheReadInputTokens":1}`
	r, err := parseClaudeOutput(claudeTestOutput(`{"model":` + model + `}`))
	if err == nil || r.Usage != nil || r.Native.UsageStatus != "invalid" {
		t.Fatalf("aggregate count boundary not enforced: %+v %v", r, err)
	}
	model = `{"inputTokens":1000000000000,"outputTokens":0,"cacheCreationInputTokens":0,"cacheReadInputTokens":0}`
	text := claudeTestOutput(`{"model":` + model + `}`)
	text = strings.Replace(text, `"usage":{"input_tokens":100,"output_tokens":40,"cache_creation_input_tokens":200,"cache_read_input_tokens":300}`, `"usage":null`, 1)
	r, err = parseClaudeOutput(text)
	if err != nil || r.Usage == nil || r.Usage.InputTokens != 1000000000000 {
		t.Fatalf("exact aggregate boundary rejected: %+v %v", r, err)
	}
}

func TestClaudeOutputRejectsIncompleteSource(t *testing.T) {
	for _, final := range []string{`{}`, `null`, `{"source":""}`, `{"source":"x","source":"y"}`, `{"source":"x","extra":true}`, "```json\n{}\n```"} {
		encoded, _ := json.Marshal(final)
		text := strings.Replace(claudeGoodOutput(), `"{\"source\":\"package example\\n\"}"`, string(encoded), 1)
		r, err := parseClaudeOutput(text)
		if err == nil || r.Source != "" || r.Usage == nil {
			t.Fatalf("malformed final source lost usage or was graded: %+v %v", r, err)
		}
	}
	for name, text := range map[string]string{
		"missing is_error": strings.Replace(claudeGoodOutput(), `"is_error":false,`, "", 1),
		"missing session":  strings.Replace(claudeGoodOutput(), `"session_id":"test-session"`, `"session_id":""`, 1),
		"not result":       strings.Replace(claudeGoodOutput(), `"type":"result"`, `"type":"assistant"`, 1),
		"truncated":        strings.Replace(claudeGoodOutput(), `"stop_reason":"end_turn"`, `"stop_reason":"max_tokens"`, 1),
		"interrupted":      strings.Replace(claudeGoodOutput(), `"terminal_reason":"completed"`, `"terminal_reason":"aborted_streaming"`, 1),
		"tool deferred":    strings.Replace(claudeGoodOutput(), `"type":`, `"deferred_tool_use":{"name":"Read"},"type":`, 1),
		"no result":        strings.Replace(claudeGoodOutput(), `"result":"{\"source\":\"package example\\n\"}"`, `"structured_output":{"source":"package ignored"}`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			r, err := parseClaudeOutput(text)
			if err == nil || r.Source != "" || r.Usage == nil {
				t.Fatalf("incomplete envelope was graded or lost usage: %+v %v", r, err)
			}
		})
	}
	for _, text := range []string{"", " ", "null", "[]", strings.Repeat(" ", maxAdapterBytes+1)} {
		if r, err := parseClaudeOutput(text); err == nil || r.Source != "" || r.Usage != nil {
			t.Fatalf("empty/oversized envelope accepted: %+v %v", r, err)
		}
	}
}
