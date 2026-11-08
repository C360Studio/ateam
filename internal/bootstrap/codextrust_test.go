package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// spec: openspec/specs/harness/spec.md#Requirement: Doctor reports Codex project hook trust
func TestCodexHookTrust(t *testing.T) {
	trust := func(path string, events ...string) string {
		var b strings.Builder
		for _, e := range events {
			b.WriteString("[hooks.state.\"" + path + ":" + e + ":0:0\"]\ntrusted_hash = \"sha256:00\"\n")
		}
		return b.String()
	}
	all := []string{"session_start", "user_prompt_submit", "pre_compact", "stop"}
	const advice = "; run codex in this checkout and approve the project hooks"
	for _, tc := range []struct {
		name        string
		config      func(hooksPath string) *string // nil: no config.toml
		hooks       string                         // replaces .codex/hooks.json when set
		userScope   bool
		wantStatus  string
		wantMissing string // "": no warning
	}{
		{name: "all four events trusted", config: func(p string) *string { return ptr(trust(p, all...)) }, wantStatus: "trusted"},
		{name: "no trust entries", config: func(string) *string { return ptr("model = \"o4\"\n") },
			wantStatus: "untrusted", wantMissing: "session_start, user_prompt_submit, pre_compact, stop"},
		{name: "partial trust", config: func(p string) *string { return ptr(trust(p, "session_start", "stop")) },
			wantStatus: "untrusted", wantMissing: "user_prompt_submit, pre_compact"},
		{name: "only the main checkout trusted, seen from a worktree",
			config: func(p string) *string {
				main := strings.Replace(p, string(filepath.Separator)+".codex", string(filepath.Separator)+"main"+string(filepath.Separator)+".codex", 1)
				return ptr(trust(main, all...))
			},
			wantStatus: "untrusted", wantMissing: "session_start, user_prompt_submit, pre_compact, stop"},
		{name: "event name as a prefix of another key", config: func(p string) *string {
			return ptr(trust(p, "session_start_extra", "user_prompt_submit", "pre_compact", "stop"))
		}, wantStatus: "untrusted", wantMissing: "session_start"},
		{name: "no config file", config: func(string) *string { return nil }, wantStatus: "not checked: no "},
		{name: "invalid TOML", config: func(string) *string { return ptr("[hooks.state\n") }, wantStatus: "not checked: "},
		{name: "codex hooks only at user scope", hooks: `{}`, userScope: true,
			config: func(string) *string { return ptr("") }, wantStatus: "not checked: .codex/hooks.json does not carry the ateam hooks"},
		{name: "codex hooks failing", hooks: `{}`,
			config: func(string) *string { return ptr("") }, wantStatus: "not checked: .codex/hooks.json does not carry the ateam hooks"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir, home, codexHome := t.TempDir(), t.TempDir(), t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("CODEX_HOME", codexHome)
			t.Setenv(hookScopeEnv, "")
			if _, err := InitRepo(context.Background(), dir, Options{}); err != nil {
				t.Fatalf("InitRepo: %v", err)
			}
			if tc.hooks != "" {
				projectHooks, err := os.ReadFile(filepath.Join(dir, ".codex", "hooks.json"))
				if err != nil {
					t.Fatal(err)
				}
				writeFiles(t, dir, map[string]string{".codex/hooks.json": tc.hooks})
				if tc.userScope {
					t.Setenv(hookScopeEnv, "user")
					writeFiles(t, home, map[string]string{".codex/hooks.json": string(projectHooks)})
				}
			}
			hooksPath := filepath.Join(dir, ".codex", "hooks.json")
			if c := tc.config(hooksPath); c != nil {
				writeFiles(t, codexHome, map[string]string{"config.toml": *c})
			}

			status, warnings := CodexHookTrust(dir)
			if !strings.HasPrefix(status, tc.wantStatus) || (!strings.HasPrefix(tc.wantStatus, "not checked") && status != tc.wantStatus) {
				t.Fatalf("status = %q, want %q", status, tc.wantStatus)
			}
			var want []string
			if tc.wantMissing != "" {
				want = []string{".codex/hooks.json: Codex has not trusted " + tc.wantMissing + " for " + hooksPath + advice}
			}
			if !reflect.DeepEqual(warnings, want) {
				t.Fatalf("warnings = %q, want %q", warnings, want)
			}
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Doctor reports Codex project hook trust
func TestCodexConfigPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", "")
	if got, _ := codexConfigPath(); got != filepath.Join(home, ".codex", "config.toml") {
		t.Fatalf("CODEX_HOME unset: %q", got)
	}
	t.Setenv("CODEX_HOME", "/opt/codex")
	if got, _ := codexConfigPath(); got != filepath.Join("/opt/codex", "config.toml") {
		t.Fatalf("CODEX_HOME set: %q", got)
	}
}
