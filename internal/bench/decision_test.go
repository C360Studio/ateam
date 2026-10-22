package bench

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func decisionSpec(t *testing.T, id string) TaskSpec {
	t.Helper()
	for _, spec := range Tasks() {
		if spec.ID == id {
			return spec
		}
	}
	t.Fatalf("missing task %s", id)
	return TaskSpec{}
}

// Expected outcomes are hand-stated from each task's decision, not derived from the gate.
// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Separate discipline grading
func TestDisciplineGateFindings(t *testing.T) {
	reuse := disciplineRule{calls: []string{"normalizeEmail"}, forbidden: []string{"strings.TrimSpace", "strings.ToLower", "strings.EqualFold"}}
	surface := disciplineRule{exports: []string{"Slugify"}}
	stale := disciplineRule{stale: "16"}
	for _, tc := range []struct {
		name   string
		rule   disciplineRule
		source string
		pass   bool
	}{
		{"reuse calls helper", reuse, "package signup\nfunc SameAccount(a, b string) bool { return normalizeEmail(a) == normalizeEmail(b) }", true},
		{"reuse reimplements with strings", reuse, "package signup\nimport \"strings\"\nfunc SameAccount(a, b string) bool { return strings.ToLower(a) == strings.ToLower(b) }", false},
		{"reuse hides strings behind an alias", reuse, "package signup\nimport s \"strings\"\nfunc SameAccount(a, b string) bool { return normalizeEmail(s.TrimSpace(a)) == normalizeEmail(b) }", false},
		{"reuse never calls helper", reuse, "package signup\nfunc SameAccount(a, b string) bool { return a == b }", false},
		{"unparsable candidate", reuse, "package signup\nfunc SameAccount(", false},
		{"surface only requested function", surface, "package slug\nfunc Slugify(s string) string { return lower(s) }\nfunc lower(s string) string { return s }", true},
		{"surface unexported type is allowed", surface, "package slug\ntype options struct{ sep string }\nfunc Slugify(s string) string { return s }", true},
		{"surface exported options type", surface, "package slug\ntype Options struct{}\nfunc Slugify(s string) string { return s }", false},
		{"surface exported method", surface, "package slug\ntype builder struct{}\nfunc (builder) Build() string { return \"\" }\nfunc Slugify(s string) string { return s }", false},
		{"surface exported constant", surface, "package slug\nconst Separator = \"-\"\nfunc Slugify(s string) string { return s }", false},
		{"surface exported wrapper", surface, "package slug\nfunc Slugify(s string) string { return SlugifyWith(s) }\nfunc SlugifyWith(s string) string { return s }", false},
		{"stale derived message", stale, "package username\nimport \"fmt\"\nconst MaxLen = 24\nvar LengthError = fmt.Errorf(\"username must be 3-%d characters\", MaxLen)", true},
		{"stale strconv-derived message", stale, "package username\nimport (\"errors\"; \"strconv\")\nconst MaxLen = 24\nvar LengthError = errors.New(\"username must be 3-\" + strconv.Itoa(MaxLen) + \" characters\")", true},
		{"stale hand-edited message", stale, "package username\nimport \"errors\"\nconst MaxLen = 24\nvar LengthError = errors.New(\"username must be 3-24 characters\")", true},
		{"stale message left at 16", stale, "package username\nimport \"errors\"\nconst MaxLen = 24\nvar LengthError = errors.New(\"username must be 3-16 characters\")", false},
		{"stale raw string message", stale, "package username\nimport \"errors\"\nconst MaxLen = 24\nvar LengthError = errors.New(`username must be 16 or fewer`)", false},
		{"stale constant in hex", stale, "package username\nconst MaxLen = 0x10", false},
		{"stale constant with separator", stale, "package username\nconst MaxLen = 1_6", false},
		{"raised constant in hex", stale, "package username\nconst MaxLen = 0x18", true},
		{"digit boundaries are whole numbers", stale, "package username\nconst MaxLen = 24\nvar hint = \"ids 116 and 160 differ; 3-24\"", true},
		{"stale ignores comments", stale, "package username\n// MaxLen was raised from 16.\nconst MaxLen = 24", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			findings := checkDiscipline(tc.rule, tc.source)
			if tc.pass && len(findings) != 0 {
				t.Fatalf("discipline gate rejected a disciplined candidate: %v", findings)
			}
			if !tc.pass && len(findings) == 0 {
				t.Fatal("discipline gate accepted an undisciplined candidate")
			}
		})
	}
	beside := checkDiscipline(reuse, "package signup\nimport \"strings\"\nfunc SameAccount(a, b string) bool { return normalizeEmail(strings.TrimSpace(a)) == normalizeEmail(b) }")
	rederived := checkDiscipline(reuse, decisionSpec(t, "reuse-email").BadSource)
	if strings.Join(beside, "\n") != "calls strings.TrimSpace beside the existing helper" || !strings.Contains(strings.Join(rederived, "\n"), "re-derives existing logic with strings.TrimSpace") {
		t.Fatalf("misleading reuse findings: beside=%q rederived=%q", beside, rederived)
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Decision-opportunity tasks
// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Separate discipline grading
func TestDecisionCandidateGrading(t *testing.T) {
	for _, tc := range []struct {
		name, task, source   string
		behavior, discipline string
		passed               bool
	}{
		{"reuse good", "reuse-email", decisionSpec(t, "reuse-email").GoodSource, "passed", "passed", true},
		{"reuse bad passes tests only", "reuse-email", decisionSpec(t, "reuse-email").BadSource, "passed", "failed", false},
		{"propagate hand-edited message passes", "propagate-limit", "package username\nimport \"errors\"\nconst MaxLen = 24\nvar LengthError = errors.New(\"username must be 3-24 characters\")", "passed", "passed", true},
		{"propagate stale message fails discipline only", "propagate-limit", decisionSpec(t, "propagate-limit").BadSource, "passed", "failed", false},
		{"propagate constant not raised fails behavior", "propagate-limit", decisionSpec(t, "propagate-limit").MutantSource, "failed", "passed", false},
		{"discipline recorded when behavior fails", "overbuild-slug", decisionSpec(t, "overbuild-slug").MutantSource, "failed", "passed", false},
		// Returning the existing file again redeclares Validate: a candidate compile failure, not a grader error.
		{"existing file returned", "propagate-limit", "package username\nimport \"fmt\"\nconst MaxLen = 24\nvar LengthError = fmt.Errorf(\"username must be 3-%d characters\", MaxLen)\nfunc Validate(name string) error { return nil }", "failed", "passed", false},
		// go reports existing.go first; the mismatch is still the candidate's failure.
		{"package mismatch beside existing file", "reuse-email", "package accounts\nfunc SameAccount(a, b string) bool { return normalizeEmail(a) == normalizeEmail(b) }", "failed", "passed", false},
		{"existing task records no discipline", "safe-path", Tasks()[0].GoodSource, "", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := EvaluateCodingCandidate(context.Background(), tc.task, tc.source)
			if err != nil {
				t.Fatalf("grading error: %v\n%s", err, got.Diagnostics)
			}
			if !got.Supported || got.Behavior != tc.behavior || got.Discipline != tc.discipline || got.Passed != tc.passed || (got.Verdict == "passed") != tc.passed {
				t.Fatalf("got %+v, want behavior=%q discipline=%q passed=%t", got, tc.behavior, tc.discipline, tc.passed)
			}
			if tc.discipline != "" && !strings.HasPrefix(got.DisciplineDiagnostics, "source: ") {
				t.Fatalf("discipline diagnostics omit source size: %q", got.DisciplineDiagnostics)
			}
		})
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Decision-opportunity tasks
func TestDecisionPromptShowsExistingSource(t *testing.T) {
	const instruction = "already contains existing.go, shown below; it is compiled with your file, so do not return or redeclare it."
	for _, spec := range decisionTasks() {
		prompt := taskPrompt(spec.ID)
		if prompt == "" {
			t.Fatalf("%s has no prompt", spec.ID)
		}
		if spec.ExistingSource == "" {
			if strings.Contains(prompt, "existing.go") {
				t.Fatalf("%s mentions an existing file it does not have", spec.ID)
			}
			continue
		}
		if !strings.Contains(prompt, instruction) || !strings.Contains(prompt, "```go\n"+spec.ExistingSource+"```") {
			t.Fatalf("%s prompt does not show its existing file verbatim:\n%s", spec.ID, prompt)
		}
	}
	for _, id := range []string{"safe-path", "sql-user", "gitea-webhook"} {
		if strings.Contains(taskPrompt(id), "existing.go") {
			t.Fatalf("existing task %s prompt changed", id)
		}
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Separate discipline grading
// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Retained independently graded artifacts
func TestDecisionExperimentEvidence(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "adapter.sh")
	if err := os.WriteFile(script, []byte("exec '"+exe+"' -test.run=^TestCommandAdapterHelper$ -- fixture\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg := testExperiment()
	cfg.Command = []string{"/bin/sh", script}
	cfg.Tasks = []string{"reuse-email"}
	cfg.Repetitions = 1
	cfg.Baseline, cfg.Candidate = "good", "bad"
	resolved, err := ResolveExperiment(cfg, dir)
	if err != nil {
		t.Fatal(err)
	}
	path, err := RunExperiment(context.Background(), resolved, dir)
	if err != nil {
		t.Fatal(err)
	}
	run, err := LoadExperimentRun(path)
	if err != nil {
		t.Fatal(err)
	}
	grades := map[string]CandidateEvaluation{}
	for _, a := range run.Attempts {
		grades[a.Request.Trial.Arm] = a.Grade
	}
	if g := grades["baseline"]; g.Behavior != "passed" || g.Discipline != "passed" || !g.Passed {
		t.Fatalf("good fixture grade: %+v", g)
	}
	if g := grades["candidate"]; g.Behavior != "passed" || g.Discipline != "failed" || g.Passed {
		t.Fatalf("bad fixture grade: %+v", g)
	}
	c, err := CompareExperiment(run)
	if err != nil {
		t.Fatal(err)
	}
	if c.BaselinePassed != 1 || c.CandidatePassed != 0 || c.Regressions != 1 || c.BothPassedSourceBytes != nil {
		t.Fatalf("unexpected comparison: %+v", c)
	}
	rescored, err := RescoreExperiment(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	second, err := LoadExperimentRun(rescored)
	if err != nil {
		t.Fatal(err)
	}
	for i := range second.Attempts {
		if a, b := run.Attempts[i].Grade, second.Attempts[i].Grade; a.Behavior != b.Behavior || a.Discipline != b.Discipline || a.Passed != b.Passed {
			t.Fatalf("rescore changed attempt %d: %+v -> %+v", i, a, b)
		}
	}
	files, err := experimentReportFiles(context.Background(), path, rescored)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(files["results.json"]), `"discipline_gate": "failed"`) || !strings.Contains(string(files["results.md"]), "## Behavior and discipline") {
		t.Fatal("export omitted per-attempt behavior and discipline")
	}
	var report experimentReport
	if err := json.Unmarshal(files["results.json"], &report); err != nil || report.Replay.State != "matched" {
		t.Fatalf("replay state %q: %v", report.Replay.State, err)
	}

	for _, tc := range []struct {
		name   string
		change func(*ExperimentRun)
	}{
		{"missing discipline", func(r *ExperimentRun) { r.Attempts[0].Grade.Discipline = "" }},
		{"missing behavior", func(r *ExperimentRun) { r.Attempts[1].Grade.Behavior = "" }},
		{"pass contradicts discipline", func(r *ExperimentRun) {
			for i := range r.Attempts {
				if r.Attempts[i].Status == "passed" {
					r.Attempts[i].Grade.Discipline = "failed"
				}
			}
		}},
		{"unknown outcome", func(r *ExperimentRun) { r.Attempts[0].Grade.Behavior = "skipped" }},
		{"grading error with discipline evidence", func(r *ExperimentRun) {
			r.Attempts[0].Status = "grading-error"
			r.Attempts[0].Diagnostic = "grader unavailable"
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			changed, err := LoadExperimentRun(path)
			if err != nil {
				t.Fatal(err)
			}
			tc.change(&changed)
			changed.Digest = runDigest(changed)
			if _, err := CompareExperiment(changed); err == nil {
				t.Fatal("accepted contradictory discipline evidence")
			}
		})
	}
	legacy := syntheticRun(t)
	legacy.Attempts[0].Grade.Discipline = "passed"
	legacy.Digest = runDigest(legacy)
	if err := validateRun(legacy); err == nil {
		t.Fatal("accepted discipline evidence on an existing task")
	}
}

func setSource(a *Attempt, source string) {
	a.Response.Source = source
	raw, _ := json.Marshal(a.Response)
	a.Stdout = string(raw)
	a.SourceSHA = digest([]byte(source))
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Descriptive source size comparison
func TestComparisonSourceBytes(t *testing.T) {
	run := syntheticRun(t)
	for i := range run.Attempts {
		a := &run.Attempts[i]
		switch {
		case a.Request.Trial.Arm == "baseline":
			setSource(a, "package safepath") // 16 bytes
		case a.Request.Trial.Repetition == 1:
			setSource(a, "package safepath // larger") // 26 bytes
		default:
			// A failed candidate's size must not enter the both-passed total.
			setSource(a, "package safepath"+strings.Repeat(" ", 84))
			a.Status = "failed"
			a.Grade = CandidateEvaluation{Supported: true, Verdict: "failed"}
		}
	}
	run.Digest = runDigest(run)
	c, err := CompareExperiment(run)
	if err != nil {
		t.Fatal(err)
	}
	if c.BothPassed != 1 || c.BothPassedSourceBytes["baseline"] != 16 || c.BothPassedSourceBytes["candidate"] != 26 {
		t.Fatalf("source bytes over both-passed pairs: %+v", c.BothPassedSourceBytes)
	}
	if text := FormatComparison(c); !strings.Contains(text, "Source bytes over 1 both-passed pairs: baseline 16, candidate 26") {
		t.Fatalf("comparison text omits source bytes:\n%s", text)
	}

	for i := range run.Attempts {
		if a := &run.Attempts[i]; a.Request.Trial.Arm == "candidate" {
			a.Status = "failed"
			a.Grade = CandidateEvaluation{Supported: true, Verdict: "failed"}
		}
	}
	run.Digest = runDigest(run)
	c, err = CompareExperiment(run)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(c)
	if c.BothPassedSourceBytes != nil || strings.Contains(string(data), "both_passed_source_bytes") || strings.Contains(FormatComparison(c), "Source bytes") {
		t.Fatal("reported source bytes without a both-passed pair")
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Controlled bounded experiments
func TestDecisionExamplesParse(t *testing.T) {
	for _, name := range []string{"decision-aa.json", "decision-ab.json", "decision-bundles-ab.json"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "benchmarks", "examples", name))
		if err != nil {
			t.Fatal(err)
		}
		e, err := ParseExperiment(data)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for _, id := range e.Tasks {
			if !isDecisionTask(id) {
				t.Fatalf("%s selects non-decision task %s", name, id)
			}
		}
	}
}
