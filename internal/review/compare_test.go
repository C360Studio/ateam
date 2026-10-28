package review

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// adrPolicy is the committed ADR 0002 table, typed independently of the repository file.
func adrPolicy() Policy {
	return Policy{
		Version: 1,
		ModelTiers: ModelTiers{
			Frontier:  []string{"claude:opus", "claude:claude-opus-5-5", "claude:fable", "claude:claude-fable-5-1", "codex:gpt-6-astra", "gemini:gemini-4-argon"},
			Workhorse: []string{"claude:sonnet", "claude:claude-sonnet-5-5", "codex:gpt-6.1-sol", "codex:gpt-6-sol"},
			Light:     []string{"claude:haiku", "claude:claude-haiku-4-5", "claude:claude-haiku-4-5-20251001", "codex:gpt-6-luna", "codex:gpt-5.4", "gemini:gemini-3.8-flash", "gemini:gemini-3.1-pro"},
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
		{name: "light below workhorse", implementers: []string{"codex:gpt-6-sol:low"}, reviewer: "claude:haiku:max", finding: "light tier, weaker"},
		{name: "equal tier and band", implementers: []string{"claude:opus:high"}, reviewer: "claude:opus:high"},
		{name: "equal tier, same band by another native name", implementers: []string{"claude:opus:max"}, reviewer: "codex:gpt-6-astra:high"},
		{name: "equal tier, higher band", implementers: []string{"claude:opus:low"}, reviewer: "gemini:gemini-4-argon:medium"},
		{name: "equal tier, lower band", implementers: []string{"claude:opus:high"}, reviewer: "claude:fable:medium", finding: "frontier tier at mid band, weaker than implementer claude (opus, high) at high band"},
		{name: "loose model text", implementers: []string{"gemini:Gemini 4 Argon:HIGH"}, reviewer: "codex:GPT-6  Astra:high"},
		// Owner ruling on #125: an unranked run passes only against another unranked run.
		{name: "unranked reviewer model against a ranked implementer", implementers: []string{"claude:opus:high"}, reviewer: "codex:gpt-7:low", finding: "unranked: reviewer codex (gpt-7, low) is not in model_tiers"},
		{name: "unranked implementer model against a ranked reviewer", implementers: []string{"claude:opus-9:high"}, reviewer: "codex:gpt-6-astra:high", finding: "unranked: implementer claude (opus-9, high) is not in model_tiers"},
		{name: "both models unranked pass with a note", implementers: []string{"claude:opus-9:high"}, reviewer: "codex:gpt-7:low", note: "neither model is in model_tiers"},
		{name: "vague model text suggests the tier entries", implementers: []string{"claude:opus:high"}, reviewer: "codex:GPT-6:high", finding: "did you mean gpt-6-astra or gpt-6.1-sol or gpt-6-sol or gpt-6-luna?"},
		{name: "equal tier, reviewer effort unmapped", implementers: []string{"claude:opus:high"}, reviewer: "claude:opus:turbo", finding: "unranked: reviewer claude (opus, turbo) is not in effort_bands"},
		{name: "equal tier, implementer effort unmapped", implementers: []string{"claude:opus:turbo"}, reviewer: "codex:gpt-6-astra:high", finding: "unranked: implementer claude (opus, turbo) is not in effort_bands"},
		{name: "equal tier, both efforts unmapped pass with a note", implementers: []string{"claude:opus:turbo"}, reviewer: "claude:fable:hyper", note: "neither effort is in effort_bands"},
		// The band only breaks a tier tie, so an unmapped effort at unequal tiers changes nothing.
		{name: "unmapped reviewer effort at a weaker tier", implementers: []string{"claude:opus:high"}, reviewer: "codex:gpt-6-sol:turbo", finding: "workhorse tier, weaker than implementer claude (opus, high)"},
		{name: "unmapped implementer effort above the reviewer tier", implementers: []string{"claude:opus:turbo"}, reviewer: "codex:gpt-6-sol:high", finding: "workhorse tier, weaker than implementer claude (opus, turbo)"},
		{name: "unmapped reviewer effort at a stronger tier", implementers: []string{"claude:sonnet:high"}, reviewer: "codex:gpt-6-astra:turbo"},
		{name: "unmapped implementer effort below the reviewer tier", implementers: []string{"claude:haiku:turbo"}, reviewer: "codex:gpt-6-astra:high"},
		// PR 114 carried "(GPT-6, unknown)" and "(opus, unknown)" and passed (#125).
		{name: "unknown effort is no effort", implementers: []string{"claude:opus:unknown"}, reviewer: "codex:gpt-6-astra:high", finding: "claude (opus, unknown) names no effort"},
		{name: "placeholder efforts are no effort", implementers: []string{"claude:opus:high"}, reviewer: "codex:gpt-6-astra:N/A", finding: "names no effort"},
		{name: "question mark effort is no effort", implementers: []string{"claude:opus:?"}, reviewer: "codex:gpt-6-astra:high", finding: "names no effort"},
		{name: "codex none is a real low effort", implementers: []string{"codex:gpt-6-sol:none"}, reviewer: "claude:sonnet:low"},
		{name: "several implementers, one stronger than the reviewer", implementers: []string{"claude:sonnet:max", "claude:opus:high"}, reviewer: "gemini:gemini-4-argon:medium", finding: "weaker than implementer claude (opus, high)"},
		{name: "several implementers, all covered", implementers: []string{"claude:sonnet:max", "claude:opus:medium"}, reviewer: "gemini:gemini-4-argon:medium"},
		{name: "reviewer effort missing", implementers: []string{"claude:opus:high"}, reviewer: "codex:gpt-6-astra:", finding: "codex (gpt-6-astra) names no effort"},
		{name: "model in the effort slot", implementers: []string{"claude:sonnet:opus"}, reviewer: "claude:sonnet:low", finding: `claude (sonnet, opus) names model "opus" where the effort goes`},
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
	for _, bad := range []string{"", "claude", "claude:opus", "openai:gpt-6:high", "claude: :high", "Claude:opus:high", "claude:a:b:high"} {
		if _, err := ParseRunSpec(bad); err == nil {
			t.Errorf("ParseRunSpec(%q) accepted", bad)
		}
	}
}

// spec: openspec/specs/model-review/spec.md#Requirement: Model tiers and effort bands policy
// spec: openspec/specs/model-review/spec.md#Requirement: Not-weaker reviewer by tier and effort band
func TestCommittedPolicyTiers(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", ".agents", "review-policy.json"))
	if err != nil {
		t.Fatalf("read committed policy: %v", err)
	}
	p, err := ParsePolicy(data)
	if err != nil {
		t.Fatalf("parse committed policy: %v", err)
	}
	want := adrPolicy()
	if !reflect.DeepEqual(p.ModelTiers, want.ModelTiers) {
		t.Fatalf("committed model_tiers = %+v, want ADR 0002 table %+v", p.ModelTiers, want.ModelTiers)
	}
	if !reflect.DeepEqual(p.EffortBands, want.EffortBands) {
		t.Fatalf("committed effort_bands = %+v, want ADR 0002 table %+v", p.EffortBands, want.EffortBands)
	}
	if !p.CrossProvider {
		t.Fatal("committed policy cross_provider = false, want true")
	}

	for _, pair := range [][2]string{
		{"claude:claude-opus-5-5:high", "claude:opus:high"},
		{"claude:claude-fable-5-1:high", "claude:fable:high"},
		{"claude:claude-sonnet-5-5:high", "claude:sonnet:high"},
		{"claude:claude-haiku-4-5:high", "claude:haiku:high"},
		{"claude:claude-haiku-4-5-20251001:high", "claude:haiku:high"},
		{"gemini:gemini-4-argon:high", "claude:claude-opus-5-5:high"},
		{"codex:gpt-6-luna:high", "claude:haiku:high"},
	} {
		a, b := run(t, pair[0]), run(t, pair[1])
		if f, _ := p.Compare([]Run{a}, b); len(f) != 0 {
			t.Fatalf("%s reviewed by %s: unexpected findings %v", pair[0], pair[1], f)
		}
		if f, _ := p.Compare([]Run{b}, a); len(f) != 0 {
			t.Fatalf("%s reviewed by %s: unexpected findings %v", pair[1], pair[0], f)
		}
	}

	for _, tc := range []struct {
		dev, rev, want string
	}{
		{"gemini:gemini-4-argon:low", "codex:gpt-6-sol:max", "workhorse tier, weaker than implementer gemini (gemini-4-argon, low) at frontier tier"},
		{"claude:opus:low", "claude:claude-sonnet-5-5:max", "workhorse tier, weaker"},
		{"claude:sonnet:low", "claude:claude-haiku-4-5:max", "light tier, weaker"},
		{"claude:sonnet:low", "claude:claude-haiku-4-5-20251001:max", "light tier, weaker"},
		{"claude:sonnet:low", "codex:gpt-6-luna:max", "light tier, weaker"},
		{"claude:opus:high", "claude:claude-opus-4-7:high", "unranked: reviewer claude (claude-opus-4-7, high) is not in model_tiers"},
		{"claude:sonnet:high", "claude:claude-sonnet-4-6:high", "unranked: reviewer claude (claude-sonnet-4-6, high) is not in model_tiers"},
	} {
		findings, _ := p.Compare([]Run{run(t, tc.dev)}, run(t, tc.rev))
		if got := strings.Join(findings, "\n"); !strings.Contains(got, tc.want) {
			t.Fatalf("%s reviewed by %s: findings %q, want substring %q", tc.dev, tc.rev, got, tc.want)
		}
	}
}
