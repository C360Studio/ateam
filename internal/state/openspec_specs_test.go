package state

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// spec: openspec/specs/harness/spec.md#Requirement: Board state and authenticated API access
func TestParseCanonicalSpecMarkdown(t *testing.T) {
	markdown := "# Harness\n\n## Purpose\nSupport café teams.\nKeep work visible.\n\n## Requirements\n### Requirement: Board state\nThe board SHALL expose state.\nIt SHALL include holds.\n\n#### Scenario: Load board\n- **WHEN** a user opens the board\n- **THEN** show état\n#### Scenario: Refresh board\n- **WHEN** refreshed\n- **THEN** show current state\n### Requirement: Empty state\nThe board SHALL allow empty state.\n#### Scenario: No cards\n- **THEN** show no cards\n## Notes\nIgnored trailing text.\n"
	want := OpenSpecSpec{
		Name: "harness", Path: "openspec/specs/harness/spec.md", Purpose: "Support café teams. Keep work visible.",
		Requirements: []OpenSpecRequirement{
			{ID: "harness:1", Name: "Board state", Spec: "harness", Statement: "The board SHALL expose state. It SHALL include holds.", Scenarios: []OpenSpecScenario{
				{Name: "Load board", Steps: []string{"WHEN a user opens the board", "THEN show état"}},
				{Name: "Refresh board", Steps: []string{"WHEN refreshed", "THEN show current state"}},
			}},
			{ID: "harness:2", Name: "Empty state", Spec: "harness", Statement: "The board SHALL allow empty state.", Scenarios: []OpenSpecScenario{{Name: "No cards", Steps: []string{"THEN show no cards"}}}},
		},
	}
	for _, newline := range []string{"\n", "\r\n"} {
		t.Run(strings.ReplaceAll(newline, "\n", "LF"), func(t *testing.T) {
			got, err := ParseSpecMarkdown("harness", want.Path, []byte(strings.ReplaceAll(markdown, "\n", newline)))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("parsed spec = %#v; want %#v", got, want)
			}
		})
	}
	got, err := ParseSpecMarkdown("empty", "empty/spec.md", nil)
	if err != nil || got.Name != "empty" || len(got.Requirements) != 0 {
		t.Fatalf("empty spec = %+v, %v", got, err)
	}
	if _, err := ParseSpecMarkdown("bad", "bad/spec.md", []byte{0xff}); err == nil || !strings.Contains(err.Error(), "bad/spec.md") {
		t.Fatalf("invalid UTF-8 error = %v", err)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Board state and authenticated API access
func TestScanCanonicalSpecsDeterministic(t *testing.T) {
	root := t.TempDir()
	if got, err := ScanOpenSpecSpecs(root); err != nil || len(got) != 0 {
		t.Fatalf("missing specs = %+v, %v", got, err)
	}
	for _, name := range []string{"zeta", ".hidden", "alpha", "missing"} {
		dir := filepath.Join(root, "openspec", "specs", name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if name != "missing" {
			if err := os.WriteFile(filepath.Join(dir, "spec.md"), []byte("## Purpose\n"+name+" purpose\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := os.WriteFile(filepath.Join(root, "openspec", "specs", "stray.md"), []byte("not a spec directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	want := []OpenSpecSpec{{Name: "alpha", Path: "openspec/specs/alpha/spec.md", Purpose: "alpha purpose"}, {Name: "zeta", Path: "openspec/specs/zeta/spec.md", Purpose: "zeta purpose"}}
	for range 2 {
		got, err := ScanOpenSpecSpecs(root)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("scan order/content = %+v, want %+v", got, want)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "openspec", "specs", "alpha", "spec.md"), []byte{0xff}, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ScanOpenSpecSpecs(root); err == nil || !strings.Contains(err.Error(), "alpha/spec.md") {
		t.Fatalf("invalid spec was not reported: %v", err)
	}
}
