package bench

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func nativeTestExperiment(t *testing.T, usage string) Experiment {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture requires Unix")
	}
	cfg := testExperiment()
	cfg.Agent, cfg.AgentVersion, cfg.Mode = "gemini", "", "aa"
	cfg.Candidate = cfg.Baseline
	cfg.Model = "gemini-3.8-flash"
	cfg.Gemini = &GeminiConfig{CLIVersion: "0.64.0", Auth: "api-key-env", Reasoning: "low", OutputFormat: "json-schema"}
	cfg.MaxTotalTokens, cfg.MaxRunSeconds = 1000, 60
	cfg.Tasks, cfg.Repetitions = []string{"safe-path"}, 1
	var source string
	for _, task := range Tasks() {
		if task.ID == "safe-path" {
			source = task.GoodSource
		}
	}
	final, _ := json.Marshal(map[string]string{"source": source})
	var counts map[string]int64
	if err := json.Unmarshal([]byte(usage), &counts); err != nil {
		t.Fatal(err)
	}
	tokens := map[string]int64{}
	if counts != nil {
		tokens = map[string]int64{"prompt": counts["input_tokens"], "candidates": counts["output_tokens"], "thoughts": 0, "cached": counts["cached_input_tokens"], "total": counts["input_tokens"] + counts["output_tokens"], "tool": 0, "input": counts["input_tokens"] - counts["cached_input_tokens"]}
	}
	output, _ := json.Marshal(map[string]any{"response": string(final), "stats": map[string]any{"models": map[string]any{cfg.Model: map[string]any{"tokens": tokens, "api": map[string]int{"totalRequests": 1, "totalErrors": 0}}}}})
	script := "#!/bin/sh\nif [ \"$1\" = --version ]; then echo '0.64.0'; exit 0; fi\ncat >/dev/null\nprintf '{\"session_id\":\"%s\",' \"$PWD\"\nprintf '%s\\n' '" + strings.ReplaceAll(string(output[1:]), "'", "'\\''") + "'\n"

	executable := filepath.Join(t.TempDir(), "gemini-fixture")
	if err := os.WriteFile(executable, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	cfg.Command = []string{executable}
	t.Setenv("GEMINI_API_KEY", "fixture-key-not-real")
	return cfg
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Bounded native pilots
// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Retained independently graded artifacts
func TestNativeBudgetStopsAndRescoresPartialHistory(t *testing.T) {
	// Independent oracle: first 15-token attempt exhausts a 10-token ceiling;
	// a missing-usage attempt also stops, never silently authorizing another call.
	for _, tc := range []struct {
		name, usage, reason string
		ceiling             int64
	}{
		{"tokens", `{"input_tokens":10,"output_tokens":5,"cached_input_tokens":3}`, "token-limit", 10},
		{"unknown", `null`, "unknown-usage", 1000},
		{"retry", `{"input_tokens":10,"output_tokens":5}`, "unknown-usage", 1000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := nativeTestExperiment(t, tc.usage)
			if tc.name == "retry" {
				script, err := os.ReadFile(cfg.Command[0])
				if err != nil {
					t.Fatal(err)
				}
				script = []byte(strings.ReplaceAll(string(script), `"totalErrors":0,"totalRequests":1`, `"totalErrors":1,"totalRequests":2`))
				if err := os.WriteFile(cfg.Command[0], script, 0700); err != nil {
					t.Fatal(err)
				}
			}
			cfg.MaxTotalTokens = tc.ceiling
			resolved, err := ResolveExperiment(cfg, ".")
			if err != nil {
				t.Fatal(err)
			}
			path, err := RunExperiment(context.Background(), resolved, t.TempDir())
			if err == nil {
				t.Fatal("budget stop must return nonzero outcome")
			}
			run, err := LoadExperimentRun(path)
			if err != nil {
				t.Fatal(err)
			}
			if len(run.Attempts) != 1 || run.StopReason != tc.reason {
				t.Fatalf("budget stop mismatch: attempts=%d reason=%s", len(run.Attempts), run.StopReason)
			}
			if run.Attempts[0].Status != "passed" {
				t.Fatalf("fixture failed: %+v", run.Attempts[0])
			}
			report, err := CompareExperiment(run)
			if err != nil {
				t.Fatal(err)
			}
			if report.Pairs != 0 {
				t.Fatal("unpaired attempt became a pair")
			}
			before, _ := os.ReadFile(path)
			rescore, err := RescoreExperiment(context.Background(), path)
			if err != nil {
				t.Fatal(err)
			}
			rescored, err := LoadExperimentRun(rescore)
			if err != nil {
				t.Fatal(err)
			}
			if rescored.StopReason != tc.reason || len(rescored.Attempts) != 1 {
				t.Fatal("rescore lost partial history")
			}
			after, _ := os.ReadFile(path)
			if string(before) != string(after) {
				t.Fatal("rescore overwrote evidence")
			}
		})
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: A/A calibration
func TestNativeIndependentSessions(t *testing.T) {
	cfg := nativeTestExperiment(t, `{"input_tokens":10,"output_tokens":5}`)
	resolved, err := ResolveExperiment(cfg, ".")
	if err != nil {
		t.Fatal(err)
	}
	path, err := RunExperiment(context.Background(), resolved, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	run, err := LoadExperimentRun(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(run.Attempts) != 2 || run.Attempts[0].Response.Native.SessionID == run.Attempts[1].Response.Native.SessionID {
		t.Fatal("A/A arms reused a native session")
	}
	if run.Attempts[0].Request.Instructions != run.Attempts[1].Request.Instructions {
		t.Fatal("A/A instructions differ")
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: A/A calibration
// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Bounded native pilots
func TestExperimentModeAndNativeBudgetValidation(t *testing.T) {
	for _, mode := range []string{"", "ab", "aa", "invalid"} {
		for _, same := range []bool{false, true} {
			cfg := testExperiment()
			cfg.Mode = mode
			if same {
				cfg.Candidate = cfg.Baseline
			}
			want := (mode == "aa" && same) || ((mode == "" || mode == "ab") && !same)
			if got := validateExperiment(cfg) == nil; got != want {
				t.Fatalf("mode=%q identical=%t valid=%t want=%t", mode, same, got, want)
			}
		}
	}
	cfg := nativeTestExperiment(t, `null`)
	for _, change := range []func(*Experiment){
		func(e *Experiment) { e.MaxTotalTokens = 0 }, func(e *Experiment) { e.MaxTotalTokens = 1000000000001 },
		func(e *Experiment) { e.MaxRunSeconds = 0 }, func(e *Experiment) { e.MaxRunSeconds = 86401 },
		func(e *Experiment) { e.Gemini = &GeminiConfig{Auth: "ambient", Reasoning: "low"} },
		func(e *Experiment) { e.Gemini = &GeminiConfig{Auth: "api-key-env", Reasoning: ""} },
		func(e *Experiment) { e.Command = append(e.Command, "exec") }, func(e *Experiment) { e.Agent = "unknown" },
		func(e *Experiment) { e.Model = "gemini-flash-latest" },
	} {
		bad := cfg
		change(&bad)
		if validateExperiment(bad) == nil {
			t.Fatalf("accepted invalid native config: %+v", bad)
		}
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Retained independently graded artifacts
func TestNativeFailureRescorePreservesOutcome(t *testing.T) {
	cfg := nativeTestExperiment(t, `{"input_tokens":10,"output_tokens":5}`)
	resolved, err := ResolveExperiment(cfg, ".")
	if err != nil {
		t.Fatal(err)
	}
	path, err := RunExperiment(context.Background(), resolved, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	run, err := LoadExperimentRun(path)
	if err != nil {
		t.Fatal(err)
	}
	// Model a process timing out after terminal JSON was emitted. Source and usage
	// remain evidence, but replaying the grader cannot turn that execution into success.
	run.Attempts[0].Status = "timeout"
	run.Attempts[0].Diagnostic = "deadline exceeded after terminal event"
	run.Attempts[0].Grade = CandidateEvaluation{}
	run.Attempts[0].GradeNanos = 0
	run.Digest = runDigest(run)
	failure := filepath.Join(t.TempDir(), "failure.json")
	if err := writeExclusiveJSON(failure, run); err != nil {
		t.Fatal(err)
	}
	calls := 0
	rescored, err := rescoreExperiment(context.Background(), failure, func(_ context.Context, _ *Attempt) { calls++ })
	if err != nil {
		t.Fatal(err)
	}
	got, err := LoadExperimentRun(rescored)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || got.Attempts[0].Status != "timeout" {
		t.Fatal("rescore reclassified execution failure")
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Bounded native pilots
func TestNativeStopHistoryConsistency(t *testing.T) {
	cfg := testExperiment()
	cfg.Gemini = &GeminiConfig{Auth: "api-key-env", Reasoning: "low"}
	cfg.MaxTotalTokens = 20
	cfg.Model = "gemini-3.8-flash"
	response := func(session string, tokens int64) *AdapterResponse {
		return &AdapterResponse{Native: &GeminiEvidence{SessionID: session, ReportedModels: []string{cfg.Model}}, Usage: &Usage{InputTokens: tokens}}
	}
	for _, tc := range []struct {
		name, reason string
		attempts     []Attempt
		valid        bool
	}{
		{"below ceiling", "token-limit", []Attempt{{Response: response("a", 19)}}, false},
		{"at ceiling", "token-limit", []Attempt{{Response: response("a", 20)}}, true},
		{"continued past ceiling", "cancelled", []Attempt{{Response: response("a", 20)}, {Response: response("b", 1)}}, false},
		{"claimed missing but known", "unknown-usage", []Attempt{{Response: response("a", 1)}}, false},
		{"genuinely missing", "unknown-usage", []Attempt{{Response: &AdapterResponse{Native: &GeminiEvidence{SessionID: "a", ReportedModels: []string{cfg.Model}}}}}, true},
		{"no invented reuse", "session-reuse", []Attempt{{Response: response("a", 1)}}, false},
		{"reuse stop", "session-reuse", []Attempt{{Response: response("a", 1)}, {Response: response("a", 1), Diagnostic: "native session ID reused across attempts"}}, true},
		{"reuse cancelled", "cancelled", []Attempt{{Response: response("a", 1)}, {Response: response("a", 1), Diagnostic: "native session ID reused across attempts"}}, true},
		{"reuse deadline", "run-timeout", []Attempt{{Response: response("a", 1)}, {Response: response("a", 1), Diagnostic: "native session ID reused across attempts"}}, true},
		{"reuse ignored", "cancelled", []Attempt{{Response: response("a", 1)}, {Response: response("a", 1)}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := ExperimentRun{Resolved: ResolvedExperiment{Experiment: cfg, Schedule: make([]Trial, 4)}, StopReason: tc.reason, Attempts: tc.attempts}
			if got := validateNativeStop(r) == nil; got != tc.valid {
				t.Fatalf("stop validity=%t want=%t", got, tc.valid)
			}
		})
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Controlled bounded experiments
func TestNativeReusedSessionStops(t *testing.T) {
	cfg := nativeTestExperiment(t, `{"input_tokens":10,"output_tokens":5}`)
	script, err := os.ReadFile(cfg.Command[0])
	if err != nil {
		t.Fatal(err)
	}
	script = []byte(strings.ReplaceAll(string(script), "$PWD", "reused-session"))
	if err := os.WriteFile(cfg.Command[0], script, 0700); err != nil {
		t.Fatal(err)
	}
	cfg.Repetitions = 2
	resolved, err := ResolveExperiment(cfg, ".")
	if err != nil {
		t.Fatal(err)
	}
	path, err := RunExperiment(context.Background(), resolved, t.TempDir())
	if err == nil {
		t.Fatal("reused session did not stop pilot")
	}
	run, err := LoadExperimentRun(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(run.Attempts) != 2 || run.StopReason != "session-reuse" || run.Attempts[1].Status != "adapter-error" {
		t.Fatal("session reuse was hidden")
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Fixed-model harness treatments
// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Retained independently graded artifacts
func TestNativeModelIdentityStopsAndRescores(t *testing.T) {
	for _, tc := range []struct{ name, replacement, reason string }{
		{"changed", "gemini-other-model", "model-mismatch"},
		{"missing", "", "model-unverified"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := nativeTestExperiment(t, `{"input_tokens":10,"output_tokens":5}`)
			script, err := os.ReadFile(cfg.Command[0])
			if err != nil {
				t.Fatal(err)
			}
			// Change only the CLI's observed model identity, leaving the requested model fixed.
			changed := strings.ReplaceAll(string(script), cfg.Model, tc.replacement)
			if tc.replacement == "" {
				changed = strings.ReplaceAll(changed, `"models":{"":{"api":{"totalErrors":0,"totalRequests":1},"tokens":{"cached":0,"candidates":5,"input":10,"prompt":10,"thoughts":0,"tool":0,"total":15}}}`, `"models":{}`)
			}
			if err := os.WriteFile(cfg.Command[0], []byte(changed), 0700); err != nil {
				t.Fatal(err)
			}
			resolved, err := ResolveExperiment(cfg, ".")
			if err != nil {
				t.Fatal(err)
			}
			path, err := RunExperiment(context.Background(), resolved, t.TempDir())
			if err == nil {
				t.Fatal("model uncertainty must stop run")
			}
			run, err := LoadExperimentRun(path)
			if err != nil {
				t.Fatal(err)
			}
			if run.StopReason != tc.reason || len(run.Attempts) != 1 || run.Attempts[0].Status != "adapter-error" || run.Attempts[0].Grade.Passed {
				t.Fatalf("wrong model was graded or run continued: %+v", run)
			}
			replay, err := RescoreExperiment(context.Background(), path)
			if err != nil {
				t.Fatal(err)
			}
			rescored, err := LoadExperimentRun(replay)
			if err != nil {
				t.Fatal(err)
			}
			if rescored.StopReason != tc.reason || rescored.Attempts[0].Status != "adapter-error" {
				t.Fatal("replay lost model refusal")
			}
		})
	}
}
