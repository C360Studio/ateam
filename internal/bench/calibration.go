package bench

import (
	"fmt"
	"math"
	"strings"
)

// DescriptiveStats describes observed values, without a population inference.
// An absent metric is nil; a measured zero remains zero with a positive Count.
type DescriptiveStats struct {
	Count        int     `json:"count"`
	Min          float64 `json:"min"`
	Max          float64 `json:"max"`
	Mean         float64 `json:"mean"`
	PopulationSD float64 `json:"population_sd"`
}

// ArmSummary retains the denominators for duration and reported usage summaries.
type ArmSummary struct {
	Attempts               int               `json:"attempts"`
	StatusCounts           map[string]int    `json:"status_counts"`
	ReportedUsageAttempts  int               `json:"reported_usage_attempts"`
	MissingUsageAttempts   int               `json:"missing_usage_attempts"`
	CachedUsageAttempts    int               `json:"cached_usage_attempts"`
	ReasoningUsageAttempts int               `json:"reasoning_usage_attempts"`
	ExecutionNanos         *DescriptiveStats `json:"execution_nanos"`
	PassedExecutionNanos   *DescriptiveStats `json:"passed_execution_nanos"`
	ReportedTokens         *DescriptiveStats `json:"reported_tokens"`
}

func summarizeArms(attempts []Attempt) map[string]ArmSummary {
	result := make(map[string]ArmSummary, 2)
	for _, arm := range []string{"baseline", "candidate"} {
		summary := ArmSummary{StatusCounts: map[string]int{}}
		var durations, passedDurations, tokens []int64
		for _, attempt := range attempts {
			if attempt.Request.Trial.Arm != arm {
				continue
			}
			summary.Attempts++
			summary.StatusCounts[attempt.Status]++
			durations = append(durations, attempt.AgentNanos)
			if attempt.Status == "passed" {
				passedDurations = append(passedDurations, attempt.AgentNanos)
			}
			if attempt.Response == nil || attempt.Response.Usage == nil {
				summary.MissingUsageAttempts++
				continue
			}
			usage := attempt.Response.Usage
			summary.ReportedUsageAttempts++
			tokens = append(tokens, usage.InputTokens+usage.OutputTokens)
			if usage.CachedInputTokens != nil {
				summary.CachedUsageAttempts++
			}
			if usage.ReasoningOutputTokens != nil {
				summary.ReasoningUsageAttempts++
			}
		}
		summary.ExecutionNanos = describeValues(durations)
		summary.PassedExecutionNanos = describeValues(passedDurations)
		summary.ReportedTokens = describeValues(tokens)
		result[arm] = summary
	}
	return result
}

func describeValues(values []int64) *DescriptiveStats {
	if len(values) == 0 {
		return nil
	}
	summary := &DescriptiveStats{Count: len(values), Min: float64(values[0]), Max: float64(values[0])}
	for _, value := range values {
		v := float64(value)
		summary.Min = math.Min(summary.Min, v)
		summary.Max = math.Max(summary.Max, v)
		summary.Mean += v / float64(summary.Count)
	}
	for _, value := range values {
		difference := float64(value) - summary.Mean
		summary.PopulationSD += difference * difference / float64(summary.Count)
	}
	summary.PopulationSD = math.Sqrt(summary.PopulationSD)
	return summary
}

func formatArmSummaries(b *strings.Builder, arms map[string]ArmSummary) {
	for _, arm := range []string{"baseline", "candidate"} {
		summary, exists := arms[arm]
		if !exists {
			continue
		}
		fmt.Fprintf(b, "%s usage completeness: %d/%d primary totals reported; %d unknown; cache breakdown %d/%d; reasoning breakdown %d/%d\n", arm, summary.ReportedUsageAttempts, summary.Attempts, summary.MissingUsageAttempts, summary.CachedUsageAttempts, summary.Attempts, summary.ReasoningUsageAttempts, summary.Attempts)
		formatStats(b, arm+" all-attempt execution", summary.ExecutionNanos, 1e9, "s")
		formatStats(b, arm+" passed execution", summary.PassedExecutionNanos, 1e9, "s")
		formatStats(b, arm+" reported primary tokens", summary.ReportedTokens, 1, "tokens")
	}
	b.WriteString("Variation is descriptive across the configured task mix; population SD describes these observations, not uncertainty or equivalence. Cache/reasoning breakdowns are subsets and are not added to primary totals.\n")
}

func formatStats(b *strings.Builder, label string, values *DescriptiveStats, scale float64, unit string) {
	if values == nil {
		fmt.Fprintf(b, "%s: unknown (n=0)\n", label)
		return
	}
	fmt.Fprintf(b, "%s: n=%d mean=%.3f min=%.3f max=%.3f population-SD=%.3f %s\n", label, values.Count, values.Mean/scale, values.Min/scale, values.Max/scale, values.PopulationSD/scale, unit)
}
