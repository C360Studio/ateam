package bootstrap

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/pelletier/go-toml/v2"
)

// CodexHookTrust reports whether Codex has trusted this checkout's project hooks (#311). Codex runs
// .codex/hooks.json only after the user approves it once, and records each approval in its config.toml as a
// [hooks.state."<absolute hooks.json path>:<event>:<i>:<j>"] table, so every checkout and worktree needs its own.
// An untrusted checkout yields a warning, never a violation: CI and Claude-only machines have no Codex config.
func CodexHookTrust(root string) (status string, warnings []string) {
	if HookSources(root)["codex"] != ".codex/hooks.json" {
		return "not checked: .codex/hooks.json does not carry the ateam hooks", nil
	}
	config, err := codexConfigPath()
	if err != nil {
		return "not checked: cannot resolve the home directory", nil
	}
	data, err := os.ReadFile(config)
	if err != nil {
		return "not checked: no " + config, nil
	}
	var parsed struct {
		Hooks struct {
			State map[string]any `toml:"state"`
		} `toml:"hooks"`
	}
	if err := toml.Unmarshal(data, &parsed); err != nil {
		return "not checked: " + config + ": invalid TOML", nil
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "not checked: " + err.Error(), nil
	}
	// Matched as written, without EvalSymlinks: a checkout reached through a symlink reads untrusted (#311 OQ2).
	hooksPath := filepath.Join(abs, ".codex", "hooks.json")
	// shortcut: trust is the key's presence only; trusted_hash is not recomputed, so an edit that revoked trust
	// still reads trusted. Compare the hash when Codex documents which bytes it covers.
	var missing []string
	for _, e := range claudeHookEvents {
		event := snakeCase(e.native)
		prefix := hooksPath + ":" + event + ":"
		found := false
		for key := range parsed.Hooks.State {
			if strings.HasPrefix(key, prefix) {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, event)
		}
	}
	if len(missing) == 0 {
		return "trusted", nil
	}
	return "untrusted", []string{fmt.Sprintf(
		".codex/hooks.json: Codex has not trusted %s for %s; run codex in this checkout and approve the project hooks",
		strings.Join(missing, ", "), hooksPath)}
}

// codexConfigPath is Codex's config.toml: under $CODEX_HOME when set, else ~/.codex.
func codexConfigPath() (string, error) {
	if dir := os.Getenv("CODEX_HOME"); dir != "" {
		return filepath.Join(dir, "config.toml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codex", "config.toml"), nil
}

// snakeCase maps a native hook event (UserPromptSubmit) to the form Codex keys trust by (user_prompt_submit).
func snakeCase(s string) string {
	var b strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				b.WriteByte('_')
			}
			r = unicode.ToLower(r)
		}
		b.WriteRune(r)
	}
	return b.String()
}
