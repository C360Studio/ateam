package hook

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/c360studio/ateam/internal/state"
)

// spec: openspec/changes/commit-statuses-302/specs/harness/spec.md#Requirement: Informational commit statuses on the PR head
func TestSessionStartMakesNoForgeCall(t *testing.T) {
	// Commit statuses are never written from a hook: a session-start with a stale forge cache and an open worktree
	// PR on both forges must not run gh or reach the Gitea API.
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()
	// Activation: the counter sees a request that does reach the server.
	if resp, err := http.Get(srv.URL); err == nil {
		resp.Body.Close()
	}
	if requests.Swap(0) != 1 {
		t.Fatal("fake Gitea did not count a request")
	}

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
	gh := state.ForgeConfig{Provider: state.ForgeGitHub, Owner: "o", Repo: "r"}
	gt := state.ForgeConfig{Provider: state.ForgeGitea, BaseURL: srv.URL, Owner: "o", Repo: "r"}
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
		t.Fatalf("session-start made %d Gitea requests, want 0", n)
	}
}
