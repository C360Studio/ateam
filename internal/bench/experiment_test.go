package bench

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testExperiment() Experiment {
	return Experiment{Version: 1, Name: "pilot", Command: []string{"/bin/cat"}, Agent: "fake", AgentVersion: "1", Model: "none", Baseline: "plain", Candidate: "check boundaries", Tasks: []string{"safe-path", "sql-user"}, Repetitions: 2, Seed: 42, TimeoutSeconds: 5}
}

func TestExperimentValidationAndSchedule(t *testing.T) {
	cfg := testExperiment()
	data, _ := json.Marshal(cfg)
	got, err := ParseExperiment(data)
	if err != nil {
		t.Fatal(err)
	}
	schedule := Schedule(got)
	if len(schedule) != 8 {
		t.Fatalf("want eight attempts, got %d", len(schedule))
	}
	for i := 0; i < len(schedule); i += 2 {
		a, b := schedule[i], schedule[i+1]
		if a.Task != b.Task || a.Repetition != b.Repetition || a.Arm == b.Arm {
			t.Fatalf("unpaired schedule: %+v", schedule)
		}
	}
	if schedule[0].Arm == schedule[4].Arm {
		t.Fatal("repetitions must counterbalance first arm")
	}
	for _, change := range []func(*Experiment){
		func(e *Experiment) { e.Version = 0 }, func(e *Experiment) { e.Repetitions = 0 }, func(e *Experiment) { e.Repetitions = 101 },
		func(e *Experiment) { e.TimeoutSeconds = 0 }, func(e *Experiment) { e.Tasks = []string{"safe-path", "safe-path"} },
		func(e *Experiment) { e.Tasks = []string{"ui-datepicker"} }, func(e *Experiment) { e.Command = nil },
	} {
		bad := cfg
		change(&bad)
		b, _ := json.Marshal(bad)
		if _, err := ParseExperiment(b); err == nil {
			t.Fatalf("accepted invalid config: %s", b)
		}
	}
	if _, err := ParseExperiment(append(data, []byte(` {}`)...)); err == nil {
		t.Fatal("accepted trailing JSON")
	}
	if _, err := ParseExperiment(bytes.Replace(data, []byte(`"name"`), []byte(`"unknown"`), 1)); err == nil {
		t.Fatal("accepted unknown field")
	}
}

func TestAdapterProtocolAndIsolation(t *testing.T) {
	if os.Getenv("ATEAM_BENCH_HELPER") == "child" {
		return
	}
	t.Setenv("ATEAM_SECRET_PROBE", "must-not-leak")
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cfg := testExperiment()
	cfg.Command = []string{exe, "-test.run=^TestCommandAdapterHelper$", "--"}
	trial := Schedule(cfg)[0]
	request := newAdapterRequest(cfg, trial)
	response, stdout, _, status, err := executeAdapter(context.Background(), cfg, request, t.TempDir())
	if err != nil || status != "completed" || response.Source != "package safepath" || response.Usage != nil {
		t.Fatalf("response=%+v status=%s err=%v stdout=%s", response, status, err, stdout)
	}
	if !strings.Contains(request.Prompt, "func Resolve") {
		t.Fatal("task API missing")
	}
}

func TestCommandAdapterHelper(_ *testing.T) {
	mode := ""
	for i, arg := range os.Args {
		if arg == "--" {
			mode = "protocol"
			if i+1 < len(os.Args) {
				mode = os.Args[i+1]
			}
			break
		}
	}
	if mode == "" {
		return
	}
	if os.Getenv("ATEAM_SECRET_PROBE") != "" {
		os.Exit(9)
	}
	cwd, _ := os.Getwd()
	home, _ := filepath.EvalSymlinks(os.Getenv("HOME"))
	cwd, _ = filepath.EvalSymlinks(cwd)
	if cwd != home {
		os.Exit(10)
	}
	var req AdapterRequest
	if json.NewDecoder(os.Stdin).Decode(&req) != nil {
		os.Exit(11)
	}
	if req.Instructions == "timeout" {
		time.Sleep(5 * time.Second)
		os.Exit(12)
	}
	if req.Instructions == "malformed" {
		_, _ = os.Stdout.WriteString(`{"version":1}`)
		os.Exit(0)
	}
	if req.Instructions == "overflow" {
		_, _ = os.Stdout.WriteString(strings.Repeat("x", maxAdapterBytes+1))
		os.Exit(0)
	}
	response := AdapterResponse{Version: 1, Source: "package safepath"}
	if mode == "fixture" {
		for _, spec := range Tasks() {
			if spec.ID == req.Trial.Task {
				response.Source = spec.GoodSource
				if req.Instructions == "bad" {
					response.Source = spec.BadSource
				}
			}
		}
	}
	_ = json.NewEncoder(os.Stdout).Encode(response)
	os.Exit(0)
}

func TestAdapterScriptProvenance(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "agent.sh")
	if err := os.WriteFile(p, []byte("exit 0\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cfg := testExperiment()
	cfg.Command = []string{"/bin/sh", "./agent.sh"}
	first, err := ResolveExperiment(cfg, dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("exit 1\n"), 0600); err != nil {
		t.Fatal(err)
	}
	second, err := ResolveExperiment(cfg, dir)
	if err != nil {
		t.Fatal(err)
	}
	if first.CommandFiles[p] == second.CommandFiles[p] {
		t.Fatal("script change absent from provenance")
	}
}

func FuzzExperimentDecode(f *testing.F) {
	b, _ := json.Marshal(testExperiment())
	f.Add(b)
	f.Add([]byte(`{"version":1}`))
	f.Add([]byte(`null`))
	f.Fuzz(func(t *testing.T, data []byte) {
		cfg, err := ParseExperiment(data)
		if err != nil {
			return
		}
		if len(Schedule(cfg)) != len(cfg.Tasks)*cfg.Repetitions*2 {
			t.Fatal("valid config schedule cardinality violated")
		}
	})
}

func TestAdapterFailuresAndUsage(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cfg := testExperiment()
	cfg.Command = []string{exe, "-test.run=^TestCommandAdapterHelper$", "--"}
	cfg.TimeoutSeconds = 1
	for _, tc := range []struct{ instruction, status string }{{"malformed", "adapter-error"}, {"timeout", "timeout"}, {"overflow", "adapter-error"}} {
		t.Run(tc.instruction, func(t *testing.T) {
			req := newAdapterRequest(cfg, Schedule(cfg)[0])
			req.Instructions = tc.instruction
			_, _, _, status, err := executeAdapter(context.Background(), cfg, req, t.TempDir())
			if err == nil || status != tc.status {
				t.Fatalf("status=%s err=%v", status, err)
			}
		})
	}
	for _, raw := range []string{`{}`, `{"input_tokens":1}`, `{"input_tokens":-1,"output_tokens":0}`, `{"input_tokens":1,"output_tokens":0,"cached_input_tokens":2}`} {
		var u Usage
		if json.Unmarshal([]byte(raw), &u) == nil {
			t.Fatalf("accepted unknown/invalid usage %s", raw)
		}
	}
}

func TestExperimentEndToEndAndRescore(t *testing.T) {
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
	cfg.Tasks = []string{"safe-path"}
	cfg.Repetitions = 1
	cfg.Baseline = "good"
	cfg.Candidate = "bad"
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
	c, err := CompareExperiment(run)
	if err != nil {
		t.Fatal(err)
	}
	if c.Pairs != 1 || c.BaselinePassed != 1 || c.Regressions != 1 || c.BothPassed != 0 || c.SuccessfulPairMeanDeltaNanos != nil || c.TotalTokenDelta != nil {
		t.Fatalf("unexpected comparison: %+v", c)
	}
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(script); err != nil {
		t.Fatal(err)
	}
	rescored, err := RescoreExperiment(context.Background(), path)
	if err != nil {
		t.Fatalf("offline rescore must not launch removed adapter: %v", err)
	}
	if rescored == path {
		t.Fatal("overwrote original")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(original, after) {
		t.Fatal("rescore modified original")
	}
	second, err := LoadExperimentRun(rescored)
	if err != nil {
		t.Fatal(err)
	}
	next, _ := CompareExperiment(second)
	if next.Regressions != 1 {
		t.Fatalf("rescore changed known outcome: %+v", next)
	}
	// A changed candidate must not be silently graded or compared using stale evidence.
	run.Attempts[0].Response.Source += "\n// modified"
	if err := validateRun(run); err == nil {
		t.Fatal("accepted corrupted artifact")
	}
}

func syntheticRun(t *testing.T) ExperimentRun {
	t.Helper()
	cfg := testExperiment()
	cfg.Tasks = []string{"safe-path"}
	cfg.Repetitions = 2
	resolved, err := ResolveExperiment(cfg, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	r := ExperimentRun{Version: 1, Created: time.Now().UTC(), Resolved: resolved}
	for _, trial := range resolved.Schedule {
		response := AdapterResponse{Version: 1, Source: "package safepath", Usage: &Usage{InputTokens: 100, OutputTokens: 10}}
		raw, _ := json.Marshal(response)
		r.Attempts = append(r.Attempts, Attempt{Request: newAdapterRequest(cfg, trial), Status: "passed", Response: &response, Stdout: string(raw), SourceSHA: digest([]byte(response.Source)), AgentNanos: 100, Grade: CandidateEvaluation{Supported: true, Passed: true, Verdict: "passed"}})
	}
	r.Digest = runDigest(r)
	return r
}

func TestComparisonOracleAndRefusals(t *testing.T) {
	run := syntheticRun(t)
	for i := range run.Attempts {
		a := &run.Attempts[i]
		if a.Request.Trial.Arm != "candidate" {
			continue
		}
		if a.Request.Trial.Repetition == 1 {
			a.AgentNanos = 130
		} else {
			a.AgentNanos = 1
			a.Status = "failed"
			a.Grade = CandidateEvaluation{Supported: true, Verdict: "failed"}
		}
	}
	run.Digest = runDigest(run)
	c, err := CompareExperiment(run)
	if err != nil {
		t.Fatal(err)
	}
	if c.Pairs != 2 || c.Regressions != 1 || c.BothPassed != 1 || c.SuccessfulPairMeanDeltaNanos == nil || *c.SuccessfulPairMeanDeltaNanos != 30 || c.UsagePairs != 2 || c.TotalTokenDelta == nil || *c.TotalTokenDelta != 0 {
		t.Fatalf("failed candidate skewed successful time: %+v", c)
	}
	for _, mutate := range []func(*ExperimentRun){
		func(r *ExperimentRun) { r.Attempts = r.Attempts[:1] },
		func(r *ExperimentRun) { r.Resolved.EvaluatorSHA = "other" },
		func(r *ExperimentRun) { r.Attempts[0].Grade.Passed = false },
		func(r *ExperimentRun) { r.Attempts[0].Request.Instructions = "unexpected" },
		func(r *ExperimentRun) { r.Attempts[0].SourceSHA = "other" },
	} {
		invalid := syntheticRun(t)
		mutate(&invalid)
		invalid.Digest = runDigest(invalid)
		if _, err := CompareExperiment(invalid); err == nil {
			t.Fatal("accepted contradictory/incomplete evidence")
		}
	}
}

func TestRescoreCancellationDuringFinalGrade(t *testing.T) {
	run := syntheticRun(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "run.json")
	if err := writeExclusiveJSON(path, run); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	_, err := rescoreExperiment(ctx, path, func(_ context.Context, a *Attempt) {
		calls++
		if calls == len(run.Attempts) {
			cancel()
			a.Status = "grading-error"
			a.Diagnostic = "cancelled"
		}
	})
	if err == nil {
		t.Fatal("final-grade cancellation returned successful rescore")
	}
	matches, _ := filepath.Glob(filepath.Join(dir, "rescore-*"))
	if len(matches) != 0 {
		t.Fatal("cancelled rescore wrote completed artifact")
	}
}

func TestComparisonDoesNotCallInfrastructureFailureQuality(t *testing.T) {
	run := syntheticRun(t)
	for i := range run.Attempts {
		if run.Attempts[i].Request.Trial.Arm == "candidate" {
			run.Attempts[i].Status = "grading-error"
			run.Attempts[i].Diagnostic = "tool unavailable"
			run.Attempts[i].Grade = CandidateEvaluation{Verdict: "inconclusive"}
		}
	}
	run.Digest = runDigest(run)
	c, err := CompareExperiment(run)
	if err != nil {
		t.Fatal(err)
	}
	if c.Regressions != 0 {
		t.Fatal("infrastructure error counted as quality regression")
	}
}
