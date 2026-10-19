package bench

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Execution failure is the primary cause even when the aborted process supplied
// no identity or usage. Relabeling it model-unverified hides the failure.
// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Bounded native pilots
func TestNativeFailureStopPrecedesAccounting(t *testing.T) {
	cfg := nativeTestExperiment(t, `null`)
	for _, tc := range []struct{ status, want string }{
		{"timeout", "attempt-timeout"},
		{"cancelled", "cancelled"},
		{"adapter-error", "adapter-error"},
	} {
		t.Run(tc.status, func(t *testing.T) {
			for _, observed := range []bool{false, true} {
				native := &NativeEvidence{}
				if observed {
					native.ReportedModels = []string{cfg.Model}
				}
				a := Attempt{Status: tc.status, Diagnostic: "native execution failed", Response: &AdapterResponse{Native: native}}
				used := int64(0)
				reason, err := nativeStop(context.Background(), cfg, a, &used, false)
				if reason != tc.want || err == nil {
					t.Fatalf("execution failure mislabeled: status=%s observed=%t reason=%s err=%v", tc.status, observed, reason, err)
				}
			}
		})
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Bounded native pilots
// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Retained independently graded artifacts
func TestNativeEarlyFailureStopAndLegacyReplay(t *testing.T) {
	cfg := nativeTestExperiment(t, `null`)
	script := "#!/bin/sh\nif [ \"$1\" = --version ]; then echo '0.43.0'; exit 0; fi\ncat >/dev/null\necho provider-unavailable >&2\nexit 1\n"
	if err := os.WriteFile(cfg.Command[0], []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	resolved, err := ResolveExperiment(cfg, ".")
	if err != nil {
		t.Fatal(err)
	}
	path, err := RunExperiment(context.Background(), resolved, t.TempDir())
	if err == nil {
		t.Fatal("provider failure did not stop run")
	}
	run, err := LoadExperimentRun(path)
	if err != nil {
		t.Fatal(err)
	}
	if run.StopReason != "adapter-error" || len(run.Attempts) != 1 || run.Attempts[0].Status != "adapter-error" {
		t.Fatalf("early failure mislabeled: reason=%s attempts=%d", run.StopReason, len(run.Attempts))
	}
	for _, tc := range []struct{ reason, status string }{
		{"adapter-error", "adapter-error"},
		{"attempt-timeout", "timeout"},
		{"cancelled", "cancelled"},
		{"model-unverified", "adapter-error"},
	} {
		run.StopReason = tc.reason
		run.Attempts[0].Status = tc.status
		run.Digest = runDigest(run)
		artifact := filepath.Join(t.TempDir(), "run.json")
		if err := writeExclusiveJSON(artifact, run); err != nil {
			t.Fatal(err)
		}
		calls := 0
		replay, err := rescoreExperiment(context.Background(), artifact, func(context.Context, *Attempt) { calls++ })
		if err != nil {
			t.Fatal(err)
		}
		replayed, err := LoadExperimentRun(replay)
		if err != nil || replayed.StopReason != tc.reason || replayed.Attempts[0].Status != tc.status || calls != 0 {
			t.Fatalf("legacy/new failure replay changed history: reason=%s calls=%d err=%v", replayed.StopReason, calls, err)
		}
	}
}

// An init side effect is the independent oracle: once per candidate evaluation.
// Clearing Grade afterward cannot hide executing the reused session's source.
// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Controlled bounded experiments
func TestNativeReusedSessionNeverExecutesCandidate(t *testing.T) {
	cfg := nativeTestExperiment(t, `{"input_tokens":10,"output_tokens":5}`)
	marker := filepath.Join(t.TempDir(), "evaluations")
	var source string
	for _, task := range Tasks() {
		if task.ID == "safe-path" {
			source = task.GoodSource
		}
	}
	if source == "" {
		t.Fatal("safe-path fixture missing")
	}
	source = strings.Replace(source, "import (", "import (\"os\";", 1)
	source += fmt.Sprintf("\nfunc init() { f, err := os.OpenFile(%q, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600); if err != nil { panic(err) }; defer f.Close(); if _, err := f.WriteString(\"evaluated\\n\"); err != nil { panic(err) } }\n", marker)
	final, _ := json.Marshal(map[string]string{"source": source})
	output, _ := json.Marshal(map[string]any{
		"session_id": "reused-session", "response": string(final),
		"stats": map[string]any{"models": map[string]any{cfg.Model: map[string]any{
			"tokens": map[string]int{"prompt": 10, "candidates": 5, "thoughts": 0, "cached": 0, "total": 15, "tool": 0, "input": 10},
			"api":    map[string]int{"totalRequests": 1, "totalErrors": 0},
		}}},
	})
	script := "#!/bin/sh\nif [ \"$1\" = --version ]; then echo '0.43.0'; exit 0; fi\ncat >/dev/null\nprintf '%s\\n' '" + strings.ReplaceAll(string(output), "'", "'\\''") + "'\n"
	if err := os.WriteFile(cfg.Command[0], []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	resolved, err := ResolveExperiment(cfg, ".")
	if err != nil {
		t.Fatal(err)
	}
	path, err := RunExperiment(context.Background(), resolved, t.TempDir())
	if err == nil {
		t.Fatal("reused session did not stop run")
	}
	run, err := LoadExperimentRun(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(run.Attempts) != 2 || run.Attempts[0].Status != "passed" || run.StopReason != "session-reuse" {
		t.Fatalf("fixture did not reach reused session: %+v", run)
	}
	data, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "evaluated\n" {
		t.Fatalf("reused session source executed: %q", data)
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Bounded native pilots
func TestNativeFailureStopHistoryValidation(t *testing.T) {
	cfg := testExperiment()
	cfg.Gemini = &GeminiConfig{Auth: "api-key-env", Reasoning: "low"}
	cfg.MaxTotalTokens = 20
	known := func(session, status string) Attempt {
		return Attempt{Status: status, Diagnostic: "native failed", Response: &AdapterResponse{
			Native: &NativeEvidence{SessionID: session, ReportedModels: []string{cfg.Model}}, Usage: &Usage{InputTokens: 1},
		}}
	}
	missing := Attempt{Status: "adapter-error", Diagnostic: "native failed", Response: &AdapterResponse{Native: &NativeEvidence{}}}
	for _, tc := range []struct {
		name, reason string
		attempts     []Attempt
		valid        bool
	}{
		{"adapter failure", "adapter-error", []Attempt{known("a", "adapter-error")}, true},
		{"timeout", "attempt-timeout", []Attempt{known("a", "timeout")}, true},
		{"missing identity failure", "adapter-error", []Attempt{missing}, true},
		{"legacy identity reason", "model-unverified", []Attempt{missing}, true},
		{"no failure", "adapter-error", []Attempt{known("a", "passed")}, false},
		{"wrong failure", "attempt-timeout", []Attempt{known("a", "adapter-error")}, false},
		{"no attempts", "adapter-error", nil, false},
		{"continued failure", "adapter-error", []Attempt{known("a", "timeout"), known("b", "adapter-error")}, false},
		{"reuse cannot become execution failure", "adapter-error", []Attempt{known("a", "passed"), known("a", "adapter-error")}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			run := ExperimentRun{Resolved: ResolvedExperiment{Experiment: cfg, Schedule: make([]Trial, 4)}, StopReason: tc.reason, Attempts: tc.attempts}
			if err := validateNativeStop(run); (err == nil) != tc.valid {
				t.Fatalf("execution-stop history valid=%t want=%t: %v", err == nil, tc.valid, err)
			}
		})
	}
}
