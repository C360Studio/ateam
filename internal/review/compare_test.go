package review

import (
	"strings"
	"testing"
)

// adrPolicy is the committed ADR 0002 table, typed independently of the repository file.
func adrPolicy() Policy {
	return Policy{
		Version: 1,
		ModelTiers: ModelTiers{
			Frontier:  []string{"claude:opus", "claude:fable", "codex:gpt-6-astra"},
			Workhorse: []string{"claude:sonnet", "codex:gpt-6.1-sol", "codex:gpt-6-sol", "gemini:gemini-4-argon"},
			Light:     []string{"claude:haiku", "codex:gpt-5.4", "gemini:gemini-3.8-flash", "gemini:gemini-3.1-pro"},
		},
		EffortBands: EffortBands{
			Claude: Bands{Low: []string{"low"}, Mid: []string{"medium"}, High: []string{"high", "xhigh", "max"}},
			Codex:  Bands{Low: []string{"none", "minimal", "low"}, Mid: []string{"medium"}, High: []string{"high", "xhigh", "max"}},
			Gemini: Bands{Low: []string{"minimal", "low"}, Mid: []string{"medium"}, High: []string{"high"}},
		},
	}
}

func run(t *testing.T, spec string) Run {
	t.Helper()
	r, err := ParseRunSpec(spec)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// spec: openspec/specs/model-review/spec.md#Requirement: Not-weaker reviewer by tier and effort band
func TestCompare(t *testing.T) {
	p := adrPolicy()
	for _, tc := range []struct {
		name         string
		implementers []string
		reviewer     string
		finding      string // empty: pass
		note         string
	}{
		{name: "stronger tier at a lower band", implementers: []string{"claude:sonnet:max"}, reviewer: "codex:gpt-6-astra:low"},
		{name: "weaker tier at a higher band", implementers: []string{"claude:opus:low"}, reviewer: "codex:gpt-6-sol:max", finding: "workhorse tier, weaker than implementer claude (opus, low) at frontier tier"},
		{name: "light below workhorse", implementers: []string{"gemini:gemini-4-argon:low"}, reviewer: "claude:haiku:max", finding: "light tier, weaker"},
		{name: "equal tier and band", implementers: []string{"claude:opus:high"}, reviewer: "claude:opus:high"},
		{name: "equal tier, same band by another native name", implementers: []string{"claude:opus:max"}, reviewer: "codex:gpt-6-astra:high"},
		{name: "equal tier, higher band", implementers: []string{"claude:sonnet:low"}, reviewer: "gemini:gemini-4-argon:medium"},
		{name: "equal tier, lower band", implementers: []string{"claude:opus:high"}, reviewer: "claude:fable:medium", finding: "frontier tier at mid band, weaker than implementer claude (opus, high) at high band"},
		{name: "loose model text", implementers: []string{"gemini:Gemini 4 Argon:HIGH"}, reviewer: "codex:GPT-6  Astra:low"},
		{name: "unranked model passes with a note", implementers: []string{"claude:opus:high"}, reviewer: "codex:gpt-7:low", note: "unranked: codex (gpt-7, low)"},
		{name: "unranked effort passes with a note", implementers: []string{"claude:opus:high"}, reviewer: "claude:opus:turbo", note: "an effort missing from effort_bands"},
		{name: "several implementers, one stronger than the reviewer", implementers: []string{"claude:haiku:max", "claude:sonnet:high"}, reviewer: "gemini:gemini-4-argon:medium", finding: "weaker than implementer claude (sonnet, high)"},
		{name: "several implementers, all covered", implementers: []string{"claude:haiku:max", "claude:sonnet:medium"}, reviewer: "gemini:gemini-4-argon:medium"},
		{name: "reviewer effort missing", implementers: []string{"claude:opus:high"}, reviewer: "codex:gpt-6-astra:", finding: "codex (gpt-6-astra) names no effort"},
		{name: "implementer effort missing", implementers: []string{"claude:opus:"}, reviewer: "codex:gpt-6-astra:high", finding: "claude (opus) names no effort"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var devs []Run
			for _, s := range tc.implementers {
				devs = append(devs, run(t, s))
			}
			findings, notes := p.Compare(devs, run(t, tc.reviewer))
			got := strings.Join(findings, "\n")
			if tc.finding == "" && got != "" {
				t.Fatalf("findings %q, want a pass", got)
			}
			if tc.finding != "" && !strings.Contains(got, tc.finding) {
				t.Fatalf("findings %q, want one containing %q", got, tc.finding)
			}
			if n := strings.Join(notes, "\n"); !strings.Contains(n, tc.note) || (tc.note == "" && n != "") {
				t.Fatalf("notes %q, want %q", n, tc.note)
			}
		})
	}
}

// spec: openspec/specs/model-review/spec.md#Requirement: Not-weaker reviewer by tier and effort band
func TestParseRunSpec(t *testing.T) {
	if r := run(t, "codex:gpt-6.1-sol:xhigh"); r != (Run{Provider: "codex", Model: "gpt-6.1-sol", Effort: "xhigh"}) {
		t.Fatalf("parsed %+v", r)
	}
	for _, bad := range []string{"", "claude", "claude:opus", "openai:gpt-6:high", "claude: :high", "Claude:opus:high"} {
		if _, err := ParseRunSpec(bad); err == nil {
			t.Errorf("ParseRunSpec(%q) accepted", bad)
		}
	}
}
