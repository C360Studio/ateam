package bench

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// The report must keep failed attempts and unknown usage visible; a plausible
// regression drops failures or interprets missing usage as zero tokens.
// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Paired descriptive comparisons
func TestCalibrationArmAccounting(t *testing.T) {
	zero, cached, reasoning := int64(0), int64(8), int64(3)
	var attempts []Attempt
	for _, arm := range []string{"baseline", "candidate"} {
		attempts = append(attempts,
			Attempt{Request: AdapterRequest{Trial: Trial{Arm: arm}}, Status: "passed", AgentNanos: 10, Response: &AdapterResponse{Usage: &Usage{InputTokens: 10, OutputTokens: 6, CachedInputTokens: &cached, ReasoningOutputTokens: &reasoning}}},
			Attempt{Request: AdapterRequest{Trial: Trial{Arm: arm}}, Status: "failed", AgentNanos: 30, Response: &AdapterResponse{Usage: &Usage{CachedInputTokens: &zero}}},
			Attempt{Request: AdapterRequest{Trial: Trial{Arm: arm}}, Status: "timeout", AgentNanos: 20},
			Attempt{Request: AdapterRequest{Trial: Trial{Arm: arm}}, Status: "adapter-error", AgentNanos: 20, Response: &AdapterResponse{}},
		)
	}
	got := summarizeArms(attempts)
	for _, arm := range []string{"baseline", "candidate"} {
		summary := got[arm]
		if summary.Attempts != 4 || summary.ReportedUsageAttempts != 2 || summary.MissingUsageAttempts != 2 || summary.CachedUsageAttempts != 2 || summary.ReasoningUsageAttempts != 1 {
			t.Fatalf("%s usage denominators: %+v", arm, summary)
		}
		if !reflect.DeepEqual(summary.StatusCounts, map[string]int{"passed": 1, "failed": 1, "timeout": 1, "adapter-error": 1}) {
			t.Fatalf("%s dropped terminal outcome: %+v", arm, summary.StatusCounts)
		}
		if summary.ExecutionNanos.Count != 4 || summary.ExecutionNanos.Mean != 20 || summary.ExecutionNanos.Min != 10 || summary.ExecutionNanos.Max != 30 {
			t.Fatalf("%s all-attempt durations: %+v", arm, summary.ExecutionNanos)
		}
		if summary.PassedExecutionNanos.Count != 1 || summary.PassedExecutionNanos.Mean != 10 {
			t.Fatalf("%s success-only durations: %+v", arm, summary.PassedExecutionNanos)
		}
		// Cache/reasoning are subsets: totals are 16 and 0, not 27 and 0.
		wantTokens := &DescriptiveStats{Count: 2, Min: 0, Max: 16, Mean: 8, PopulationSD: 8}
		if !reflect.DeepEqual(summary.ReportedTokens, wantTokens) {
			t.Fatalf("%s reported token statistics: got %+v, want %+v", arm, summary.ReportedTokens, wantTokens)
		}
	}
	// Counterbalanced execution order must not change arm attribution/statistics.
	for i, j := 0, len(attempts)-1; i < j; i, j = i+1, j-1 {
		attempts[i], attempts[j] = attempts[j], attempts[i]
	}
	if reversed := summarizeArms(attempts); !reflect.DeepEqual(got, reversed) {
		t.Fatalf("counterbalanced order changed arm summaries: %+v", reversed)
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Paired descriptive comparisons
func TestCalibrationUnknownAndZero(t *testing.T) {
	if got := describeValues(nil); got != nil {
		t.Fatalf("missing measurements became zero: %+v", got)
	}
	want := &DescriptiveStats{Count: 1}
	if got := describeValues([]int64{0}); !reflect.DeepEqual(got, want) {
		t.Fatalf("measured zero became unknown: %+v", got)
	}
	for _, summary := range summarizeArms(nil) {
		if summary.Attempts != 0 || summary.ExecutionNanos != nil || summary.PassedExecutionNanos != nil || summary.ReportedTokens != nil {
			t.Fatalf("empty arm fabricated evidence: %+v", summary)
		}
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Paired descriptive comparisons
func TestCalibrationEveryOutcomeHasUsageDenominator(t *testing.T) {
	for _, arm := range []string{"baseline", "candidate"} {
		for _, status := range []string{"passed", "failed", "grading-error", "adapter-error", "timeout", "cancelled"} {
			for _, known := range []bool{false, true} {
				a := Attempt{Request: AdapterRequest{Trial: Trial{Arm: arm}}, Status: status, AgentNanos: 5}
				if known {
					a.Response = &AdapterResponse{Usage: &Usage{InputTokens: 7, OutputTokens: 2}}
				}
				s := summarizeArms([]Attempt{a})[arm]
				if s.Attempts != 1 || s.StatusCounts[status] != 1 || s.ExecutionNanos.Count != 1 || s.ExecutionNanos.Mean != 5 {
					t.Fatalf("terminal attempt hidden (%s/%s, known=%v): %+v", arm, status, known, s)
				}
				if known {
					if s.ReportedUsageAttempts != 1 || s.MissingUsageAttempts != 0 || s.ReportedTokens == nil || s.ReportedTokens.Mean != 9 {
						t.Fatalf("reported failure usage hidden (%s/%s): %+v", arm, status, s)
					}
				} else if s.ReportedUsageAttempts != 0 || s.MissingUsageAttempts != 1 || s.ReportedTokens != nil {
					t.Fatalf("unknown failure usage fabricated (%s/%s): %+v", arm, status, s)
				}
				if (s.PassedExecutionNanos != nil) != (status == "passed") {
					t.Fatalf("success-only timing contains %s", status)
				}
			}
		}
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: A/A calibration
// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Paired descriptive comparisons
func TestCalibrationAAComparison(t *testing.T) {
	run := syntheticRun(t)
	run.Resolved.Original.Mode = "aa"
	run.Resolved.Original.Candidate = run.Resolved.Original.Baseline
	run.Resolved.Experiment.Mode = "aa"
	run.Resolved.Experiment.Candidate = run.Resolved.Experiment.Baseline
	run.Resolved.ConfigSHA = jsonDigest(run.Resolved.Original)
	for i := range run.Attempts {
		a := &run.Attempts[i]
		a.Request = newAdapterRequest(run.Resolved.Experiment, a.Request.Trial)
		if a.Request.Trial.Arm == "candidate" {
			a.Response.Source = "package safepath\n// stochastic output"
			a.SourceSHA = digest([]byte(a.Response.Source))
			raw, err := json.Marshal(a.Response)
			if err != nil {
				t.Fatal(err)
			}
			a.Stdout = string(raw)
			a.Status = "failed"
			a.Grade = CandidateEvaluation{Supported: true, Verdict: "failed"}
		}
	}
	run.Digest = runDigest(run)
	c, err := CompareExperiment(run)
	if err != nil {
		t.Fatal(err)
	}
	if c.Mode != "aa" || c.Pairs != 2 || c.Regressions != 2 || c.Arms["candidate"].PassedExecutionNanos != nil {
		t.Fatalf("A/A rejected or hid stochastic outcomes: %+v", c)
	}
	formatted := FormatComparison(c)
	for _, fragment := range []string{"A/A calibration", "Quality discordance", "2 baseline-only passes", "stochastic outcomes may differ", "no statistical significance, quality equivalence", "candidate passed execution: unknown (n=0)"} {
		if !strings.Contains(formatted, fragment) {
			t.Fatalf("calibration report missing %q: %s", fragment, formatted)
		}
	}
	if strings.Contains(formatted, "Quality changes:") {
		t.Fatal("A/A labeled stochastic differences treatment improvements")
	}
	ab, err := CompareExperiment(syntheticRun(t))
	if err != nil || ab.Mode != "ab" {
		t.Fatalf("legacy experiment mode: %q, %v", ab.Mode, err)
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Paired descriptive comparisons
func TestCalibrationPartialSchedule(t *testing.T) {
	for _, retained := range []int{0, 1, 2, 3} {
		run := syntheticRun(t)
		run.Attempts = run.Attempts[:retained]
		run.StopReason = "cancelled"
		run.Digest = runDigest(run)
		comparison, err := CompareExperiment(run)
		if err != nil {
			t.Fatalf("retained %d: %v", retained, err)
		}
		if comparison.CompletedAttempts != retained || comparison.ScheduledAttempts != 4 || comparison.UnpairedAttempts != retained%2 || comparison.Pairs != retained/2 {
			t.Fatalf("partial history denominators: %+v", comparison)
		}
		if comparison.Arms["baseline"].Attempts+comparison.Arms["candidate"].Attempts != retained || comparison.BaselinePassed+comparison.CandidatePassed != retained {
			t.Fatalf("unpaired attempt hidden from arm totals: %+v", comparison)
		}
		if comparison.StatusCounts["baseline/passed"]+comparison.StatusCounts["candidate/passed"] != retained {
			t.Fatalf("unpaired terminal outcome missing: %+v", comparison.StatusCounts)
		}
		if !strings.Contains(FormatComparison(comparison), "Run stopped: cancelled") {
			t.Fatal("report hid stop reason")
		}
	}
}
