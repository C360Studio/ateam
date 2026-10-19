package bootstrap

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// Shared client files may contain permissions, secrets, or other servers. Never
// replace them with templates, even under --force; migration is a manual merge.
func isClientConfig(rel string) bool {
	switch rel {
	case ".codex/config.toml", ".codex/hooks.json", ".mcp.json", ".claude/settings.json", ".gemini/settings.json":
		return true
	default:
		return false
	}
}

type mcpRegistration struct {
	Command string   `json:"command" toml:"command"`
	Args    []string `json:"args" toml:"args"`
	Enabled *bool    `json:"enabled" toml:"enabled"`
	Type    string   `json:"type" toml:"type"`
	URL     string   `json:"url" toml:"url"`
	HTTPURL string   `json:"httpUrl" toml:"httpUrl"`
}

// checkMCPConfigs checks project-local configuration, never starts a server or
// claims that a client trusts, enables, or connects to it. Use a real TOML parser
// so comments, quoted keys and unrelated tables cannot impersonate registration.
func checkMCPConfigs(root string) []string {
	var out []string
	for _, rel := range []string{".codex/config.toml", ".mcp.json", ".gemini/settings.json"} {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			out = append(out, fmt.Sprintf("%s: cannot read MCP registration: %v; merge the registration from https://github.com/C360Studio/ateam/blob/main/docs/mcp.md", rel, err))
			continue
		}
		var servers map[string]mcpRegistration
		if rel == ".codex/config.toml" {
			var cfg struct {
				Servers map[string]mcpRegistration `toml:"mcp_servers"`
			}
			err = toml.Unmarshal(data, &cfg)
			servers = cfg.Servers
		} else {
			var cfg struct {
				Servers map[string]mcpRegistration `json:"mcpServers"`
			}
			err = json.Unmarshal(data, &cfg)
			servers = cfg.Servers
		}
		if err != nil {
			out = append(out, fmt.Sprintf("%s: invalid MCP configuration: %v", rel, err))
			continue
		}
		server, ok := servers["ateam"]
		if !ok || (filepath.Base(server.Command) != "ateam" && filepath.Base(server.Command) != "ateam.exe") ||
			len(server.Args) != 1 || server.Args[0] != "mcp" ||
			(server.Enabled != nil && !*server.Enabled) ||
			(server.Type != "" && server.Type != "stdio") || server.URL != "" || server.HTTPURL != "" {
			out = append(out, fmt.Sprintf("%s: expected enabled ateam stdio registration with command ateam (or path to it) and args [mcp]; preserve other settings and merge https://github.com/C360Studio/ateam/blob/main/docs/mcp.md", rel))
		}
	}
	return out
}
