package state

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHandoffPickupRoundTrip(t *testing.T) {
	dir := t.TempDir()
	st, err := Load(dir)
	if err != nil {
		t.Fatalf("Load empty dir: %v", err)
	}
	if st.Mode != ModeFull {
		t.Fatalf("default mode = %q, want %q", st.Mode, ModeFull)
	}

	h := Handoff{
		Goal:          "PR #12: implement universal hooks and Gitea sync",
		Addresses:     "branch gemini/hooks, SHA abc1234",
		Constraints:   "Hold: none; write ownership transferred to codex",
		DoneWhen:      "task verify passes on clean tree",
		ChecksToRun:   "task test:unit && task spec:check",
		Agent:         "gemini",
		Branch:        "gemini/hooks",
		CheckpointSHA: "abc1234",
		UpdatedAt:     time.Now().UTC(),
	}
	st.LastHandoff = &h
	st.Mode = ModeUltra
	if err := Save(dir, st); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load saved: %v", err)
	}
	if loaded.Mode != ModeUltra {
		t.Fatalf("loaded mode = %q, want %q", loaded.Mode, ModeUltra)
	}
	if loaded.LastHandoff == nil || loaded.LastHandoff.Goal != h.Goal {
		t.Fatalf("loaded handoff mismatch: %+v", loaded.LastHandoff)
	}

	block := FormatFiveFieldPrompt(*loaded.LastHandoff)
	for _, want := range []string{
		"Goal: PR #12: implement universal hooks and Gitea sync",
		"Addresses: branch gemini/hooks, SHA abc1234",
		"Constraints: Hold: none; write ownership transferred to codex",
		"Done when: task verify passes on clean tree",
		"Checks to run: task test:unit && task spec:check",
	} {
		if !strings.Contains(block, want) {
			t.Errorf("FormatFiveFieldPrompt missing %q in:\n%s", want, block)
		}
	}
}

func TestScanShortcuts(t *testing.T) {
	dir := t.TempDir()
	src := `package sample

// shortcut: in-memory cache only, upgrade to NATS KV when multi-node is enabled
func compute() int {
	return 42
}
`
	if err := os.WriteFile(filepath.Join(dir, "sample.go"), []byte(src), 0o644); err != nil {
		t.Fatalf("write sample.go: %v", err)
	}
	items, err := ScanShortcuts(dir)
	if err != nil {
		t.Fatalf("ScanShortcuts: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("got %d shortcuts, want 1", len(items))
	}
	if items[0].File != "sample.go" || items[0].Line != 3 {
		t.Errorf("unexpected location: %+v", items[0])
	}
	if items[0].Limit != "in-memory cache only" || items[0].RevisitTrigger != "upgrade to NATS KV when multi-node is enabled" {
		t.Errorf("unexpected parsed shortcut: %+v", items[0])
	}
}
