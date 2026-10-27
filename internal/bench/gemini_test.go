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

const geminiTestKey = "unit-test-key\"with\\escape"

func fakeGeminiExperiment(t *testing.T) Experiment {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake executable uses a POSIX launcher; native parser tests remain portable")
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	launcher := filepath.Join(t.TempDir(), "fake-gemini")
	quoted := "'" + strings.ReplaceAll(exe, "'", "'\"'\"'") + "'"
	if err := os.WriteFile(launcher, []byte("#!/bin/sh\nexec "+quoted+" -test.run=^TestGeminiCLIHelper$ -- \"$@\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	e := testExperiment()
	e.Command = []string{launcher}
	e.Model = "explicit-test-model"
	e.Gemini = &GeminiConfig{CLIVersion: "0.64.0", Reasoning: "low", Auth: "api-key-env", OutputFormat: "json-schema"}
	t.Setenv("GEMINI_API_KEY", geminiTestKey)
	t.Setenv("ATEAM_SECRET_PROBE", "must-not-inherit")
	for _, name := range []string{"OPENAI_API_KEY", "GOOGLE_API_KEY", "GOOGLE_APPLICATION_CREDENTIALS", "GOOGLE_GENAI_USE_VERTEXAI", "NODE_OPTIONS"} {
		t.Setenv(name, "must-not-inherit")
	}
	return e
}

// Explicit API auth reaches the CLI, but ambient config/secrets and prior arm files must not.
// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Native agent evidence
// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Generated workspace trust
func TestGeminiExecutionIsolationAndRedaction(t *testing.T) {
	e := fakeGeminiExperiment(t)
	t.Setenv("GEMINI_CLI_TRUST_WORKSPACE", "parent-must-not-change")
	previousSession := ""
	for _, arm := range []string{"baseline", "candidate"} {
		req := newAdapterRequest(e, Trial{Task: "safe-path", Repetition: 1, Arm: arm})
		req.Instructions = "same-visible-instructions"
		r, stdout, stderr, status, err := executeGemini(context.Background(), e, req, t.TempDir())
		if err != nil || status != "completed" {
			t.Fatalf("status=%s error=%v stderr=%s stdout=%s", status, err, stderr, stdout)
		}
		if r.Source != "package safepath // key=[REDACTED]" || r.Usage == nil || r.Usage.InputTokens != 100 || r.Usage.OutputTokens != 25 {
			t.Fatal("decoded source must redact a doubly JSON-escaped API credential and retain native usage")
		}
		encoded, _ := json.Marshal(geminiTestKey)
		if strings.Contains(stdout+stderr, geminiTestKey) || strings.Contains(stdout+stderr, string(encoded[1:len(encoded)-1])) || !strings.Contains(stdout, "[REDACTED]") || !strings.Contains(stderr, "[REDACTED]") {
			t.Fatal("persisted output did not redact exact API credential")
		}
		if r.Native.SessionID == "" || r.Native.SessionID == previousSession {
			t.Fatal("independent attempts reused a session")
		}
		previousSession = r.Native.SessionID
		if os.Getenv("GEMINI_CLI_TRUST_WORKSPACE") != "parent-must-not-change" {
			t.Fatal("workspace trust escaped the child environment")
		}
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Native agent evidence
func TestGeminiExecutionFailureEvidence(t *testing.T) {
	e := fakeGeminiExperiment(t)
	for _, tc := range []struct{ instruction, status string }{{"failure", "adapter-error"}, {"timeout", "timeout"}, {"overflow", "adapter-error"}, {"stdout-cap", "adapter-error"}, {"stderr-cap", "adapter-error"}, {"malformed", "adapter-error"}} {
		t.Run(tc.instruction, func(t *testing.T) {
			e.TimeoutSeconds = 5
			if tc.instruction == "timeout" {
				e.TimeoutSeconds = 1
			}
			req := newAdapterRequest(e, Schedule(e)[0])
			req.Instructions = tc.instruction
			r, stdout, _, status, err := executeGemini(context.Background(), e, req, t.TempDir())
			if err == nil || status != tc.status || r.Native == nil || len(stdout) > maxAdapterBytes {
				t.Fatalf("status=%s response=%+v error=%v", status, r, err)
			}
			if tc.instruction == "failure" && (r.Usage == nil || r.Usage.InputTokens != 41 || r.Native.Terminal != "failed") {
				t.Fatal("failed process lost reported native usage")
			}
			if strings.HasSuffix(tc.instruction, "-cap") && (r.Usage != nil || !r.Native.OutputTruncated || r.Native.ReportedUsage == nil || r.Native.UsageStatus != "truncated") {
				t.Fatal("truncated trace must not supply accountable usage from a completed prefix")
			}
		})
	}
	t.Setenv("GEMINI_API_KEY", "")
	_, stdout, _, status, err := executeGemini(context.Background(), e, AdapterRequest{}, t.TempDir())
	if err == nil || status != "adapter-error" || stdout != "" || !strings.Contains(err.Error(), "GEMINI_API_KEY") {
		t.Fatalf("missing explicit auth did not fail before process launch: status=%s error=%v", status, err)
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Native agent evidence
func TestGeminiExecutionCancellation(t *testing.T) {
	e := fakeGeminiExperiment(t)
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := newAdapterRequest(e, Schedule(e)[0])
	req.Instructions = "timeout"
	result := make(chan string, 1)
	go func() {
		_, _, _, status, _ := executeGemini(ctx, e, req, dir)
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

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Native agent evidence
func TestGeminiCLIHelper(_ *testing.T) {
	separator := slices.Index(os.Args, "--")
	if separator < 0 {
		return
	}
	args := os.Args[separator+1:]
	if err := validateFakeGeminiInvocation(args); err != nil {
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
	_, _ = os.Stderr.WriteString(os.Getenv("GEMINI_API_KEY"))
	answer, _ := json.Marshal(map[string]string{"source": "package safepath // key=" + os.Getenv("GEMINI_API_KEY")})
	tokens := map[string]int{"prompt": 100, "candidates": 10, "thoughts": 15, "total": 125, "cached": 80, "input": 20, "tool": 0}
	root := map[string]any{"session_id": filepath.Base(cwd), "response": string(answer), "stats": map[string]any{
		"models": map[string]any{"explicit-test-model": map[string]any{"tokens": tokens, "api": map[string]int{"totalRequests": 1, "totalErrors": 0, "totalLatencyMs": 10}}},
		"tools":  map[string]any{"totalCalls": 0, "byName": map[string]any{}},
	}}
	if strings.Contains(string(prompt), "failure") {
		tokens["prompt"], tokens["candidates"], tokens["thoughts"], tokens["total"], tokens["cached"], tokens["input"] = 41, 7, 0, 48, 0, 41
		root["error"] = map[string]string{"type": "APIError", "message": "failure fixture"}
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

func validateFakeGeminiInvocation(args []string) error {
	for _, name := range []string{"ATEAM_SECRET_PROBE", "OPENAI_API_KEY", "GOOGLE_API_KEY", "GOOGLE_APPLICATION_CREDENTIALS", "GOOGLE_GENAI_USE_VERTEXAI", "NODE_OPTIONS"} {
		if os.Getenv(name) != "" {
			return fmt.Errorf("ambient secret inherited")
		}
	}
	if os.Getenv("GEMINI_CLI_TRUST_WORKSPACE") != "true" {
		return fmt.Errorf("generated workspace lacks session-only trust")
	}
	if os.Getenv("GEMINI_API_KEY") != geminiTestKey {
		return fmt.Errorf("explicit credential missing")
	}
	cwd, _ := os.Getwd()
	cwd, _ = filepath.EvalSymlinks(cwd)
	home, _ := filepath.EvalSymlinks(os.Getenv("HOME"))
	geminiHome, _ := filepath.EvalSymlinks(os.Getenv("GEMINI_CLI_HOME"))
	if cwd != home || geminiHome != home {
		return fmt.Errorf("home or CLI config was not isolated")
	}
	marker := filepath.Join(geminiHome, ".gemini", "attempt-marker")
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		return fmt.Errorf("previous attempt state leaked")
	}
	if err := os.WriteFile(marker, nil, 0600); err != nil {
		return err
	}
	for _, required := range []string{"--output-format", "json", "--approval-mode", "default", "--extensions", "none", "--model", "explicit-test-model", "--prompt"} {
		if !slices.Contains(args, required) {
			return fmt.Errorf("missing required argument %s", required)
		}
	}
	for _, name := range []string{".env", filepath.Join(".gemini", ".env")} {
		b, err := os.ReadFile(filepath.Join(home, name))
		if err != nil || len(b) != 0 {
			return fmt.Errorf("dotenv search can inherit ambient parent files")
		}
	}
	settingsBytes, err := os.ReadFile(filepath.Join(home, ".gemini", "settings.json"))
	if err != nil {
		return err
	}
	var settings struct {
		Security struct {
			Auth struct {
				SelectedType string `json:"selectedType"`
			} `json:"auth"`
		} `json:"security"`
		Advanced struct {
			IgnoreLocalEnv bool `json:"ignoreLocalEnv"`
		} `json:"advanced"`
		Skills struct {
			Enabled bool `json:"enabled"`
		} `json:"skills"`
		HooksConfig struct {
			Enabled bool `json:"enabled"`
		} `json:"hooksConfig"`
		Experimental struct {
			EnableAgents bool `json:"enableAgents"`
		} `json:"experimental"`
		Tools struct {
			Core []string `json:"core"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(settingsBytes, &settings); err != nil {
		return err
	}
	if settings.Security.Auth.SelectedType != "gemini-api-key" || !settings.Advanced.IgnoreLocalEnv || settings.Skills.Enabled || settings.HooksConfig.Enabled || settings.Experimental.EnableAgents {
		return fmt.Errorf("authentication or isolation policy is wrong")
	}
	var native map[string]json.RawMessage
	if err := json.Unmarshal(settingsBytes, &native); err != nil {
		return err
	}
	if !strings.Contains(string(native["modelConfigs"]), `"responseMimeType": "application/json"`) || !strings.Contains(string(native["modelConfigs"]), `"responseJsonSchema"`) {
		return fmt.Errorf("native source schema was not delivered")
	}
	if !slices.Equal(settings.Tools.Core, []string{"list_directory", "read_file", "grep_search", "glob"}) {
		return fmt.Errorf("unexpected tools enabled")
	}
	return nil
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Generated workspace trust
func TestGeminiRejectsManagedSettings(t *testing.T) {
	e := fakeGeminiExperiment(t)
	for _, name := range []string{"GEMINI_CLI_SYSTEM_SETTINGS_PATH", "GEMINI_CLI_SYSTEM_DEFAULTS_PATH"} {
		t.Run(name, func(t *testing.T) {
			t.Setenv(name, filepath.Join(t.TempDir(), "managed.json"))
			_, stdout, _, status, err := executeGemini(context.Background(), e, AdapterRequest{}, t.TempDir())
			if status != "adapter-error" || err == nil || stdout != "" || !strings.Contains(err.Error(), "policy review") {
				t.Fatal("managed policy was bypassed")
			}
		})
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Fixed-model harness treatments
// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Generated workspace trust
func TestGeminiSettingsPinExecutionPolicy(t *testing.T) {
	e := testExperiment()
	e.Model = "gemini-3.8-flash"
	e.Gemini = &GeminiConfig{Auth: "api-key-env", Reasoning: "low", OutputFormat: "json-schema"}
	b, err := geminiSettings(e)
	if err != nil {
		t.Fatal(err)
	}
	var settings map[string]json.RawMessage
	if err := json.Unmarshal(b, &settings); err != nil {
		t.Fatal(err)
	}
	var contextPolicy struct {
		FileName      string   `json:"fileName"`
		Boundaries    []string `json:"memoryBoundaryMarkers"`
		DirectoryTree bool     `json:"includeDirectoryTree"`
	}
	if err := json.Unmarshal(settings["context"], &contextPolicy); err != nil {
		t.Fatal(err)
	}
	if contextPolicy.FileName != "ATEAM_BENCHMARK_CONTEXT_DISABLED" || contextPolicy.Boundaries == nil || len(contextPolicy.Boundaries) != 0 || contextPolicy.DirectoryTree {
		t.Fatal("ambient instructions or parent context may load")
	}
	var model struct {
		Name        string `json:"name"`
		MaxTurns    int    `json:"maxSessionTurns"`
		DisableLoop bool   `json:"disableLoopDetection"`
		SkipNext    bool   `json:"skipNextSpeakerCheck"`
	}
	if err := json.Unmarshal(settings["model"], &model); err != nil {
		t.Fatal(err)
	}
	if model.Name != "gemini-3.8-flash" || model.MaxTurns != 4 || !model.DisableLoop || !model.SkipNext {
		t.Fatal("model or session policy was not pinned")
	}
	var configs struct {
		Overrides []struct {
			Match  map[string]string `json:"match"`
			Config struct {
				Model      string `json:"model"`
				Generation struct {
					MaxTokens int `json:"maxOutputTokens"`
					Thinking  struct {
						Level   string `json:"thinkingLevel"`
						Include bool   `json:"includeThoughts"`
					} `json:"thinkingConfig"`
				} `json:"generateContentConfig"`
			} `json:"modelConfig"`
		} `json:"customOverrides"`
	}
	if err := json.Unmarshal(settings["modelConfigs"], &configs); err != nil {
		t.Fatal(err)
	}
	if len(configs.Overrides) != 2 || configs.Overrides[0].Match["overrideScope"] != "core" || configs.Overrides[1].Match["model"] != "gemini-3.8-flash" {
		t.Fatal("missing core or selected-model overrides")
	}
	for _, override := range configs.Overrides {
		if override.Config.Model != "gemini-3.8-flash" || override.Config.Generation.MaxTokens != 8192 || override.Config.Generation.Thinking.Level != "LOW" || override.Config.Generation.Thinking.Include {
			t.Fatal("generation model, thinking, or output cap drifted")
		}
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Native agent evidence
func TestGeminiNativeSourceSchema(t *testing.T) {
	e := testExperiment()
	e.Model = "gemini-3.8-flash"
	if err := json.Unmarshal([]byte(`{"auth":"api-key-env","reasoning":"low","output_format":"json-schema"}`), &e.Gemini); err != nil {
		t.Fatal(err)
	}
	data, err := geminiSettings(e)
	if err != nil {
		t.Fatal(err)
	}
	var settings struct {
		ModelConfigs struct {
			Overrides []struct {
				Config struct {
					Generation map[string]json.RawMessage `json:"generateContentConfig"`
				} `json:"modelConfig"`
			} `json:"customOverrides"`
		} `json:"modelConfigs"`
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatal(err)
	}
	if len(settings.ModelConfigs.Overrides) != 2 {
		t.Fatal("missing native model overrides")
	}
	schema := []byte(`{"type":"object","properties":{"source":{"type":"string"}},"required":["source"],"additionalProperties":false}`)
	for _, override := range settings.ModelConfigs.Overrides {
		if string(override.Config.Generation["responseMimeType"]) != `"application/json"` || !jsonEquivalent(schema, override.Config.Generation["responseJsonSchema"]) {
			t.Fatal("native source schema is absent or does not constrain the final JSON object")
		}
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Controlled bounded experiments
func TestGeminiLegacyExecutionRefused(t *testing.T) {
	e := fakeGeminiExperiment(t)
	e.Gemini.OutputFormat = ""
	dir := t.TempDir()
	_, stdout, _, status, err := executeGemini(context.Background(), e, AdapterRequest{}, dir)
	if err == nil || status != "adapter-error" || stdout != "" || !strings.Contains(err.Error(), "output_format json-schema") {
		t.Fatalf("legacy execution not refused before process launch: %s %v", status, err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".gemini")); !os.IsNotExist(err) {
		t.Fatal("legacy execution reached workspace preparation")
	}
	e.Command = []string{"/does-not-exist/no-version-probe"}
	if err := resolveNative(context.Background(), &ResolvedExperiment{}, e); err == nil || !strings.Contains(err.Error(), "output_format json-schema") {
		t.Fatalf("legacy resolution reached CLI probe: %v", err)
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Retained independently graded artifacts
func TestGeminiLegacySettingsReplay(t *testing.T) {
	e := testExperiment()
	e.Model = "gemini-3.8-flash"
	e.Gemini = &GeminiConfig{Auth: "api-key-env", Reasoning: "low"}
	data, err := geminiSettings(e)
	if err != nil {
		t.Fatal(err)
	}
	// Captured native settings from the real, prompt-only capacity canary.
	previous, err := os.ReadFile(filepath.Join("testdata", "gemini", "legacy-settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !jsonEquivalent(data, previous) {
		t.Fatal("legacy Gemini settings no longer reproduce saved canary provenance")
	}
	raw, err := json.Marshal(e.Gemini)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "output_format") {
		t.Fatal("absent output format changed legacy config digest")
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Controlled bounded experiments
func TestGeminiOutputFormatValidation(t *testing.T) {
	e := testExperiment()
	e.Agent, e.Model = "gemini", "gemini-3.8-flash"
	e.MaxTotalTokens, e.MaxRunSeconds = 1000, 60
	e.Gemini = &GeminiConfig{Auth: "api-key-env", Reasoning: "low"}
	for _, format := range []string{"", "json-schema", "unknown"} {
		e.Gemini.OutputFormat = format
		data, _ := json.Marshal(e)
		_, err := ParseExperiment(data)
		if (err == nil) != (format != "unknown") {
			t.Fatalf("output format %q validation: %v", format, err)
		}
	}
}
