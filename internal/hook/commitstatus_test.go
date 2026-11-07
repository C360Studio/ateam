package hook

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/c360studio/ateam/internal/state"
)

// spec: openspec/specs/harness/spec.md#Requirement: Informational commit statuses on the PR head
func TestSessionStartMakesNoForgeCall(t *testing.T) {
	// Commit statuses are never written from a hook: a session-start with a stale forge cache and an open worktree
	// PR on both forges must not run gh or reach the Gitea API.
	// Only requests for this test's own repository count: under full-suite load another test process can reach this
	// port after its own server closed and the port was reused (#317). Every request is kept to name it on failure.
	const owner, repo = "hook-317-owner", "hook-317-repo"
	repoPath := "/api/v1/repos/" + owner + "/" + repo
	var requests atomic.Int32
	var mu sync.Mutex
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.Method+" "+r.URL.Path)
		mu.Unlock()
		if strings.HasPrefix(r.URL.Path, repoPath) {
			requests.Add(1)
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()
	// Activation: the counter sees a request for the repository and ignores one for another.
	for _, path := range []string{repoPath + "/statuses/abc1234", "/api/v1/repos/o/r/statuses/abc1234"} {
		if resp, err := http.Get(srv.URL + path); err == nil {
			resp.Body.Close()
		}
	}
	if requests.Swap(0) != 1 {
		t.Fatal("fake Gitea did not count exactly the repository's request")
	}
	mu.Lock()
	seen = nil
	mu.Unlock()

	bin, marker := t.TempDir(), filepath.Join(t.TempDir(), "gh-ran")
	fake := filepath.Join(bin, "gh")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\necho \"$*\" >> '"+marker+"'\nprintf '[]'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	// Activation: the fake is the gh a subprocess would find, and it leaves the marker.
	if err := exec.Command("gh", "probe").Run(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(marker); err != nil {
		t.Fatalf("fake gh did not record a run: %v", err)
	}

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git", "refs", "heads"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".git", "HEAD"), []byte("ref: refs/heads/claude/pr-7\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gh := state.ForgeConfig{Provider: state.ForgeGitHub, Owner: owner, Repo: repo}
	gt := state.ForgeConfig{Provider: state.ForgeGitea, BaseURL: srv.URL, Owner: owner, Repo: repo}
	old := time.Now().Add(-24 * time.Hour)
	if err := state.Save(dir, &state.State{
		Mode: state.ModeOn, Forge: gh, Forges: []state.ForgeConfig{gh, gt}, ForgesConfigured: true,
		ForgeSyncedAt: map[state.ForgeProvider]time.Time{state.ForgeGitHub: old, state.ForgeGitea: old},
		Git:           state.GitStatus{Branch: "claude/pr-7", HeadSHA: "abc1234"},
		Handoffs:      map[string]*state.Handoff{"claude@claude/pr-7": {Goal: "g", Branch: "claude/pr-7", CheckpointSHA: "abc1234"}},
		Cards: []state.BoardCard{
			{ID: "GH-PR-7", Number: 7, Type: "pr", Forge: "github", Column: "in-progress", Branch: "claude/pr-7", HeadSHA: "abc1234"},
			{ID: "GT-PR-7", Number: 7, Type: "pr", Forge: "gitea", Column: "in-progress", Branch: "claude/pr-7", HeadSHA: "abc1234"},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := Handle(dir, "session-start", strings.NewReader(`{"session_id":"s"}`)); err != nil {
		t.Fatalf("Handle session-start: %v", err)
	}
	if data, err := os.ReadFile(marker); err == nil {
		t.Fatalf("session-start ran gh: %s", data)
	}
	if n := requests.Load(); n != 0 {
		mu.Lock()
		defer mu.Unlock()
		t.Fatalf("session-start made %d Gitea requests for %s, want 0; all requests: %q", n, repoPath, seen)
	}
}
