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
	// A sentence that tells the reader to read the protocol, first or fully,
	// without naming a section is a full-read mandate, however the path is
	// formatted.
	sentenceEnd = regexp.MustCompile(`[.;:]\s|\n`)
	fullRead    = regexp.MustCompile(`(?i)\bread\b.*\b(first|fully)\b`)
	// Unnumbered protocol headings that skills and contracts name in italics.
	namedProtocolSections = []string{"Test-to-spec attribution", "Unattended and fan-out runs"}
	// pointerFallback sends an agent whose protocol lacks the numbered
	// sections, as a brownfield adopter's own protocol does, to the whole file (#371).
	pointerFallback = "If your protocol has no such numbered sections, read all of it."
)

// readingViolations lists full-protocol read mandates and section pointers in
// a skill or contract that the protocol does not resolve.
func readingViolations(protocol, doc string) []string {
	var out []string
	for _, sentence := range sentenceEnd.Split(doc, -1) {
		if fullRead.MatchString(sentence) && strings.Contains(sentence, "protocol.md") && !strings.Contains(sentence, "§") {
			out = append(out, "full read: "+strings.TrimSpace(sentence))
		}
	}
	if strings.Contains(doc, "read §") && !strings.Contains(doc, pointerFallback) {
		out = append(out, "section pointer without the whole-protocol fallback")
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
	protocol := "# P\n\n## 1. State\n\n## 2. Lifecycle\n\n## Test-to-spec attribution\n"
	for _, tc := range []struct {
		doc  string
		want int
	}{
		{"From protocol, read § 1 and *Test-to-spec attribution*.", 0},
		{"The review-pr skill § 5 gives the commands.", 0},
		{"Read § 3.", 1},
		{"Also read *Unattended and fan-out runs*.", 1},
		{"Read AGENTS.md and .agents/protocol.md fully.", 1},
		{"Read [the shared protocol](../../protocol.md) first.", 1},
		{"Read `.agents/protocol.md` fully.", 1},
		{"Read [the shared protocol](../../protocol.md) and [the reviewer contract](../../contracts/reviewer.md) first.", 1},
		{"Read [the shared protocol](../../protocol.md) and [the architect contract](../../contracts/architect.md) first. Use this skill.", 1},
		{"Read [the reviewer contract](../../contracts/reviewer.md) first; it names the protocol sections a review needs.", 0},
		{"From `.agents/protocol.md`, read § 1 and § 2 step 1 (Clarify).", 0},
		{"From [the shared protocol](../../protocol.md), read § 1 and § 2 steps 1–2 (Clarify, Claim).", 0},
		{"For unclaimed work, follow the Claim gate in [protocol.md](../../protocol.md).", 0},
	} {
		if got := readingViolations(protocol, tc.doc); len(got) != tc.want {
			t.Errorf("%q: got %v, want %d violations", tc.doc, got, tc.want)
		}
	}
}
