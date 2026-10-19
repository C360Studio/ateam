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
	cfg.Agent, cfg.AgentVersion, cfg.Mode = "codex", "", "aa"
	cfg.Candidate = cfg.Baseline
	cfg.Codex = &CodexConfig{Auth: "api-key-env", Reasoning: "low"}
	cfg.MaxTotalTokens, cfg.MaxRunSeconds = 1000, 60
	cfg.Tasks, cfg.Repetitions = []string{"safe-path"}, 1
	var source string
	for _, task := range Tasks() {
		if task.ID == "safe-path" {
			source = task.GoodSource
		}
	}
	final, _ := json.Marshal(map[string]string{"source": source})
	event, _ := json.Marshal(map[string]any{"type": "item.completed", "item": map[string]string{"id": "answer", "type": "agent_message", "text": string(final)}})
	script := "#!/bin/sh\nif [ \"$1\" = --version ]; then echo 'codex-cli fixture'; exit 0; fi\ncat >/dev/null\nprintf '{\"type\":\"thread.started\",\"thread_id\":\"%s\"}\\n' \"$PWD\"\necho '{\"type\":\"turn.started\"}'\n" + "printf '%s\\n' '" + strings.ReplaceAll(string(event), "'", "'\\''") + "'\nprintf '%s\\n' '{\"type\":\"turn.completed\",\"usage\":" + usage + "}'\n"
	executable := filepath.Join(t.TempDir(), "codex-fixture")
	if err := os.WriteFile(executable, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	cfg.Command = []string{executable}
	t.Setenv("CODEX_API_KEY", "fixture-key-not-real")
	return cfg
}

func TestNativeBudgetStopsAndRescoresPartialHistory(t *testing.T) {
	// Independent oracle: first 15-token attempt exhausts a 10-token ceiling;
	// a missing-usage attempt also stops, never silently authorizing another call.
	for _, tc := range []struct {
		name, usage, reason string
		ceiling             int64
	}{
		{"tokens", `{"input_tokens":10,"output_tokens":5,"cached_input_tokens":3}`, "token-limit", 10},
		{"unknown", `null`, "unknown-usage", 1000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := nativeTestExperiment(t, tc.usage)
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
		func(e *Experiment) { e.Codex = &CodexConfig{Auth: "ambient", Reasoning: "low"} },
		func(e *Experiment) { e.Codex = &CodexConfig{Auth: "api-key-env", Reasoning: ""} },
		func(e *Experiment) { e.Command = append(e.Command, "exec") }, func(e *Experiment) { e.Agent = "unknown" },
	} {
		bad := cfg
		change(&bad)
		if validateExperiment(bad) == nil {
			t.Fatalf("accepted invalid native config: %+v", bad)
		}
	}
}

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

func TestNativeStopHistoryConsistency(t *testing.T) {
	cfg := testExperiment()
	cfg.Codex = &CodexConfig{Auth: "api-key-env", Reasoning: "low"}
	cfg.MaxTotalTokens = 20
	response := func(session string, tokens int64) *AdapterResponse {
		return &AdapterResponse{Native: &CodexEvidence{SessionID: session}, Usage: &Usage{InputTokens: tokens}}
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
		{"genuinely missing", "unknown-usage", []Attempt{{Response: &AdapterResponse{Native: &CodexEvidence{SessionID: "a"}}}}, true},
		{"no invented reuse", "session-reuse", []Attempt{{Response: response("a", 1)}}, false},
		{"reuse stop", "session-reuse", []Attempt{{Response: response("a", 1)}, {Response: response("a", 1), Diagnostic: "native session ID reused across attempts"}}, true},
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
