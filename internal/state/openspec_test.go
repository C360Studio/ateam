package state

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

// spec: openspec/specs/harness/spec.md#Requirement: OpenSpec queue and stray Hold detection
func TestOpenSpecQueueAndCheck(t *testing.T) {
	dir := t.TempDir()
	changeDir := filepath.Join(dir, "openspec", "changes", "setup-01")
	if err := os.MkdirAll(changeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "openspec", "config.yaml"), []byte("schema: spec-driven\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tasksMD := `# Tasks

This header explains Hold: markers above the first section heading.

## 1. Implementation

- [x] 1.1 Completed task mentioning halt condition that already passed.
- [ ] 1.2 Implement core state parser.
  - Hold: waiting on PR #48 to merge.
- [~] 1.3 Deliberately skipped legacy adapter.
- [ ] 1.4 If pre-v1 window closes, halt: record the missed window.
- [ ] 1.5 RED — integration suite is failing on port check.
- [ ] 1.6 **STILL OPEN** — decide whether Gitea webhook rides v0.1.
`
	if err := os.WriteFile(filepath.Join(changeDir, "tasks.md"), []byte(tasksMD), 0o644); err != nil {
		t.Fatal(err)
	}

	changes, err := ScanOpenSpecQueue(dir)
	if err != nil {
		t.Fatalf("ScanOpenSpecQueue: %v", err)
	}
	if len(changes) != 1 {
		t.Fatalf("got %d changes, want 1", len(changes))
	}
	ch := changes[0]
	if ch.CompletedTasks != 1 || ch.TotalTasks != 6 {
		t.Errorf("counts = %d/%d, want 1/6", ch.CompletedTasks, ch.TotalTasks)
	}
	if len(ch.Caveats) != 5 {
		t.Fatalf("got %d caveats, want 5: %+v", len(ch.Caveats), ch.Caveats)
	}

	wantMarkers := []string{"BLOCKED", "WONTDO", "HALT", "RED", "OPEN-Q"}
	for i, want := range wantMarkers {
		if ch.Caveats[i].Marker != want {
			t.Errorf("caveat[%d] marker = %q, want %q (%+v)", i, ch.Caveats[i].Marker, want, ch.Caveats[i])
		}
	}

	out, code, err := CheckOpenSpec(context.Background(), dir)
	if err != nil || code != 0 {
		t.Fatalf("CheckOpenSpec clean failed: code=%d err=%v out=%s", code, err, out)
	}
	if !strings.Contains(out, "holds: ok (1 tasks.md read)") {
		t.Errorf("unexpected check output: %q", out)
	}

	// Add a stray Hold: below the ## heading outside any task block.
	strayMD := tasksMD + "\nHold: stray hold outside every task.\n"
	if err := os.WriteFile(filepath.Join(changeDir, "tasks.md"), []byte(strayMD), 0o644); err != nil {
		t.Fatal(err)
	}
	out, code, err = CheckOpenSpec(context.Background(), dir)
	if err == nil || code != 1 {
		t.Fatalf("expected exit code 1 for stray hold, got code=%d err=%v out=%s", code, err, out)
	}
	if !strings.Contains(out, "Hold: outside every task") {
		t.Errorf("missing stray hold message in %q", out)
	}

	// Write invalid UTF-8 and verify exit code 2.
	if err := os.WriteFile(filepath.Join(changeDir, "tasks.md"), []byte{0xff, 0xfe}, 0o644); err != nil {
		t.Fatal(err)
	}
	_, code, err = CheckOpenSpec(context.Background(), dir)
	if err == nil || code != 2 {
		t.Fatalf("expected exit code 2 for invalid UTF-8, got code=%d err=%v", code, err)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: OpenSpec queue and stray Hold detection
func TestCleanTaskTextKeepsUTF8AtTheCut(t *testing.T) {
	got := cleanTaskText("- [ ] 1.1 RED: " + strings.Repeat("é", 60))
	if !utf8.ValidString(got) || len(got) > 104 {
		t.Fatalf("cleanTaskText = %q (len %d), want valid UTF-8 of at most 104 bytes", got, len(got))
	}
}
