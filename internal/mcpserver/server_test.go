package mcpserver

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

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
