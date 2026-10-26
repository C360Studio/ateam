package review

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
)

// syntheticRecord is a minimal REVIEW-002 implementation record; the fixture is invented.
func syntheticRecord(verdict, reviewer string) string {
	return "## Verdict\n\n- **Stage**: implementation\n- **Verdict**: " + verdict + "\n- **Reviewer**: " + reviewer + "\n- **Date**: 2026-10-26\n- **Round**: 1\n\n" +
		"## Manifest\n\n```text\n" + strings.Repeat("a", 64) + "  internal/example/example.go\n```\n\n" +
		"## Findings\n\n| ID | Severity | Location | Rule | Finding | Status |\n|----|----------|----------|------|---------|--------|\n| I-1 | ADVISORY | example.go:1 | 3 | Example. | closed |\n"
}

func approvalFor(stage, record string) []byte {
	return []byte(fmt.Sprintf("| Stage | Approver | Date | Review digest |\n|-------|----------|------|---------------|\n| %s | person@example.com | 2026-10-26 | %x |\n", stage, sha256.Sum256([]byte(record))))
}

// spec: openspec/specs/model-review/spec.md#Requirement: Reviewer run from a review record
func TestCheckImplementationRecord(t *testing.T) {
	pass := syntheticRecord("PASS", "claude (opus, high), agent type example-reviewer")
	outside := strings.Replace(pass, "- **Verdict**: PASS\n", "", 1) + "\n- **Verdict**: PASS\n"
	for _, tc := range []struct {
		name     string
		record   string
		approval []byte
		wantRun  string // "" means no run read
		want     string // substring of the only finding; "" means none
	}{
		{"approved pass", pass, approvalFor("implementation", pass), "claude:opus:high", ""},
		{"fail verdict", syntheticRecord("FAIL", "claude (opus, high)"), approvalFor("implementation", syntheticRecord("FAIL", "claude (opus, high)")), "claude:opus:high", `Verdict is "FAIL"`},
		{"lower-case pass", syntheticRecord("pass", "claude (opus, high)"), approvalFor("implementation", syntheticRecord("pass", "claude (opus, high)")), "claude:opus:high", `Verdict is "pass"`},
		{"missing approval", pass, nil, "claude:opus:high", "no approval.md"},
		{"empty approval", pass, []byte{}, "claude:opus:high", "no implementation row"},
		{"approval of another record", pass, approvalFor("implementation", pass+"\n"), "claude:opus:high", "no implementation row"},
		{"approval of the spec stage", pass, approvalFor("specification", pass), "claude:opus:high", "no implementation row"},
		{"reviewer without an agent", syntheticRecord("PASS", "example-reviewer (claude-opus-5-5, agent type example-reviewer)"), approvalFor("implementation", syntheticRecord("PASS", "example-reviewer (claude-opus-5-5, agent type example-reviewer)")), "", "must start with <agent> (<model>, <effort>)"},
		{"reviewer with two runs", syntheticRecord("PASS", "claude (opus, high, sonnet, high)"), approvalFor("implementation", syntheticRecord("PASS", "claude (opus, high, sonnet, high)")), "", "must start with"},
		{"spec stage record", strings.Replace(pass, "implementation", "specification", 1), approvalFor("implementation", strings.Replace(pass, "implementation", "specification", 1)), "claude:opus:high", `Stage is "specification"`},
		{"second verdict field", strings.Replace(pass, "- **Date**", "- **Verdict**: FAIL\n- **Date**", 1), approvalFor("implementation", strings.Replace(pass, "- **Date**", "- **Verdict**: FAIL\n- **Date**", 1)), "claude:opus:high", "2 Verdict fields"},
		{"verdict only in the findings section", outside, approvalFor("implementation", outside), "claude:opus:high", "0 Verdict fields"},
		{"no verdict section", strings.Replace(pass, "## Verdict", "## Summary", 1), approvalFor("implementation", pass), "", "0 \"## Verdict\" sections"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			run, ok, findings := CheckImplementationRecord([]byte(tc.record), tc.approval)
			got := ""
			if ok {
				got = run.Provider + ":" + run.Model + ":" + run.Effort
			}
			if got != tc.wantRun {
				t.Errorf("run %q, want %q", got, tc.wantRun)
			}
			if tc.want == "" {
				if len(findings) != 0 {
					t.Errorf("findings %q, want none", findings)
				}
				return
			}
			if len(findings) != 1 || !strings.Contains(findings[0], tc.want) {
				t.Errorf("findings %q, want one containing %q", findings, tc.want)
			}
		})
	}
}
