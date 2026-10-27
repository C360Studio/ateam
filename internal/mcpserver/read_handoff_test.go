package mcpserver

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/c360studio/ateam/internal/state"
)

// spec: openspec/specs/harness/spec.md#Requirement: Read another agent's handoff without changing ownership
func TestMCPReadHandoffIsReadOnly(t *testing.T) {
	// #196: a Codex MCP session reads Claude's slot without refreshing, syncing, or claiming anything.
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
	// Prove the fake records calls, so an unchanged log after the read means no forge call.
	_ = exec.Command("gh", "probe").Run()
	ghBefore, _ := os.ReadFile(ghLog)
	if string(ghBefore) != "probe\n" {
		t.Fatalf("fake gh log = %q, want the probe call", ghBefore)
	}
	cache := func() [32]byte {
		data, err := os.ReadFile(state.CacheFilePath(dir))
		if err != nil {
			t.Fatal(err)
		}
		return sha256.Sum256(data)
	}
	cacheBefore := cache()

	text, isErr := execStateTool(context.Background(), dir, json.RawMessage(`{"action":"read_handoff","slot":"claude@main"}`), "codex-conn", "test")
	if isErr || !strings.Contains(text, "Handoff claude@main (") || !strings.Contains(text, "Goal: Ship the #196 read path\n") {
		t.Fatalf("read_handoff isErr=%v:\n%s", isErr, text)
	}
	text, isErr = execStateTool(context.Background(), dir, json.RawMessage(`{"action":"read_handoff","slot":"gemini@main"}`), "codex-conn", "test")
	if !isErr || !strings.Contains(text, "existing slots: claude@main") {
		t.Fatalf("missing slot isErr=%v, text %q; want a tool error listing claude@main", isErr, text)
	}

	if after := cache(); after != cacheBefore {
		t.Fatalf("cache SHA-256 changed: %x -> %x", cacheBefore, after)
	}
	// CurrentAgent and CurrentSessionID are not serialized; the lease is the persisted identity.
	if got, _ := os.ReadFile(state.WriterLeasePath(dir)); string(got) != string(lease) {
		t.Fatalf("writer lease changed:\n%s", got)
	}
	if got, _ := os.ReadFile(ghLog); string(got) != string(ghBefore) {
		t.Fatalf("forge fake recorded calls: %q", got)
	}
}
