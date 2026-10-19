package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
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
