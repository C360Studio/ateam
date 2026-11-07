package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/c360studio/ateam/internal/state"
)

// statusFixture is a GitHub-remote repository on main with a linked worktree on claude/pr-7, a fake gh answering
// for PR #7 at the worktree head, and a fresh handoff in the worktree's cache.
type statusFixture struct {
	root, wt, head, log string
}

func gitRun(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@t"}, args...)...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func newStatusFixture(t *testing.T, statusJSON string) statusFixture {
	t.Helper()
	for _, k := range []string{"ATEAM_FORGE", "GITEA_URL", "ATEAM_COMMIT_STATUSES", "FAKE_POST_EXIT", "GH_TOKEN", "ATEAM_TOKEN_READ", "ATEAM_TOKEN_WRITE",
		"ATEAM_SESSION_ID", "CLAUDE_SESSION_ID", "CLAUDE_CODE_SESSION_ID", "CODEX_SESSION_ID", "GEMINI_SESSION_ID", "HERDR_ENV"} {
		t.Setenv(k, "")
	}
	f := statusFixture{root: t.TempDir(), wt: filepath.Join(t.TempDir(), "pr-7")}
	gitRun(t, f.root, "init", "-q", "-b", "main")
	gitRun(t, f.root, "remote", "add", "origin", "https://github.com/o/r.git")
	gitRun(t, f.root, "commit", "-q", "--allow-empty", "-m", "init")
	gitRun(t, f.root, "update-ref", "refs/remotes/origin/main", "main")
	gitRun(t, f.root, "worktree", "add", "-q", "-b", "claude/pr-7", f.wt)
	gitRun(t, f.wt, "commit", "-q", "--allow-empty", "-m", "work")
	f.head = gitRun(t, f.wt, "rev-parse", "HEAD")

	bin := t.TempDir()
	f.log = filepath.Join(bin, "calls")
	statusFile := filepath.Join(bin, "status.json")
	if err := os.WriteFile(statusFile, []byte(statusJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	prList := `[{"number":7,"title":"PR 7","state":"OPEN","isDraft":true,"headRefName":"claude/pr-7","headRefOid":"` + f.head + `","url":"https://github.com/o/r/pull/7"}]`
	prView := `{"number":7,"headRefOid":"` + f.head + `","url":"https://github.com/o/r/pull/7"}`
	script := "#!/bin/sh\n" +
		"{ printf '%s' \"$GH_TOKEN\"; for a in \"$@\"; do printf '|%s' \"$a\"; done; printf '\\n'; } >> '" + f.log + "'\n" +
		"case \"$*\" in\n" +
		"'pr list --state open'*) printf '%s' '" + prList + "';;\n" +
		"'pr comment'*) echo 'https://github.com/o/r/pull/7#issuecomment-42';;\n" +
		"'pr view claude/pr-7 --json number,headRefOid,url'*) printf '%s' '" + prView + "';;\n" +
		"'pr view 7 --json comments'*) printf '%s' '{\"comments\":[]}';;\n" +
		"'api -X POST '*) echo 'response body secret-body' >&2; exit \"${FAKE_POST_EXIT:-0}\";;\n" +
		"'api repos/'*) cat '" + statusFile + "';;\n" +
		"*) printf '[]';;\nesac\n"
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return f
}

func (f statusFixture) saveHandoff(t *testing.T, at time.Time) {
	t.Helper()
	if err := state.Save(f.wt, &state.State{Handoffs: map[string]*state.Handoff{
		"claude@claude/pr-7": {Goal: "g", Agent: "claude", Branch: "claude/pr-7", CheckpointSHA: f.head[:7], UpdatedAt: at},
	}}); err != nil {
		t.Fatal(err)
	}
}

// apiCalls returns the logged gh api calls, each as token|arg|arg....
func (f statusFixture) apiCalls(t *testing.T) []string {
	t.Helper()
	data, err := os.ReadFile(f.log)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	var out []string
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if strings.Contains(line, "|api|") {
			out = append(out, line)
		}
	}
	return out
}

// syncJSON runs ateam state sync from the main checkout and returns its exit code, decoded stdout, and stderr.
func syncJSON(t *testing.T, root string) (int, map[string]any, string) {
	t.Helper()
	var out string
	code, stderr := captureBenchError(t, func() int {
		c, o := captureStdout(t, func() int { return runStateSync(root, nil) })
		out = o
		return c
	})
	var decoded map[string]any
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("sync stdout is not JSON: %v\n%s\nstderr: %s", err, out, stderr)
	}
	return code, decoded, stderr
}

// spec: openspec/changes/commit-statuses-302/specs/harness/spec.md#Requirement: Informational commit statuses on the PR head
func TestStateSyncCommitStatuses(t *testing.T) {
	at := time.Date(2026, 11, 7, 18, 42, 0, 0, time.UTC)
	handoffDesc := func(f statusFixture) string { return "handoff at " + f.head[:7] + ", 2026-11-07T18:42Z" }

	t.Run("posted", func(t *testing.T) {
		f := newStatusFixture(t, `{"statuses":[]}`)
		f.saveHandoff(t, at)
		code, out, stderr := syncJSON(t, f.root)
		if code != 0 || stderr != "" {
			t.Fatalf("sync = %d, stderr %q; want 0 and silent", code, stderr)
		}
		if want := map[string]any{"posted": 1.0, "unchanged": 0.0, "errors": 0.0}; !reflect.DeepEqual(out["statuses"], want) {
			t.Fatalf("statuses = %v, want %v", out["statuses"], want)
		}
		want := []string{
			"|api|repos/o/r/commits/" + f.head + "/status?per_page=100",
			"|api|-X|POST|repos/o/r/statuses/" + f.head + "|-f|state=success|-f|context=ateam/handoff|-f|description=" + handoffDesc(f) + "|-f|target_url=https://github.com/o/r/pull/7",
		}
		if got := f.apiCalls(t); !reflect.DeepEqual(got, want) {
			t.Fatalf("gh api calls =\n%q\nwant\n%q", got, want)
		}
	})

	t.Run("unchanged", func(t *testing.T) {
		f := newStatusFixture(t, "")
		f.saveHandoff(t, at)
		if err := os.WriteFile(filepath.Join(filepath.Dir(f.log), "status.json"),
			[]byte(`{"statuses":[{"context":"ateam/handoff","state":"success","description":"`+handoffDesc(f)+`"}]}`), 0o644); err != nil {
			t.Fatal(err)
		}
		code, out, stderr := syncJSON(t, f.root)
		if code != 0 || stderr != "" {
			t.Fatalf("sync = %d, stderr %q; want 0 and silent", code, stderr)
		}
		if want := map[string]any{"posted": 0.0, "unchanged": 1.0, "errors": 0.0}; !reflect.DeepEqual(out["statuses"], want) {
			t.Fatalf("statuses = %v, want %v", out["statuses"], want)
		}
		if got := f.apiCalls(t); len(got) != 1 {
			t.Fatalf("an unchanged status was written again: %q", got)
		}
	})

	t.Run("disabled", func(t *testing.T) {
		f := newStatusFixture(t, `{"statuses":[]}`)
		f.saveHandoff(t, at)
		t.Setenv("ATEAM_COMMIT_STATUSES", "0")
		code, out, stderr := syncJSON(t, f.root)
		if code != 0 || stderr != "" {
			t.Fatalf("sync = %d, stderr %q; want 0 and silent", code, stderr)
		}
		if _, ok := out["statuses"]; ok {
			t.Fatalf("disabled sync reported statuses: %v", out)
		}
		if got := f.apiCalls(t); len(got) != 0 {
			t.Fatalf("disabled sync read or wrote a status: %q", got)
		}
		// Activation: the sync did reach the PR whose statuses it skipped.
		if out["synced"] != 1.0 {
			t.Fatalf("synced = %v, want the one PR", out["synced"])
		}
	})

	t.Run("write error keeps exit 0", func(t *testing.T) {
		f := newStatusFixture(t, `{"statuses":[]}`)
		f.saveHandoff(t, at)
		t.Setenv("ATEAM_TOKEN_WRITE", "fake-write-secret")
		t.Setenv("FAKE_POST_EXIT", "1")
		code, out, stderr := syncJSON(t, f.root)
		if code != 0 {
			t.Fatalf("sync exited %d on a status write error, want 0", code)
		}
		want := "ateam state sync: warning: commit statuses: ateam/handoff on " + f.head[:7] + ": gh exit 1 (posted 0, unchanged 0, errors 1)\n"
		if stderr != want {
			t.Fatalf("stderr = %q, want one warning line %q", stderr, want)
		}
		if want := map[string]any{"posted": 0.0, "unchanged": 0.0, "errors": 1.0}; !reflect.DeepEqual(out["statuses"], want) {
			t.Fatalf("statuses = %v, want %v", out["statuses"], want)
		}
		warnings := statusWriteWarnings(f.root)
		if len(warnings) != 1 || !strings.Contains(warnings[0], "ateam/handoff on "+f.head[:7]+": gh exit 1") || strings.Contains(warnings[0], "secret") {
			t.Fatalf("doctor warnings = %q, want the cached error without a token or body", warnings)
		}
	})
}

// spec: openspec/changes/commit-statuses-302/specs/harness/spec.md#Requirement: Informational commit statuses on the PR head
func TestStateHandoffCommitStatuses(t *testing.T) {
	t.Run("write error keeps exit 0", func(t *testing.T) {
		f := newStatusFixture(t, `{"statuses":[]}`)
		t.Setenv("FAKE_POST_EXIT", "1")
		code, stderr := captureBenchError(t, func() int {
			c, _ := captureStdout(t, func() int {
				return runStateHandoff(f.wt, []string{"--agent", "claude", "--session", "S", "--goal", "g"})
			})
			return c
		})
		if code != 0 {
			t.Fatalf("handoff exited %d on a status write error, want 0", code)
		}
		want := "ateam state handoff: warning: commit statuses: ateam/handoff on " + f.head[:7] + ": gh exit 1\n"
		if stderr != want {
			t.Fatalf("stderr = %q, want one warning line %q", stderr, want)
		}
		calls := f.apiCalls(t)
		if len(calls) != 2 || !strings.Contains(calls[1], "|target_url=https://github.com/o/r/pull/7#issuecomment-42") {
			t.Fatalf("gh api calls = %q, want one read and one write linking the posted comment", calls)
		}
	})

	t.Run("no-post skips statuses", func(t *testing.T) {
		f := newStatusFixture(t, `{"statuses":[]}`)
		code, stderr := captureBenchError(t, func() int {
			c, _ := captureStdout(t, func() int {
				return runStateHandoff(f.wt, []string{"--agent", "claude", "--session", "S", "--goal", "g", "--no-post"})
			})
			return c
		})
		if code != 0 || stderr != "" {
			t.Fatalf("handoff --no-post = %d, stderr %q", code, stderr)
		}
		if data, _ := os.ReadFile(f.log); len(data) != 0 {
			t.Fatalf("handoff --no-post called gh: %s", data)
		}
	})

	t.Run("disabled skips statuses but posts the comment", func(t *testing.T) {
		f := newStatusFixture(t, `{"statuses":[]}`)
		t.Setenv("ATEAM_COMMIT_STATUSES", "0")
		if code := runStateHandoff(f.wt, []string{"--agent", "claude", "--session", "S", "--goal", "g"}); code != 0 {
			t.Fatalf("handoff = %d", code)
		}
		data, _ := os.ReadFile(f.log)
		if !strings.Contains(string(data), "|pr|comment|claude/pr-7|") || len(f.apiCalls(t)) != 0 {
			t.Fatalf("want the comment and no status call, got:\n%s", data)
		}
	})
}

// spec: openspec/changes/commit-statuses-302/specs/harness/spec.md#Requirement: Informational commit statuses on the PR head
func TestDoctorStatusWriteWarningOnly(t *testing.T) {
	dir := t.TempDir()
	gitRun(t, dir, "init", "-q", "-b", "main")
	doctor := func() (passed bool, violations, warnings []string) {
		t.Helper()
		_, out := captureStdout(t, func() int { return runDoctor(dir, []string{"--check"}) })
		var report struct {
			Passed     bool     `json:"passed"`
			Violations []string `json:"violations"`
			Warnings   []string `json:"warnings"`
		}
		if err := json.Unmarshal([]byte(out), &report); err != nil {
			t.Fatalf("doctor output: %v\n%s", err, out)
		}
		for _, v := range report.Violations {
			if !strings.Contains(v, "latency") { // wall-clock, so not part of this comparison
				violations = append(violations, v)
			}
		}
		return report.Passed, violations, report.Warnings
	}
	hasStatusWarning := func(ws []string) bool {
		for _, w := range ws {
			if strings.HasPrefix(w, "last commit status write failed (ateam/handoff on abc1234: HTTP 403)") {
				return true
			}
		}
		return false
	}
	passedBefore, violationsBefore, warningsBefore := doctor()
	if hasStatusWarning(warningsBefore) {
		t.Fatalf("status warning present before any error: %q", warningsBefore)
	}
	if err := state.Save(dir, &state.State{StatusWriteError: "ateam/handoff on abc1234: HTTP 403"}); err != nil {
		t.Fatal(err)
	}
	passed, violations, warnings := doctor()
	if !hasStatusWarning(warnings) {
		t.Fatalf("doctor warnings lack the cached status error: %q", warnings)
	}
	if passed != passedBefore || !reflect.DeepEqual(violations, violationsBefore) {
		t.Fatalf("the status warning changed doctor's verdict: passed %t -> %t, violations %q -> %q", passedBefore, passed, violationsBefore, violations)
	}
	if err := state.Save(dir, &state.State{}); err != nil {
		t.Fatal(err)
	}
	if _, _, warnings := doctor(); hasStatusWarning(warnings) {
		t.Fatalf("status warning still present after a clean publication cleared it: %q", warnings)
	}
}
