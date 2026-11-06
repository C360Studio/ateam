package review

import (
	"reflect"
	"strings"
	"testing"
)

// spec: openspec/changes/review-launch-295/specs/model-review/spec.md#Requirement: Review launch policy
func TestReviewLaunchPolicy(t *testing.T) {
	tiers := `"model_tiers":{"frontier":["claude:opus"]},"effort_bands":{"claude":{"high":["high"]},"codex":{"high":["high"]},"gemini":{"high":["high"]}}`
	for _, tc := range []struct {
		name, field, want string
		ok                bool
	}{
		{"absent", ``, "", true},
		{"manual", `"review_launch":"manual",`, LaunchManual, true},
		{"agent", `"review_launch":"agent",`, LaunchAgent, true},
		{"invalid", `"review_launch":"auto",`, "", false},
		{"case", `"review_launch":"Agent",`, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, err := ParsePolicy([]byte(`{"version":1,` + tc.field + tiers + `}`))
			if !tc.ok {
				if err == nil || !strings.Contains(err.Error(), `review_launch`) || !strings.Contains(err.Error(), `"manual" or "agent"`) {
					t.Fatalf("invalid review_launch: err %v, want the two accepted values named", err)
				}
				return
			}
			if err != nil || p.ReviewLaunch != tc.want {
				t.Fatalf("review_launch = %q, err %v; want %q", p.ReviewLaunch, err, tc.want)
			}
		})
	}
}

// spec: openspec/changes/review-launch-295/specs/model-review/spec.md#Requirement: Review request spec
func TestEligibleReviewers(t *testing.T) {
	c := func(agent, model, effort, band string) Candidate {
		return Candidate{Agent: agent, Model: model, Effort: effort, Band: band}
	}
	for _, tc := range []struct {
		name  string
		cross bool
		impl  []string
		want  []Candidate
	}{
		{"frontier-cross", true, []string{"claude:opus:high"},
			[]Candidate{c("codex", "gpt-6-astra", "high", "high"), c("gemini", "gemini-4-argon", "high", "high")}},
		{"workhorse-cross", true, []string{"claude:sonnet:medium"},
			[]Candidate{c("codex", "gpt-6-astra", "medium", "mid"), c("gemini", "gemini-4-argon", "medium", "mid"), c("codex", "gpt-6.1-sol", "medium", "mid"), c("codex", "gpt-6-sol", "medium", "mid")}},
		{"two-implementers", true, []string{"claude:sonnet:low", "codex:gpt-6-astra:medium"},
			[]Candidate{c("gemini", "gemini-4-argon", "medium", "mid")}},
		{"same-provider-allowed", false, []string{"codex:gpt-6-astra:xhigh"},
			[]Candidate{c("claude", "opus", "high", "high"), c("claude", "claude-opus-5-5", "high", "high"), c("claude", "fable", "high", "high"), c("claude", "claude-fable-5-1", "high", "high"), c("codex", "gpt-6-astra", "high", "high"), c("gemini", "gemini-4-argon", "high", "high")}},
		{"unranked-implementer", true, []string{"claude:mystery:high"}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := adrPolicy()
			p.CrossProvider = tc.cross
			var impl []Run
			for _, s := range tc.impl {
				impl = append(impl, run(t, s))
			}
			got := p.EligibleReviewers(impl)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("EligibleReviewers(%v) =\n%v\nwant\n%v", tc.impl, got, tc.want)
			}
			// Independent oracle: every listed run passes Compare against the implementers.
			for _, cand := range got {
				if findings, _ := p.Compare(impl, Run{Provider: cand.Agent, Model: cand.Model, Effort: cand.Effort}); len(findings) > 0 {
					t.Errorf("listed %v fails Compare: %v", cand, findings)
				}
			}
		})
	}
}
