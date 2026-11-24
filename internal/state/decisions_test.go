package state

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeChange writes an open change under a fresh openspec root; an empty body skips that file.
func writeChange(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	all := map[string]string{
		"openspec/config.yaml":         "schema: spec-driven\n",
		"openspec/changes/c1/tasks.md": "## 1. Tasks\n\n- [ ] 1.1 Do it\n",
	}
	for rel, body := range files {
		all[rel] = body
	}
	for rel, body := range all {
		if body == "" {
			continue
		}
		abs := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// spec: openspec/specs/harness/spec.md#Requirement: Proposal decisions section and spec text length
func TestProposalDecisionsSection(t *testing.T) {
	// Fixtures are not strict-valid; keep a host openspec on PATH out of this check (#205).
	t.Setenv("PATH", t.TempDir())
	const (
		design    = "# Design\n"
		missing   = "openspec/changes/c1/proposal.md: no \"## Decisions you are making by approving\" section with a bullet"
		fewWords  = "has fewer than 5 plain words"
		heading   = "## Decisions you are making by approving\n\n"
		goodValue = "- Users see a queue limit\n" // exactly 5 plain words
	)
	for _, tc := range []struct {
		name, proposal, design string
		code                   int
		want                   []string // substrings the output must contain
		noWarn                 bool     // the output must hold no warning at all
	}{
		{"design track, no section", "# P\n\n## Why\n", design, 1, []string{missing + "; list in plain words what the owner decides by approving (required on the design track)"}, false},
		{"design track, heading without bullet", "# P\n\n" + heading + "Prose only.\n", design, 1, []string{missing}, false},
		{"design track, heading only inside a code fence", "# P\n\n```\n" + heading + goodValue + "```\n", design, 1, []string{missing}, false},
		{"quick track, no section", "# P\n\n## Why\n", "", 0, []string{"warning: " + missing}, false},
		{"design track, five plain words", "# P\n\n" + heading + goodValue, design, 0, nil, true},
		{"heading in another case, star bullet", "# P\n\n## decisions YOU are making by approving\n\n* Users see a queue limit\n", design, 0, nil, true},
		{"code and numbers only", "# P\n\n" + heading + "- `MaxQueue` = 64\n", design, 0, []string{"warning: openspec/changes/c1/proposal.md: decision \"`MaxQueue` = 64\" " + fewWords}, false},
		{"four plain words", "# P\n\n" + heading + "- Users see queue limit `MaxQueue` 64\n", "", 0, []string{fewWords}, false},
		{"bullets after the next heading do not count", "# P\n\n" + heading + "## Why\n\n" + goodValue, design, 1, []string{missing}, false},
	} {
		dir := writeChange(t, map[string]string{
			"openspec/changes/c1/proposal.md": tc.proposal,
			"openspec/changes/c1/design.md":   tc.design,
		})
		out, code, err := CheckOpenSpec(context.Background(), dir)
		if code != tc.code || (code == 0) != (err == nil) {
			t.Fatalf("%s: code=%d err=%v, want code %d; output:\n%s", tc.name, code, err, tc.code, out)
		}
		for _, w := range tc.want {
			if !strings.Contains(out, w) {
				t.Errorf("%s: output lacks %q:\n%s", tc.name, w, out)
			}
		}
		if tc.noWarn && (strings.Contains(out, "warning:") || strings.Contains(out, "Decisions")) {
			t.Errorf("%s: unexpected warning:\n%s", tc.name, out)
		}
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Proposal decisions section and spec text length
func TestDeltaSpecLengthWarnings(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	const proposal = "# P\n\n## Decisions you are making by approving\n\n- Nothing changes for any user here.\n"
	// body returns a Requirement body of exactly n characters.
	body := func(n int) string { return "The tool SHALL " + strings.Repeat("x", n-len("The tool SHALL ")) }
	// step returns a scenario step of exactly n words after the bold WHEN marker.
	step := func(n int) string { return "- **WHEN** " + strings.TrimSpace(strings.Repeat("w ", n-1)) }
	delta := func(section, reqBody, scenarioStep string) string {
		return "## " + section + " Requirements\n\n### Requirement: Long one\n\n" + reqBody + "\n\n#### Scenario: S\n\n" + scenarioStep + "\n- **THEN** ok\n"
	}
	const file = "openspec/changes/c1/specs/harness/spec.md"
	for _, tc := range []struct {
		name, spec string
		want       string // "" means no warning
	}{
		{"ADDED at 500 characters", delta("ADDED", body(500), step(60)), ""},
		{"ADDED at 501 characters", delta("ADDED", body(501), step(60)), "warning: " + file + ": Requirement \"Long one\" is 501 characters, over the limit of 500"},
		{"MODIFIED at 501 characters", delta("MODIFIED", body(501), step(60)), "is 501 characters, over the limit of 500"},
		{"61-word step", delta("ADDED", body(100), step(61)), "warning: " + file + ": Requirement \"Long one\" scenario \"S\" has a 61-word step, over the limit of 60"},
		{"REMOVED section is not measured", delta("REMOVED", body(501), step(61)), ""},
	} {
		dir := writeChange(t, map[string]string{"openspec/changes/c1/proposal.md": proposal, file: tc.spec})
		out, code, err := CheckOpenSpec(context.Background(), dir)
		if code != 0 || err != nil {
			t.Fatalf("%s: code=%d err=%v; length must warn, never fail; output:\n%s", tc.name, code, err, out)
		}
		if tc.want == "" {
			if strings.Contains(out, "warning:") {
				t.Errorf("%s: unexpected warning:\n%s", tc.name, out)
			}
			continue
		}
		if !strings.Contains(out, tc.want) || strings.Count(out, "warning:") != 1 {
			t.Errorf("%s: want exactly one warning containing %q, got:\n%s", tc.name, tc.want, out)
		}
	}
}
