package bench

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// Comparison summarizes paired pilot evidence with explicit denominators and no inference claims.
type Comparison struct {
	UnevaluablePairs             int            `json:"unevaluable_pairs"`
	Pairs                        int            `json:"pairs"`
	BaselinePassed               int            `json:"baseline_passed"`
	CandidatePassed              int            `json:"candidate_passed"`
	Improvements                 int            `json:"improvements"`
	Regressions                  int            `json:"regressions"`
	BothPassed                   int            `json:"both_passed"`
	NeitherPassed                int            `json:"neither_passed"`
	StatusCounts                 map[string]int `json:"status_counts"`
	SuccessfulPairMeanDeltaNanos *float64       `json:"successful_pair_mean_delta_nanos"`
	UsagePairs                   int            `json:"usage_pairs"`
	MissingUsagePairs            int            `json:"missing_usage_pairs"`
	TotalTokenDelta              *int64         `json:"total_token_delta"`
}

// CompareExperiment compares the two configured arms only after complete-history validation.
func CompareExperiment(run ExperimentRun) (Comparison, error) {
	c := Comparison{StatusCounts: map[string]int{}}
	if err := validateRun(run); err != nil {
		return c, err
	}
	var timeDelta int64
	var tokenDelta int64
	for i := 0; i < len(run.Attempts); i += 2 {
		a, b := run.Attempts[i], run.Attempts[i+1]
		if a.Request.Trial.Arm == "candidate" {
			a, b = b, a
		}
		c.Pairs++
		c.StatusCounts["baseline/"+a.Status]++
		c.StatusCounts["candidate/"+b.Status]++
		if a.Status == "passed" {
			c.BaselinePassed++
		}
		if b.Status == "passed" {
			c.CandidatePassed++
		}
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
	fmt.Fprintf(&b, "Paired pilot: %d pairs\nBaseline passed: %d/%d; candidate passed: %d/%d\nQuality changes: %d improvements, %d regressions, %d both passed, %d neither passed\n", c.Pairs, c.BaselinePassed, c.Pairs, c.CandidatePassed, c.Pairs, c.Improvements, c.Regressions, c.BothPassed, c.NeitherPassed)
	fmt.Fprintf(&b, "Unevaluable quality pairs: %d (execution/grading errors or timeouts)\n", c.UnevaluablePairs)
	keys := make([]string, 0, len(c.StatusCounts))
	for key := range c.StatusCounts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Fprintf(&b, "%s: %d\n", key, c.StatusCounts[key])
	}
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
