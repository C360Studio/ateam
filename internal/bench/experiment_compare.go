package bench

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// Comparison summarizes paired pilot evidence with explicit denominators and no inference claims.
type Comparison struct {
	Accounting                   *UsageAccounting      `json:"accounting,omitempty"`
	Mode                         string                `json:"mode"`
	Arms                         map[string]ArmSummary `json:"arms"`
	StopReason                   string                `json:"stop_reason,omitempty"`
	ScheduledAttempts            int                   `json:"scheduled_attempts"`
	CompletedAttempts            int                   `json:"completed_attempts"`
	UnpairedAttempts             int                   `json:"unpaired_attempts"`
	UnevaluablePairs             int                   `json:"unevaluable_pairs"`
	Pairs                        int                   `json:"pairs"`
	BaselinePassed               int                   `json:"baseline_passed"`
	CandidatePassed              int                   `json:"candidate_passed"`
	Improvements                 int                   `json:"improvements"`
	Regressions                  int                   `json:"regressions"`
	BothPassed                   int                   `json:"both_passed"`
	NeitherPassed                int                   `json:"neither_passed"`
	StatusCounts                 map[string]int        `json:"status_counts"`
	SuccessfulPairMeanDeltaNanos *float64              `json:"successful_pair_mean_delta_nanos"`
	UsagePairs                   int                   `json:"usage_pairs"`
	MissingUsagePairs            int                   `json:"missing_usage_pairs"`
	TotalTokenDelta              *int64                `json:"total_token_delta"`
}

// UsageAccounting describes observed primary usage without implying complete provider billing.
type UsageAccounting struct {
	Basis                 string `json:"basis"`
	ReportedUsageAttempts int    `json:"reported_usage_attempts"`
	UnknownUsageAttempts  int    `json:"unknown_usage_attempts"`
	KnownTokenSubtotal    *int64 `json:"known_token_subtotal"`
	CostStatus            string `json:"cost_status"`
}

func summarizeUsageAccounting(attempts []Attempt) *UsageAccounting {
	accounting := &UsageAccounting{Basis: "cli-reported-best-effort", CostStatus: "unknown"}
	var subtotal int64
	for _, attempt := range attempts {
		if attempt.Response == nil || attempt.Response.Usage == nil {
			accounting.UnknownUsageAttempts++
			continue
		}
		accounting.ReportedUsageAttempts++
		subtotal += attempt.Response.Usage.InputTokens + attempt.Response.Usage.OutputTokens
	}
	if accounting.ReportedUsageAttempts > 0 {
		accounting.KnownTokenSubtotal = &subtotal
	}
	return accounting
}

// CompareExperiment compares the two configured arms only after retained-history validation.
func CompareExperiment(run ExperimentRun) (Comparison, error) {
	c := Comparison{StatusCounts: map[string]int{}}
	if err := validateRun(run); err != nil {
		return c, err
	}
	if cfg := run.Resolved.Experiment; cfg.Gemini != nil && cfg.Gemini.CLIVersion == "0.64.0" {
		c.Accounting = summarizeUsageAccounting(run.Attempts)
	}
	c.Mode = run.Resolved.Experiment.Mode
	if c.Mode == "" {
		c.Mode = "ab"
	}
	c.Arms = summarizeArms(run.Attempts)
	c.StopReason = run.StopReason
	c.ScheduledAttempts = len(run.Resolved.Schedule)
	c.CompletedAttempts = len(run.Attempts)
	c.UnpairedAttempts = len(run.Attempts) % 2
	c.BaselinePassed = c.Arms["baseline"].StatusCounts["passed"]
	c.CandidatePassed = c.Arms["candidate"].StatusCounts["passed"]
	for _, a := range run.Attempts {
		c.StatusCounts[a.Request.Trial.Arm+"/"+a.Status]++
	}
	var timeDelta int64
	var tokenDelta int64
	for i := 0; i+1 < len(run.Attempts); i += 2 {
		a, b := run.Attempts[i], run.Attempts[i+1]
		if a.Request.Trial.Arm == "candidate" {
			a, b = b, a
		}
		c.Pairs++
		switch {
		case a.Status == "passed" && b.Status == "passed":
			c.BothPassed++
			timeDelta += b.AgentNanos - a.AgentNanos
		case a.Status == "passed" && b.Status == "failed":
			c.Regressions++
		case b.Status == "passed" && a.Status == "failed":
			c.Improvements++
		case a.Status == "failed" && b.Status == "failed":
			c.NeitherPassed++
		default:
			c.UnevaluablePairs++
		}
		if a.Response != nil && b.Response != nil && a.Response.Usage != nil && b.Response.Usage != nil {
			c.UsagePairs++
			tokenDelta += b.Response.Usage.InputTokens + b.Response.Usage.OutputTokens - a.Response.Usage.InputTokens - a.Response.Usage.OutputTokens
		} else {
			c.MissingUsagePairs++
		}
	}
	if c.BothPassed > 0 {
		delta := float64(timeDelta) / float64(c.BothPassed)
		c.SuccessfulPairMeanDeltaNanos = &delta
	}
	if c.UsagePairs > 0 {
		c.TotalTokenDelta = &tokenDelta
	}
	return c, nil
}

// FormatComparison renders descriptive results without declaring a winner from a pilot.
func FormatComparison(c Comparison) string {
	var b strings.Builder
	if c.Mode == "aa" {
		b.WriteString("A/A calibration: identical configured treatments; stochastic outcomes may differ.\n")
	}
	fmt.Fprintf(&b, "Attempts retained: %d/%d scheduled; %d unpaired\n", c.CompletedAttempts, c.ScheduledAttempts, c.UnpairedAttempts)
	if c.StopReason != "" {
		fmt.Fprintf(&b, "Run stopped: %s\n", c.StopReason)
	}
	fmt.Fprintf(&b, "Paired pilot: %d pairs\nBaseline passed: %d/%d; candidate passed: %d/%d\n", c.Pairs, c.BaselinePassed, c.Arms["baseline"].Attempts, c.CandidatePassed, c.Arms["candidate"].Attempts)
	if c.Mode == "aa" {
		fmt.Fprintf(&b, "Quality discordance: %d candidate-only passes, %d baseline-only passes, %d both passed, %d neither passed\n", c.Improvements, c.Regressions, c.BothPassed, c.NeitherPassed)
	} else {
		fmt.Fprintf(&b, "Quality changes: %d improvements, %d regressions, %d both passed, %d neither passed\n", c.Improvements, c.Regressions, c.BothPassed, c.NeitherPassed)
	}
	fmt.Fprintf(&b, "Unevaluable quality pairs: %d (execution/grading errors or timeouts)\n", c.UnevaluablePairs)
	keys := make([]string, 0, len(c.StatusCounts))
	for key := range c.StatusCounts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Fprintf(&b, "%s: %d\n", key, c.StatusCounts[key])
	}
	formatArmSummaries(&b, c.Arms)
	if c.SuccessfulPairMeanDeltaNanos != nil {
		fmt.Fprintf(&b, "Mean candidate-minus-baseline execution time: %.3f s (%d both-passed pairs)\n", *c.SuccessfulPairMeanDeltaNanos/1e9, c.BothPassed)
	} else {
		b.WriteString("Execution-time comparison: unknown (no both-passed pairs)\n")
	}
	if c.TotalTokenDelta != nil {
		fmt.Fprintf(&b, "Candidate-minus-baseline reported tokens: %d (%d known-usage pairs; %d missing)\n", *c.TotalTokenDelta, c.UsagePairs, c.MissingUsagePairs)
	} else {
		fmt.Fprintf(&b, "Token comparison: unknown (%d missing-usage pairs)\n", c.MissingUsagePairs)
	}
	if accounting := c.Accounting; accounting != nil {
		fmt.Fprintf(&b, "Accounting (%s): primary usage reported for %d/%d retained attempts; %d unknown.\n", accounting.Basis, accounting.ReportedUsageAttempts, c.CompletedAttempts, accounting.UnknownUsageAttempts)
		if accounting.KnownTokenSubtotal == nil {
			b.WriteString("Known token subtotal: unknown (no reported primary usage); cost is unknown.\n")
		} else {
			fmt.Fprintf(&b, "Known token subtotal: %d; cost is unknown.\n", *accounting.KnownTokenSubtotal)
		}
		b.WriteString("These are known observations from CLI telemetry, not complete provider usage or actual cost; retries and failed calls may be unaccounted for.\n")
	}
	b.WriteString("Exploratory evidence only: no statistical significance, quality equivalence, or actual tool-activation claim. Usage is adapter-reported; cost is unknown.\n")
	return b.String()
}

func readBounded(r io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("artifact exceeds %d bytes", limit)
	}
	return data, nil
}
