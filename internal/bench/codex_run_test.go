package bench

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func codexTestExperiment() Experiment {
	e := testExperiment()
	e.Agent, e.AgentVersion, e.Model, e.Mode = "codex", "", "gpt-6-luna", "aa"
	e.Candidate = e.Baseline
	e.Command = []string{"codex"}
	e.Codex = &CodexConfig{Auth: "api-key-env", Reasoning: "low"}
	e.MaxTotalTokens, e.MaxRunSeconds = 100000, 3600
	return e
}

func TestCodexExperimentValidation(t *testing.T) {
	good := codexTestExperiment()
	data, _ := json.Marshal(good)
	if _, err := ParseExperiment(data); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*Experiment)
	}{
		{"gemini also selected", func(e *Experiment) { e.Gemini = &GeminiConfig{Auth: "api-key-env", Reasoning: "low"} }},
		{"claude also selected", func(e *Experiment) { e.Claude = &ClaudeConfig{Auth: "api-key-env", Reasoning: "low"} }},
		{"model alias", func(e *Experiment) { e.Model = "luna" }},
		{"wrong agent", func(e *Experiment) { e.Agent = "claude" }},
		{"subscription", func(e *Experiment) { e.Codex.Auth = "subscription" }},
		{"implicit effort", func(e *Experiment) { e.Codex.Reasoning = "" }},
		{"unsupported effort", func(e *Experiment) { e.Codex.Reasoning = "invalid" }},
		{"extra argv", func(e *Experiment) { e.Command = []string{"codex", "resume"} }},
		{"no tokens", func(e *Experiment) { e.MaxTotalTokens = 0 }},
		{"no run deadline", func(e *Experiment) { e.MaxRunSeconds = 0 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := codexTestExperiment()
			tc.change(&e)
			data, _ := json.Marshal(e)
			if _, err := ParseExperiment(data); err == nil {
				t.Fatal("accepted invalid Codex configuration")
			}
		})
	}
}

// The independent oracle is a native response matching all three selections.
// Merely knowing the request, or a different provider's model usage, is insufficient.
func TestCodexIdentityRequiresNativeResolvedEvidence(t *testing.T) {
	e := codexTestExperiment()
	for _, tc := range []struct {
		name, model, effort, provider, source, want string
	}{
		{"resolved", "gpt-6-luna", "low", "benchmark_openai", "cli-thread-start", ""},
		{"missing model", "", "low", "benchmark_openai", "cli-thread-start", "model-unverified"},
		{"missing effort", "gpt-6-luna", "", "benchmark_openai", "cli-thread-start", "model-unverified"},
		{"missing provider", "gpt-6-luna", "low", "", "cli-thread-start", "model-unverified"},
		{"requested is not resolved", "gpt-6-luna", "low", "benchmark_openai", "requested", "model-unverified"},
		{"different model", "gpt-6-sol", "low", "benchmark_openai", "cli-thread-start", "model-mismatch"},
		{"different effort", "gpt-6-luna", "high", "benchmark_openai", "cli-thread-start", "model-mismatch"},
		{"different provider", "gpt-6-luna", "low", "openai", "cli-thread-start", "model-mismatch"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n := &NativeEvidence{Codex: &CodexEvidence{IdentitySource: tc.source}}
			if tc.model != "" {
				n.ResolvedModel = &tc.model
			}
			if tc.effort != "" {
				n.ResolvedReasoning = &tc.effort
			}
			if tc.provider != "" {
				n.Codex.ModelProvider = &tc.provider
			}
			if got := nativeIdentityStatus(e, n); got != tc.want {
				t.Fatalf("identity verdict=%s, want=%s", got, tc.want)
			}
		})
	}
	if got := nativeIdentityStatus(e, &NativeEvidence{ReportedModels: []string{e.Model}}); got != "model-unverified" {
		t.Fatal("Codex accepted model accounting without native resolved configuration")
	}
	if got := nativeIdentityStatus(e, nil); got != "model-unverified" {
		t.Fatal("Codex accepted absent evidence")
	}
	gemini := e
	gemini.Codex = nil
	gemini.Gemini = &GeminiConfig{}
	if got := nativeIdentityStatus(gemini, &NativeEvidence{ResolvedModel: &e.Model}); got != "model-unverified" {
		t.Fatal("Codex identity exception weakened existing provider guard")
	}
}

// The test-owned history table specifies the number of allowed attempts and
// terminal state. Replay must not launch the helper or overwrite old evidence.
func TestCodexRunStopsAndReplay(t *testing.T) {
	for _, tc := range []struct {
		name, scenario, reason, status string
		attempts, pairs                int
	}{
		{"complete", "complete", "", "passed", 2, 1},
		{"tokens", "complete", "token-limit", "passed", 1, 0},
		{"unknown", "unknownusage", "unknown-usage", "passed", 1, 0},
		{"model", "mismatchedmodel", "model-mismatch", "adapter-error", 1, 0},
		{"failed", "failed", "adapter-error", "adapter-error", 1, 0},
		{"reuse", "reusedsession", "session-reuse", "adapter-error", 2, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := fakeCodexExperimentScenario(t, tc.scenario)
			e.Mode, e.Candidate = "aa", e.Baseline
			e.Tasks, e.Repetitions, e.MaxTotalTokens, e.MaxRunSeconds = []string{"safe-path"}, 1, 1000, 60
			if tc.name == "tokens" {
				e.MaxTotalTokens = 15
			}
			if tc.name == "reuse" {
				e.Repetitions = 2
			}
			resolved, err := ResolveExperiment(e, ".")
			if err != nil {
				t.Fatal(err)
			}
			if resolved.CodexVersion != "codex-cli 0.160.1" || resolved.GeminiVersion != "" || resolved.ClaudeVersion != "" || !strings.Contains(resolved.BudgetPolicy, "backend identity unknown") {
				t.Fatal("wrong Codex provenance or identity assurance")
			}
			path, err := RunExperiment(context.Background(), resolved, t.TempDir())
			if (err != nil) != (tc.reason != "") {
				t.Fatalf("run error=%v, want stop=%s", err, tc.reason)
			}
			run, err := LoadExperimentRun(path)
			if err != nil {
				t.Fatal(err)
			}
			if len(run.Attempts) != tc.attempts || run.StopReason != tc.reason || run.Attempts[len(run.Attempts)-1].Status != tc.status {
				t.Fatalf("history mismatch attempts=%d stop=%s final=%s", len(run.Attempts), run.StopReason, run.Attempts[len(run.Attempts)-1].Status)
			}
			if tc.name == "complete" {
				if run.Attempts[0].Response.Native.SessionID == run.Attempts[1].Response.Native.SessionID || run.Attempts[0].Request.Instructions != run.Attempts[1].Request.Instructions {
					t.Fatal("A/A independence or treatment mismatch")
				}
				u := run.Attempts[0].Response.Usage
				if u.InputTokens != 10 || u.OutputTokens != 5 {
					t.Fatal("Codex primary totals double-counted cached input or reasoning")
				}
				if len(run.Attempts[0].Response.Native.ReportedModels) != 0 {
					t.Fatal("CLI-resolved model became provider-reported identity")
				}
			}
			comparison, err := CompareExperiment(run)
			if err != nil || comparison.Pairs != tc.pairs {
				t.Fatalf("comparison pairs=%d err=%v", comparison.Pairs, err)
			}
			before, _ := os.ReadFile(path)
			// If rescore accidentally executes the adapter, removing its launcher
			// makes that path fail; grading uses its separately recorded Go tool.
			if err := os.Remove(e.Command[0]); err != nil {
				t.Fatal(err)
			}
			replay, err := RescoreExperiment(context.Background(), path)
			if err != nil {
				t.Fatal(err)
			}
			rescored, err := LoadExperimentRun(replay)
			if err != nil || rescored.StopReason != tc.reason || rescored.Attempts[len(rescored.Attempts)-1].Status != tc.status {
				t.Fatalf("replay changed outcome: %v", err)
			}
			after, _ := os.ReadFile(path)
			if string(before) != string(after) {
				t.Fatal("replay mutated original evidence")
			}
		})
	}
}
