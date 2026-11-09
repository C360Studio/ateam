package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var (
	protocolPointer = regexp.MustCompile(`(skill )?§ (\d+)`)
	// Unnumbered protocol headings that skills and contracts name in italics.
	namedProtocolSections = []string{"Test-to-spec attribution", "Unattended and fan-out runs"}
)

// readingViolations lists full-protocol read mandates and section pointers in
// a skill or contract that the protocol does not resolve.
func readingViolations(protocol, doc string) []string {
	var out []string
	for _, full := range []string{"protocol.md fully", "protocol.md) first"} {
		if strings.Contains(doc, full) {
			out = append(out, "full read: "+full)
		}
	}
	for _, m := range protocolPointer.FindAllStringSubmatch(doc, -1) {
		if m[1] != "" {
			continue // a skill's own section, e.g. "review-pr skill § 5"
		}
		if !strings.Contains(protocol, "\n## "+m[2]+". ") {
			out = append(out, "dangling § "+m[2])
		}
	}
	for _, name := range namedProtocolSections {
		if strings.Contains(doc, "*"+name+"*") && !regexp.MustCompile(`(?m)^#+ `+regexp.QuoteMeta(name)+`$`).MatchString(protocol) {
			out = append(out, "dangling *"+name+"*")
		}
	}
	return out
}

// Skills and contracts point at protocol sections instead of a full read, and
// every pointer resolves, in shipped templates and in this repository (#200).
// spec: openspec/specs/harness/spec.md#Requirement: Task-scoped instruction reads
func TestTaskScopedInstructionReads(t *testing.T) {
	root := t.TempDir()
	if _, err := InitRepo(context.Background(), root, Options{}); err != nil {
		t.Fatalf("init: %v", err)
	}
	for _, dir := range []string{root, filepath.Join("..", "..")} {
		protocol, err := os.ReadFile(filepath.Join(dir, ".agents", "protocol.md"))
		if err != nil {
			t.Fatal(err)
		}
		docs, _ := filepath.Glob(filepath.Join(dir, ".agents", "skills", "*", "SKILL.md"))
		contracts, _ := filepath.Glob(filepath.Join(dir, ".agents", "contracts", "*.md"))
		docs = append(docs, contracts...)
		if len(docs) < len(canonicalSkills)+3 {
			t.Fatalf("%s: found %d skills and contracts", dir, len(docs))
		}
		pointers := 0
		for _, path := range docs {
			doc, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			pointers += len(protocolPointer.FindAllString(string(doc), -1))
			if v := readingViolations(string(protocol), string(doc)); len(v) != 0 {
				t.Errorf("%s: %v", path, v)
			}
		}
		if pointers == 0 {
			t.Fatalf("%s: no section pointers found", dir)
		}
	}
}

// The checker flags full reads and dangling pointers and skips a skill's own sections.
// spec: openspec/specs/harness/spec.md#Requirement: Task-scoped instruction reads
func TestReadingViolationsSensitivity(t *testing.T) {
	protocol := "# P\n\n## 1. State\n\n## Test-to-spec attribution\n"
	for _, tc := range []struct {
		doc  string
		want int
	}{
		{"From protocol, read § 1 and *Test-to-spec attribution*.", 0},
		{"The review-pr skill § 5 gives the commands.", 0},
		{"Read § 2.", 1},
		{"Also read *Unattended and fan-out runs*.", 1},
		{"Read AGENTS.md and .agents/protocol.md fully.", 1},
		{"Read [the shared protocol](../../protocol.md) first.", 1},
	} {
		if got := readingViolations(protocol, tc.doc); len(got) != tc.want {
			t.Errorf("%q: got %v, want %d violations", tc.doc, got, tc.want)
		}
	}
}
