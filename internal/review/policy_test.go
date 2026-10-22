package review

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func testPolicy() Policy {
	return Policy{Version: 1, Models: []Model{{ID: "small", Provider: "codex"}, {ID: "medium", Provider: "codex", StrongerThan: []string{"small"}}, {ID: "large", Provider: "claude", StrongerThan: []string{"medium"}}}, Routes: Routes{Developer: Route{Model: "small", Effort: "high"}, Reviewer: Route{Model: "medium", Effort: "low"}}, Owners: []string{"owner"}, ReviewAuthors: []string{"reviewer-login"}, RequiredChecks: []string{"test"}}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// Rank is independent of effort/provider: equal or weaker reviewers must fail.
// spec: openspec/specs/model-review/spec.md#Requirement: Stronger independent reviewer
func TestEligible(t *testing.T) {
	cases := []struct {
		name, model, provider string
		cross, want           bool
	}{
		{"same-provider-stronger", "medium", "codex", false, true},
		{"transitive-stronger", "large", "claude", false, true},
		{"equal-higher-effort", "small", "codex", false, false},
		{"unknown", "unknown", "claude", false, false},
		{"inherit", "inherit", "claude", false, false},
		{"spoof-provider", "medium", "claude", false, false},
		{"cross-blocks-same", "medium", "codex", true, false},
		{"cross-stronger", "large", "claude", true, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := testPolicy()
			p.CrossProvider = tc.cross
			err := Eligible(p, []Run{{ID: "dev", Model: "small", Provider: "codex", Effort: "low"}}, Run{ID: "review", Model: tc.model, Provider: tc.provider, Effort: "ultra"})
			if (err == nil) != tc.want {
				t.Fatalf("eligibility got %v, want allowed=%v", err, tc.want)
			}
		})
	}
}

// spec: openspec/specs/model-review/spec.md#Requirement: Stronger independent reviewer
// spec: openspec/specs/model-review/spec.md#Requirement: Explicit setup and portable routing
// spec: openspec/specs/model-review/spec.md#Requirement: Explicit owner exception at the model ceiling
func TestValidatePolicy(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Policy)
	}{
		{"version", func(p *Policy) { p.Version = 2 }},
		{"empty-models", func(p *Policy) { p.Models = nil }},
		{"too-many", func(p *Policy) { p.Models = make([]Model, 257) }},
		{"duplicate", func(p *Policy) { p.Models = append(p.Models, p.Models[0]) }},
		{"unknown-edge", func(p *Policy) { p.Models[1].StrongerThan = []string{"unknown"} }},
		{"duplicate-edge", func(p *Policy) { p.Models[1].StrongerThan = []string{"small", "small"} }},
		{"cycle", func(p *Policy) { p.Models[0].StrongerThan = []string{"large"} }},
		{"self-cycle", func(p *Policy) { p.Models[0].StrongerThan = []string{"small"} }},
		{"unresolved", func(p *Policy) { p.Models[0].ID = "inherit" }},
		{"unknown-route", func(p *Policy) { p.Routes.Reviewer.Model = "unknown" }},
		{"missing-effort", func(p *Policy) { p.Routes.Developer.Effort = "" }},
		{"owners", func(p *Policy) { p.Owners = nil }},
		{"review-authors", func(p *Policy) { p.ReviewAuthors = nil }},
		{"duplicate-owners", func(p *Policy) { p.Owners = []string{"owner", "owner"} }},
		{"checks", func(p *Policy) { p.RequiredChecks = nil }},
		{"bad-check", func(p *Policy) { p.RequiredChecks = []string{" test "} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := testPolicy()
			tc.mutate(&p)
			if _, err := ParsePolicy(mustJSON(t, p)); err == nil {
				t.Fatal("invalid policy accepted")
			}
		})
	}
	p := testPolicy()
	p.Routes.Reviewer = p.Routes.Developer
	if _, err := ParsePolicy(mustJSON(t, p)); err != nil {
		t.Fatalf("ceiling policy must parse: %v", err)
	}
	if err := p.ValidateRouting(); err == nil {
		t.Fatal("ceiling route approved without exception")
	}
	p = testPolicy()
	if err := p.ValidateRouting(); err != nil {
		t.Fatal(err)
	}
}

// spec: openspec/specs/model-review/spec.md#Requirement: Portable enforcement and honest trust boundaries
// spec: openspec/specs/model-review/spec.md#Requirement: Current and independently attributed review evidence
func TestStrictJSON(t *testing.T) {
	valid := string(mustJSON(t, testPolicy()))
	// Nil slices encode as null, deliberately forbidden at trust boundaries.
	valid = strings.ReplaceAll(valid, ":null", ":[]")
	if _, err := ParsePolicy([]byte(valid)); err != nil {
		t.Fatal(err)
	}
	if _, err := ParsePolicy([]byte(valid + strings.Repeat(" ", MaxJSONBytes-len(valid)))); err != nil {
		t.Fatalf("exact byte limit rejected: %v", err)
	}
	cases := []string{"", "null", valid + " {}", strings.Replace(valid, `"version":1`, `"version":1,"version":1`, 1), strings.Replace(valid, `"version":1`, `"Version":1`, 1), strings.Replace(valid, `"version":1`, `"version":1,"owner":"fake"`, 1), strings.Replace(valid, `"version":1`, `"version":"1"`, 1), strings.Replace(valid, `"models":[`, `"models":null,"x":[`, 1), strings.Repeat(" ", MaxJSONBytes+1)}
	for i, input := range cases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			if _, err := ParsePolicy([]byte(input)); err == nil {
				t.Fatal("malformed policy accepted")
			}
		})
	}
	if PolicyDigest([]byte("abc")) != "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad" {
		t.Fatal("policy SHA-256 mismatch")
	}
	if PolicyDigest([]byte(valid)) == PolicyDigest([]byte(valid+"\n")) {
		t.Fatal("policy bytes not bound exactly")
	}
}

// A generated linear order has a simple integer oracle, independent of graph traversal.
// spec: openspec/specs/model-review/spec.md#Requirement: Stronger independent reviewer
func TestGeneratedRankComparisons(t *testing.T) {
	checks := 0
	for count := 2; count <= 12; count++ {
		p := testPolicy()
		p.Models = nil
		for i := 0; i < count; i++ {
			m := Model{ID: fmt.Sprint(i), Provider: "codex"}
			if i > 0 {
				m.StrongerThan = []string{fmt.Sprint(i - 1)}
			}
			p.Models = append(p.Models, m)
		}
		p.Routes = Routes{Developer: Route{Model: "0", Effort: "low"}, Reviewer: Route{Model: "1", Effort: "high"}}
		for dev := 0; dev < count; dev++ {
			for reviewer := 0; reviewer < count; reviewer++ {
				err := Eligible(p, []Run{{ID: "dev", Model: fmt.Sprint(dev), Provider: "codex", Effort: "ultra"}}, Run{ID: "reviewer", Model: fmt.Sprint(reviewer), Provider: "codex", Effort: "low"})
				if (err == nil) != (reviewer > dev) {
					t.Fatalf("generated rank developer=%d reviewer=%d: %v", dev, reviewer, err)
				}
				checks++
			}
		}
	}
	if checks != 649 {
		t.Fatalf("assertion activation: got %d checks", checks)
	}
	t.Logf("completed %d comparisons; equal, adjacent, transitive and ceiling boundaries reached", checks)
}

// spec: openspec/specs/model-review/spec.md#Requirement: Explicit setup and portable routing
func FuzzPolicyJSON(f *testing.F) {
	valid, err := json.Marshal(testPolicy())
	if err != nil {
		f.Fatal(err)
	}
	f.Add(valid)
	f.Add([]byte(`{"version":1,"models":[]}`))
	f.Add([]byte(`{"version":1,"version":1}`))
	f.Add([]byte(`{"models":[{"id":"inherit"}]}`))
	f.Add([]byte(tiersPolicy))
	f.Fuzz(func(t *testing.T, data []byte) {
		p, err := ParsePolicy(data)
		if err != nil {
			return
		}
		if err := ValidatePolicy(p); err != nil {
			t.Fatalf("parser returned invalid policy: %v", err)
		}
		encoded := mustJSON(t, p)
		if _, err := ParsePolicy(encoded); err != nil {
			t.Fatalf("accepted policy does not round trip: %v", err)
		}
	})
}

// spec: openspec/specs/model-review/spec.md#Requirement: Explicit setup and portable routing
func TestPolicyModelLimit(t *testing.T) {
	p := testPolicy()
	p.Models = nil
	for i := 0; i < 256; i++ {
		p.Models = append(p.Models, Model{ID: fmt.Sprint(i), Provider: "codex"})
	}
	p.Routes = Routes{Developer: Route{Model: "0", Effort: "low"}, Reviewer: Route{Model: "255", Effort: "high"}}
	if err := ValidatePolicy(p); err != nil {
		t.Fatalf("exact model limit rejected: %v", err)
	}
	p.Models = append(p.Models, Model{ID: "256", Provider: "codex"})
	if err := ValidatePolicy(p); err == nil {
		t.Fatal("excess model accepted")
	}
}

const tiersPolicy = `{"version":1,
"model_tiers":{"frontier":["claude:opus","codex:gpt-6-astra"],"workhorse":["gemini:gemini-4-argon"],"light":["claude:haiku"]},
"effort_bands":{"claude":{"low":["low"],"mid":["medium"],"high":["high","xhigh","max"]},
"codex":{"low":["none","minimal","low"],"mid":["medium"],"high":["high","xhigh","max"]},
"gemini":{"low":["low"],"mid":["medium"],"high":["high"]}},
"cross_provider":false}`

// A tiers-only policy (ADR 0002) parses without routes; malformed tiers are rejected.
// spec: openspec/specs/model-review/spec.md#Requirement: Model tiers and effort bands policy
func TestValidateTiersPolicy(t *testing.T) {
	p, err := ParsePolicy([]byte(tiersPolicy))
	if err != nil {
		t.Fatalf("tiers-only policy must parse: %v", err)
	}
	if p.HasRoutes() || !p.HasTiers() {
		t.Fatalf("tiers-only policy: HasRoutes=%v HasTiers=%v, want false true", p.HasRoutes(), p.HasTiers())
	}
	for name, mutate := range map[string]func(string) string{
		"no provider prefix":   func(s string) string { return strings.Replace(s, `"claude:haiku"`, `"haiku"`, 1) },
		"unsupported provider": func(s string) string { return strings.Replace(s, `"claude:haiku"`, `"mistral:small"`, 1) },
		"empty model":          func(s string) string { return strings.Replace(s, `"claude:haiku"`, `"claude:"`, 1) },
		"model in two tiers":   func(s string) string { return strings.Replace(s, `"claude:haiku"`, `"claude:OPUS"`, 1) },
		"repeated effort value": func(s string) string {
			return strings.Replace(s, `"mid":["medium"],"high":["high"]`, `"mid":["medium"],"high":["Medium"]`, 1)
		},
		"provider without bands": func(s string) string {
			return strings.Replace(s, `"gemini":{"low":["low"],"mid":["medium"],"high":["high"]}`, `"gemini":{}`, 1)
		},
		"unknown band": func(s string) string {
			return strings.Replace(s, `"mid":["medium"],"high":["high"]}}`, `"mid":["medium"],"top":["high"]}}`, 1)
		},
		"owners without models": func(s string) string {
			return strings.Replace(s, `"cross_provider":false`, `"cross_provider":false,"owners":["o"]`, 1)
		},
		"neither tiers nor models": func(string) string {
			return `{"version":1,"cross_provider":false}`
		},
	} {
		t.Run(name, func(t *testing.T) {
			data := mutate(tiersPolicy)
			if data == tiersPolicy {
				t.Fatal("mutation did not apply")
			}
			if _, err := ParsePolicy([]byte(data)); err == nil {
				t.Fatalf("expected rejection for %s", name)
			}
		})
	}
	// The dormant path still parses alongside tiers.
	full := testPolicy()
	full.ModelTiers = ModelTiers{Frontier: []string{"claude:large"}}
	full.EffortBands = EffortBands{Claude: Bands{High: []string{"max"}}, Codex: Bands{High: []string{"max"}}, Gemini: Bands{High: []string{"high"}}}
	if _, err := ParsePolicy(mustJSON(t, full)); err != nil {
		t.Fatalf("policy with models and tiers must parse: %v", err)
	}
	// Bands supplied without tiers are still validated (Codex review of b44186d).
	orphan := testPolicy()
	orphan.EffortBands = EffortBands{Claude: Bands{Low: []string{"high"}, High: []string{"high"}}, Codex: Bands{High: []string{"max"}}, Gemini: Bands{High: []string{"high"}}}
	if _, err := ParsePolicy(mustJSON(t, orphan)); err == nil {
		t.Fatal("contradictory effort bands on a policy without tiers must be rejected")
	}
}
