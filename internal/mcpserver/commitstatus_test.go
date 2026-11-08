package mcpserver

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/c360studio/ateam/internal/state"
)

// mcpStatusRepo is a GitHub-remote repository on claude/pr-7 with a fake gh that answers for PR #7 at its head and
// rejects every status write; it returns the root, the head, and the gh call log.
func mcpStatusRepo(t *testing.T) (root, head, log string) {
	t.Helper()
	for _, k := range []string{"ATEAM_FORGE", "GITEA_URL", "ATEAM_COMMIT_STATUSES", "FAKE_COMMENT_SLEEP", "FAKE_HANDOFF_COMMENTS", "GH_TOKEN", "ATEAM_TOKEN_READ", "ATEAM_TOKEN_WRITE",
		"ATEAM_SESSION_ID", "CLAUDE_SESSION_ID", "CLAUDE_CODE_SESSION_ID", "CODEX_SESSION_ID", "GEMINI_SESSION_ID", "HERDR_ENV"} {
		t.Setenv(k, "")
	}
	root = t.TempDir()
	git := func(args ...string) string {
		cmd := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@t"}, args...)...)
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-q", "-b", "main")
	git("remote", "add", "origin", "https://github.com/o/r.git")
	git("commit", "-q", "--allow-empty", "-m", "init")
	git("update-ref", "refs/remotes/origin/main", "main")
	git("checkout", "-q", "-b", "claude/pr-7")
	git("commit", "-q", "--allow-empty", "-m", "work")
	head = git("rev-parse", "HEAD")

	bin := t.TempDir()
	log = filepath.Join(bin, "calls")
	script := "#!/bin/sh\n" +
		"{ for a in \"$@\"; do printf '%s|' \"$a\"; done; printf '\\n'; } >> '" + log + "'\n" +
		"case \"$*\" in\n" +
		"'pr comment'*) [ -n \"$FAKE_COMMENT_SLEEP\" ] && exec sleep \"$FAKE_COMMENT_SLEEP\"; echo 'https://github.com/o/r/pull/7#issuecomment-42';;\n" +
		"'pr view claude/pr-7 --json comments') printf '%s' \"${FAKE_HANDOFF_COMMENTS:-[]}\";;\n" +
		"'pr view claude/pr-7 --json number,headRefOid,url'*) printf '%s' '{\"number\":7,\"headRefOid\":\"" + head + "\",\"url\":\"https://github.com/o/r/pull/7\"}';;\n" +
		"'api -X POST '*) echo 'response body secret-body' >&2; exit 1;;\n" +
		"'api repos/'*) printf '%s' '{\"statuses\":[]}';;\n" +
		"*) printf '[]';;\nesac\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return root, head, log
}

// spec: openspec/specs/harness/spec.md#Requirement: Informational commit statuses on the PR head
func TestMCPHandoffReportsStatusWriteError(t *testing.T) {
	root, head, log := mcpStatusRepo(t)
	text, isErr := execStateTool(context.Background(), root, json.RawMessage(`{"action":"handoff","agent":"claude","session_id":"S","goal":"g"}`), "", "test")
	if isErr {
		t.Fatalf("a status write error failed the handoff:\n%s", text)
	}
	want := "PR comment: posted\ncommit statuses: error (ateam/handoff on " + head[:7] + ": gh exit 1; posted 0, unchanged 0, errors 1)\n"
	if !strings.HasSuffix(text, want) {
		t.Fatalf("handoff text =\n%s\nwant suffix\n%s", text, want)
	}
	if strings.Contains(text, "secret") {
		t.Fatalf("handoff text leaks the response body:\n%s", text)
	}

	if err := os.Remove(log); err != nil {
		t.Fatal(err)
	}
	text, isErr = execStateTool(context.Background(), root, json.RawMessage(`{"action":"handoff","agent":"claude","session_id":"S","goal":"g2","post":false}`), "", "test")
	if isErr || !strings.Contains(text, "PR comment: skipped (post=false)") {
		t.Fatalf("post:false handoff isErr=%v:\n%s", isErr, text)
	}
	if data, _ := os.ReadFile(log); len(data) != 0 {
		t.Fatalf("post:false handoff called gh: %s", data)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Durable shared state and five-field handoff
func TestMCPHandoffPostOutcome(t *testing.T) {
	args := json.RawMessage(`{"action":"handoff","agent":"claude","session_id":"S","goal":"g","addresses":"PR #7","checks_to_run":"task verify"}`)

	t.Run("a timed-out post is uncertain", func(t *testing.T) {
		root, _, log := mcpStatusRepo(t)
		t.Setenv("FAKE_COMMENT_SLEEP", "10")
		text, isErr := execStateTool(context.Background(), root, args, "", "test")
		if want := "PR comment: uncertain (gh pr comment claude/pr-7 timed out: outcome unknown)\n"; isErr || !strings.HasSuffix(text, want) {
			t.Fatalf("handoff isErr=%v text =\n%s\nwant suffix %q", isErr, text, want)
		}
		if data, _ := os.ReadFile(log); strings.Contains(string(data), "api|-X|POST") {
			t.Fatalf("an uncertain post wrote a commit status:\n%s", data)
		}
	})

	t.Run("a repeat of the newest handoff is not posted again", func(t *testing.T) {
		root, _, log := mcpStatusRepo(t)
		body := state.FormatFiveFieldPrompt(state.Handoff{Goal: "g", Addresses: "PR #7", ChecksToRun: "task verify"})
		raw, _ := json.Marshal(map[string]any{"comments": []map[string]string{{"body": body, "url": "https://github.com/o/r/pull/7#issuecomment-41"}}})
		t.Setenv("FAKE_HANDOFF_COMMENTS", string(raw))
		text, isErr := execStateTool(context.Background(), root, args, "", "test")
		if isErr || !strings.Contains(text, "PR comment: posted (already on PR)\n") {
			t.Fatalf("handoff isErr=%v text =\n%s", isErr, text)
		}
		if data, _ := os.ReadFile(log); strings.Contains(string(data), "pr|comment|") {
			t.Fatalf("a repeat posted a second comment:\n%s", data)
		}
	})
}
