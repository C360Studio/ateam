package mcpserver

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/c360studio/ateam/internal/state"
)

// spec: openspec/specs/harness/spec.md#Requirement: Two-tool MCP context budget
func TestMCPTokenBudget(t *testing.T) {
	tokens, bytesLen, err := EstimateSchemaTokens()
	if err != nil {
		t.Fatalf("EstimateSchemaTokens: %v", err)
	}
	if len(Tools()) != 2 {
		t.Fatalf("got %d tools, want 2", len(Tools()))
	}
	if tokens >= MaxSchemaTokens {
		t.Fatalf("MCP tool schema uses %d tokens (%d bytes), exceeds %d token budget", tokens, bytesLen, MaxSchemaTokens)
	}
	t.Logf("MCP 2-tool schema footprint: %d tokens (%d bytes) < %d budget", tokens, bytesLen, MaxSchemaTokens)
}

// spec: openspec/specs/harness/spec.md#Requirement: Durable shared state and five-field handoff
func TestMCPStdioRoundTrip(t *testing.T) {
	dir := t.TempDir()
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"ateam_state","arguments":{"action":"handoff","agent":"codex","goal":"Ship v0.1","done_when":"task verify green"}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"ateam_state","arguments":{"action":"handoff","agent":"claude","goal":"Ship v0.1 opt-out","done_when":"task verify green","post":false}}}`,
		`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"ateam_state","arguments":{"action":"pickup","agent":"claude"}}}`,
	}, "\n") + "\n"

	var out bytes.Buffer
	if err := Serve(context.Background(), dir, strings.NewReader(input), &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 5 {
		t.Fatalf("got %d responses, want 5:\n%s", len(lines), out.String())
	}
	if !strings.Contains(lines[2], "Goal: Ship v0.1") || !strings.Contains(lines[2], "PR comment: skipped (local forge)") {
		t.Errorf("handoff response missing Goal or local forge status line: %s", lines[2])
	}
	if !strings.Contains(lines[3], "PR comment: skipped (post=false)") {
		t.Errorf("handoff post=false response missing status line: %s", lines[3])
	}
	if !strings.Contains(lines[4], "Goal: Ship v0.1 opt-out") || !strings.Contains(lines[4], "other handoffs: codex@main") {
		t.Errorf("pickup response missing handoff Goal or other handoffs line: %s", lines[4])
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Two-tool MCP context budget
// spec: openspec/specs/harness/spec.md#Requirement: Durable shared state and five-field handoff
func TestMCPWriterLeaseSameSessionNoWarning(t *testing.T) {
	dir := t.TempDir()
	// Write a lease with a live PID that differs from ResolveClientPID() (os.Getpid() vs os.Getppid()),
	// and verify MCP session resolution via ATEAM_SESSION_ID suppresses false competing-writer warnings.
	t.Setenv("ATEAM_SESSION_ID", "sess-hook-123")
	_ = os.MkdirAll(filepath.Join(dir, ".ateam"), 0o755)
	leaseRaw := []byte(`{"agent":"claude","session_id":"sess-hook-123","pid":` + strconv.Itoa(os.Getpid()) + `,"last_seen":"` + time.Now().UTC().Format(time.RFC3339) + `"}`)
	if err := os.WriteFile(state.WriterLeasePath(dir), leaseRaw, 0o644); err != nil {
		t.Fatalf("write lease: %v", err)
	}

	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"ateam_state","arguments":{"action":"pickup"}}}`,
	}, "\n") + "\n"

	var out bytes.Buffer
	if err := Serve(context.Background(), dir, strings.NewReader(input), &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	if strings.Contains(out.String(), "WARNING: another writer may be active") {
		t.Fatalf("expected MCP pickup from same client process not to warn on own writer lease, got:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "Writer lease: agent=claude") {
		t.Fatalf("expected Writer lease summary in pickup output, got:\n%s", out.String())
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Two-tool MCP context budget
// spec: openspec/specs/harness/spec.md#Requirement: Dual-forge GitHub and Gitea state sync
func TestMCPBoardAndPickupStaleCacheFallback(t *testing.T) {
	dir := t.TempDir()
	binDir := t.TempDir()
	fakeGH := filepath.Join(binDir, "gh")
	if err := os.WriteFile(fakeGH, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	oldSync := time.Date(2026, 10, 19, 13, 3, 0, 0, time.UTC)
	if _, err := state.Update(dir, func(cur *state.State) error {
		if err := cur.UpsertForge(state.ForgeConfig{Provider: state.ForgeGitHub, Owner: "c360", Repo: "ateam"}); err != nil {
			return err
		}
		cur.BoardSyncedAt = oldSync
		cur.Cards = []state.BoardCard{{ID: "GH-1", Number: 1, Title: "Cached card", Column: "backlog", Forge: "github"}}
		return nil
	}); err != nil {
		t.Fatalf("state.Update: %v", err)
	}

	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"ateam_state","arguments":{"action":"board"}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"resources/read","params":{"uri":"ateam://state/board"}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"ateam_state","arguments":{"action":"pickup"}}}`,
	}, "\n") + "\n"

	var out bytes.Buffer
	if err := Serve(context.Background(), dir, strings.NewReader(input), &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d responses, want 3:\n%s", len(lines), out.String())
	}
	for i, name := range []string{"board tool", "board resource"} {
		for _, want := range []string{"2026-10-19T13:03:00Z", "cached_fallback", "sync_error", "GH-1"} {
			if !strings.Contains(lines[i], want) {
				t.Fatalf("%s response missing %q:\n%s", name, want, lines[i])
			}
		}
	}
	if !strings.Contains(lines[2], "synced=2026-10-19T13:03:00Z") || !strings.Contains(lines[2], "WARNING: Forge board sync failed") {
		t.Fatalf("pickup response missing sync timestamp or fallback warning:\n%s", lines[2])
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Two-tool MCP context budget
func TestMCPPickupBuildProvenanceAndStaleBinaryWarning(t *testing.T) {
	dir := t.TempDir()
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "ateam"), []byte("#!/bin/sh\necho 'ateam v0.1.0 (e37563b)'\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	origBuild := state.BuildCommit
	state.BuildCommit = "7606b7e"
	t.Cleanup(func() { state.BuildCommit = origBuild })

	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"ateam_state","arguments":{"action":"pickup"}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"resources/read","params":{"uri":"ateam://state/pickup"}}`,
	}, "\n") + "\n"

	var out bytes.Buffer
	if err := Serve(context.Background(), dir, strings.NewReader(input), &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d responses, want 2:\n%s", len(lines), out.String())
	}
	for i, name := range []string{"pickup tool", "pickup resource"} {
		for _, want := range []string{
			"build=e37563b mcp=7606b7e",
			"WARNING: MCP server binary is stale; restart the client to pick up e37563b",
		} {
			if !strings.Contains(lines[i], want) {
				t.Fatalf("%s response missing %q:\n%s", name, want, lines[i])
			}
		}
	}
	if live := state.LoadLiveMCPProcesses(dir); len(live) != 0 {
		t.Fatalf("expected MCP process registry to be cleaned up after Serve exits, got: %+v", live)
	}
}
