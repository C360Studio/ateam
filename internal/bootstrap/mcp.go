package bootstrap

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	out = append(out, checkContextConfigs(root)...)
	return out
}

func checkContextConfigs(root string) []string {
	var out []string
	if data, err := os.ReadFile(filepath.Join(root, ".claude/settings.json")); err == nil {
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.UseNumber()
		var raw map[string]any
		if err := dec.Decode(&raw); err == nil {
			if val, ok := raw["autoCompactWindow"]; ok {
				num, ok := val.(json.Number)
				n, err := num.Int64()
				if !ok || err != nil || strings.ContainsAny(num.String(), ".eE") || n < 100000 || n > 1000000 {
					out = append(out, ".claude/settings.json: autoCompactWindow must be an integer between 100000 and 1000000")
				}
			}
		}
	}
	if data, err := os.ReadFile(filepath.Join(root, ".codex/config.toml")); err == nil {
		var raw map[string]any
		if err := toml.Unmarshal(data, &raw); err == nil {
			wVal, hasW := raw["model_context_window"]
			cVal, hasC := raw["model_auto_compact_token_limit"]
			w, wOK := wVal.(int64)
			c, cOK := cVal.(int64)
			if (hasW && (!wOK || w <= 0)) || (hasC && (!cOK || c <= 0)) || (hasW && hasC && c > w) {
				out = append(out, ".codex/config.toml: model_context_window and model_auto_compact_token_limit must be positive integers with model_auto_compact_token_limit <= model_context_window")
			}
		}
	}
	if data, err := os.ReadFile(filepath.Join(root, ".gemini/settings.json")); err == nil {
		var raw map[string]any
		if err := json.Unmarshal(data, &raw); err == nil {
			if mMap, ok := raw["model"].(map[string]any); ok {
				if ccMap, ok := mMap["chatCompression"].(map[string]any); ok {
					if tVal, ok := ccMap["contextPercentageThreshold"]; ok {
						if thresh, ok := tVal.(float64); !ok || thresh <= 0 || thresh > 1 {
							out = append(out, ".gemini/settings.json: model.chatCompression.contextPercentageThreshold must be a number in (0, 1]")
						}
					}
				}
			}
		}
	}
	return out
}
