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

type recordCase struct {
	name     string
	record   string
	approval []byte
	wantRun  string // "" means no run read
	want     string // substring of the only finding; "" means none
}

// spec: openspec/specs/model-review/spec.md#Requirement: Reviewer run from a review record
func TestCheckImplementationRecord(t *testing.T) {
	pass := syntheticRecord("PASS", "claude (opus, high), agent type example-reviewer")
	outside := strings.Replace(pass, "- **Verdict**: PASS\n", "", 1) + "\n- **Verdict**: PASS\n"
	var cases []recordCase
	for _, tc := range []recordCase{
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
		cases = append(cases, tc)
	}
	// Fenced examples never count, with either fence style (Codex review on #175).
	for _, f := range []string{"```", "~~~"} {
		example := pass + "\n" + f + "markdown\n## Verdict\n\n- **Stage**: implementation\n- **Verdict**: FAIL\n- **Reviewer**: gemini (gemini-3.8-flash, low)\n" + f + "\n"
		wrapped := f + "markdown\n" + pass + f + "\n"
		fencedApproval := append(append([]byte(f+"text\n"), approvalFor("implementation", pass)...), []byte(f+"\n")...)
		cases = append(cases,
			recordCase{"fenced example " + f, example, approvalFor("implementation", example), "claude:opus:high", ""},
			recordCase{"record wrapped in " + f, wrapped, approvalFor("implementation", wrapped), "", "0 \"## Verdict\" sections"},
			recordCase{"approval row only in " + f, pass, fencedApproval, "claude:opus:high", "no implementation row"},
		)
	}
	// A blank duplicate before or after a populated field is still a second field.
	for _, field := range []string{"Stage", "Verdict", "Reviewer"} {
		populated := "- **" + field + "**: "
		for _, blank := range []string{"- **" + field + "**:", "- **" + field + "**: "} {
			before := strings.Replace(pass, populated, blank+"\n"+populated, 1)
			i := strings.Index(pass, populated)
			eol := i + strings.Index(pass[i:], "\n")
			after := pass[:eol+1] + blank + "\n" + pass[eol+1:]
			wantRun := "claude:opus:high"
			if field == "Reviewer" {
				wantRun = ""
			}
			cases = append(cases,
				recordCase{"blank " + field + " before " + fmt.Sprintf("%q", blank), before, approvalFor("implementation", before), wantRun, "2 " + field + " fields"},
				recordCase{"blank " + field + " after " + fmt.Sprintf("%q", blank), after, approvalFor("implementation", after), wantRun, "2 " + field + " fields"},
			)
		}
	}
	empty := strings.Replace(pass, "- **Reviewer**: claude (opus, high), agent type example-reviewer", "- **Reviewer**:", 1)
	cases = append(cases, recordCase{"empty reviewer", empty, approvalFor("implementation", empty), "", "Reviewer is empty"})
	for _, tc := range cases {
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
