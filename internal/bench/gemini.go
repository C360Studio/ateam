package bench

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// GeminiConfig opts into the built-in, API-key-only Gemini CLI adapter.
type GeminiConfig struct {
	CLIVersion      string `json:"cli_version,omitempty"`
	Reasoning       string `json:"reasoning"`
	Auth            string `json:"auth"`
	OutputFormat    string `json:"output_format,omitempty"`
	MaxOutputTokens *int   `json:"max_output_tokens,omitempty"`
}

func geminiMaxOutputTokens(config *GeminiConfig) int {
	if config.MaxOutputTokens != nil {
		return *config.MaxOutputTokens
	}
	return 8192
}

func executeGemini(ctx context.Context, e Experiment, request AdapterRequest, dir string) (AdapterResponse, string, string, string, error) {
	empty, _ := parseGeminiOutput("")
	if e.Gemini == nil || e.Gemini.Auth != "api-key-env" || len(e.Command) != 1 {
		return empty, "", "", "adapter-error", fmt.Errorf("gemini requires api-key-env authentication and one executable")
	}
	if err := validateNewGeminiExecution(e.Gemini); err != nil {
		return empty, "", "", "adapter-error", err
	}
	key := os.Getenv("GEMINI_API_KEY")
	if key == "" {
		return empty, "", "", "adapter-error", fmt.Errorf("GEMINI_API_KEY is required for the explicit api-key-env policy")
	}
	if err := checkGeminiManagedSettings(); err != nil {
		return empty, "", "", "adapter-error", err
	}
	if err := os.MkdirAll(filepath.Join(dir, ".gemini"), 0700); err != nil {
		return empty, "", "", "adapter-error", err
	}
	settings, err := geminiSettings(e)
	if err != nil {
		return empty, "", "", "adapter-error", err
	}
	if err := os.WriteFile(filepath.Join(dir, ".gemini", "settings.json"), settings, 0600); err != nil {
		return empty, "", "", "adapter-error", err
	}
	// Gemini's dotenv search climbs parents even with ignoreLocalEnv. Empty local
	// sentinels stop both trusted (.gemini/.env) and untrusted (.env) searches.
	for _, name := range []string{".env", filepath.Join(".gemini", ".env")} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0600); err != nil {
			return empty, "", "", "adapter-error", err
		}
	}
	bounded, cancel := context.WithTimeout(ctx, time.Duration(e.TimeoutSeconds)*time.Second)
	defer cancel()
	cmd := exec.CommandContext(bounded, e.Command[0], geminiArguments(e)...)
	configureBenchmarkProcess(cmd)
	defer cleanupBenchmarkProcess(cmd)
	cmd.Dir = dir
	// Trust only this runner-created workspace for this child, without persisting a trust policy.
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + dir, "GEMINI_CLI_HOME=" + dir, "GEMINI_API_KEY=" + key, "GEMINI_CLI_TRUST_WORKSPACE=true",
		"TMPDIR=" + dir, "XDG_CONFIG_HOME=" + dir, "XDG_CACHE_HOME=" + dir, "GOWORK=off", "GOENV=off"}
	// Arm names and repetition numbers do not enter the model prompt.
	cmd.Stdin = strings.NewReader("Instructions:\n" + request.Instructions + "\n\nTask:\n" + request.Prompt + "\n\nReturn a JSON object with source containing the complete Go source file.")
	var stdout, stderr limitedBuffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	runErr := cmd.Run()
	out, diagnostic := redactGeminiOutput(stdout.String(), key), redactGeminiOutput(stderr.String(), key)
	response, parseErr := parseGeminiVersionedOutput(e.Gemini, out)
	if stdout.overflow || stderr.overflow {
		markGeminiTruncated(&response)
		return response, out, diagnostic, "adapter-error", fmt.Errorf("gemini output exceeded 1 MiB per stream")
	}
	if bounded.Err() != nil {
		status := "timeout"
		if ctx.Err() != nil {
			status = "cancelled"
		}
		return response, out, diagnostic, status, bounded.Err()
	}
	if runErr != nil {
		var exit *exec.ExitError
		if out == "" && errors.As(runErr, &exit) && exit.ExitCode() == 53 {
			if failure := parseGeminiTurnLimit(diagnostic); failure != nil {
				response.Native.GeminiFailure = failure
				return response, out, diagnostic, "adapter-error", fmt.Errorf("%s", geminiTurnLimitDiagnostic)
			}
		}
		return response, out, diagnostic, "adapter-error", fmt.Errorf("gemini execution: %w", runErr)
	}
	if parseErr != nil {
		return response, out, diagnostic, "adapter-error", parseErr
	}
	return response, out, diagnostic, "completed", nil
}

func markGeminiTruncated(response *AdapterResponse) {
	response.Native.OutputTruncated = true
	response.Native.UsageStatus = "truncated"
	response.Usage = nil
}

// geminiArguments and geminiSettings are persisted in resolved provenance. The
// prompt arrives on stdin so arm labels and secrets never become command args.
func geminiArguments(e Experiment) []string {
	return []string{"--model", e.Model, "--prompt", "Complete the task supplied on stdin.", "--output-format", "json", "--approval-mode", "default", "--extensions", "none"}
}

func geminiSettings(e Experiment) ([]byte, error) {
	if e.Gemini == nil {
		return nil, fmt.Errorf("gemini config is required")
	}
	generation := map[string]any{
		"maxOutputTokens": geminiMaxOutputTokens(e.Gemini),
		"thinkingConfig":  map[string]any{"thinkingLevel": strings.ToUpper(e.Gemini.Reasoning), "includeThoughts": false},
	}
	if e.Gemini.OutputFormat == "json-schema" {
		generation["responseMimeType"] = "application/json"
		generation["responseJsonSchema"] = map[string]any{
			"type": "object", "properties": map[string]any{"source": map[string]any{"type": "string"}},
			"required": []string{"source"}, "additionalProperties": false,
		}
	}
	settings := map[string]any{
		"security":     map[string]any{"auth": map[string]any{"selectedType": "gemini-api-key"}},
		"advanced":     map[string]any{"ignoreLocalEnv": true},
		"general":      map[string]any{"checkpointing": map[string]any{"enabled": false}},
		"telemetry":    map[string]any{"enabled": false},
		"skills":       map[string]any{"enabled": false},
		"hooksConfig":  map[string]any{"enabled": false},
		"experimental": map[string]any{"enableAgents": false, "contextManagement": false, "extensionManagement": false},
		"context":      map[string]any{"fileName": "ATEAM_BENCHMARK_CONTEXT_DISABLED", "memoryBoundaryMarkers": []string{}, "includeDirectoryTree": false, "loadMemoryFromIncludeDirectories": false},
		"tools":        map[string]any{"core": []string{"list_directory", "read_file", "grep_search", "glob"}},
		"model":        map[string]any{"name": e.Model, "maxSessionTurns": 4, "disableLoopDetection": true, "skipNextSpeakerCheck": true, "compressionThreshold": 1},
		"modelConfigs": map[string]any{"customOverrides": []any{
			map[string]any{"match": map[string]any{"overrideScope": "core"}, "modelConfig": map[string]any{"model": e.Model, "generateContentConfig": generation}},
			map[string]any{"match": map[string]any{"model": e.Model}, "modelConfig": map[string]any{"model": e.Model, "generateContentConfig": generation}},
		}},
	}
	if geminiCLIVersion(e.Gemini) == geminiStableVersion {
		settings["privacy"] = map[string]any{"usageStatisticsEnabled": false}
	}
	return json.MarshalIndent(settings, "", "  ")
}

// Refuse managed installations instead of bypassing their restrictions with an
// alternate system-settings path. A reviewed policy/provenance integration is
// required before benchmarks can run on those installations.
func checkGeminiManagedSettings() error {
	for _, name := range []string{"GEMINI_CLI_SYSTEM_SETTINGS_PATH", "GEMINI_CLI_SYSTEM_DEFAULTS_PATH"} {
		if os.Getenv(name) != "" {
			return fmt.Errorf("gemini managed configuration %s is set; benchmark policy review is required", name)
		}
	}
	root := "/etc/gemini-cli"
	switch runtime.GOOS {
	case "darwin":
		root = "/Library/Application Support/GeminiCli"
	case "windows":
		root = `C:\ProgramData\gemini-cli`
	}
	for _, name := range []string{"settings.json", "system-defaults.json", "policies"} {
		path := filepath.Join(root, name)
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("gemini managed configuration at %s requires benchmark policy review", path)
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("inspect gemini managed configuration: %w", err)
		}
	}
	return nil
}

func redactGeminiOutput(output, key string) string {
	if key == "" {
		return output
	}
	// A final-message JSON string embeds another JSON document. Match escaped
	// forms from deepest to shallowest so replacing one form cannot expose another.
	forms := []string{key}
	for depth := 0; depth < 64; depth++ {
		encoded, _ := json.Marshal(forms[len(forms)-1])
		next := string(encoded[1 : len(encoded)-1])
		if next == forms[len(forms)-1] || len(next) > maxAdapterBytes {
			break
		}
		forms = append(forms, next)
	}
	for i := len(forms) - 1; i >= 0; i-- {
		output = strings.ReplaceAll(output, forms[i], "[REDACTED]")
	}
	return output
}
