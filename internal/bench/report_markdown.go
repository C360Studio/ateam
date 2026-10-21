package bench

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// reportText keeps arbitrary metadata in a single Markdown text cell. Encoding
// punctuation also prevents reference links, autolinks and injected table rows.
func reportText(value string) string {
	if value == "" {
		return "not recorded"
	}
	var b strings.Builder
	for i, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == ' ' && i > 0 && i < len(value)-1:
			b.WriteRune(r)
		case unicode.IsControl(r) || unicode.Is(unicode.Cf, r):
			fmt.Fprintf(&b, "[U+%04X]", r)
		default:
			fmt.Fprintf(&b, "&#x%X;", r)
		}
	}
	return b.String()
}

func reportNumber(value *int64) string {
	if value == nil {
		return "unknown"
	}
	return strconv.FormatInt(*value, 10)
}

func formatExperimentReport(r experimentReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Benchmark experiment report\n\n%s — %s\n\n", reportText(r.Identity.Name), reportText(r.Comparison.Mode))
	b.WriteString("Descriptive observations only. No winner, statistical significance, equivalence, ROI, or skill-activation claim. Cost is unknown.\n\n")
	b.WriteString("## Treatment and provenance\n\nField | Recorded value\n--- | ---\n")
	identity := r.Identity
	for _, field := range [][2]string{
		{"Agent", identity.Agent}, {"Agent version", identity.AgentVersion}, {"Requested model", identity.Model},
		{"Runtime", identity.Runtime}, {"Native version", identity.NativeVersion}, {"Bridge version", identity.BridgeVersion},
		{"Configuration SHA-256", identity.ConfigSHA}, {"Baseline instruction SHA-256", identity.BaselineSHA},
		{"Candidate instruction SHA-256", identity.CandidateSHA}, {"Runner SHA-256", identity.RunnerSHA},
		{"Evaluator SHA-256", identity.EvaluatorSHA}, {"Go driver SHA-256", identity.GoDriverSHA}, {"Go driver version", identity.GoDriverVersion},
		{"Native settings SHA-256", identity.NativeSettingsSHA}, {"Native arguments SHA-256", identity.NativeArgumentsSHA},
		{"Command file SHA-256 (paths omitted)", strings.Join(identity.CommandFileSHAs, ", ")},
		{"Tasks", strings.Join(identity.Tasks, ", ")}, {"Budget policy", identity.BudgetPolicy},
	} {
		fmt.Fprintf(&b, "%s | %s\n", field[0], reportText(field[1]))
	}
	fmt.Fprintf(&b, "\nRepetitions: %d; seed: %d; per-attempt limit: %d s; run limit: %d s; token ceiling: %d. Zero run/token limits mean no configured ceiling for command adapters.\n\n", identity.Repetitions, identity.Seed, identity.TimeoutSeconds, identity.MaxRunSeconds, identity.MaxTotalTokens)
	c := r.Counts
	fmt.Fprintf(&b, "## Outcomes and denominators\n\nScheduled: %d; retained: %d; unstarted: %d. Graded: %d (%d passed, %d failed); unevaluable: %d.\n\n", c.Scheduled, c.Retained, c.Unstarted, c.Graded, c.Passed, c.Failed, c.Unevaluable)
	stop := r.Comparison.StopReason
	if stop == "" {
		stop = "none"
	}
	fmt.Fprintf(&b, "Stop reason: %s. Replay: %s; changed attempt indexes: %s (zero-based).\n\n", reportText(stop), reportText(r.Replay.State), reportText(fmt.Sprint(r.Replay.ChangedAttempts)))
	a := r.Accounting
	fmt.Fprintf(&b, "Accounting basis: %s. Primary usage known for %d/%d retained attempts; unknown for %d. Known primary token subtotal: %s; cost: unknown. Cached input and reasoning output are subsets, never added again. Partial provider observations without normalized usage are excluded.\n\n", reportText(a.Basis), a.ReportedUsageAttempts, c.Retained, a.UnknownUsageAttempts, reportNumber(a.KnownTokenSubtotal))
	// Existing comparison owns aggregation. Render every line as escaped text,
	// replacing only its generic accounting disclaimer with the scoped one above.
	comparison := r.Comparison
	comparison.Accounting = nil
	lines := strings.Split(strings.TrimSpace(FormatComparison(comparison)), "\n")
	for _, line := range lines[:len(lines)-1] {
		fmt.Fprintf(&b, "%s\n\n", reportText(line))
	}
	b.WriteString("## Retained attempts\n\nIndex | Task | Repetition | Arm | Status | Quality passed | Execution ns | Grading ns | Input | Output | Cached subset | Reasoning subset | Source SHA-256\n--- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | ---\n")
	for _, a := range r.Attempts {
		quality := "unknown"
		if a.Passed != nil {
			quality = strconv.FormatBool(*a.Passed)
		}
		input, output, cached, reasoning := "unknown", "unknown", "unknown", "unknown"
		if a.Usage != nil {
			input = strconv.FormatInt(a.Usage.Input, 10)
			output = strconv.FormatInt(a.Usage.Output, 10)
			cached = reportNumber(a.Usage.Cached)
			reasoning = reportNumber(a.Usage.Reasoning)
		}
		fmt.Fprintf(&b, "%d | %s | %d | %s | %s | %s | %d | %d | %s | %s | %s | %s | %s\n", a.Index, reportText(a.Trial.Task), a.Trial.Repetition, reportText(a.Trial.Arm), reportText(a.Status), quality, a.AgentNanos, a.GradeNanos, input, output, cached, reasoning, reportText(a.SourceSHA))
	}
	b.WriteString("\n## Matched pairs\n\nBaseline index | Candidate index | Quality eligible | Both-passed execution delta ns | Known-usage token delta\n--- | --- | --- | --- | ---\n")
	for _, p := range r.Pairs {
		fmt.Fprintf(&b, "%d | %d | %t | %s | %s\n", p.Baseline, p.Candidate, p.QualityEligible, reportNumber(p.TimeDelta), reportNumber(p.TokenDelta))
	}
	b.WriteString("\nDeltas are candidate minus baseline. Execution deltas include only both-passed pairs; token deltas include pairs with both normalized usage observations, including failures. Unstarted attempts have no invented outcomes.\n\n## Schedule\n\nIndex | Task | Repetition | Arm\n--- | --- | --- | ---\n")
	for i, t := range r.Schedule {
		fmt.Fprintf(&b, "%d | %s | %d | %s\n", i, reportText(t.Task), t.Repetition, reportText(t.Arm))
	}
	b.WriteString("\n## Evidence and interpretation\n\nOriginal and replay artifacts are not included. Rederivation requires separately retained originals and the matching evaluator version; this report alone is not replayable. See evidence.json for byte digests and results.json for projected facts. A matched replay describes recorded outcome agreement, not proof of independent execution.\n\nRaw instructions, generated source, traces, diagnostics, environment, paths and native session IDs are omitted. Included metadata still requires human review before publication. Keep hypotheses and the next decision in a separate human-authored interpretation.md.\n\nClassifier decisions and classifier overhead are not recorded by this artifact schema and remain unmeasured. Execution time must not be presented as isolated classifier latency.\n")
	return b.String()
}
