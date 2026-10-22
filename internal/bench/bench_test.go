package bench

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Honest evaluator self-tests
// spec: openspec/changes/benchmark-decision-tasks/specs/benchmark-experiments/spec.md#Requirement: Separate discipline grading
func TestSelfTestAndMutation(t *testing.T) {
	rep, err := RunSelfTest(context.Background())
	if err != nil {
		t.Fatalf("RunSelfTest failed: %v", err)
	}
	if !rep.SelfTestPassed {
		t.Fatalf("expected SelfTestPassed=true, got %+v", rep)
	}
	if rep.TotalTasks != 10 || rep.EvaluatedTasks != 9 || rep.SkippedTasks != 1 || rep.PassedTasks != 9 {
		t.Fatalf("expected nine evaluated reference checks and one unsupported task: %+v", rep)
	}
	decision := 0
	for _, task := range rep.Tasks {
		if task.Tier != "decision-opportunity" {
			continue
		}
		decision++
		// Good passes both gates; bad passes behavior and is caught by discipline alone; mutant fails behavior.
		if task.Evaluation != "discipline" || !task.GoodPassedSafety || !task.GoodPassedDiscipline || !task.BadPassedBehavior || !task.BadCaughtByDiscipline || !task.BadCaughtByGate || !task.MutantCaught || !task.Passed {
			t.Errorf("decision task %s did not prove both gates: %+v", task.ID, task)
		}
	}
	if decision != 3 {
		t.Fatalf("want three decision-opportunity tasks, got %d", decision)
	}
	if rep.SafetyPassRatePct != 100 {
		t.Errorf("SafetyPassRatePct = %d, want 100", rep.SafetyPassRatePct)
	}
	if rep.MutationCatchPct != 100 {
		t.Errorf("MutationCatchPct = %d, want 100", rep.MutationCatchPct)
	}
	md := FormatMarkdown(rep)
	if !strings.Contains(md, "relay-openspec-hold") || !strings.Contains(md, "gitea-webhook") {
		t.Errorf("FormatMarkdown missing expected task rows:\n%s", md)
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Honest evaluator self-tests
func TestCommentSpoofingAndEarlyReturnBypassRejected(t *testing.T) {
	// Probe 8: An always-accepting path resolver with expected keywords in a comment must fail evaluation.
	spoofed := `package safepath
import "path/filepath"
// filepath.Separator HasPrefix !strings.HasPrefix
func Resolve(base, rel string) (string, error) {
	return filepath.Join(base, rel), nil
}`
	if CheckCodingCandidate("safe-path", spoofed) {
		t.Fatal("CheckCodingCandidate accepted comment-spoofed always-accepting path resolver")
	}

	// Astra #4: Unconditional/early return (`if true`) before safety check must be rejected.
	earlyReturnBypass := `package safepath
import ("errors"
 "fmt"
 "path/filepath"; "path/filepath"; "strings")
func Resolve(base, rel string) (string, error) {
	if true { return rel, nil }
	absBase, err := filepath.Abs(base)
	if err != nil { return "", err }
	target, err := filepath.Abs(filepath.Join(absBase, rel))
	if err != nil { return "", err }
	if target != absBase && !strings.HasPrefix(target, absBase+string(filepath.Separator)) {
		return "", errors.New("blocked")
	}
	return target, nil
}`
	if CheckCodingCandidate("safe-path", earlyReturnBypass) {
		t.Fatal("CheckCodingCandidate accepted if true early-return bypass before safety guard")
	}

	// G-03: `if 1 == 1` early-return bypass must also be rejected behaviorally.
	oneEqOneBypass := `package safepath
import ("errors"
 "fmt"
 "path/filepath"; "path/filepath"; "strings")
func Resolve(base, rel string) (string, error) {
	if 1 == 1 { return rel, nil }
	absBase, err := filepath.Abs(base)
	if err != nil { return "", err }
	target, err := filepath.Abs(filepath.Join(absBase, rel))
	if err != nil { return "", err }
	if target != absBase && !strings.HasPrefix(target, absBase+string(filepath.Separator)) {
		return "", errors.New("blocked")
	}
	return target, nil
}`
	if CheckCodingCandidate("safe-path", oneEqOneBypass) {
		t.Fatal("CheckCodingCandidate accepted if 1 == 1 early-return bypass before safety guard")
	}

	nonConstEarlyReturn := `package safepath
import ("errors"
 "fmt"
 "path/filepath"; "path/filepath"; "strings")
func Resolve(base, rel string) (string, error) {
	if rel != "" { return rel, nil }
	absBase, err := filepath.Abs(base)
	if err != nil { return "", err }
	target, err := filepath.Abs(filepath.Join(absBase, rel))
	if err != nil { return "", err }
	if target != absBase && !strings.HasPrefix(target, absBase+string(filepath.Separator)) {
		return "", errors.New("blocked")
	}
	return target, nil
}`
	if CheckCodingCandidate("safe-path", nonConstEarlyReturn) {
		t.Fatal("CheckCodingCandidate accepted non-constant early nil-error return before safety guard")
	}
}

// Continuity checks do not exercise an adversarial or mutated fixture.
// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Honest evaluator self-tests
func TestContinuityDoesNotClaimMutationDetection(t *testing.T) {
	got := evaluateContinuityTask(Tasks()[5])
	if !got.Passed {
		t.Fatal("continuity fixture did not pass")
	}
	if got.BadCaughtByGate || got.MutantCaught {
		t.Fatal("continuity falsely claims adversarial or mutation detection")
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Honest evaluator self-tests
func TestMarkdownDoesNotInventSuccess(t *testing.T) {
	md := FormatMarkdown(SuiteReport{SafetyPassRatePct: 20})
	if strings.Contains(md, "100% of bad fixtures caught") {
		t.Fatal("markdown invents perfect safety")
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Honest evaluator self-tests
func TestUnsupportedSvelteCandidateRejected(t *testing.T) {
	if CheckCodingCandidate("ui-datepicker", `<div>type="date" bind:value $bindable</div>`) {
		t.Fatal("keyword-only Svelte candidate accepted without behavioral grader")
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Controlled bounded experiments
func TestSelfTestHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := RunSelfTest(ctx); err == nil {
		t.Fatal("canceled selftest did not return an error")
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Honest evaluator self-tests
// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Controlled bounded experiments
func TestCandidateEvaluationOutcomes(t *testing.T) {
	// These examples distinguish trusted assertion failure from an unavailable grader.
	for _, tc := range []struct {
		name, id, source, verdict string
		passed, supported         bool
	}{
		{"good", "safe-path", Tasks()[0].GoodSource, "passed", true, true},
		{"assertion", "safe-path", Tasks()[0].BadSource, "failed", false, true},
		{"compile", "safe-path", "not Go source", "failed", false, true},
		{"missing-function", "safe-path", "package safepath", "failed", false, true},
		{"wrong-package", "safe-path", "package wrong", "failed", false, true},
		{"unsupported", "ui-datepicker", "anything", "unsupported", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := EvaluateCodingCandidate(context.Background(), tc.id, tc.source)
			if err != nil || got.Passed != tc.passed || got.Supported != tc.supported || got.Verdict != tc.verdict || got.Diagnostics == "" {
				t.Fatalf("candidate outcome = %+v, %v", got, err)
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	got, err := EvaluateCodingCandidate(ctx, "safe-path", Tasks()[0].GoodSource)
	if !errors.Is(err, context.Canceled) || got.Passed || got.Verdict != "inconclusive" {
		t.Fatalf("canceled grade = %+v, %v", got, err)
	}
	got, err = EvaluateCodingCandidate(context.Background(), "unknown", "")
	if err == nil || got.Supported || got.Passed {
		t.Fatalf("unknown grade = %+v, %v", got, err)
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Honest evaluator self-tests
// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Retained independently graded artifacts
func TestCandidateRequiresExecutedTrustedTest(t *testing.T) {
	spec := TaskSpec{ID: "empty", TestSource: "package fixture\nimport \"testing\"\nfunc TestOther(t *testing.T) {}"}
	got, err := evaluateCandidate(context.Background(), spec, "package fixture")
	if err != nil || got.Passed || got.Verdict != "failed" {
		t.Fatalf("missing trusted test accepted: %+v, %v", got, err)
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Retained independently graded artifacts
func TestSaveSelfTestRunIsImmutable(t *testing.T) {
	rep := SuiteReport{Timestamp: time.Unix(100, 0)}
	root := t.TempDir()
	first, err := SaveSelfTestRun(root, rep)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	rep.PassedTasks = 42
	second, err := SaveSelfTestRun(root, rep)
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	if first == second || !bytes.Equal(before, after) {
		t.Fatal("saving another selftest replaced existing evidence")
	}
	if _, err := SaveSelfTestRun(first, rep); err == nil {
		t.Fatal("save error was ignored")
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Honest evaluator self-tests
// spec: openspec/changes/benchmark-decision-tasks/specs/benchmark-experiments/spec.md#Requirement: Separate discipline grading
func TestParseSelfTestReportRejectsContradictions(t *testing.T) {
	// Construct evidence from the contract, independent of the production evaluator.
	rep := SuiteReport{SchemaVersion: 1, Timestamp: time.Unix(100, 0), Mode: "selftest-reference-fixtures", TotalTasks: 10,
		EvaluatedTasks: 9, SkippedTasks: 1, PassedTasks: 9, SafetyPassRatePct: 100, MutationCatchPct: 100, SelfTestPassed: true, MCPTokenTax: 200}
	for _, spec := range Tasks() {
		r := TaskResult{ID: spec.ID, Tier: spec.Tier, Evaluation: "behavioral", Passed: true, GoodPassedSafety: true, BadCaughtByGate: true, MutantCaught: true}
		if spec.ID == "ui-datepicker" {
			r.Evaluation = "unsupported"
			r.Passed, r.GoodPassedSafety, r.BadCaughtByGate, r.MutantCaught = false, false, false, false
		}
		if spec.Tier == "handoff-continuity" {
			r.Evaluation = "continuity"
			r.BadCaughtByGate, r.MutantCaught = false, false
		}
		if spec.Tier == "decision-opportunity" {
			r.Evaluation = "discipline"
			r.GoodPassedDiscipline, r.BadPassedBehavior, r.BadCaughtByDiscipline = true, true, true
		}
		rep.Tasks = append(rep.Tasks, r)
	}
	data, err := json.Marshal(rep)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseSelfTestReport(data); err != nil {
		t.Fatalf("valid reference report: %v", err)
	}
	for _, tc := range []struct {
		name   string
		change func(*SuiteReport)
	}{
		{"version", func(r *SuiteReport) { r.SchemaVersion = 99 }},
		{"summary", func(r *SuiteReport) { r.PassedTasks = 0 }},
		{"percentage", func(r *SuiteReport) { r.MutationCatchPct = 0 }},
		{"duplicate", func(r *SuiteReport) { r.Tasks[1] = r.Tasks[0] }},
		{"continuity", func(r *SuiteReport) { r.Tasks[5].MutantCaught = true }},
		{"unsupported", func(r *SuiteReport) { r.Tasks[4].Passed = true }},
		{"evidence", func(r *SuiteReport) { r.Tasks[0].MutantCaught = false }},
		{"bad caught by behavior", func(r *SuiteReport) { r.Tasks[7].BadPassedBehavior = false }},
		{"good fails discipline", func(r *SuiteReport) { r.Tasks[8].GoodPassedDiscipline = false }},
		{"discipline claimed on coding task", func(r *SuiteReport) { r.Tasks[0].BadCaughtByDiscipline = true }},
		{"discipline task graded as coding", func(r *SuiteReport) { r.Tasks[9].Evaluation = "behavioral" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var changed SuiteReport
			if err := json.Unmarshal(data, &changed); err != nil {
				t.Fatal(err)
			}
			tc.change(&changed)
			bad, err := json.Marshal(changed)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := ParseSelfTestReport(bad); err == nil {
				t.Fatal("contradictory selftest evidence accepted")
			}
		})
	}
	for _, bad := range [][]byte{[]byte(`{"selftest_passed":true}`), append(append([]byte{}, data...), []byte(` {}`)...), []byte(`{"unexpected":1}`)} {
		if _, err := ParseSelfTestReport(bad); err == nil {
			t.Fatal("invalid selftest schema accepted")
		}
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Controlled bounded experiments
func TestGraderDoesNotInheritSecrets(t *testing.T) {
	t.Setenv("ATEAM_TEST_SECRET", "must-not-leak")
	spec := TaskSpec{ID: "env", TestSource: `package fixture
import ("os"; "testing")
func TestFixture(t *testing.T) {
 if os.Getenv("ATEAM_TEST_SECRET") != "" { t.Fatal("inherited secret") }
 if os.Getenv("HOME") == "" || os.Getenv("TMPDIR") == "" { t.Fatal("missing isolated home or temp") }
}`}
	got, err := evaluateCandidate(context.Background(), spec, "package fixture")
	if err != nil || !got.Passed {
		t.Fatalf("grader environment = %+v, %v", got, err)
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Controlled bounded experiments
func TestDiagnosticBufferBoundsOutput(t *testing.T) {
	var b diagnosticBuffer
	payload := bytes.Repeat([]byte("x"), (1<<20)+10)
	if n, err := b.Write(payload); err != nil || n != len(payload) {
		t.Fatalf("write = %d, %v", n, err)
	}
	if b.Len() != 1<<20 || !b.truncated {
		t.Fatal("grader output was not bounded")
	}
	if _, err := b.Write([]byte("more")); err != nil || b.Len() != 1<<20 {
		t.Fatal("continued writes exceeded bound")
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Honest evaluator self-tests
func TestCandidateAcceptsAlternativeImplementation(t *testing.T) {
	// filepath.Rel provides an independent implementation, with no mutation reach requirement.
	candidate := `package safepath
import ("fmt"; "path/filepath"; "strings")
func Resolve(base, rel string) (string, error) {
 base, err := filepath.Abs(base); if err != nil { return "", err }
 target := filepath.Join(base, rel)
 relative, err := filepath.Rel(base, target); if err != nil { return "", err }
 if relative == ".." || strings.HasPrefix(relative, ".." + string(filepath.Separator)) { return "", fmt.Errorf("outside base") }
 return target, nil
}`
	got, err := EvaluateCodingCandidate(context.Background(), "safe-path", candidate)
	if err != nil || !got.Passed {
		t.Fatalf("alternative valid solution rejected: %+v, %v", got, err)
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Controlled bounded experiments
// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Honest evaluator self-tests
func TestCandidateCancellationDuringExecution(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "started")
	spec := TaskSpec{ID: "blocking", TestSource: fmt.Sprintf(`package fixture
import ("os"; "testing"; "time")
func TestFixture(t *testing.T) {
 if err := os.WriteFile(%q, []byte("ready"), 0600); err != nil { t.Fatal(err) }
 time.Sleep(time.Hour)
}`, marker)}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	type outcome struct {
		result CandidateEvaluation
		err    error
	}
	done := make(chan outcome, 1)
	go func() { result, err := evaluateCandidate(ctx, spec, "package fixture"); done <- outcome{result, err} }()
	deadline := time.NewTimer(10 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
waiting:
	for {
		select {
		case <-ticker.C:
			if _, err := os.Stat(marker); err == nil {
				break waiting
			}
		case got := <-done:
			t.Fatalf("grader returned before readiness: %+v", got)
		case <-deadline.C:
			t.Fatal("grader never reached synchronized cancellation boundary")
		}
	}
	cancel()
	select {
	case got := <-done:
		if !errors.Is(got.err, context.Canceled) || got.result.Passed || got.result.Verdict != "inconclusive" {
			t.Fatalf("canceled grader = %+v", got)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("grader failed to return after cancellation")
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Honest evaluator self-tests
func TestGraderInfrastructureFailure(t *testing.T) {
	t.Setenv("GOCACHE", filepath.Join(t.TempDir(), "cache-file"))
	if err := os.WriteFile(os.Getenv("GOCACHE"), []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := EvaluateCodingCandidate(context.Background(), "safe-path", Tasks()[0].GoodSource)
	if err == nil || got.Passed || got.Verdict != "inconclusive" {
		t.Fatalf("infrastructure failure counted as candidate failure: %+v, %v", got, err)
	}
}
