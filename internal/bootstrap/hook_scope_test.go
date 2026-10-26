package bootstrap

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// pinnedClaudeSettings stands in for an adopter's byte-pinned .claude/settings.json with no ateam hooks (#128).
const pinnedClaudeSettings = `{"permissions": {"deny": ["Bash(rm -rf:*)"]}}`

// hookScopeRepo is a freshly initialised repository whose project .claude/settings.json carries no ateam
// hooks, with HOME pointed at an empty temporary directory. It returns the repository, the home directory,
// and the init-written Claude settings that do carry the hooks.
func hookScopeRepo(t *testing.T) (dir, home, claudeHooks string) {
	t.Helper()
	dir, home = t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	if _, err := InitRepo(context.Background(), dir, Options{}); err != nil {
		t.Fatalf("InitRepo: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, ".claude", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	writeFiles(t, dir, map[string]string{".claude/settings.json": pinnedClaudeSettings})
	return dir, home, string(data)
}

func readRepoFile(t *testing.T, dir, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// spec: openspec/specs/harness/spec.md#Requirement: Hooks registered outside the project settings
func TestHookScopeUserPasses(t *testing.T) {
	cases := []struct {
		name  string
		setup func(t *testing.T, dir, home, claudeHooks string)
		want  map[string]string
	}{
		{name: "Claude hooks in the user settings",
			setup: func(t *testing.T, _, home, claudeHooks string) {
				writeFiles(t, home, map[string]string{".claude/settings.json": claudeHooks})
			},
			want: map[string]string{"claude": "~/.claude/settings.json"}},
		{name: "Claude hooks in the local settings",
			setup: func(t *testing.T, dir, _, claudeHooks string) {
				writeFiles(t, dir, map[string]string{".claude/settings.local.json": claudeHooks})
			},
			want: map[string]string{"claude": ".claude/settings.local.json"}},
		{name: "local settings win over the user settings",
			setup: func(t *testing.T, dir, home, claudeHooks string) {
				writeFiles(t, dir, map[string]string{".claude/settings.local.json": claudeHooks})
				writeFiles(t, home, map[string]string{".claude/settings.json": claudeHooks})
			},
			want: map[string]string{"claude": ".claude/settings.local.json"}},
		{name: "project hooks still win",
			setup: func(t *testing.T, dir, home, claudeHooks string) {
				writeFiles(t, dir, map[string]string{".claude/settings.json": claudeHooks})
				writeFiles(t, home, map[string]string{".claude/settings.json": "{not json"})
			},
			want: map[string]string{"claude": ".claude/settings.json"}},
		{name: "Codex hooks in the user hooks file",
			setup: func(t *testing.T, dir, home, claudeHooks string) {
				writeFiles(t, home, map[string]string{
					".claude/settings.json": claudeHooks,
					".codex/hooks.json":     readRepoFile(t, dir, ".codex/hooks.json"),
				})
				if err := os.Remove(filepath.Join(dir, ".codex", "hooks.json")); err != nil {
					t.Fatal(err)
				}
			},
			want: map[string]string{"claude": "~/.claude/settings.json", "codex": "~/.codex/hooks.json"}},
		{name: "Gemini hooks in the user settings",
			setup: func(t *testing.T, dir, home, claudeHooks string) {
				gemini := readRepoFile(t, dir, ".gemini/settings.json")
				writeFiles(t, home, map[string]string{".claude/settings.json": claudeHooks, ".gemini/settings.json": gemini})
				// The project file keeps contextFileName and the MCP registration, and loses the hooks.
				writeFiles(t, dir, map[string]string{".gemini/settings.json": strings.ReplaceAll(gemini, "ateam hook --event", "ateam hook --evnt")})
			},
			want: map[string]string{"claude": "~/.claude/settings.json", "gemini": "~/.gemini/settings.json"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir, home, claudeHooks := hookScopeRepo(t)
			t.Setenv(hookScopeEnv, "user")
			c.setup(t, dir, home, claudeHooks)
			if v := CheckParity(dir); len(v) > 0 {
				t.Fatalf("CheckParity:\n- %s", strings.Join(v, "\n- "))
			}
			got := HookSources(dir)
			want := map[string]string{"claude": ".claude/settings.json", "codex": ".codex/hooks.json", "gemini": ".gemini/settings.json", "agy": ".agents/hooks.json"}
			for agent, src := range c.want {
				want[agent] = src
			}
			if len(got) != len(want) {
				t.Errorf("HookSources = %q, want %q", got, want)
			}
			for agent, src := range want {
				if got[agent] != src {
					t.Errorf("HookSources[%s] = %q, want %q", agent, got[agent], src)
				}
			}
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Hooks registered outside the project settings
func TestHookScopeSensitivity(t *testing.T) {
	const turnEndOnly = `{"hooks": {"Stop": [{"hooks": [{"type": "command", "command": "ateam hook --event turn-end --agent claude"}]}]}}`
	cases := []struct {
		name   string
		scope  string
		setup  func(t *testing.T, dir, home, claudeHooks string)
		want   []string
		absent string // a client that must not be reported as satisfied
	}{
		{name: "knob unset ignores the user settings", scope: "",
			setup: func(t *testing.T, _, home, claudeHooks string) {
				writeFiles(t, home, map[string]string{".claude/settings.json": claudeHooks})
			},
			want:   []string{".claude/settings.json: missing hook command for event session-start", ".claude/settings.json: missing hook command for event turn-end"},
			absent: "claude"},
		{name: "unknown knob value is a violation and reads no user file", scope: "global",
			setup: func(t *testing.T, _, home, claudeHooks string) {
				writeFiles(t, home, map[string]string{".claude/settings.json": claudeHooks})
			},
			want:   []string{"ATEAM_HOOK_SCOPE=global: unknown value", ".claude/settings.json: missing hook command for event session-start"},
			absent: "claude"},
		{name: "malformed user settings", scope: "user",
			setup: func(t *testing.T, _, home, claudeHooks string) {
				writeFiles(t, home, map[string]string{".claude/settings.json": strings.TrimSuffix(strings.TrimSpace(claudeHooks), "}")})
			},
			want:   []string{"~/.claude/settings.json: invalid JSON"},
			absent: "claude"},
		{name: "user settings missing one event", scope: "user",
			setup: func(t *testing.T, _, home, claudeHooks string) {
				writeFiles(t, home, map[string]string{".claude/settings.json": strings.ReplaceAll(claudeHooks, "ateam hook --event pre-compact", "ateam hook --event pre-kompact")})
			},
			want:   []string{"~/.claude/settings.json: missing hook command for event pre-compact"},
			absent: "claude"},
		{name: "user settings registering another agent", scope: "user",
			setup: func(t *testing.T, _, home, claudeHooks string) {
				writeFiles(t, home, map[string]string{".claude/settings.json": strings.ReplaceAll(claudeHooks, "--agent claude", "--agent codex")})
			},
			want:   []string{"~/.claude/settings.json: missing --agent claude in hook commands"},
			absent: "claude"},
		{name: "no user-scope file at all", scope: "user",
			setup:  func(*testing.T, string, string, string) {},
			want:   []string{".claude/settings.json: missing hook command for event session-start", ".claude/settings.local.json: missing hook config file", "~/.claude/settings.json: missing hook config file"},
			absent: "claude"},
		{name: "hooks split across the local and user settings", scope: "user",
			setup: func(t *testing.T, dir, home, claudeHooks string) {
				writeFiles(t, dir, map[string]string{".claude/settings.local.json": strings.ReplaceAll(claudeHooks, "ateam hook --event turn-end", "ateam hook --event turn-xnd")})
				writeFiles(t, home, map[string]string{".claude/settings.json": turnEndOnly})
			},
			want:   []string{".claude/settings.local.json: missing hook command for event turn-end", "~/.claude/settings.json: missing hook command for event session-start"},
			absent: "claude"},
		{name: "client without user-scope hooks", scope: "user",
			setup: func(t *testing.T, dir, home, claudeHooks string) {
				writeFiles(t, home, map[string]string{
					".claude/settings.json":     claudeHooks,
					".gemini/config/hooks.json": readRepoFile(t, dir, ".agents/hooks.json"),
				})
				writeFiles(t, dir, map[string]string{".agents/hooks.json": `{}`})
			},
			want:   []string{".agents/hooks.json: missing hook command for event session-start", ".agents/hooks.json: ATEAM_HOOK_SCOPE=user, but agy has no user-scope hooks doctor reads"},
			absent: "agy"},
		{name: "Gemini user hooks without contextFileName and no project file", scope: "user",
			setup: func(t *testing.T, dir, home, claudeHooks string) {
				gemini := strings.ReplaceAll(readRepoFile(t, dir, ".gemini/settings.json"), `"contextFileName"`, `"contextFile"`)
				writeFiles(t, home, map[string]string{".claude/settings.json": claudeHooks, ".gemini/settings.json": gemini})
				if err := os.Remove(filepath.Join(dir, ".gemini", "settings.json")); err != nil {
					t.Fatal(err)
				}
			},
			want: []string{".gemini/settings.json: missing contextFileName AGENTS.md"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir, home, claudeHooks := hookScopeRepo(t)
			t.Setenv(hookScopeEnv, c.scope)
			c.setup(t, dir, home, claudeHooks)
			requireViolation(t, CheckParity(dir), c.want...)
			if c.absent != "" {
				if src, ok := HookSources(dir)[c.absent]; ok {
					t.Errorf("HookSources[%s] = %q, want no satisfying file", c.absent, src)
				}
			}
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Hooks registered outside the project settings
func TestHookSourcesEmptyMarshalsAsObject(t *testing.T) {
	dir := t.TempDir()
	got, err := json.Marshal(map[string]any{"hook_sources": HookSources(dir)})
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"hook_sources":{}}`; string(got) != want {
		t.Fatalf("doctor hook_sources JSON = %s, want %s", got, want)
	}
}
