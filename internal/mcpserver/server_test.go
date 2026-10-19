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
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"ateam_state","arguments":{"action":"handoff","goal":"Ship v0.1","done_when":"task verify green"}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"ateam_state","arguments":{"action":"handoff","goal":"Ship v0.1 opt-out","done_when":"task verify green","post":false}}}`,
		`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"ateam_state","arguments":{"action":"pickup"}}}`,
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
	if !strings.Contains(lines[4], "Goal: Ship v0.1 opt-out") {
		t.Errorf("pickup response missing handoff Goal: %s", lines[4])
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
