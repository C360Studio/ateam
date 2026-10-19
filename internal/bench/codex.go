package bench

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// CodexConfig selects an isolated Codex app-server using only an OpenAI API key.
type CodexConfig struct {
	Auth      string `json:"auth"`
	Reasoning string `json:"reasoning"`
}

func codexSchema() map[string]any {
	return map[string]any{"type": "object", "properties": map[string]any{"source": map[string]string{"type": "string"}}, "required": []string{"source"}, "additionalProperties": false}
}

func codexOverrides(e Experiment) []string {
	reasoning := ""
	if e.Codex != nil {
		reasoning = e.Codex.Reasoning
	}
	quote := func(s string) string { b, _ := json.Marshal(s); return string(b) }
	settings := []string{"model=" + quote(e.Model), "model_reasoning_effort=" + quote(reasoning), `model_provider="benchmark_openai"`, `forced_login_method="api"`, `cli_auth_credentials_store="ephemeral"`,
		`model_providers.benchmark_openai.name="OpenAI benchmark API"`, `model_providers.benchmark_openai.base_url="https://api.openai.com/v1"`, `model_providers.benchmark_openai.env_key="OPENAI_API_KEY"`, `model_providers.benchmark_openai.requires_openai_auth=false`, `model_providers.benchmark_openai.wire_api="responses"`,
		"model_providers.benchmark_openai.request_max_retries=0", "model_providers.benchmark_openai.stream_max_retries=0", `approval_policy="never"`, `sandbox_mode="read-only"`, `web_search="disabled"`, "project_doc_max_bytes=0", `project_root_markers=[".ateam-benchmark-root"]`, `shell_environment_policy.inherit="none"`, "analytics.enabled=false", "check_for_update_on_startup=false", `history.persistence="none"`}
	for _, feature := range []string{"hooks", "plugins", "remote_plugin", "apps", "multi_agent", "multi_agent_v2", "shell_tool", "shell_snapshot", "browser_use", "computer_use", "image_generation", "view_image", "artifact", "skill_search", "skill_mcp_dependency_install", "goals", "unbounded_connection_retries", "code_mode_host"} {
		settings = append(settings, "features."+feature+"=false")
	}
	settings = append(settings, "features.code_mode.enabled=false", "features.skip_host_skill_discovery=true")
	// Codex seeds these four bundled skills into fresh homes. The native disable
	// policy matches SKILL.md files, not their containing directories.
	var skills []string
	for _, name := range []string{"imagegen", "openai-docs", "skill-creator", "skill-installer"} {
		skills = append(skills, `{path=".codex/skills/.system/`+name+`/SKILL.md",enabled=false}`)
	}
	return append(settings, "skills.config=["+strings.Join(skills, ",")+"]")
}

func codexArguments(e Experiment) []string {
	args := []string{"app-server", "--stdio", "--strict-config"}
	for _, setting := range codexOverrides(e) {
		args = append(args, "-c", setting)
	}
	return args
}

func codexSettings(e Experiment) ([]byte, error) {
	if e.Codex == nil {
		return nil, fmt.Errorf("codex config is required")
	}
	return json.MarshalIndent(map[string]any{"config_overrides": codexOverrides(e), "output_schema": codexSchema(), "ephemeral": true, "allow_provider_model_fallback": false, "dynamic_tools": []string{}, "environments": []string{}}, "", "  ")
}

func executeCodex(ctx context.Context, e Experiment, request AdapterRequest, dir string) (AdapterResponse, string, string, string, error) {
	empty, _ := parseCodexOutput("")
	if e.Codex == nil || e.Codex.Auth != "api-key-env" || len(e.Command) != 1 {
		return empty, "", "", "adapter-error", fmt.Errorf("codex requires api-key-env authentication and one executable")
	}
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		return empty, "", "", "adapter-error", fmt.Errorf("OPENAI_API_KEY is required for the explicit api-key-env policy")
	}
	if err := checkCodexManagedSettings(); err != nil {
		return empty, "", "", "adapter-error", err
	}
	configDir := filepath.Join(dir, ".codex")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return empty, "", "", "adapter-error", fmt.Errorf("create isolated codex home: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".ateam-benchmark-root"), nil, 0600); err != nil {
		return empty, "", "", "adapter-error", fmt.Errorf("create codex instruction boundary: %w", err)
	}
	bounded, cancel := context.WithTimeout(ctx, time.Duration(e.TimeoutSeconds)*time.Second)
	defer cancel()
	cmd := exec.CommandContext(bounded, e.Command[0], codexArguments(e)...)
	configureBenchmarkProcess(cmd)
	defer cleanupBenchmarkProcess(cmd)
	cmd.Dir = dir
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + dir, "CODEX_HOME=" + configDir, "OPENAI_API_KEY=" + key, "TMPDIR=" + dir, "XDG_CONFIG_HOME=" + dir, "XDG_CACHE_HOME=" + dir, "GOWORK=off", "GOENV=off"}
	var stdout, stderr limitedBuffer
	cmd.Stderr = &stderr
	input, err := cmd.StdinPipe()
	if err != nil {
		return empty, "", "", "adapter-error", fmt.Errorf("open codex input: %w", err)
	}
	defer input.Close()
	output, err := cmd.StdoutPipe()
	if err != nil {
		return empty, "", "", "adapter-error", fmt.Errorf("open codex output: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return empty, "", "", "adapter-error", fmt.Errorf("start codex: %w", err)
	}
	protocolErr := driveCodex(input, output, &stdout, e, request, dir)
	if protocolErr != nil {
		cancel()
	}
	runErr := cmd.Wait()
	out, diagnostic := redactGeminiOutput(stdout.String(), key), redactGeminiOutput(stderr.String(), key)
	response, parseErr := parseCodexOutput(out)
	if stdout.overflow || stderr.overflow {
		markGeminiTruncated(&response)
		return response, out, diagnostic, "adapter-error", fmt.Errorf("codex output exceeded 1 MiB per stream")
	}
	if ctx.Err() != nil {
		return response, out, diagnostic, "cancelled", ctx.Err()
	}
	if bounded.Err() == context.DeadlineExceeded {
		return response, out, diagnostic, "timeout", bounded.Err()
	}
	if protocolErr != nil {
		return response, out, diagnostic, "adapter-error", fmt.Errorf("%s", redactGeminiOutput(protocolErr.Error(), key))
	}
	if runErr != nil {
		return response, out, diagnostic, "adapter-error", fmt.Errorf("codex execution: %w", runErr)
	}
	if parseErr != nil {
		return response, out, diagnostic, "adapter-error", parseErr
	}
	return response, out, diagnostic, "completed", nil
}

// Send the paid turn only after the native thread response proves the requested
// model, effort and sandbox. Never approve a server-initiated request.
func driveCodex(input io.WriteCloser, output io.Reader, trace *limitedBuffer, e Experiment, request AdapterRequest, dir string) error {
	encoder := json.NewEncoder(input)
	send := func(id int, method string, params any) error {
		message := map[string]any{"method": method, "params": params}
		if id != 0 {
			message["id"] = id
		}
		if err := encoder.Encode(message); err != nil {
			return fmt.Errorf("write codex protocol: %w", err)
		}
		return nil
	}
	if err := send(1, "initialize", map[string]any{"clientInfo": map[string]string{"name": "ateam-benchmark", "version": "1"}, "capabilities": map[string]bool{"experimentalApi": true}}); err != nil {
		return err
	}
	scanner := bufio.NewScanner(output)
	scanner.Buffer(make([]byte, 4096), maxAdapterBytes+1)
	history := newCodexHistory()
	completed := false
	for scanner.Scan() {
		line := scanner.Bytes()
		_, _ = trace.Write(line)
		_, _ = trace.Write([]byte{'\n'})
		if trace.overflow {
			return fmt.Errorf("codex trace exceeded retention limit")
		}
		if err := history.consume(line); err != nil {
			return err
		}
		var event codexRPC
		if err := json.Unmarshal(line, &event); err != nil {
			return fmt.Errorf("decode validated codex envelope: %w", err)
		}
		if event.ID != nil {
			switch *event.ID {
			case 1:
				if err := send(0, "initialized", map[string]any{}); err != nil {
					return err
				}
				if err := send(2, "thread/start", map[string]any{"model": e.Model, "modelProvider": "benchmark_openai", "cwd": dir, "ephemeral": true, "approvalPolicy": "never", "sandbox": "read-only", "allowProviderModelFallback": false, "environments": []any{}, "dynamicTools": []any{}}); err != nil {
					return err
				}
			case 2:
				thread, err := validateCodexThreadStart(line, e.Model, e.Codex.Reasoning)
				if err != nil {
					return err
				}
				prompt := "Instructions:\n" + request.Instructions + "\n\nTask:\n" + request.Prompt + "\n\nReturn a JSON object with source containing the complete Go source file."
				if err := send(3, "turn/start", map[string]any{"threadId": thread, "input": []any{map[string]any{"type": "text", "text": prompt, "text_elements": []any{}}}, "outputSchema": codexSchema()}); err != nil {
					return err
				}
			}
		}
		if event.Method == "turn/completed" {
			completed = true
			if err := input.Close(); err != nil {
				return fmt.Errorf("close codex input: %w", err)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		if errors.Is(err, bufio.ErrTooLong) {
			trace.overflow = true
		}
		return fmt.Errorf("read codex protocol: %w", err)
	}
	if !completed {
		return fmt.Errorf("codex ended without a terminal notification")
	}
	return nil
}

func checkCodexManagedSettings() error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("codex Windows managed policy inspection is not supported; benchmark policy review is required")
	}
	paths := []string{"/etc/codex/config.toml", "/etc/codex/requirements.toml", "/etc/codex/managed_config.toml", "/etc/codex/skills"}
	if runtime.GOOS == "darwin" {
		current, err := user.Current()
		if err != nil {
			return fmt.Errorf("inspect codex managed user preferences: %w", err)
		}
		paths = append(paths, "/Library/Managed Preferences/com.openai.codex.plist", filepath.Join("/Library/Managed Preferences", current.Username, "com.openai.codex.plist"))
	}
	return checkCodexManagedPaths(paths)
}

func checkCodexManagedPaths(paths []string) error {
	for _, path := range paths {
		if _, err := os.Lstat(path); err == nil {
			return fmt.Errorf("codex managed configuration at %s requires benchmark policy review", path)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("inspect codex managed configuration: %w", err)
		}
	}
	return nil
}
