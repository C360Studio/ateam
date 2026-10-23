package review

import (
	"fmt"
	"strings"
)

// tierNames lists model tiers from the weakest to the strongest; the index is the rank.
var tierNames = []string{"light", "workhorse", "frontier"}

// bandNames lists effort bands from the lowest to the highest; the index is the rank.
var bandNames = []string{"low", "mid", "high"}

// ParseRunSpec reads an agent:model:effort run as named on a PR line. Effort may be empty.
func ParseRunSpec(s string) (Run, error) {
	parts := strings.SplitN(s, ":", 3)
	if len(parts) != 3 || !supportedProvider(parts[0]) || strings.TrimSpace(parts[1]) == "" || strings.Contains(parts[2], ":") {
		return Run{}, fmt.Errorf("run %q must be claude:, codex: or gemini: then model:effort, with no colon in the effort", s)
	}
	return Run{Provider: parts[0], Model: strings.TrimSpace(parts[1]), Effort: strings.TrimSpace(parts[2])}, nil
}

// Compare applies ADR 0002 rule 2: the reviewer passes when it is not weaker than
// every implementer, by tier and then by effort band. A missing effort or a weaker
// reviewer is a finding. A model or effort the policy does not map passes only when
// both sides of the comparison are unmapped (owner ruling on #125, 2026-10-22): an
// unranked run against a ranked one is a finding, since nothing shows it is not weaker.
func (p Policy) Compare(implementers []Run, reviewer Run) (findings, notes []string) {
	for _, r := range append([]Run{reviewer}, implementers...) {
		switch {
		case noEffort(r.Effort):
			findings = append(findings, fmt.Sprintf("%s names no effort; an agent that cannot read its own effort names the band it was launched at", describe(r)))
		case p.isModel(r.Provider, r.Effort):
			// "(sonnet, opus)" read as pairs puts a model where the effort goes, and
			// an unranked effort would pass it without comparing opus at all.
			findings = append(findings, fmt.Sprintf("%s names model %q where the effort goes; write model, effort pairs", describe(r), r.Effort))
		}
	}
	if len(findings) > 0 {
		return findings, nil
	}
	rt, rtok := p.tier(reviewer)
	rb, rbok := p.band(reviewer)
	for _, dev := range implementers {
		dt, dtok := p.tier(dev)
		db, dbok := p.band(dev)
		switch {
		case !rtok && !dtok:
			notes = append(notes, fmt.Sprintf("unranked: %s against %s; neither model is in model_tiers, so they pass unranked", describe(reviewer), describe(dev)))
		case !rtok || !dtok:
			findings = append(findings, p.unrankedFinding("model_tiers", reviewer, rtok, dev))
		case rt > dt:
		case rt < dt:
			findings = append(findings, fmt.Sprintf("reviewer %s is %s tier, weaker than implementer %s at %s tier", describe(reviewer), tierNames[rt], describe(dev), tierNames[dt]))
		case !rbok && !dbok:
			notes = append(notes, fmt.Sprintf("unranked: %s against %s; neither effort is in effort_bands, so they pass unranked", describe(reviewer), describe(dev)))
		case !rbok || !dbok:
			findings = append(findings, p.unrankedFinding("effort_bands", reviewer, rbok, dev))
		case rb < db:
			findings = append(findings, fmt.Sprintf("reviewer %s is %s tier at %s band, weaker than implementer %s at %s band", describe(reviewer), tierNames[rt], bandNames[rb], describe(dev), bandNames[db]))
		}
	}
	return findings, notes
}

// unrankedFinding names which side of a reviewer-implementer pair is missing from table,
// with the tier entries its model text could have meant.
func (p Policy) unrankedFinding(table string, reviewer Run, reviewerRanked bool, dev Run) string {
	side, r := "reviewer", reviewer
	if reviewerRanked {
		side, r = "implementer", dev
	}
	msg := fmt.Sprintf("unranked: %s %s is not in %s, so it cannot be shown not weaker than the ranked side (reviewer %s, implementer %s)", side, describe(r), table, describe(reviewer), describe(dev))
	if table == "model_tiers" {
		if guesses := p.modelsLike(r); len(guesses) > 0 {
			msg += fmt.Sprintf("; did you mean %s?", strings.Join(guesses, " or "))
		}
	}
	return msg
}

// modelsLike lists the provider's tier entries whose loose text begins with r's model, so
// a vague "GPT-6" suggests gpt-6-astra and its siblings.
func (p Policy) modelsLike(r Run) []string {
	prefix := r.Provider + ":" + loose(r.Model)
	var out []string
	for _, entries := range [][]string{p.ModelTiers.Frontier, p.ModelTiers.Workhorse, p.ModelTiers.Light} {
		for _, entry := range entries {
			if strings.HasPrefix(loose(entry), prefix) {
				out = append(out, strings.TrimPrefix(entry, r.Provider+":"))
			}
		}
	}
	return out
}

// noEffort reports an empty effort or a placeholder for one: an agent writing
// "unknown" has named no effort (#125). "none" is not here: it is a codex effort.
func noEffort(effort string) bool {
	switch strings.ToLower(strings.TrimSpace(effort)) {
	case "", "unknown", "n/a", "na", "?":
		return true
	}
	return false
}

func describe(r Run) string {
	if r.Effort == "" {
		return fmt.Sprintf("%s (%s)", r.Provider, r.Model)
	}
	return fmt.Sprintf("%s (%s, %s)", r.Provider, r.Model, r.Effort)
}

// loose matches model text to tier entries ignoring case, and treating a run of
// spaces as one hyphen, so "GPT-6 Astra" matches "gpt-6-astra" (ADR 0002 rule 3).
func loose(s string) string { return strings.Join(strings.Fields(strings.ToLower(s)), "-") }

func (p Policy) tier(r Run) (int, bool) {
	want := r.Provider + ":" + loose(r.Model)
	for rank, entries := range [][]string{p.ModelTiers.Light, p.ModelTiers.Workhorse, p.ModelTiers.Frontier} {
		for _, entry := range entries {
			if loose(entry) == want {
				return rank, true
			}
		}
	}
	return 0, false
}

// isModel reports whether s names a tier entry of the provider.
func (p Policy) isModel(provider, s string) bool {
	_, ok := p.tier(Run{Provider: provider, Model: s})
	return ok
}

func (p Policy) band(r Run) (int, bool) {
	var bands Bands
	switch r.Provider {
	case "claude":
		bands = p.EffortBands.Claude
	case "codex":
		bands = p.EffortBands.Codex
	case "gemini":
		bands = p.EffortBands.Gemini
	}
	for rank, values := range [][]string{bands.Low, bands.Mid, bands.High} {
		for _, value := range values {
			if strings.EqualFold(value, strings.TrimSpace(r.Effort)) {
				return rank, true
			}
		}
	}
	return 0, false
}
