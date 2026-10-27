package main

import (
	"crypto/sha256"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/c360studio/ateam/internal/state"
)

// spec: openspec/changes/handoff-read-196/specs/harness/spec.md#Requirement: Read another agent's handoff without changing ownership
func TestStateReadHandoffIsReadOnly(t *testing.T) {
	// #196: Codex reads Claude's slot; nothing Claude owns, and no forge, may be touched.
	dir := t.TempDir()
	binDir := t.TempDir()
	ghLog := filepath.Join(binDir, "gh.log")
	if err := os.WriteFile(filepath.Join(binDir, "gh"), []byte("#!/bin/sh\necho \"$@\" >> '"+ghLog+"'\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	for k, v := range map[string]string{"ATEAM_AGENT": "codex", "CODEX_SESSION_ID": "codex-sess", "ATEAM_SESSION_ID": "", "CLAUDE_SESSION_ID": "", "CLAUDECODE": "", "GEMINI_SESSION_ID": ""} {
		t.Setenv(k, v)
	}
	if _, err := state.Update(dir, func(cur *state.State) error {
		if err := cur.UpsertForge(state.ForgeConfig{Provider: state.ForgeGitHub, Owner: "c360", Repo: "ateam"}); err != nil {
			return err
		}
		cur.Git.Branch, cur.Git.HeadSHA = "main", "1111111"
		cur.SetHandoff(state.Handoff{Goal: "Ship the #196 read path", ChecksToRun: "task verify", Agent: "claude", SessionID: "claude-sess", Branch: "main", CheckpointSHA: "1111111", UpdatedAt: time.Now().UTC()})
		return nil
	}); err != nil {
		t.Fatalf("seed state: %v", err)
	}
	lease := []byte(`{"agent":"claude","session_id":"claude-sess","pid":` + strconv.Itoa(os.Getpid()) + `,"last_seen":"` + time.Now().UTC().Format(time.RFC3339) + `"}`)
	if err := os.WriteFile(state.WriterLeasePath(dir), lease, 0o644); err != nil {
		t.Fatal(err)
	}
	// Prove the fake records calls, so an empty log after the read means no forge call.
	if err := exec.Command("gh", "probe").Run(); err == nil {
		t.Fatal("fake gh should exit 1")
	}
	ghBefore := readFile(t, ghLog)
	if ghBefore != "probe\n" {
		t.Fatalf("fake gh log = %q, want the probe call", ghBefore)
	}
	cacheBefore := sha256.Sum256([]byte(readFile(t, state.CacheFilePath(dir))))

	code, out := captureStdout(t, func() int { return runState(dir, []string{"read-handoff", "claude@main"}) })
	if code != 0 || !strings.Contains(out, "Handoff claude@main (") || !strings.Contains(out, "Goal: Ship the #196 read path\n") {
		t.Fatalf("read-handoff = %d, output:\n%s", code, out)
	}
	code, stderr := captureBenchError(t, func() int { return runState(dir, []string{"read-handoff", "gemini@main"}) })
	if code != 1 || !strings.Contains(stderr, "existing slots: claude@main") {
		t.Fatalf("missing slot = %d, stderr %q; want 1 listing claude@main", code, stderr)
	}

	if after := sha256.Sum256([]byte(readFile(t, state.CacheFilePath(dir)))); after != cacheBefore {
		t.Fatalf("cache SHA-256 changed: %x -> %x", cacheBefore, after)
	}
	// CurrentAgent and CurrentSessionID are not serialized; the lease is the persisted identity.
	if got := readFile(t, state.WriterLeasePath(dir)); got != string(lease) {
		t.Fatalf("writer lease changed:\n%s", got)
	}
	if got := readFile(t, ghLog); got != ghBefore {
		t.Fatalf("forge fake recorded calls: %q", got)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return string(data)
}
