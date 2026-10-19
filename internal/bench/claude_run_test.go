package bench

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func claudeRunFixture(t *testing.T, scenario string) Experiment {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture and managed policy inspection require Unix")
	}
	e := testExperiment()
	e.Agent, e.AgentVersion, e.Model, e.Mode = "claude", "", "claude-haiku-5-5", "aa"
	e.Candidate = e.Baseline
	e.Claude = &ClaudeConfig{Auth: "api-key-env", Reasoning: "low", OutputFormat: "json-schema"}
	e.Tasks, e.Repetitions, e.MaxTotalTokens, e.MaxRunSeconds = []string{"safe-path"}, 1, 1000, 60
	var source string
	for _, task := range Tasks() {
		if task.ID == "safe-path" {
			source = task.GoodSource
		}
	}
	answer, _ := json.Marshal(map[string]string{"source": source})
	counts := map[string]int{"inputTokens": 10, "outputTokens": 5, "cacheReadInputTokens": 3, "cacheCreationInputTokens": 2}
	models := map[string]any{e.Model: counts}
	result := map[string]any{"type": "result", "subtype": "success", "is_error": false, "num_turns": 1, "result": string(answer), "structured_output": map[string]string{"source": source}, "modelUsage": models}
	switch scenario {
	case "unknown":
		delete(counts, "outputTokens")
	case "tokens":
		e.MaxTotalTokens = 20
	case "model":
		models["claude-other-model"] = counts
	case "failed":
		result["subtype"], result["is_error"] = "error_during_execution", true
	case "reuse":
		e.Repetitions = 2
	}
	data, _ := json.Marshal(result)
	script := "#!/bin/sh\nif [ \"$1\" = --version ]; then echo '2.1.292 (Claude Code)'; exit 0; fi\ncat >/dev/null\nprintf '{\"session_id\":\"%s\",' \"$PWD\"\nprintf '%s\\n' '" + strings.ReplaceAll(string(data[1:]), "'", "'\\''") + "'\n"
	if scenario == "reuse" {
		script = strings.ReplaceAll(script, "$PWD", "reused")
	}
	e.Command = []string{filepath.Join(t.TempDir(), "claude-fixture")}
	if err := os.WriteFile(e.Command[0], []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ANTHROPIC_API_KEY", "fixture-not-real")
	return e
}

// A test-owned history model specifies exactly how many attempts each boundary
// permits. Replaying must neither call an adapter nor erase an execution failure.
func TestClaudeRunStopsAndReplay(t *testing.T) {
	for _, tc := range []struct {
		name, reason, finalStatus string
		attempts, pairs           int
	}{
		{"complete", "", "passed", 2, 1},
		{"tokens", "token-limit", "passed", 1, 0},
		{"unknown", "unknown-usage", "passed", 1, 0},
		{"model", "model-mismatch", "adapter-error", 1, 0},
		{"failed", "adapter-error", "adapter-error", 1, 0},
		{"reuse", "session-reuse", "adapter-error", 2, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := claudeRunFixture(t, tc.name)
			resolved, err := ResolveExperiment(e, ".")
			if err != nil {
				t.Fatal(err)
			}
			if resolved.ClaudeVersion != "2.1.292 (Claude Code)" || resolved.GeminiVersion != "" || !strings.Contains(resolved.BudgetPolicy, "ANTHROPIC_API_KEY only") {
				t.Fatal("wrong native provider provenance")
			}
			path, err := RunExperiment(context.Background(), resolved, t.TempDir())
			if (err != nil) != (tc.reason != "") {
				t.Fatalf("run error=%v, expected stop=%s", err, tc.reason)
			}
			run, err := LoadExperimentRun(path)
			if err != nil {
				t.Fatal(err)
			}
			if len(run.Attempts) != tc.attempts || run.StopReason != tc.reason || run.Attempts[len(run.Attempts)-1].Status != tc.finalStatus {
				t.Fatalf("history mismatch: attempts=%d stop=%s final=%s", len(run.Attempts), run.StopReason, run.Attempts[len(run.Attempts)-1].Status)
			}
			if tc.name == "complete" {
				if run.Attempts[0].Response.Native.SessionID == run.Attempts[1].Response.Native.SessionID || run.Attempts[0].Request.Instructions != run.Attempts[1].Request.Instructions {
					t.Fatal("A/A independence or treatment mismatch")
				}
				if run.Attempts[0].Response.Usage.InputTokens != 15 || run.Attempts[0].Response.Usage.OutputTokens != 5 {
					t.Fatal("cache accounting must total 15 input + 5 output")
				}
			}
			comparison, err := CompareExperiment(run)
			if err != nil || comparison.Pairs != tc.pairs {
				t.Fatalf("comparison pairs=%d err=%v", comparison.Pairs, err)
			}
			before, _ := os.ReadFile(path)
			replay, err := RescoreExperiment(context.Background(), path)
			if err != nil {
				t.Fatal(err)
			}
			rescored, err := LoadExperimentRun(replay)
			if err != nil || rescored.StopReason != tc.reason || rescored.Attempts[len(rescored.Attempts)-1].Status != tc.finalStatus {
				t.Fatalf("replay changed history: %v", err)
			}
			after, _ := os.ReadFile(path)
			if string(before) != string(after) {
				t.Fatal("replay mutated original evidence")
			}
		})
	}
}

func TestClaudeExperimentValidation(t *testing.T) {
	e := claudeRunFixture(t, "complete")
	for _, change := range []func(*Experiment){
		func(e *Experiment) { e.Gemini = &GeminiConfig{} },
		func(e *Experiment) { e.Model = "haiku" },
		func(e *Experiment) { e.Agent = "gemini" },
		func(e *Experiment) { e.Claude = &ClaudeConfig{Auth: "subscription", Reasoning: "low"} },
		func(e *Experiment) { e.Claude = &ClaudeConfig{Auth: "api-key-env"} },
		func(e *Experiment) {
			e.Claude = &ClaudeConfig{Auth: "api-key-env", Reasoning: "low", OutputFormat: "text"}
		},
		func(e *Experiment) { e.MaxTotalTokens = 0 },
		func(e *Experiment) { e.MaxRunSeconds = 0 },
		func(e *Experiment) { e.Command = append(e.Command, "--resume") },
	} {
		bad := e
		change(&bad)
		if validateExperiment(bad) == nil {
			t.Fatal("accepted invalid Claude experiment")
		}
	}
}

func TestClaudeLegacyReplayCannotExecute(t *testing.T) {
	e := claudeRunFixture(t, "complete")
	resolved, err := ResolveExperiment(e, ".")
	if err != nil {
		t.Fatal(err)
	}
	// Model the first pilot's exact text-output contract. This config remains
	// readable, but must never start a new native process after the schema fix.
	e.Claude.OutputFormat = ""
	if err := validateExperiment(e); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveExperiment(e, "."); err == nil || !strings.Contains(err.Error(), "replay-only") {
		t.Fatal("legacy text contract was executable")
	}
	resolved.Original.Claude.OutputFormat = ""
	resolved.Experiment.Claude.OutputFormat = ""
	resolved.ConfigSHA = jsonDigest(resolved.Original)
	resolved.NativeArguments = claudeArguments(e)
	raw := strings.ReplaceAll(claudeGoodOutput(), "claude-haiku-4-5-20251001", e.Model)
	var envelope map[string]any
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		t.Fatal(err)
	}
	envelope["result"] = "```json\n{\"source\":\"package legacy\"}\n```"
	data, _ := json.Marshal(envelope)
	response, parseErr := parseNativeOutput(e, string(data))
	if parseErr == nil {
		t.Fatal("legacy fenced reply was reinterpreted as success")
	}
	run := ExperimentRun{Version: 1, Created: time.Now().UTC(), Resolved: resolved, StopReason: "cancelled",
		Attempts: []Attempt{{Request: newAdapterRequest(e, resolved.Schedule[0]), Status: "adapter-error", Response: &response, Stdout: string(data), Diagnostic: parseErr.Error()}}}
	run.Digest = runDigest(run)
	path := filepath.Join(t.TempDir(), "legacy.json")
	if err := writeExclusiveJSON(path, run); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	rescored, err := rescoreExperiment(context.Background(), path, func(context.Context, *Attempt) { t.Fatal("legacy failure was graded") })
	if err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(path)
	got, err := LoadExperimentRun(rescored)
	if err != nil || got.Attempts[0].Status != "adapter-error" || string(before) != string(after) {
		t.Fatalf("legacy evidence changed: %v", err)
	}
}
