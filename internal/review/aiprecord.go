package review

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
)

// recordField is one REVIEW-002 Verdict-section item: "- **<Name>**: <value>".
var recordField = regexp.MustCompile(`^- \*\*([A-Za-z]+)\*\*: (\S.*)$`)

// CheckImplementationRecord reads the reviewer run from a REVIEW-002
// review-implementation.md and checks it is an approved pass (owner ruling on
// #130): one Verdict section whose Stage is implementation and whose Verdict is
// PASS, and a sibling approval.md (nil when it does not exist) with an
// implementation row whose Review digest is the record's SHA-256. The Reviewer
// value must start with <agent> (<model>, <effort>). ok reports that the
// reviewer run was read; findings lists every reason the record does not count.
// This does not validate the rest of REVIEW-002 (manifest, findings table): the
// project's own gate owns that.
func CheckImplementationRecord(record, approval []byte) (reviewer Run, ok bool, findings []string) {
	fields, sections := map[string][]string{}, 0
	in := false
	for _, line := range strings.Split(strings.ReplaceAll(string(record), "\r\n", "\n"), "\n") {
		if strings.HasPrefix(line, "# ") || strings.HasPrefix(line, "## ") {
			in = line == "## Verdict"
			if in {
				sections++
			}
			continue
		}
		if m := recordField.FindStringSubmatch(line); in && m != nil {
			fields[m[1]] = append(fields[m[1]], m[2])
		}
	}
	if sections != 1 {
		return Run{}, false, []string{fmt.Sprintf("the record has %d \"## Verdict\" sections, want exactly one", sections)}
	}
	field := func(name, want string) string {
		values := fields[name]
		switch {
		case len(values) != 1:
			findings = append(findings, fmt.Sprintf("the Verdict section has %d %s fields, want exactly one", len(values), name))
			return ""
		case want != "" && values[0] != want:
			findings = append(findings, fmt.Sprintf("%s is %q, want %q", name, values[0], want))
		}
		return values[0]
	}
	field("Stage", "implementation")
	field("Verdict", "PASS")
	if name := field("Reviewer", ""); name != "" {
		runs := gateRuns(name)
		if len(runs) == 1 {
			reviewer, ok = mustRun(runs[0])
		}
		if !ok {
			findings = append(findings, fmt.Sprintf("Reviewer %q must start with <agent> (<model>, <effort>), agent claude, codex or gemini", name))
		}
	}
	if approval == nil {
		return reviewer, ok, append(findings, "no approval.md beside the record: a person has not approved it")
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(record))
	if !approvedDigest(approval, digest) {
		findings = append(findings, "approval.md has no implementation row whose Review digest is the record's SHA-256 "+digest)
	}
	return reviewer, ok, findings
}

// mustRun parses an agent:model:effort run that gateRuns built.
func mustRun(spec string) (Run, bool) {
	r, err := ParseRunSpec(spec)
	return r, err == nil
}

// approvedDigest reports an approval.md table row "| implementation | <approver> | <date> | <digest> |".
func approvedDigest(approval []byte, digest string) bool {
	for _, line := range strings.Split(string(approval), "\n") {
		cells := strings.Split(strings.TrimSpace(line), "|")
		if len(cells) != 6 || cells[0] != "" || cells[5] != "" {
			continue
		}
		if strings.TrimSpace(cells[1]) == "implementation" && strings.TrimSpace(cells[4]) == digest {
			return true
		}
	}
	return false
}
