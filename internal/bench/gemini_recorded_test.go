package bench

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Versioned Gemini best-effort accounting
func TestGeminiStableRecordedUsage(t *testing.T) {
	for _, tc := range []struct {
		file                           string
		input, output, cache, thoughts int64
	}{
		{"success.json", 100, 25, 80, 15}, {"tools.json", 307, 70, 130, 40},
	} {
		t.Run(tc.file, func(t *testing.T) {
			data, err := os.ReadFile("testdata/gemini064/" + tc.file)
			if err != nil {
				t.Fatal(err)
			}
			cfg := Experiment{Gemini: &GeminiConfig{CLIVersion: "0.64.0"}}
			response, err := parseNativeOutput(cfg, string(data))
			if err != nil || response.Usage == nil {
				t.Fatalf("recorded usage refused: %v", err)
			}
			u := response.Usage
			if u.InputTokens != tc.input || u.OutputTokens != tc.output || u.CachedInputTokens == nil || *u.CachedInputTokens != tc.cache || u.ReasoningOutputTokens == nil || *u.ReasoningOutputTokens != tc.thoughts {
				t.Fatalf("recorded usage double counted or lost fields: %+v", u)
			}
		})
	}
}

// The installed CLI turns omitted upstream usage into zero counters. A nonempty
// completion cannot establish measured zero from that ambiguous native record.
// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Versioned Gemini best-effort accounting
func TestGeminiStableMissingUsageIsNotZero(t *testing.T) {
	data, err := os.ReadFile("testdata/gemini064/missing-usage.json")
	if err != nil {
		t.Fatal(err)
	}
	cfg := Experiment{Gemini: &GeminiConfig{CLIVersion: "0.64.0"}}
	r, err := parseNativeOutput(cfg, string(data))
	if err != nil || r.Source == "" || r.Native.ReportedUsage == nil {
		t.Fatalf("unknown usage lost quality/raw evidence: %v", err)
	}
	if r.Usage != nil || r.Native.UsageStatus != "partial" {
		t.Fatal("ambiguous native zero became reported usage")
	}
	cfg.Gemini.CLIVersion = ""
	old, err := parseNativeOutput(cfg, string(data))
	if err != nil || old.Usage == nil || old.Usage.InputTokens != 0 || old.Usage.OutputTokens != 0 {
		t.Fatal("historical parser interpretation changed")
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Versioned Gemini best-effort accounting
func TestGeminiStableRecordedFatalOutput(t *testing.T) {
	stderr, err := os.ReadFile("testdata/gemini064/turn-limit.stderr.txt")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(stderr), "FatalTurnLimitedError") || strings.Contains(string(stderr), `"stats"`) {
		t.Fatal("recorded fatal fixture does not show missing accounting")
	}
	r, err := parseNativeOutput(Experiment{Gemini: &GeminiConfig{CLIVersion: "0.64.0"}}, "")
	if err == nil || r.Usage != nil || r.Source != "" {
		t.Fatal("empty fatal stdout fabricated usage or source")
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Versioned Gemini best-effort accounting
func TestGeminiStableZeroFailureUsage(t *testing.T) {
	data, err := os.ReadFile("testdata/gemini064/missing-usage.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []string{"malformed-final", "empty-source", "terminal-error"} {
		t.Run(scenario, func(t *testing.T) {
			var envelope map[string]any
			if err := json.Unmarshal(data, &envelope); err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "malformed-final":
				envelope["response"] = "{incomplete"
			case "empty-source":
				envelope["response"] = `{"source":""}`
			case "terminal-error":
				delete(envelope, "response")
				envelope["error"] = map[string]string{"type": "NativeFailure", "message": "fixture"}
			}
			raw, err := json.Marshal(envelope)
			if err != nil {
				t.Fatal(err)
			}
			r, err := parseNativeOutput(Experiment{Gemini: &GeminiConfig{CLIVersion: "0.64.0"}}, string(raw))
			if err == nil || r.Native.ReportedUsage == nil {
				t.Fatal("failure or raw evidence was lost")
			}
			if r.Usage != nil || r.Native.UsageStatus != "partial" {
				t.Fatal("zero failure snapshot became known usage")
			}
			old, oldErr := parseNativeOutput(Experiment{Gemini: &GeminiConfig{}}, string(raw))
			if oldErr == nil || old.Usage == nil {
				t.Fatal("historical failed-zero interpretation changed")
			}
		})
	}
}
