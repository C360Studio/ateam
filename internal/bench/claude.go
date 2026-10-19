package bench

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// ClaudeConfig opts into the built-in, API-key-only Claude Code adapter.
type ClaudeConfig struct {
	Reasoning    string `json:"reasoning"`
	Auth         string `json:"auth"`
	OutputFormat string `json:"output_format,omitempty"`
}

func executeClaude(ctx context.Context, e Experiment, request AdapterRequest, dir string) (AdapterResponse, string, string, string, error) {
	empty, _ := parseClaudeOutput("")
	if e.Claude == nil || e.Claude.Auth != "api-key-env" || len(e.Command) != 1 {
		return empty, "", "", "adapter-error", fmt.Errorf("claude requires api-key-env authentication and one executable")
	}
	if e.Claude.OutputFormat != "json-schema" {
		return empty, "", "", "adapter-error", fmt.Errorf("claude execution requires output_format json-schema; legacy artifacts support offline replay only")
	}
	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		return empty, "", "", "adapter-error", fmt.Errorf("ANTHROPIC_API_KEY is required for the explicit api-key-env policy")
	}
	if err := checkClaudeManagedSettings(); err != nil {
		return empty, "", "", "adapter-error", err
	}
	configDir := filepath.Join(dir, ".claude")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return empty, "", "", "adapter-error", fmt.Errorf("create isolated claude configuration: %w", err)
	}
	settings, err := claudeSettings(e)
	if err != nil {
		return empty, "", "", "adapter-error", err
	}
	if err := os.WriteFile(filepath.Join(configDir, "benchmark-settings.json"), settings, 0600); err != nil {
		return empty, "", "", "adapter-error", fmt.Errorf("write isolated claude settings: %w", err)
	}
	bounded, cancel := context.WithTimeout(ctx, time.Duration(e.TimeoutSeconds)*time.Second)
	defer cancel()
	cmd := exec.CommandContext(bounded, e.Command[0], claudeArguments(e)...)
	configureBenchmarkProcess(cmd)
	defer cleanupBenchmarkProcess(cmd)
	cmd.Dir = dir
	// --bare rejects OAuth/keychain auth; the allowlist also excludes alternate
	// providers, API endpoints, proxy settings, helpers, and inherited sessions.
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + dir, "CLAUDE_CONFIG_DIR=" + configDir, "ANTHROPIC_API_KEY=" + key,
		"TMPDIR=" + dir, "XDG_CONFIG_HOME=" + dir, "XDG_CACHE_HOME=" + dir, "GOWORK=off", "GOENV=off",
		"CLAUDE_CODE_MAX_OUTPUT_TOKENS=8192", "CLAUDE_CODE_MAX_RETRIES=0", "MAX_STRUCTURED_OUTPUT_RETRIES=1", "DISABLE_AUTOUPDATER=1", "DISABLE_TELEMETRY=1", "DISABLE_ERROR_REPORTING=1"}
	cmd.Stdin = strings.NewReader("Instructions:\n" + request.Instructions + "\n\nTask:\n" + request.Prompt + "\n\nReturn a JSON object with source containing the complete Go source file.")
	var stdout, stderr limitedBuffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	runErr := cmd.Run()
	out, diagnostic := redactGeminiOutput(stdout.String(), key), redactGeminiOutput(stderr.String(), key)
	response, parseErr := parseNativeOutput(e, out)
	if stdout.overflow || stderr.overflow {
		markGeminiTruncated(&response)
		return response, out, diagnostic, "adapter-error", fmt.Errorf("claude output exceeded 1 MiB per stream")
	}
	if bounded.Err() != nil {
		status := "timeout"
		if ctx.Err() != nil {
			status = "cancelled"
		}
		return response, out, diagnostic, status, bounded.Err()
	}
	if runErr != nil {
		return response, out, diagnostic, "adapter-error", fmt.Errorf("claude execution: %w", runErr)
	}
	if parseErr != nil {
		return response, out, diagnostic, "adapter-error", parseErr
	}
	return response, out, diagnostic, "completed", nil
}

// Arguments and settings are retained in resolved provenance. The prompt is
// supplied on stdin so arm labels and credentials never become command args.
func claudeArguments(e Experiment) []string {
	reasoning := ""
	if e.Claude != nil {
		reasoning = e.Claude.Reasoning
	}
	args := []string{"--model", e.Model, "--effort", reasoning, "--print", "--output-format", "json",
		"--bare", "--safe-mode", "--restricted", "--setting-sources", "", "--settings", ".claude/benchmark-settings.json",
		"--strict-mcp-config", "--mcp-config", `{"mcpServers":{}}`, "--disable-slash-commands", "--no-chrome",
		"--tools", "Read,Glob,Grep", "--allowedTools", "Read,Glob,Grep", "--permission-mode", "dontAsk", "--permission-prompts", "none",
		"--max-turns", "4", "--no-session-persistence"}
	if e.Claude != nil && e.Claude.OutputFormat == "json-schema" {
		args = append(args, "--json-schema", `{"type":"object","properties":{"source":{"type":"string"}},"required":["source"],"additionalProperties":false}`)
	}
	return args
}

func claudeSettings(e Experiment) ([]byte, error) {
	if e.Claude == nil {
		return nil, fmt.Errorf("claude config is required")
	}
	return json.MarshalIndent(map[string]any{"model": e.Model, "effortLevel": e.Claude.Reasoning,
		"permissions": map[string]any{"allow": []string{"Read", "Glob", "Grep"}, "defaultMode": "dontAsk"}}, "", "  ")
}

// Refuse local managed policy rather than redirecting or overriding it. Server
// policies remain enforced by Claude Code and require separate provenance review.
func checkClaudeManagedSettings() error {
	root := "/etc/claude-code"
	switch runtime.GOOS {
	case "darwin":
		root = "/Library/Application Support/ClaudeCode"
	case "windows":
		return fmt.Errorf("claude Windows managed registry policy inspection is not supported; benchmark policy review is required")
	}
	paths := []string{filepath.Join(root, "managed-settings.json"), filepath.Join(root, "managed-settings.d"), filepath.Join(root, "managed-mcp.json")}
	if runtime.GOOS == "darwin" {
		current, err := user.Current()
		if err != nil {
			return fmt.Errorf("inspect claude managed user preferences: %w", err)
		}
		paths = append(paths, "/Library/Managed Preferences/com.anthropic.claudecode.plist", filepath.Join("/Library/Managed Preferences", current.Username, "com.anthropic.claudecode.plist"))
	}
	return checkClaudeManagedPaths(paths)
}

func checkClaudeManagedPaths(paths []string) error {
	for _, path := range paths {
		if _, err := os.Lstat(path); err == nil {
			return fmt.Errorf("claude managed configuration at %s requires benchmark policy review", path)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("inspect claude managed configuration: %w", err)
		}
	}
	return nil
}
