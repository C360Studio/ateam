package bench

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Versioned Gemini best-effort accounting
func TestGeminiAccountingComparison(t *testing.T) {
	// Totals are hand-calculated from primary usage: 12+7=19, 20+8=28,
	// and 5+6=11. Cache and thinking are subsets, not extra charges.
	known := `{"api":{"totalRequests":1,"totalErrors":0},"tokens":{"prompt":10,"tool":2,"candidates":4,"thoughts":3,"cached":4,"input":6,"total":19}}`
	failed := `{"api":{"totalRequests":1,"totalErrors":0},"tokens":{"prompt":20,"tool":0,"candidates":6,"thoughts":2,"cached":5,"input":15,"total":28}}`
	last := `{"api":{"totalRequests":1,"totalErrors":0},"tokens":{"prompt":5,"tool":0,"candidates":5,"thoughts":1,"total":11}}`
	partial := `{"api":{"totalRequests":2,"totalErrors":1},"tokens":{"prompt":90,"tool":0,"candidates":9,"thoughts":1,"total":100}}`
	zero := `{"api":{"totalRequests":1,"totalErrors":0},"tokens":{"prompt":0,"tool":0,"candidates":0,"thoughts":0,"total":0}}`
	for _, tc := range []struct {
		name, stop, want  string
		metrics, statuses []string
	}{
		{"no attempts", "cancelled", `{"basis":"cli-reported-best-effort","reported_usage_attempts":0,"unknown_usage_attempts":0,"known_token_subtotal":null,"cost_status":"unknown"}`, nil, nil},
		{"missing", "adapter-error", `{"basis":"cli-reported-best-effort","reported_usage_attempts":0,"unknown_usage_attempts":1,"known_token_subtotal":null,"cost_status":"unknown"}`, []string{`null`}, []string{"adapter-error"}},
		{"ambiguous native zeros", "unknown-usage", `{"basis":"cli-reported-best-effort","reported_usage_attempts":0,"unknown_usage_attempts":1,"known_token_subtotal":null,"cost_status":"unknown"}`, []string{zero}, []string{"passed"}},
		{"mixed with failed pair and unknown unpaired", "unknown-usage", `{"basis":"cli-reported-best-effort","reported_usage_attempts":2,"unknown_usage_attempts":1,"known_token_subtotal":47,"cost_status":"unknown"}`, []string{known, failed, partial}, []string{"passed", "failed", "passed"}},
		{"known unpaired execution failure", "adapter-error", `{"basis":"cli-reported-best-effort","reported_usage_attempts":3,"unknown_usage_attempts":0,"known_token_subtotal":58,"cost_status":"unknown"}`, []string{known, failed, last}, []string{"passed", "failed", "adapter-error"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			run := accountingTestRun(t, "0.64.0", tc.stop, tc.metrics, tc.statuses)
			comparison, err := CompareExperiment(run)
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(comparison)
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(encoded, &fields); err != nil {
				t.Fatal(err)
			}
			if !jsonEquivalent(fields["accounting"], []byte(tc.want)) {
				t.Fatalf("accounting = %s, want %s", fields["accounting"], tc.want)
			}
			text := FormatComparison(comparison)
			var expected struct {
				Reported int    `json:"reported_usage_attempts"`
				Unknown  int    `json:"unknown_usage_attempts"`
				Subtotal *int64 `json:"known_token_subtotal"`
			}
			if err := json.Unmarshal([]byte(tc.want), &expected); err != nil {
				t.Fatal(err)
			}
			required := []string{
				fmt.Sprintf("Accounting (cli-reported-best-effort): primary usage reported for %d/%d retained attempts; %d unknown.", expected.Reported, len(tc.metrics), expected.Unknown),
				"known observations", "cost is unknown",
			}
			if expected.Subtotal == nil {
				required = append(required, "Known token subtotal: unknown (no reported primary usage)")
			} else {
				required = append(required, fmt.Sprintf("Known token subtotal: %d;", *expected.Subtotal))
			}
			for _, fragment := range required {
				if !strings.Contains(text, fragment) {
					t.Fatalf("missing %q in comparison:\n%s", fragment, text)
				}
			}
		})
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Versioned Gemini best-effort accounting
func TestGeminiAccountingReportedZero(t *testing.T) {
	// The report preserves a real normalized zero as distinct from no evidence.
	// Native completed-output zeros are ambiguous upstream and tested separately.
	accounting := summarizeUsageAccounting([]Attempt{{Response: &AdapterResponse{Usage: &Usage{}}}})
	data, err := json.Marshal(accounting)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"basis":"cli-reported-best-effort","reported_usage_attempts":1,"unknown_usage_attempts":0,"known_token_subtotal":0,"cost_status":"unknown"}`
	if !jsonEquivalent(data, []byte(want)) {
		t.Fatalf("accounting = %s, want %s", data, want)
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Versioned Gemini best-effort accounting
func TestGeminiAccountingLegacyComparison(t *testing.T) {
	for _, version := range []string{"", "0.43.0"} {
		t.Run("version="+version, func(t *testing.T) {
			run := accountingTestRun(t, version, "cancelled", nil, nil)
			comparison, err := CompareExperiment(run)
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(comparison)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(encoded), `"accounting"`) {
				t.Fatal("historical comparison gained accounting JSON")
			}
			want := "A/A calibration: identical configured treatments; stochastic outcomes may differ.\nAttempts retained: 0/4 scheduled; 0 unpaired\nRun stopped: cancelled\nPaired pilot: 0 pairs\nBaseline passed: 0/0; candidate passed: 0/0\nQuality discordance: 0 candidate-only passes, 0 baseline-only passes, 0 both passed, 0 neither passed\nUnevaluable quality pairs: 0 (execution/grading errors or timeouts)\nbaseline usage completeness: 0/0 primary totals reported; 0 unknown; cache breakdown 0/0; reasoning breakdown 0/0\nbaseline all-attempt execution: unknown (n=0)\nbaseline passed execution: unknown (n=0)\nbaseline reported primary tokens: unknown (n=0)\ncandidate usage completeness: 0/0 primary totals reported; 0 unknown; cache breakdown 0/0; reasoning breakdown 0/0\ncandidate all-attempt execution: unknown (n=0)\ncandidate passed execution: unknown (n=0)\ncandidate reported primary tokens: unknown (n=0)\nVariation is descriptive across the configured task mix; population SD describes these observations, not uncertainty or equivalence. Cache/reasoning breakdowns are subsets and are not added to primary totals.\nExecution-time comparison: unknown (no both-passed pairs)\nToken comparison: unknown (0 missing-usage pairs)\nExploratory evidence only: no statistical significance, quality equivalence, or actual tool-activation claim. Usage is adapter-reported; cost is unknown.\n"
			if got := FormatComparison(comparison); got != want {
				t.Fatalf("historical text changed:\n%s", got)
			}
		})
	}
}

func accountingTestRun(t *testing.T, version, stop string, metrics, statuses []string) ExperimentRun {
	t.Helper()
	cfg := nativeTestExperiment(t, `null`)
	cfg.Repetitions = 2
	cfg.Gemini.CLIVersion = "0.64.0"
	resolved, err := ResolveExperiment(cfg, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if version != "0.64.0" {
		legacy := *resolved.Experiment.Gemini
		legacy.CLIVersion = version
		resolved.Experiment.Gemini = &legacy
		original := *resolved.Original.Gemini
		original.CLIVersion = version
		resolved.Original.Gemini = &original
		resolved.GeminiVersion = "0.43.0"
		resolved.BudgetPolicy = "GEMINI_API_KEY only; GEMINI_CLI_TRUST_WORKSPACE=true only in the generated workspace child process; stop between attempts at max_total_tokens or unknown usage; max_run_seconds includes grading; per-attempt timeout_seconds; no in-flight monetary guarantee; CLI-reported usage; 4 model turns/attempt and 8192 output tokens/turn; no fallback accepted; cost unknown"
		resolved.NativeSettings, err = nativeSettings(resolved.Experiment)
		if err != nil {
			t.Fatal(err)
		}
		resolved.NativeArguments = nativeArguments(resolved.Experiment)
		resolved.ConfigSHA = jsonDigest(resolved.Original)
		cfg = resolved.Experiment
	}
	run := ExperimentRun{Version: 1, Created: time.Now().UTC(), Resolved: resolved, StopReason: stop}
	for i, metric := range metrics {
		stdout := fmt.Sprintf(`{"session_id":"accounting-%d","response":"{\"source\":\"package safepath\"}","stats":{"models":{"%s":%s}}}`, i, cfg.Model, metric)
		response, err := parseNativeOutput(cfg, stdout)
		if err != nil {
			t.Fatal(err)
		}
		status := statuses[i]
		attempt := Attempt{Request: newAdapterRequest(cfg, resolved.Schedule[i]), Status: status, Response: &response, Stdout: stdout, SourceSHA: digest([]byte(response.Source)), AgentNanos: 100}
		switch status {
		case "passed":
			attempt.Grade = CandidateEvaluation{Supported: true, Passed: true, Verdict: "passed"}
		case "failed":
			attempt.Grade = CandidateEvaluation{Supported: true, Verdict: "failed"}
		case "adapter-error":
			attempt.Diagnostic = "fixture process exited after terminal output"
		}
		run.Attempts = append(run.Attempts, attempt)
	}
	run.Digest = runDigest(run)
	path := filepath.Join(t.TempDir(), "run.json")
	if err := writeExclusiveJSON(path, run); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadExperimentRun(path)
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}
