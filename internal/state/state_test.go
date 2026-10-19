package state

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
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

func TestConcurrentUpdateAndCorruptCachePreservation(t *testing.T) {
	dir := t.TempDir()

	// Probe 2: Concurrent / sequential state.Update calls must preserve both handoff and mode without clobbering.
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, err := Update(dir, func(st *State) error {
			st.LastHandoff = &Handoff{Goal: "Preserve handoff across concurrent updates"}
			return nil
		})
		if err != nil {
			t.Errorf("Update handoff: %v", err)
		}
	}()
	go func() {
		defer wg.Done()
		_, err := Update(dir, func(st *State) error {
			st.Mode = ModeUltra
			return nil
		})
		if err != nil {
			t.Errorf("Update mode: %v", err)
		}
	}()
	wg.Wait()

	st, err := Load(dir)
	if err != nil {
		t.Fatalf("Load after concurrent Update: %v", err)
	}
	if st.Mode != ModeUltra {
		t.Fatalf("st.Mode = %q, want %q", st.Mode, ModeUltra)
	}
	if st.LastHandoff == nil || st.LastHandoff.Goal != "Preserve handoff across concurrent updates" {
		t.Fatalf("LastHandoff lost after concurrent Update: %+v", st.LastHandoff)
	}

	// Probe 3: Malformed cache must not be overwritten by Refresh.
	corruptRaw := []byte(`{"mode":"ultra","last_handoff":{"goal":"partialcorrupt`)
	if err := os.WriteFile(CacheFilePath(dir), corruptRaw, 0o644); err != nil {
		t.Fatalf("write corrupt cache: %v", err)
	}
	if _, err := Refresh(context.Background(), dir, false); err == nil {
		t.Fatal("expected Refresh to fail on malformed state.json, got nil")
	}
	afterBytes, err := os.ReadFile(CacheFilePath(dir))
	if err != nil {
		t.Fatalf("read cache after failed Refresh: %v", err)
	}
	if string(afterBytes) != string(corruptRaw) {
		t.Fatalf("Refresh overwrote corrupt cache: got %q, want %q", string(afterBytes), string(corruptRaw))
	}
}

func TestReadFastGitHeadLinkedWorktree(t *testing.T) {
	base := t.TempDir()
	mainGit := filepath.Join(base, "main-repo", ".git")
	wtGitDir := filepath.Join(mainGit, "worktrees", "wt-gemini")
	wtRoot := filepath.Join(base, "wt-gemini")

	if err := os.MkdirAll(filepath.Join(mainGit, "refs", "heads", "gemini"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(wtGitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(wtRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	// Linked worktree .git file pointing to wtGitDir, and commondir pointing to ../..
	if err := os.WriteFile(filepath.Join(wtRoot, ".git"), []byte("gitdir: ../main-repo/.git/worktrees/wt-gemini\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wtGitDir, "commondir"), []byte("../..\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wtGitDir, "HEAD"), []byte("ref: refs/heads/gemini/worktree-fix\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mainGit, "refs", "heads", "gemini", "worktree-fix"), []byte("deadbeefcafe1234567890\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	br, sha := ReadFastGitHead(wtRoot)
	if br != "gemini/worktree-fix" || sha != "deadbee" {
		t.Fatalf("ReadFastGitHead linked worktree loose ref = (%q, %q), want (\"gemini/worktree-fix\", \"deadbee\")", br, sha)
	}

	// Also verify packed-refs fallback via commondir.
	_ = os.Remove(filepath.Join(mainGit, "refs", "heads", "gemini", "worktree-fix"))
	packed := "# pack-refs with: peeled fully-peeled sorted\n9876543210abcdef1234567890abcdef12345678 refs/heads/gemini/worktree-fix\n"
	if err := os.WriteFile(filepath.Join(mainGit, "packed-refs"), []byte(packed), 0o644); err != nil {
		t.Fatal(err)
	}
	br, sha = ReadFastGitHead(wtRoot)
	if br != "gemini/worktree-fix" || sha != "9876543" {
		t.Fatalf("ReadFastGitHead linked worktree packed-ref = (%q, %q), want (\"gemini/worktree-fix\", \"9876543\")", br, sha)
	}
}
