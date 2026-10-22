package bench

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

// spec: openspec/specs/benchmark-reports/spec.md#Requirement: Offline validated report projection
func TestReportRejectsInvalidBeforeOutput(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "invalid.json")
	if err := os.WriteFile(input, []byte(`{"version":1}`), 0600); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(dir, "report")
	if err := ExportExperimentReport(context.Background(), input, "", dest); err == nil {
		t.Fatal("invalid evidence accepted")
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Fatal("invalid evidence created destination")
	}
}

// reportFixture is synthetic retained evidence, not a claim that an adapter or
// grader executed. Its fixed outcomes and timings are the independent oracle.
func reportFixture() ExperimentRun {
	cfg := testExperiment()
	cfg.Name = "Synthetic prescribed-task A/B"
	cfg.Command = []string{"/synthetic/not-executable"}
	cfg.Tasks = []string{"safe-path"}
	cfg.Baseline = "private-baseline-instructions"
	cfg.Candidate = "private-candidate-instructions"
	resolved := ResolvedExperiment{Original: cfg, Experiment: cfg, ConfigSHA: jsonDigest(cfg), EvaluatorSHA: evaluatorDigest(cfg.Tasks), RunnerSHA: digest([]byte("synthetic runner")), Runtime: "synthetic runtime", Grader: GraderIdentity{Path: "/synthetic/private-go", SHA: digest([]byte("synthetic Go driver")), Version: "synthetic Go"}, CommandFiles: map[string]string{"/synthetic/not-executable": digest([]byte("synthetic adapter"))}, Schedule: Schedule(cfg)}
	run := ExperimentRun{Version: 1, Created: time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC), Resolved: resolved}
	for _, trial := range resolved.Schedule {
		source := "package safepath // private-source-sentinel"
		u := &Usage{InputTokens: 10, OutputTokens: 20, CachedInputTokens: ptrInt64(4), ReasoningOutputTokens: ptrInt64(9)}
		pass := true
		elapsed := int64(100)
		if trial.Arm == "candidate" {
			u.InputTokens = 15
			u.OutputTokens = 25
			elapsed = 130
			if trial.Repetition == 2 {
				pass = false
				elapsed = 1000
			}
		}
		response := &AdapterResponse{Version: 1, Source: source, Usage: u}
		raw, _ := json.Marshal(response)
		status := "passed"
		if !pass {
			status = "failed"
		}
		run.Attempts = append(run.Attempts, Attempt{Request: newAdapterRequest(cfg, trial), Status: status, Response: response, Stdout: string(raw), Stderr: "private-stderr-sentinel", Diagnostic: "private-diagnostic-sentinel", SourceSHA: digest([]byte(source)), AgentNanos: elapsed, GradeNanos: 7, Grade: CandidateEvaluation{Supported: true, Passed: pass, Verdict: status, Diagnostics: "private-grade-sentinel"}})
	}
	run.Digest = runDigest(run)
	return run
}

func ptrInt64(n int64) *int64 { return &n }

func reportInput(t *testing.T, r ExperimentRun) string {
	t.Helper()
	r.Digest = runDigest(r)
	path := filepath.Join(t.TempDir(), "run.json")
	if err := writeExclusiveJSON(path, r); err != nil {
		t.Fatal(err)
	}
	return path
}

func changeReportUsage(a *Attempt, u *Usage) {
	a.Response.Usage = u
	raw, _ := json.Marshal(a.Response)
	a.Stdout = string(raw)
}

// spec: openspec/specs/benchmark-reports/spec.md#Requirement: Honest accounting and scoped publication
func TestReportProjectionOracle(t *testing.T) {
	r := reportFixture()
	got, err := projectExperimentReport(r, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Counts != (reportCounts{Scheduled: 4, Retained: 4, Graded: 4, Passed: 3, Failed: 1}) {
		t.Fatalf("counts: %+v", got.Counts)
	}
	if got.Accounting.KnownTokenSubtotal == nil || *got.Accounting.KnownTokenSubtotal != 140 || got.Accounting.ReportedUsageAttempts != 4 || got.Accounting.UnknownUsageAttempts != 0 || got.Accounting.Basis != "adapter-reported" || got.Accounting.CostStatus != "unknown" {
		t.Fatalf("accounting: %+v", got.Accounting)
	}
	if got.Comparison.BothPassed != 1 || got.Comparison.Regressions != 1 || got.Comparison.TotalTokenDelta == nil || *got.Comparison.TotalTokenDelta != 20 || got.Comparison.SuccessfulPairMeanDeltaNanos == nil || *got.Comparison.SuccessfulPairMeanDeltaNanos != 30 {
		t.Fatalf("comparison oracle: %+v", got.Comparison)
	}
	if len(got.Pairs) != 2 || got.Pairs[0].TimeDelta == nil || *got.Pairs[0].TimeDelta != 30 || got.Pairs[1].TimeDelta != nil || got.Pairs[1].TokenDelta == nil || *got.Pairs[1].TokenDelta != 10 {
		t.Fatalf("pair eligibility: %+v", got.Pairs)
	}
	path := reportInput(t, r)
	files, err := experimentReportFiles(context.Background(), path, "")
	if err != nil {
		t.Fatal(err)
	}
	for name, data := range files {
		for _, secret := range []string{"private-baseline-instructions", "private-candidate-instructions", "private-source-sentinel", "private-stderr-sentinel", "private-diagnostic-sentinel", "private-grade-sentinel", "/synthetic/"} {
			if bytes.Contains(data, []byte(secret)) {
				t.Fatalf("private field escaped into %s: %s", name, secret)
			}
		}
	}
	var evidence reportEvidence
	if err := json.Unmarshal(files["evidence.json"], &evidence); err != nil {
		t.Fatal(err)
	}
	original, _ := os.ReadFile(path)
	if evidence.Original.FileSHA != digest(original) || evidence.Original.RunSHA != r.Digest || evidence.Files["results.json"] != digest(files["results.json"]) || evidence.Files["results.md"] != digest(files["results.md"]) || evidence.OriginalsIncluded {
		t.Fatal("manifest did not bind actual bytes")
	}
	if _, err := parseExperimentRun(files["results.json"]); err == nil {
		t.Fatal("summary accepted as original")
	}
}

// spec: openspec/specs/benchmark-reports/spec.md#Requirement: Honest accounting and scoped publication
// spec: openspec/specs/benchmark-reports/spec.md#Requirement: Offline validated report projection
func TestReportPartialUnknownAndZero(t *testing.T) {
	for _, count := range []int{0, 1, 3, 4} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			r := reportFixture()
			r.StopReason = "cancelled"
			r.Attempts = r.Attempts[:count]
			for i := range r.Attempts {
				changeReportUsage(&r.Attempts[i], nil)
			}
			if count > 1 {
				changeReportUsage(&r.Attempts[1], &Usage{})
			}
			if count > 2 {
				a := &r.Attempts[2]
				a.Status = "adapter-error"
				a.Response = nil
				a.Stdout = "private-partial-provider-data"
				a.SourceSHA = ""
				a.Grade = CandidateEvaluation{}
				a.GradeNanos = 0
			}
			r.Digest = runDigest(r)
			got, err := projectExperimentReport(r, nil)
			if err != nil {
				t.Fatal(err)
			}
			if got.Counts.Retained != count || got.Counts.Unstarted != 4-count || got.Comparison.UnpairedAttempts != count%2 {
				t.Fatalf("partial denominator %+v", got)
			}
			known := 0
			if count > 1 {
				known = 1
			}
			if got.Accounting.ReportedUsageAttempts != known || got.Accounting.UnknownUsageAttempts != count-known {
				t.Fatalf("usage denominator %+v", got.Accounting)
			}
			if known == 0 && got.Accounting.KnownTokenSubtotal != nil {
				t.Fatal("unknown became zero")
			}
			if known == 1 && (got.Accounting.KnownTokenSubtotal == nil || *got.Accounting.KnownTokenSubtotal != 0 || got.Attempts[1].Usage == nil) {
				t.Fatal("reported zero became unknown")
			}
			if count > 2 && (got.Counts.Unevaluable != 1 || got.Attempts[2].Passed != nil) {
				t.Fatal("execution failure became quality result")
			}
		})
	}
	r := reportFixture()
	r.Attempts[0].Status = "grading-error"
	r.Attempts[0].Grade = CandidateEvaluation{}
	r.Digest = runDigest(r)
	got, err := projectExperimentReport(r, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Counts.Unevaluable != 1 || got.Counts.Graded != 3 || got.Pairs[0].QualityEligible || got.Pairs[0].TimeDelta != nil || got.Pairs[0].TokenDelta == nil {
		t.Fatal("grading error eligibility wrong")
	}
}

// spec: openspec/specs/benchmark-reports/spec.md#Requirement: Explicit replay association and outcome
func TestReportReplayAssociation(t *testing.T) {
	original := reportFixture()
	clone := func() ExperimentRun {
		var r ExperimentRun
		b, _ := json.Marshal(original)
		_ = json.Unmarshal(b, &r)
		return r
	}
	for _, tc := range []struct {
		name   string
		change func(*ExperimentRun)
		state  string
		refuse bool
	}{
		{"metadata", func(r *ExperimentRun) {
			r.Created = r.Created.Add(time.Hour)
			r.Attempts[0].GradeNanos++
			r.Attempts[0].Diagnostic = "new diagnostic"
			r.Attempts[0].Grade.Diagnostics = "new grading detail"
		}, "matched", false},
		{"outcome", func(r *ExperimentRun) {
			a := &r.Attempts[0]
			a.Status = "failed"
			a.Grade = CandidateEvaluation{Supported: true, Verdict: "failed"}
		}, "different-results", false},
		{"timing", func(r *ExperimentRun) { r.Attempts[0].AgentNanos++ }, "", true},
		{"source", func(r *ExperimentRun) {
			a := &r.Attempts[0]
			a.Response.Source += "\n// changed"
			a.SourceSHA = digest([]byte(a.Response.Source))
			changeReportUsage(a, a.Response.Usage)
		}, "", true},
		{"raw-stderr", func(r *ExperimentRun) { r.Attempts[0].Stderr = "changed" }, "", true},
		{"go-driver", func(r *ExperimentRun) { r.Resolved.Grader.SHA = digest([]byte("another driver")) }, "", true},
		{"runner", func(r *ExperimentRun) { r.Resolved.RunnerSHA = digest([]byte("another runner")) }, "", true},
		{"config", func(r *ExperimentRun) {
			r.Resolved.Original.Name = "changed"
			r.Resolved.Experiment.Name = "changed"
			r.Resolved.ConfigSHA = jsonDigest(r.Resolved.Original)
		}, "", true},
		{"order", func(r *ExperimentRun) { r.Attempts[0], r.Attempts[1] = r.Attempts[1], r.Attempts[0] }, "", true},
		{"evaluator", func(r *ExperimentRun) { r.Resolved.EvaluatorSHA = "changed" }, "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := clone()
			tc.change(&r)
			r.Digest = runDigest(r)
			got, err := projectExperimentReport(original, &reportSnapshot{Run: r})
			if tc.refuse {
				if err == nil {
					t.Fatal("incompatible replay accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.Replay.State != tc.state {
				t.Fatalf("replay state %+v", got.Replay)
			}
			if tc.state == "different-results" && (len(got.Replay.ChangedAttempts) != 1 || got.Replay.ChangedAttempts[0] != 0 || got.Attempts[0].Status != "passed") {
				t.Fatal("replay disagreement lost or original overwritten")
			}
		})
	}
	// Actual rescore mutates only gradable attempts; failed executions must match exactly.
	failed := clone()
	failed.Attempts[0].Status = "adapter-error"
	failed.Attempts[0].Response = nil
	failed.Attempts[0].SourceSHA = ""
	failed.Attempts[0].Grade = CandidateEvaluation{}
	failed.Digest = runDigest(failed)
	replay := failed
	replay.Attempts = append([]Attempt(nil), failed.Attempts...)
	replay.Attempts[0].Diagnostic = "changed execution diagnostic"
	replay.Digest = runDigest(replay)
	if _, err := associateReportReplay(failed, &reportSnapshot{Run: replay}); err == nil {
		t.Fatal("execution failure rewritten as regrade")
	}
}

// Original and replay differ only in discipline; passed and verdict are identical.
// spec: openspec/specs/benchmark-reports/spec.md#Requirement: Explicit replay association and outcome
// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Separate discipline grading
func TestReportReplayDisciplineChange(t *testing.T) {
	build := func(discipline string) ExperimentRun {
		r := reportFixture()
		cfg := r.Resolved.Experiment
		cfg.Tasks = []string{"reuse-email"}
		r.Resolved.Original, r.Resolved.Experiment = cfg, cfg
		r.Resolved.ConfigSHA = jsonDigest(cfg)
		r.Resolved.EvaluatorSHA = evaluatorDigest(cfg.Tasks)
		r.Resolved.Schedule = Schedule(cfg)
		for i := range r.Attempts {
			a := &r.Attempts[i]
			a.Request = newAdapterRequest(cfg, r.Resolved.Schedule[i])
			a.Status = "failed"
			a.Grade = CandidateEvaluation{Supported: true, Verdict: "failed", Behavior: "failed", Discipline: "passed"}
		}
		r.Attempts[0].Grade.Discipline = discipline
		r.Digest = runDigest(r)
		return r
	}
	original := build("passed")
	same, err := projectExperimentReport(original, &reportSnapshot{Run: build("passed")})
	if err != nil || same.Replay.State != "matched" {
		t.Fatalf("identical replay: %+v %v", same.Replay, err)
	}
	got, err := projectExperimentReport(original, &reportSnapshot{Run: build("failed")})
	if err != nil {
		t.Fatal(err)
	}
	if got.Replay.State != "different-results" || len(got.Replay.ChangedAttempts) != 1 || got.Replay.ChangedAttempts[0] != 0 {
		t.Fatalf("discipline-only replay change not reported: %+v", got.Replay)
	}
}

// spec: openspec/specs/benchmark-reports/spec.md#Requirement: Honest accounting and scoped publication
func TestReportHostileMetadata(t *testing.T) {
	label := "</td>\n# WINNER | [click](https://bad) & \x1b\u202e"
	got := reportText(label)
	for _, needle := range []string{"<", ">", "\n", "# WINNER", "|", "[click]", "https://", "\x1b", "\u202e"} {
		if strings.Contains(got, needle) {
			t.Fatalf("markup escaped boundary: %q", got)
		}
	}
	if !strings.Contains(got, "[U+001B]") || !strings.Contains(got, "[U+202E]") {
		t.Fatal("control evidence lost")
	}
	r := reportFixture()
	r.Resolved.Experiment.Name = label
	r.Resolved.Original.Name = label
	r.Resolved.ConfigSHA = jsonDigest(r.Resolved.Original)
	r.Digest = runDigest(r)
	files, err := experimentReportFiles(context.Background(), reportInput(t, r), "")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(files["results.md"]), "\n# WINNER") || !strings.Contains(string(files["results.md"]), got) {
		t.Fatal("label not rendered as data")
	}
}

// spec: openspec/specs/benchmark-reports/spec.md#Requirement: Deterministic non-overwriting output and read-only checks
func TestReportWorkedExamples(t *testing.T) {
	for _, name := range []string{"complete", "partial"} {
		t.Run(name, func(t *testing.T) {
			input := filepath.Join("testdata", "report", name+"-run.json")
			files, err := experimentReportFiles(context.Background(), input, "")
			if err != nil {
				t.Fatal(err)
			}
			for filename, got := range files {
				want, err := os.ReadFile(filepath.Join("testdata", "report", name, filename))
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(got, want) {
					t.Fatalf("worked example drift: %s/%s", name, filename)
				}
			}
			if err := CheckExperimentReport(context.Background(), input, "", filepath.Join("testdata", "report", name)); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// spec: openspec/specs/benchmark-reports/spec.md#Requirement: Honest accounting and scoped publication
// spec: openspec/specs/benchmark-reports/spec.md#Requirement: Deterministic non-overwriting output and read-only checks
func FuzzReportTextAndDeterminism(f *testing.F) {
	for _, seed := range []string{"plain", "\n|[x](bad)<script>", "\x00\x1b\u202e", "α & _ * `"} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, label string) {
		if len(label) > 4096 || !utf8.ValidString(label) {
			return
		}
		escaped := reportText(label)
		if strings.ContainsAny(escaped, "\n\r\t<>|`") {
			t.Fatal("unsafe report text")
		}
		r := reportFixture()
		r.Resolved.Experiment.Name = "fixture " + label
		r.Resolved.Original.Name = r.Resolved.Experiment.Name
		r.Resolved.ConfigSHA = jsonDigest(r.Resolved.Original)
		r.Digest = runDigest(r)
		projected, err := projectExperimentReport(r, nil)
		if err != nil {
			t.Fatal(err)
		}
		a := formatExperimentReport(projected)
		b := formatExperimentReport(projected)
		if a != b || !strings.Contains(a, reportText(r.Resolved.Experiment.Name)) {
			t.Fatal("rendering law failed")
		}
		if projected.Counts.Retained != 4 || projected.Accounting.KnownTokenSubtotal == nil || *projected.Accounting.KnownTokenSubtotal != 140 {
			t.Fatal("label affected facts")
		}
	})
}

// spec: openspec/specs/benchmark-reports/spec.md#Requirement: Honest accounting and scoped publication
// spec: openspec/specs/benchmark-reports/spec.md#Requirement: Offline validated report projection
func TestReportNativeAndAAAccounting(t *testing.T) {
	// Native envelopes here are public synthetic parser fixtures. Nothing executes.
	for _, adapter := range []string{"mercury", "gemini"} {
		for _, unknown := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/unknown=%t", adapter, unknown), func(t *testing.T) {
				r := reportFixture()
				cfg := r.Resolved.Experiment
				cfg.Mode = "aa"
				cfg.Candidate = cfg.Baseline
				cfg.Repetitions = 1
				cfg.MaxTotalTokens = 10000
				cfg.MaxRunSeconds = 60
				switch adapter {
				case "mercury":
					cfg.Agent = "claude"
					cfg.Model = "mercury-3"
					cfg.Claude = &ClaudeConfig{Auth: "api-key-env", Reasoning: "low", OutputFormat: "json-schema", Mercury: &MercuryConfig{Bridge: "/synthetic/private-bridge"}}
					r.Resolved.MercuryBridgeVersion = mercuryBridgeVersion
					r.Resolved.ClaudeVersion = "2.1.292 (Claude Code)"
					r.Resolved.CommandFiles[cfg.Claude.Mercury.Bridge] = digest([]byte("synthetic bridge"))
				case "gemini":
					cfg.Agent = "gemini"
					cfg.Model = "gemini-3.8-flash"
					cfg.Gemini = &GeminiConfig{Auth: "api-key-env", Reasoning: "low", CLIVersion: "0.64.0", OutputFormat: "json-schema"}
					r.Resolved.GeminiVersion = "0.64.0"
				}
				r.Resolved.Original = cfg
				r.Resolved.Experiment = cfg
				r.Resolved.ConfigSHA = jsonDigest(cfg)
				r.Resolved.Schedule = Schedule(cfg)
				r.Resolved.BudgetPolicy = "synthetic native budget"
				settings, err := nativeSettings(cfg)
				if err != nil {
					t.Fatal(err)
				}
				r.Resolved.NativeSettings = settings
				r.Resolved.NativeArguments = nativeArguments(cfg)
				r.Attempts = nil
				for i, trial := range r.Resolved.Schedule {
					raw := ""
					if adapter == "mercury" {
						usage := mercuryTestUsage
						if unknown && i == 1 {
							usage = `{"prompt_tokens":20}`
						}
						raw = mercuryTestEnvelope(t, mercuryTestJournal(t, mercuryTestBody(usage)), "")
						raw = strings.ReplaceAll(raw, "native-session", fmt.Sprintf("private-session-%d", i))
					} else {
						metrics := `{"api":{"totalRequests":1,"totalErrors":0},"tokens":{"input":6,"prompt":10,"tool":0,"candidates":11,"total":30,"cached":4,"thoughts":9}}`
						if unknown && i == 1 {
							metrics = `{"api":{"totalRequests":2,"totalErrors":1},"tokens":{"prompt":90,"tool":0,"candidates":9,"thoughts":1,"total":100}}`
						}
						raw = fmt.Sprintf(`{"session_id":"private-session-%d","response":"{\"source\":\"package safepath\"}","stats":{"models":{"gemini-3.8-flash":%s}}}`, i, metrics)
					}
					response, parseErr := parseNativeOutput(cfg, raw)
					a := Attempt{Request: newAdapterRequest(cfg, trial), Response: &response, Stdout: raw, SourceSHA: digest([]byte(response.Source)), Status: "passed", AgentNanos: 100, GradeNanos: 7, Grade: CandidateEvaluation{Supported: true, Passed: true, Verdict: "passed"}}
					if parseErr != nil {
						a.Status = "adapter-error"
						a.Diagnostic = "private-native-diagnostic"
						a.Grade = CandidateEvaluation{}
						a.GradeNanos = 0
						r.StopReason = "adapter-error"
					}
					if unknown && i == 1 && r.StopReason == "" {
						r.StopReason = "unknown-usage"
					}
					r.Attempts = append(r.Attempts, a)
				}
				r.Digest = runDigest(r)
				got, err := projectExperimentReport(r, nil)
				if err != nil {
					t.Fatal(err)
				}
				basis := "native-cli-reported"
				each := int64(30)
				if adapter == "mercury" {
					basis = "captured-provider-wire"
					each = 25
				}
				known := 2
				if unknown {
					known = 1
				}
				if got.Comparison.Mode != "aa" || got.Identity.BaselineSHA != got.Identity.CandidateSHA || got.Accounting.Basis != basis || got.Accounting.ReportedUsageAttempts != known || got.Accounting.UnknownUsageAttempts != 2-known || got.Accounting.KnownTokenSubtotal == nil || *got.Accounting.KnownTokenSubtotal != int64(known)*each {
					t.Fatalf("native accounting oracle: %+v", got.Accounting)
				}
				if unknown && (got.Attempts[1].Usage != nil || got.Pairs[0].TokenDelta != nil) {
					t.Fatal("partial native observation counted as authoritative usage")
				}
				files, err := experimentReportFiles(context.Background(), reportInput(t, r), "")
				if err != nil {
					t.Fatal(err)
				}
				for name, b := range files {
					for _, secret := range []string{"private-session-", "private-native-diagnostic", "private-baseline-instructions", "private-bridge", "package safepath", "package answer", "/synthetic/"} {
						if bytes.Contains(b, []byte(secret)) {
							t.Fatalf("native private field %s in %s", secret, name)
						}
					}
				}
			})
		}
	}
}

// spec: openspec/specs/benchmark-reports/spec.md#Requirement: Honest accounting and scoped publication
func TestReportMissingMetadataCells(t *testing.T) {
	r := reportFixture()
	r.Attempts[0].Status = "adapter-error"
	r.Attempts[0].Response = nil
	r.Attempts[0].SourceSHA = ""
	r.Attempts[0].Grade = CandidateEvaluation{}
	r.Digest = runDigest(r)
	projected, err := projectExperimentReport(r, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(formatExperimentReport(projected), "\n") {
		if strings.TrimRight(line, " \t") != line {
			t.Fatalf("empty report cell creates trailing whitespace: %q", line)
		}
	}
	if strings.HasPrefix(reportText("  value  "), " ") || strings.HasSuffix(reportText("  value  "), " ") {
		t.Fatal("edge whitespace became Markdown structure")
	}
	if reportText("") != "not recorded" {
		t.Fatal("missing metadata lacks explicit marker")
	}
}
