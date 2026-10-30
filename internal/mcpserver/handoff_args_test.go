package mcpserver

import (
	"bytes"
	"context"
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

// spec: openspec/changes/mcp-handoff-args-266/specs/harness/spec.md#Requirement: Two-tool MCP context budget
func TestMCPHandoffRefusesInvalidArgumentsBeforeAnyWrite(t *testing.T) {
	// #266: a blank or wrong-type goal must not replace a useful checkpoint, release the lease, or post.
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"remote", "add", "origin", "https://github.com/c360/ateam.git"},
		{"-c", "user.email=t@t", "-c", "user.name=t", "commit", "-q", "--allow-empty", "-m", "seed"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	// A fake gh that records every call and succeeds, so a post would land in the log.
	binDir := t.TempDir()
	ghLog := filepath.Join(binDir, "gh.log")
	if err := os.WriteFile(filepath.Join(binDir, "gh"), []byte("#!/bin/sh\necho \"$@\" >> '"+ghLog+"'\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	for k, v := range map[string]string{"ATEAM_AGENT": "claude", "ATEAM_SESSION_ID": "sess-266", "CLAUDE_SESSION_ID": "", "CODEX_SESSION_ID": "", "GEMINI_SESSION_ID": "", "ATEAM_FORGE": "", "GITEA_URL": ""} {
		t.Setenv(k, v)
	}
	if _, err := state.Update(dir, func(cur *state.State) error {
		if err := cur.UpsertForge(state.ForgeConfig{Provider: state.ForgeGitHub, Owner: "c360", Repo: "ateam"}); err != nil {
			return err
		}
		cur.SetHandoff(state.Handoff{Goal: "Useful checkpoint", ChecksToRun: "task verify", Agent: "claude", SessionID: "sess-266", Branch: "main", CheckpointSHA: "1111111", UpdatedAt: time.Now().UTC()})
		return nil
	}); err != nil {
		t.Fatalf("seed state: %v", err)
	}
	lease := []byte(`{"agent":"claude","session_id":"sess-266","pid":` + strconv.Itoa(os.Getpid()) + `,"last_seen":"` + time.Now().UTC().Format(time.RFC3339) + `"}`)
	if err := os.WriteFile(state.WriterLeasePath(dir), lease, 0o644); err != nil {
		t.Fatal(err)
	}
	cacheBefore, err := os.ReadFile(state.CacheFilePath(dir))
	if err != nil {
		t.Fatal(err)
	}

	for name, raw := range map[string]string{
		"missing goal":    `{"action":"handoff","agent":"claude"}`,
		"empty goal":      `{"action":"handoff","agent":"claude","goal":""}`,
		"whitespace goal": `{"action":"handoff","agent":"claude","goal":"   "}`,
		"numeric goal":    `{"action":"handoff","agent":"claude","goal":123}`,
		"string post":     `{"action":"handoff","agent":"claude","goal":"g","post":"yes"}`,
		"numeric post":    `{"action":"handoff","agent":"claude","goal":"g","post":123}`,
	} {
		text, isErr := execStateTool(context.Background(), dir, json.RawMessage(raw), "", "test")
		if !isErr {
			t.Errorf("%s: isError=false, text %q", name, text)
		}
		wantText := "handoff: goal is required"
		if strings.Contains(raw, "123") || strings.Contains(raw, `"yes"`) {
			wantText = "invalid arguments: "
		}
		if !strings.HasPrefix(text, wantText) {
			t.Errorf("%s: text %q, want prefix %q", name, text, wantText)
		}
		if got, _ := os.ReadFile(state.CacheFilePath(dir)); !bytes.Equal(got, cacheBefore) {
			t.Fatalf("%s: cache changed", name)
		}
		if got, _ := os.ReadFile(state.WriterLeasePath(dir)); !bytes.Equal(got, lease) {
			t.Fatalf("%s: writer lease changed: %q", name, got)
		}
		if got, _ := os.ReadFile(ghLog); len(got) != 0 {
			t.Fatalf("%s: forge fake recorded calls: %q", name, got)
		}
	}

	// A valid goal with post omitted replaces the same-session checkpoint and posts, proving the
	// fake forge and the cache comparison above would have caught a write or a post.
	text, isErr := execStateTool(context.Background(), dir, json.RawMessage(`{"action":"handoff","agent":"claude","goal":"Replacement goal"}`), "", "test")
	if isErr || !strings.Contains(text, "Goal: Replacement goal") || !strings.Contains(text, "PR comment: posted") {
		t.Fatalf("valid handoff isErr=%v:\n%s", isErr, text)
	}
	st, err := state.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if h := st.Handoffs["claude@main"]; h == nil || h.Goal != "Replacement goal" || h.SessionID != "sess-266" {
		t.Fatalf("slot after valid handoff = %+v, want the replacement goal", h)
	}
	if got, _ := os.ReadFile(ghLog); !strings.Contains(string(got), "pr comment main") {
		t.Fatalf("valid handoff made no forge post: %q", got)
	}
	if _, err := os.Stat(state.WriterLeasePath(dir)); !os.IsNotExist(err) {
		t.Fatalf("valid handoff kept the writer lease: %v", err)
	}
}

// spec: openspec/changes/mcp-handoff-args-266/specs/harness/spec.md#Requirement: Two-tool MCP context budget
func TestMCPTaskToolRefusesWrongTypeTarget(t *testing.T) {
	dir := t.TempDir()
	// A target that would leave a marker if task ran it.
	taskfile := "version: '3'\ntasks:\n  touch:\n    cmds:\n      - touch ran.marker\n"
	if err := os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte(taskfile), 0o644); err != nil {
		t.Fatal(err)
	}
	text, isErr := execTaskTool(context.Background(), dir, json.RawMessage(`{"target":123}`))
	if !isErr || !strings.HasPrefix(text, "invalid arguments: ") {
		t.Fatalf("numeric target isErr=%v, text %q", isErr, text)
	}
	text, isErr = execTaskTool(context.Background(), dir, json.RawMessage(`{"target":"touch","args":[1]}`))
	if !isErr || !strings.HasPrefix(text, "invalid arguments: ") {
		t.Fatalf("numeric args isErr=%v, text %q", isErr, text)
	}
	if _, err := os.Stat(filepath.Join(dir, "ran.marker")); !os.IsNotExist(err) {
		t.Fatalf("a task ran despite invalid arguments: %v", err)
	}
}

// spec: openspec/changes/mcp-handoff-args-266/specs/harness/spec.md#Requirement: Two-tool MCP context budget
func TestMCPToolCallWithoutArgumentsStillWorks(t *testing.T) {
	// Absent arguments decode to nothing, not an error, so the decode check must not reject them.
	dir := t.TempDir()
	input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"ateam_state"}}` + "\n" +
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"ateam_state","arguments":null}}` + "\n"
	var out bytes.Buffer
	if err := Serve(context.Background(), dir, strings.NewReader(input), &out); err != nil {
		t.Fatalf("Serve: %v", err)
	}
	if strings.Contains(out.String(), "invalid arguments") || strings.Count(out.String(), "unsupported action: ") != 2 {
		t.Fatalf("argument-less calls should reach action dispatch:\n%s", out.String())
	}
}
