package bench

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
)

const claudeTestKey = "unit-test-key\"with\\escape"

func fakeClaudeExperiment(t *testing.T) Experiment {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake executable uses a POSIX launcher; native parser tests remain portable")
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	launcher := filepath.Join(t.TempDir(), "fake-claude")
	quoted := "'" + strings.ReplaceAll(exe, "'", "'\"'\"'") + "'"
	if err := os.WriteFile(launcher, []byte("#!/bin/sh\nexec "+quoted+" -test.run=^TestClaudeCLIHelper$ -- \"$@\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	e := testExperiment()
	e.Command = []string{launcher}
	e.Model = "explicit-test-model"
	e.Claude = &ClaudeConfig{Reasoning: "low", Auth: "api-key-env", OutputFormat: "json-schema"}
	t.Setenv("ANTHROPIC_API_KEY", claudeTestKey)
	t.Setenv("ATEAM_SECRET_PROBE", "must-not-inherit")
	for _, name := range []string{"OPENAI_API_KEY", "GOOGLE_API_KEY", "GOOGLE_APPLICATION_CREDENTIALS", "GOOGLE_GENAI_USE_VERTEXAI", "NODE_OPTIONS", "ANTHROPIC_AUTH_TOKEN", "CLAUDE_CODE_OAUTH_TOKEN", "ANTHROPIC_BASE_URL", "CLAUDE_CODE_USE_BEDROCK", "CLAUDE_CODE_USE_VERTEX", "HTTP_PROXY", "HTTPS_PROXY"} {
		t.Setenv(name, "must-not-inherit")
	}
	return e
}

// Explicit API auth reaches the CLI, but ambient config/secrets and prior arm files must not.
// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Controlled bounded experiments
func TestClaudeExecutionIsolationAndRedaction(t *testing.T) {
	e := fakeClaudeExperiment(t)
	t.Setenv("CLAUDE_CODE_SAFE_MODE", "parent-must-not-change")
	previousSession := ""
	for _, arm := range []string{"baseline", "candidate"} {
		req := newAdapterRequest(e, Trial{Task: "safe-path", Repetition: 1, Arm: arm})
		req.Instructions = "same-visible-instructions"
		r, stdout, stderr, status, err := executeClaude(context.Background(), e, req, t.TempDir())
		if err != nil || status != "completed" {
			t.Fatalf("status=%s error=%v stderr=%s stdout=%s", status, err, stderr, stdout)
		}
		if r.Source != "package safepath // key=[REDACTED]" || r.Usage == nil || r.Usage.InputTokens != 100 || r.Usage.OutputTokens != 25 {
			t.Fatal("decoded source must redact a doubly JSON-escaped API credential and retain native usage")
		}
		encoded, _ := json.Marshal(claudeTestKey)
		if strings.Contains(stdout+stderr, claudeTestKey) || strings.Contains(stdout+stderr, string(encoded[1:len(encoded)-1])) || !strings.Contains(stdout, "[REDACTED]") || !strings.Contains(stderr, "[REDACTED]") {
			t.Fatal("persisted output did not redact exact API credential")
		}
		if r.Native.SessionID == "" || r.Native.SessionID == previousSession {
			t.Fatal("independent attempts reused a session")
		}
		previousSession = r.Native.SessionID
		if os.Getenv("CLAUDE_CODE_SAFE_MODE") != "parent-must-not-change" {
			t.Fatal("child policy changed the parent environment")
		}
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Controlled bounded experiments
// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Retained independently graded artifacts
func TestClaudeExecutionFailureEvidence(t *testing.T) {
	e := fakeClaudeExperiment(t)
	for _, tc := range []struct{ instruction, status string }{{"failure", "adapter-error"}, {"timeout", "timeout"}, {"overflow", "adapter-error"}, {"stdout-cap", "adapter-error"}, {"stderr-cap", "adapter-error"}, {"malformed", "adapter-error"}} {
		t.Run(tc.instruction, func(t *testing.T) {
			e.TimeoutSeconds = 5
			if tc.instruction == "timeout" {
				e.TimeoutSeconds = 1
			}
			req := newAdapterRequest(e, Schedule(e)[0])
			req.Instructions = tc.instruction
			r, stdout, _, status, err := executeClaude(context.Background(), e, req, t.TempDir())
			if err == nil || status != tc.status || r.Native == nil || len(stdout) > maxAdapterBytes {
				t.Fatalf("status=%s response=%+v error=%v", status, r, err)
			}
			if tc.instruction == "failure" && (r.Usage != nil || r.Native.ClaudeReportedUsage == nil || *r.Native.ClaudeReportedUsage.Models["explicit-test-model"].Input != 41 || r.Native.Terminal != "failed") {
				t.Fatal("failed process must retain raw usage without claiming complete accounting")
			}
			if strings.HasSuffix(tc.instruction, "-cap") && (r.Usage != nil || !r.Native.OutputTruncated || r.Native.ClaudeReportedUsage == nil || r.Native.UsageStatus != "truncated") {
				t.Fatal("truncated trace must not supply accountable usage from a completed prefix")
			}
		})
	}
	t.Setenv("ANTHROPIC_API_KEY", "")
	_, stdout, _, status, err := executeClaude(context.Background(), e, AdapterRequest{}, t.TempDir())
	if err == nil || status != "adapter-error" || stdout != "" || !strings.Contains(err.Error(), "ANTHROPIC_API_KEY") {
		t.Fatalf("missing explicit auth did not fail before process launch: status=%s error=%v", status, err)
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Controlled bounded experiments
func TestClaudeExecutionCancellation(t *testing.T) {
	e := fakeClaudeExperiment(t)
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := newAdapterRequest(e, Schedule(e)[0])
	req.Instructions = "timeout"
	result := make(chan string, 1)
	go func() {
		_, _, _, status, _ := executeClaude(ctx, e, req, dir)
		result <- status
	}()
	deadline := time.NewTimer(4 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(5 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case status := <-result:
			t.Fatalf("process ended before cancellation synchronization: %s", status)
		case <-deadline.C:
			t.Fatal("fake CLI did not signal readiness")
		case <-tick.C:
			if _, err := os.Stat(filepath.Join(dir, "ready")); err == nil {
				cancel()
				select {
				case status := <-result:
					if status != "cancelled" {
						t.Fatalf("cancelled process status=%s", status)
					}
				case <-deadline.C:
					t.Fatal("cancelled process did not exit")
				}
				return
			}
		}
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Controlled bounded experiments
func TestClaudeCLIHelper(_ *testing.T) {
	separator := slices.Index(os.Args, "--")
	if separator < 0 {
		return
	}
	args := os.Args[separator+1:]
	if err := validateFakeClaudeInvocation(args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(9)
	}
	prompt, _ := io.ReadAll(os.Stdin)
	if strings.Contains(string(prompt), `"arm"`) || strings.Contains(string(prompt), "baseline") || strings.Contains(string(prompt), "candidate") || !strings.Contains(string(prompt), "func Resolve") {
		os.Exit(10)
	}
	cwd, _ := os.Getwd()
	if err := os.WriteFile(filepath.Join(cwd, "bundle-delivered-prompt.txt"), prompt, 0600); err != nil {
		os.Exit(12)
	}
	if strings.Contains(string(prompt), "timeout") {
		_ = os.WriteFile(filepath.Join(cwd, "ready"), nil, 0600)
		time.Sleep(8 * time.Second)
		os.Exit(11)
	}
	if strings.Contains(string(prompt), "overflow") {
		_, _ = os.Stdout.WriteString(strings.Repeat("x", maxAdapterBytes+1))
		os.Exit(0)
	}
	if strings.Contains(string(prompt), "malformed") {
		_, _ = os.Stdout.WriteString("{")
		os.Exit(0)
	}
	_, _ = os.Stderr.WriteString(os.Getenv("ANTHROPIC_API_KEY"))
	answer, _ := json.Marshal(map[string]string{"source": "package safepath // key=" + os.Getenv("ANTHROPIC_API_KEY")})
	usage := map[string]int{"input_tokens": 100, "output_tokens": 25, "cache_read_input_tokens": 0, "cache_creation_input_tokens": 0}
	modelUsage := map[string]int{"inputTokens": 100, "outputTokens": 25, "cacheReadInputTokens": 0, "cacheCreationInputTokens": 0}
	root := map[string]any{"type": "result", "subtype": "success", "is_error": false, "result": "```json\n" + string(answer) + "\n```", "structured_output": map[string]string{"source": "package safepath // key=" + os.Getenv("ANTHROPIC_API_KEY")}, "session_id": filepath.Base(cwd), "num_turns": 1, "total_cost_usd": 0.001, "usage": usage, "modelUsage": map[string]any{"explicit-test-model": modelUsage}}
	if strings.Contains(string(prompt), "failure") {
		usage["input_tokens"], modelUsage["inputTokens"] = 41, 41
		root["subtype"], root["is_error"] = "error_during_execution", true
		_ = json.NewEncoder(os.Stdout).Encode(root)
		os.Exit(7)
	}
	_ = json.NewEncoder(os.Stdout).Encode(root)
	if strings.Contains(string(prompt), "stdout-cap") {
		_, _ = os.Stdout.WriteString(strings.Repeat(" ", maxAdapterBytes))
	}
	if strings.Contains(string(prompt), "stderr-cap") {
		_, _ = os.Stderr.WriteString(strings.Repeat(" ", maxAdapterBytes))
	}
	os.Exit(0)
}

func validateFakeClaudeInvocation(args []string) error {
	for _, name := range []string{"ATEAM_SECRET_PROBE", "OPENAI_API_KEY", "GOOGLE_API_KEY", "GOOGLE_APPLICATION_CREDENTIALS", "GOOGLE_GENAI_USE_VERTEXAI", "NODE_OPTIONS", "ANTHROPIC_AUTH_TOKEN", "CLAUDE_CODE_OAUTH_TOKEN", "ANTHROPIC_BASE_URL", "CLAUDE_CODE_USE_BEDROCK", "CLAUDE_CODE_USE_VERTEX", "HTTP_PROXY", "HTTPS_PROXY"} {
		if os.Getenv(name) != "" {
			return fmt.Errorf("ambient secret or routing inherited: %s", name)
		}
	}
	allowedEnv := map[string]bool{}
	for _, name := range []string{"PATH", "HOME", "CLAUDE_CONFIG_DIR", "ANTHROPIC_API_KEY", "TMPDIR", "XDG_CONFIG_HOME", "XDG_CACHE_HOME", "GOWORK", "GOENV", "CLAUDE_CODE_MAX_OUTPUT_TOKENS", "CLAUDE_CODE_MAX_RETRIES", "MAX_STRUCTURED_OUTPUT_RETRIES", "DISABLE_AUTOUPDATER", "DISABLE_TELEMETRY", "DISABLE_ERROR_REPORTING", "PWD", "SHLVL", "_"} {
		allowedEnv[name] = true
	}
	for _, assignment := range os.Environ() {
		name, _, _ := strings.Cut(assignment, "=")
		if !allowedEnv[name] {
			return fmt.Errorf("unexpected child environment: %s", name)
		}
	}
	if os.Getenv("ANTHROPIC_API_KEY") != claudeTestKey {
		return fmt.Errorf("explicit credential missing")
	}
	cwd, _ := os.Getwd()
	cwd, _ = filepath.EvalSymlinks(cwd)
	home, _ := filepath.EvalSymlinks(os.Getenv("HOME"))
	configDir, _ := filepath.EvalSymlinks(os.Getenv("CLAUDE_CONFIG_DIR"))
	if cwd != home || configDir != filepath.Join(home, ".claude") {
		return fmt.Errorf("home or CLI config was not isolated")
	}
	marker := filepath.Join(configDir, "attempt-marker")
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		return fmt.Errorf("previous attempt state leaked")
	}
	if err := os.WriteFile(marker, nil, 0600); err != nil {
		return err
	}
	// This literal policy is the oracle, independent of claudeArguments.
	expected := []string{"--model", "explicit-test-model", "--effort", "low", "--print", "--output-format", "json", "--bare", "--safe-mode", "--restricted", "--setting-sources", "", "--settings", ".claude/benchmark-settings.json", "--strict-mcp-config", "--mcp-config", `{"mcpServers":{}}`, "--disable-slash-commands", "--no-chrome", "--tools", "Read,Glob,Grep", "--allowedTools", "Read,Glob,Grep", "--permission-mode", "dontAsk", "--permission-prompts", "none", "--max-turns", "4", "--no-session-persistence", "--json-schema", `{"type":"object","properties":{"source":{"type":"string"}},"required":["source"],"additionalProperties":false}`}
	if !slices.Contains(args, "--json-schema") {
		return fmt.Errorf("native JSON schema constraint missing")
	}
	if !slices.Equal(args, expected) {
		return fmt.Errorf("execution policy arguments differ")
	}
	for key, want := range map[string]string{"CLAUDE_CODE_MAX_OUTPUT_TOKENS": "8192", "CLAUDE_CODE_MAX_RETRIES": "0", "MAX_STRUCTURED_OUTPUT_RETRIES": "1", "DISABLE_AUTOUPDATER": "1", "DISABLE_TELEMETRY": "1", "DISABLE_ERROR_REPORTING": "1"} {
		if os.Getenv(key) != want {
			return fmt.Errorf("execution limit missing: %s", key)
		}
	}
	b, err := os.ReadFile(filepath.Join(configDir, "benchmark-settings.json"))
	if err != nil {
		return err
	}
	var settings struct {
		Model       string `json:"model"`
		Effort      string `json:"effortLevel"`
		Permissions struct {
			Allow       []string `json:"allow"`
			DefaultMode string   `json:"defaultMode"`
		} `json:"permissions"`
	}
	if err := json.Unmarshal(b, &settings); err != nil {
		return err
	}
	if settings.Model != "explicit-test-model" || settings.Effort != "low" || settings.Permissions.DefaultMode != "dontAsk" || !slices.Equal(settings.Permissions.Allow, []string{"Read", "Glob", "Grep"}) {
		return fmt.Errorf("settings changed model or readonly policy")
	}
	return nil
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Controlled bounded experiments
func TestClaudeManagedPaths(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing")
	if err := checkClaudeManagedPaths([]string{missing}); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"file", "directory", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			path := filepath.Join(dir, kind)
			var err error
			switch kind {
			case "file":
				err = os.WriteFile(path, nil, 0600)
			case "directory":
				err = os.Mkdir(path, 0700)
			case "symlink":
				err = os.Symlink(missing, path)
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := checkClaudeManagedPaths([]string{missing, path}); err == nil || !strings.Contains(err.Error(), "policy review") {
				t.Fatal("managed policy was not refused")
			}
		})
	}
	// A child path under a regular file is not an absent configuration.
	if err := checkClaudeManagedPaths([]string{filepath.Join(dir, "file", "child")}); err == nil {
		t.Fatal("filesystem inspection error was ignored")
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Controlled bounded experiments
func TestClaudeMissingConfiguration(t *testing.T) {
	e := testExperiment()
	if _, err := claudeSettings(e); err == nil {
		t.Fatal("missing native configuration was accepted")
	}
	_, _, _, status, err := executeClaude(context.Background(), e, AdapterRequest{}, t.TempDir())
	if err == nil || status != "adapter-error" {
		t.Fatal("invalid authentication config was accepted")
	}
}

// Legacy provenance still derives its original arguments, but cannot launch a
// new native run without the explicit structured-output policy.
// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Controlled bounded experiments
func TestClaudeLegacyExecutionRefused(t *testing.T) {
	e := fakeClaudeExperiment(t)
	for _, format := range []string{"", "unknown"} {
		e.Claude.OutputFormat = format
		if slices.Contains(claudeArguments(e), "--json-schema") {
			t.Fatal("legacy provenance arguments changed")
		}
		dir := t.TempDir()
		_, stdout, _, status, err := executeClaude(context.Background(), e, AdapterRequest{}, dir)
		if err == nil || status != "adapter-error" || stdout != "" || !strings.Contains(err.Error(), "output_format json-schema") {
			t.Fatalf("legacy execution not refused: %s %v", status, err)
		}
		if _, err := os.Stat(filepath.Join(dir, ".claude")); !os.IsNotExist(err) {
			t.Fatal("legacy execution reached workspace preparation")
		}
	}
}
