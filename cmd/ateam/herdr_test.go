package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// fakeHerdr records herdr invocations and answers `pane get` with a cwd.
type fakeHerdr struct {
	calls   [][]string
	paneCWD string
}

func (f *fakeHerdr) run(_ context.Context, bin string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, append([]string{bin}, args...))
	if len(args) >= 2 && args[0] == "pane" && args[1] == "get" {
		return json.Marshal(map[string]any{"result": map[string]any{"type": "pane_info", "pane": map[string]any{"pane_id": args[2], "cwd": f.paneCWD}}})
	}
	return nil, nil
}

func (f *fakeHerdr) toasts() []string {
	var out []string
	for _, c := range f.calls {
		if len(c) > 3 && c[1] == "notification" && c[2] == "show" {
			out = append(out, c[3])
		}
	}
	return out
}

func writeTasks(t *testing.T, repo, body string) {
	t.Helper()
	dir := filepath.Join(repo, "openspec", "changes", "demo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tasks.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func setHerdrEnv(t *testing.T, event, payload string) {
	t.Helper()
	t.Setenv("HERDR_ENV", "1")
	t.Setenv("HERDR_BIN_PATH", "/fake/herdr")
	t.Setenv("HERDR_PLUGIN_STATE_DIR", t.TempDir())
	t.Setenv("HERDR_PLUGIN_EVENT", event)
	t.Setenv("HERDR_PLUGIN_EVENT_JSON", payload)
}

func TestHerdrOnEventGateNeverInvokesHerdr(t *testing.T) {
	repo := t.TempDir()
	writeTasks(t, repo, "# Tasks\n\n- [ ] 1.1 Build is RED on main\n")
	payload := `{"event":"worktree.created","data":{"worktree":{"path":"` + repo + `","branch":"claude/x"}}}`
	tests := []struct {
		name    string
		herdr   string
		binPath string
	}{
		{name: "HERDR_ENV unset", herdr: "", binPath: "/fake/herdr"},
		{name: "HERDR_ENV not 1", herdr: "0", binPath: "/fake/herdr"},
		{name: "HERDR_BIN_PATH unset", herdr: "1", binPath: ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setHerdrEnv(t, "worktree.created", payload)
			t.Setenv("HERDR_ENV", tc.herdr)
			t.Setenv("HERDR_BIN_PATH", tc.binPath)
			f := &fakeHerdr{}
			var out bytes.Buffer
			if err := herdrOnEvent(context.Background(), f.run, &out); err != nil {
				t.Fatalf("herdrOnEvent: %v", err)
			}
			if len(f.calls) != 0 || out.Len() != 0 {
				t.Fatalf("gate leaked: calls=%v output=%q", f.calls, out.String())
			}
		})
	}
}

func TestHerdrOnEventToastsOnlyNewCaveats(t *testing.T) {
	repo := t.TempDir()
	writeTasks(t, repo, "# Tasks\n\n- [ ] 1.1 Build is RED on main\n- [ ] 1.2 Plain open task\n")
	setHerdrEnv(t, "pane.agent_status_changed", `{"event":"pane.agent_status_changed","data":{"pane_id":"p_1","workspace_id":"w_1","agent_status":"idle"}}`)
	f := &fakeHerdr{paneCWD: repo}
	var out bytes.Buffer

	if err := herdrOnEvent(context.Background(), f.run, &out); err != nil {
		t.Fatalf("first event: %v", err)
	}
	if got := f.toasts(); len(got) != 1 || got[0] != "ateam: demo RED" {
		t.Fatalf("first event toasts = %q, want [ateam: demo RED]", got)
	}

	if err := herdrOnEvent(context.Background(), f.run, &out); err != nil {
		t.Fatalf("repeat event: %v", err)
	}
	if got := f.toasts(); len(got) != 1 {
		t.Fatalf("repeat event toasted again: %q", got)
	}

	writeTasks(t, repo, "# Tasks\n\n- [ ] 1.1 Build is RED on main\n- [ ] 1.2 Plain open task\n  Hold: waiting on owner ruling\n")
	if err := herdrOnEvent(context.Background(), f.run, &out); err != nil {
		t.Fatalf("new caveat event: %v", err)
	}
	if got := f.toasts(); len(got) != 2 || got[1] != "ateam: demo BLOCKED" {
		t.Fatalf("new caveat toasts = %q, want a second toast [ateam: demo BLOCKED]", got)
	}
	if out.Len() != 0 {
		t.Fatalf("agent status events should print nothing, got %q", out.String())
	}
}

func TestHerdrOnEventSkipsNonAteamDir(t *testing.T) {
	plain := t.TempDir()
	setHerdrEnv(t, "worktree.created", `{"worktree":{"path":"`+plain+`","branch":"x"}}`)
	f := &fakeHerdr{}
	var out bytes.Buffer
	if err := herdrOnEvent(context.Background(), f.run, &out); err != nil {
		t.Fatalf("herdrOnEvent: %v", err)
	}
	if len(f.calls) != 0 || out.Len() != 0 {
		t.Fatalf("non-ateam dir should be ignored: calls=%v output=%q", f.calls, out.String())
	}
}

func TestHerdrOnEventWorktreeCreatedPrintsPickup(t *testing.T) {
	repo := t.TempDir()
	writeTasks(t, repo, "# Tasks\n\n- [ ] 1.1 Plain open task\n")
	setHerdrEnv(t, "worktree.created", `{"worktree":{"path":"`+repo+`","branch":"claude/x"}}`)
	f := &fakeHerdr{}
	var out bytes.Buffer
	if err := herdrOnEvent(context.Background(), f.run, &out); err != nil {
		t.Fatalf("herdrOnEvent: %v", err)
	}
	if !strings.HasPrefix(out.String(), "[ateam] mode=") || !strings.Contains(out.String(), "demo (0/1 tasks") {
		t.Fatalf("worktree.created should print the pickup summary, got %q", out.String())
	}
	if len(f.toasts()) != 0 {
		t.Fatalf("no caveats, no toasts; got %q", f.toasts())
	}
}

func TestHerdrOpenPane(t *testing.T) {
	t.Setenv("HERDR_ENV", "1")
	t.Setenv("HERDR_BIN_PATH", "/fake/herdr")
	t.Setenv("HERDR_WORKSPACE_ID", "w_3")
	t.Setenv("HERDR_PANE_ID", "")
	f := &fakeHerdr{}
	if err := herdrOpenPane(context.Background(), f.run, "pickup"); err != nil {
		t.Fatalf("open-pane pickup: %v", err)
	}
	want := "/fake/herdr plugin pane open --plugin ateam --entrypoint pickup --focus --workspace w_3"
	if len(f.calls) != 1 || strings.Join(f.calls[0], " ") != want {
		t.Fatalf("calls = %v, want [%s]", f.calls, want)
	}
	if err := herdrOpenPane(context.Background(), f.run, "rm -rf"); err == nil {
		t.Fatal("unknown pane id should be rejected")
	}
	t.Setenv("HERDR_ENV", "")
	if err := herdrOpenPane(context.Background(), f.run, "pickup"); err == nil || len(f.calls) != 1 {
		t.Fatalf("outside herdr open-pane should fail without invoking herdr: err=%v calls=%v", err, f.calls)
	}
}

func agentStatusEvent(t *testing.T) {
	t.Helper()
	t.Setenv("HERDR_PLUGIN_EVENT", "pane.agent_status_changed")
	t.Setenv("HERDR_PLUGIN_EVENT_JSON", `{"pane_id":"p_1","workspace_id":"w_1","agent_status":"idle"}`)
}

func TestHerdrOnEventStateDirFailuresToastAtMostOnce(t *testing.T) {
	tests := []struct {
		name      string
		stateDir  func(t *testing.T) string
		wantExact bool
	}{
		{name: "absent dir is created", stateDir: func(t *testing.T) string {
			return filepath.Join(t.TempDir(), "missing", "state")
		}, wantExact: true},
		{name: "unwritable dir", stateDir: func(t *testing.T) string {
			if os.Geteuid() == 0 {
				t.Skip("root ignores directory permissions")
			}
			dir := t.TempDir()
			if err := os.Chmod(dir, 0o500); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
			return dir
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := t.TempDir()
			writeTasks(t, repo, "# Tasks\n\n- [ ] 1.1 Build is RED on main\n")
			setHerdrEnv(t, "", "")
			agentStatusEvent(t)
			stateDir := tc.stateDir(t)
			t.Setenv("HERDR_PLUGIN_STATE_DIR", stateDir)
			f := &fakeHerdr{paneCWD: repo}
			for i := 0; i < 2; i++ {
				// The unwritable dir is expected to error; the toast count is the contract there.
				if err := herdrOnEvent(context.Background(), f.run, &bytes.Buffer{}); err != nil && tc.wantExact {
					t.Fatalf("event %d: %v", i, err)
				}
			}
			got := f.toasts()
			if len(got) > 1 || (tc.wantExact && len(got) != 1) {
				t.Fatalf("toasts across two events = %q, want at most one (exactly one: %v)", got, tc.wantExact)
			}
			if tc.wantExact {
				if _, err := os.Stat(filepath.Join(stateDir, "notified.json")); err != nil {
					t.Fatalf("record not written: %v", err)
				}
			}
		})
	}
}

// linkedWorktree returns a linked worktree of repo, built with git when available.
func linkedWorktree(t *testing.T, repo string) string {
	t.Helper()
	wt := filepath.Join(t.TempDir(), "wt")
	if _, err := exec.LookPath("git"); err == nil {
		git := func(args ...string) {
			cmd := exec.Command("git", append([]string{"-C", repo, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false"}, args...)...)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("git %v: %v\n%s", args, err, out)
			}
		}
		git("init", "-q")
		git("add", "-A")
		git("commit", "-q", "-m", "init")
		git("worktree", "add", "-q", "-b", "claude/x", wt)
		return wt
	}
	gitDir := filepath.Join(repo, ".git", "worktrees", "wt")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "commondir"), []byte("../..\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(wt, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wt, ".git"), []byte("gitdir: "+gitDir+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeTasks(t, wt, "# Tasks\n\n- [ ] 1.1 Build is RED on main\n")
	return wt
}

func TestHerdrOnEventKeysByRepository(t *testing.T) {
	repo := t.TempDir()
	writeTasks(t, repo, "# Tasks\n\n- [ ] 1.1 Build is RED on main\n")
	wt := linkedWorktree(t, repo)
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(repo, link); err != nil {
		t.Fatal(err)
	}
	setHerdrEnv(t, "", "")
	agentStatusEvent(t)
	f := &fakeHerdr{}
	for _, dir := range []string{repo, repo + "/", link, wt, repo} {
		f.paneCWD = dir
		if err := herdrOnEvent(context.Background(), f.run, &bytes.Buffer{}); err != nil {
			t.Fatalf("event from %s: %v", dir, err)
		}
	}
	if got := f.toasts(); len(got) != 1 {
		t.Fatalf("same caveat across spellings and worktrees toasted %d times: %q", len(got), got)
	}
}

func TestHerdrOnEventDuplicateCaveatTextToastsOnce(t *testing.T) {
	repo := t.TempDir()
	writeTasks(t, repo, "# Tasks\n\n- [ ] Build is RED\n- [ ] Build is RED\n")
	setHerdrEnv(t, "", "")
	agentStatusEvent(t)
	f := &fakeHerdr{paneCWD: repo}
	if err := herdrOnEvent(context.Background(), f.run, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	if got := f.toasts(); len(got) != 1 {
		t.Fatalf("duplicate caveat text toasted %d times: %q", len(got), got)
	}
}

func TestHerdrOnEventCorruptRecordIsReplaced(t *testing.T) {
	repo := t.TempDir()
	writeTasks(t, repo, "# Tasks\n\n- [ ] 1.1 Build is RED on main\n")
	setHerdrEnv(t, "", "")
	agentStatusEvent(t)
	if err := os.WriteFile(filepath.Join(os.Getenv("HERDR_PLUGIN_STATE_DIR"), "notified.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	f := &fakeHerdr{paneCWD: repo}
	for i := 0; i < 2; i++ {
		if err := herdrOnEvent(context.Background(), f.run, &bytes.Buffer{}); err != nil {
			t.Fatalf("event %d with corrupt record: %v", i, err)
		}
	}
	if got := f.toasts(); len(got) != 1 {
		t.Fatalf("corrupt record: toasts = %q, want exactly one", got)
	}
}

func TestHerdrOpenPaneUsesPaneCWD(t *testing.T) {
	t.Setenv("HERDR_ENV", "1")
	t.Setenv("HERDR_BIN_PATH", "/fake/herdr")
	t.Setenv("HERDR_WORKSPACE_ID", "")
	t.Setenv("HERDR_PANE_ID", "p_5")
	f := &fakeHerdr{paneCWD: "/work/repo"}
	if err := herdrOpenPane(context.Background(), f.run, "spec-queue"); err != nil {
		t.Fatalf("open-pane: %v", err)
	}
	want := "/fake/herdr plugin pane open --plugin ateam --entrypoint spec-queue --focus --cwd /work/repo"
	if len(f.calls) != 2 || strings.Join(f.calls[0], " ") != "/fake/herdr pane get p_5" || strings.Join(f.calls[1], " ") != want {
		t.Fatalf("calls = %v, want [pane get p_5] then [%s]", f.calls, want)
	}
}
