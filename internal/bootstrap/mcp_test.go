package bootstrap

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
