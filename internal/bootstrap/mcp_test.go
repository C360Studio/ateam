package bootstrap

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

// spec: openspec/specs/harness/spec.md#Requirement: Client-readable MCP registration
func TestMCPRegistrationFiles(t *testing.T) {
	root := t.TempDir()
	if _, err := InitRepo(context.Background(), root, Options{}); err != nil {
		t.Fatal(err)
	}
	// Client-owned locations, not strings found in an unrelated hooks file.
	data, err := os.ReadFile(filepath.Join(root, ".codex/config.toml"))
	if err != nil {
		t.Fatalf("Codex client registration missing: %v", err)
	}
	if !strings.Contains(string(data), "[mcp_servers.ateam]") {
		t.Fatal("Codex registration table missing")
	}
	for _, rel := range []string{".mcp.json", ".gemini/settings.json"} {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatal(err)
		}
		var cfg struct {
			Servers map[string]struct {
				Command string
				Args    []string
			} `json:"mcpServers"`
		}
		if err := json.Unmarshal(data, &cfg); err != nil {
			t.Fatal(err)
		}
		got := cfg.Servers["ateam"]
		if got.Command != "ateam" || len(got.Args) != 1 || got.Args[0] != "mcp" {
			t.Fatalf("client registration missing in %s: %+v", rel, got)
		}
	}
	for _, rel := range []string{".codex/hooks.json", ".claude/settings.json"} {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatal(err)
		}
		var cfg map[string]any
		if err := json.Unmarshal(data, &cfg); err != nil {
			t.Fatal(err)
		}
		if _, ok := cfg["mcpServers"]; ok {
			t.Fatalf("misplaced MCP entry in %s", rel)
		}
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Client-readable MCP registration
func TestInitPreservesClientSettings(t *testing.T) {
	root := t.TempDir()
	// Independent model: every pre-existing client file must remain byte-identical,
	// including comments, unrelated settings and an owner's custom ateam command.
	originals := map[string]string{
		".codex/config.toml":    "# owner config\nmodel = 'custom'\n[mcp_servers.ateam]\ncommand = '/custom/ateam'\nargs = ['mcp']\n",
		".mcp.json":             "{\"mcpServers\":{\"other\":{\"command\":\"custom\"}}}\n",
		".gemini/settings.json": "{\"theme\":\"custom\",\"mcpServers\":{}}\n",
		".claude/settings.json": "{\"permissions\":{\"deny\":[\"Read(secret)\"]}}\n",
		".codex/hooks.json":     "{\"hooks\":{}}\n",
	}
	for rel, data := range originals {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, force := range []bool{false, false, true, false} {
		if _, err := InitRepo(context.Background(), root, Options{Force: force}); err != nil {
			t.Fatal(err)
		}
		for rel, want := range originals {
			got, err := os.ReadFile(filepath.Join(root, rel))
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != want {
				t.Fatalf("client settings overwritten: %s force=%v", rel, force)
			}
		}
	}
	requireViolation(t, CheckParity(root), ".mcp.json", ".gemini/settings.json")
}

// spec: openspec/specs/harness/spec.md#Requirement: Client-readable MCP registration
func TestMCPConfigValidation(t *testing.T) {
	cases := []struct {
		name, file, body string
		valid            bool
	}{
		{"Codex quoted table and multiline array", ".codex/config.toml", "model = 'custom'\n[mcp_servers.\"ateam\"]\ncommand = '/opt/bin/ateam'\nargs = [\n 'mcp', # comment\n]\n", true},
		{"Codex comment is not registration", ".codex/config.toml", "# [mcp_servers.ateam]\n# command = 'ateam'\n# args = ['mcp']\n", false},
		{"Codex unrelated table", ".codex/config.toml", "[other.ateam]\ncommand = 'ateam'\nargs = ['mcp']\n", false},
		{"Codex disabled", ".codex/config.toml", "[mcp_servers.ateam]\ncommand = 'ateam'\nargs = ['mcp']\nenabled = false\n", false},
		{"Codex wrong argument", ".codex/config.toml", "[mcp_servers.ateam]\ncommand = 'ateam'\nargs = ['serve']\n", false},
		{"Codex invalid TOML", ".codex/config.toml", "[mcp_servers.ateam\n", false},
		{"Claude correct nested server", ".mcp.json", `{"unrelated":true,"mcpServers":{"ateam":{"command":"/opt/bin/ateam","args":["mcp"],"type":"stdio"},"other":{"command":"anything"}}}`, true},
		{"Claude misleading strings", ".mcp.json", `{"ateam":"mcp","mcpServers":{}}`, false},
		{"Claude wrong server", ".mcp.json", `{"mcpServers":{"other":{"command":"ateam","args":["mcp"]}}}`, false},
		{"Claude wrong command", ".mcp.json", `{"mcpServers":{"ateam":{"command":"not-ateam","args":["mcp"]}}}`, false},
		{"Claude bad args type", ".mcp.json", `{"mcpServers":{"ateam":{"command":"ateam","args":"mcp"}}}`, false},
		{"Claude extra args", ".mcp.json", `{"mcpServers":{"ateam":{"command":"ateam","args":["mcp","--bad"]}}}`, false},
		{"Claude remote transport", ".mcp.json", `{"mcpServers":{"ateam":{"command":"ateam","args":["mcp"],"type":"http","url":"https://example.invalid"}}}`, false},
		{"Gemini wrong nesting", ".gemini/settings.json", `{"hooks":{"mcpServers":{"ateam":{"command":"ateam","args":["mcp"]}}}}`, false},
		{"Gemini httpUrl", ".gemini/settings.json", `{"mcpServers":{"ateam":{"command":"ateam","args":["mcp"],"httpUrl":"https://example.invalid"}}}`, false},
		{"Gemini malformed JSON", ".gemini/settings.json", `{`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			if _, err := InitRepo(context.Background(), root, Options{}); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, tc.file), []byte(tc.body), 0o600); err != nil {
				t.Fatal(err)
			}
			got := CheckParity(root)
			if tc.valid {
				if len(got) != 0 {
					t.Fatalf("valid registration rejected: %v", got)
				}
			} else {
				requireViolation(t, got, tc.file)
			}
		})
	}
	for _, rel := range []string{".codex/config.toml", ".mcp.json"} {
		t.Run("missing "+rel, func(t *testing.T) {
			root := t.TempDir()
			if _, err := InitRepo(context.Background(), root, Options{}); err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(filepath.Join(root, rel)); err != nil {
				t.Fatal(err)
			}
			requireViolation(t, CheckParity(root), rel)
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Client-readable MCP registration
func TestInitPreservesConfigSymlink(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(t.TempDir(), "owner.json")
	if err := os.Symlink(target, filepath.Join(root, ".mcp.json")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := InitRepo(context.Background(), root, Options{Force: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("dangling settings symlink was followed: %v", err)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Default client context compaction limits
func TestClientContextCompactionLimits(t *testing.T) {
	root := t.TempDir()
	if _, err := InitRepo(context.Background(), root, Options{}); err != nil {
		t.Fatal(err)
	}

	var claudeCfg struct {
		AutoCompactWindow int `json:"autoCompactWindow"`
	}
	claudeBytes, err := os.ReadFile(filepath.Join(root, ".claude/settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(claudeBytes, &claudeCfg); err != nil {
		t.Fatal(err)
	}
	if claudeCfg.AutoCompactWindow != 500000 {
		t.Fatalf("Claude autoCompactWindow = %d, want 500000", claudeCfg.AutoCompactWindow)
	}

	var codexCfg struct {
		ContextWindow    int64 `toml:"model_context_window"`
		AutoCompactLimit int64 `toml:"model_auto_compact_token_limit"`
	}
	codexBytes, err := os.ReadFile(filepath.Join(root, ".codex/config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := toml.Unmarshal(codexBytes, &codexCfg); err != nil {
		t.Fatal(err)
	}
	if codexCfg.ContextWindow != 500000 || codexCfg.AutoCompactLimit != 450000 {
		t.Fatalf("Codex context limits = (%d, %d), want (500000, 450000)", codexCfg.ContextWindow, codexCfg.AutoCompactLimit)
	}

	var geminiCfg struct {
		Model struct {
			ChatCompression struct {
				Threshold float64 `json:"contextPercentageThreshold"`
			} `json:"chatCompression"`
		} `json:"model"`
	}
	geminiBytes, err := os.ReadFile(filepath.Join(root, ".gemini/settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(geminiBytes, &geminiCfg); err != nil {
		t.Fatal(err)
	}
	if geminiCfg.Model.ChatCompression.Threshold != 0.5 {
		t.Fatalf("Gemini contextPercentageThreshold = %v, want 0.5", geminiCfg.Model.ChatCompression.Threshold)
	}

	cases := []struct {
		name  string
		file  string
		edit  func(string) string
		valid bool
	}{
		{
			name: "Claude lower boundary 100000",
			file: ".claude/settings.json",
			edit: func(s string) string {
				return strings.Replace(s, `"autoCompactWindow": 500000`, `"autoCompactWindow": 100000`, 1)
			},
			valid: true,
		},
		{
			name: "Claude upper boundary 1000000",
			file: ".claude/settings.json",
			edit: func(s string) string {
				return strings.Replace(s, `"autoCompactWindow": 500000`, `"autoCompactWindow": 1000000`, 1)
			},
			valid: true,
		},
		{
			name: "Claude below lower boundary 99999",
			file: ".claude/settings.json",
			edit: func(s string) string {
				return strings.Replace(s, `"autoCompactWindow": 500000`, `"autoCompactWindow": 99999`, 1)
			},
			valid: false,
		},
		{
			name: "Claude above upper boundary 1000001",
			file: ".claude/settings.json",
			edit: func(s string) string {
				return strings.Replace(s, `"autoCompactWindow": 500000`, `"autoCompactWindow": 1000001`, 1)
			},
			valid: false,
		},
		{
			name: "Claude fractional value rejected",
			file: ".claude/settings.json",
			edit: func(s string) string {
				return strings.Replace(s, `"autoCompactWindow": 500000`, `"autoCompactWindow": 500000.5`, 1)
			},
			valid: false,
		},
		{
			name: "Codex auto compact equals context window",
			file: ".codex/config.toml",
			edit: func(s string) string {
				return strings.Replace(s, "model_auto_compact_token_limit = 450000", "model_auto_compact_token_limit = 500000", 1)
			},
			valid: true,
		},
		{
			name: "Codex auto compact exceeds context window",
			file: ".codex/config.toml",
			edit: func(s string) string {
				return strings.Replace(s, "model_auto_compact_token_limit = 450000", "model_auto_compact_token_limit = 500001", 1)
			},
			valid: false,
		},
		{
			name: "Codex zero context window rejected",
			file: ".codex/config.toml",
			edit: func(s string) string {
				return strings.Replace(s, "model_context_window = 500000", "model_context_window = 0", 1)
			},
			valid: false,
		},
		{
			name: "Gemini upper boundary 1.0",
			file: ".gemini/settings.json",
			edit: func(s string) string {
				return strings.Replace(s, `"contextPercentageThreshold": 0.5`, `"contextPercentageThreshold": 1.0`, 1)
			},
			valid: true,
		},
		{
			name: "Gemini zero threshold rejected",
			file: ".gemini/settings.json",
			edit: func(s string) string {
				return strings.Replace(s, `"contextPercentageThreshold": 0.5`, `"contextPercentageThreshold": 0`, 1)
			},
			valid: false,
		},
		{
			name: "Gemini threshold above 1 rejected",
			file: ".gemini/settings.json",
			edit: func(s string) string {
				return strings.Replace(s, `"contextPercentageThreshold": 0.5`, `"contextPercentageThreshold": 1.01`, 1)
			},
			valid: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sub := t.TempDir()
			if _, err := InitRepo(context.Background(), sub, Options{}); err != nil {
				t.Fatal(err)
			}
			p := filepath.Join(sub, tc.file)
			orig, err := os.ReadFile(p)
			if err != nil {
				t.Fatal(err)
			}
			updated := tc.edit(string(orig))
			if updated == string(orig) {
				t.Fatalf("edit did not modify %s", tc.file)
			}
			if err := os.WriteFile(p, []byte(updated), 0o600); err != nil {
				t.Fatal(err)
			}
			got := CheckParity(sub)
			if tc.valid {
				if len(got) != 0 {
					t.Fatalf("valid context config rejected: %v", got)
				}
			} else {
				requireViolation(t, got, tc.file)
			}
		})
	}
}
