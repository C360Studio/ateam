package bench

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func geminiBudgetConfigJSON(t *testing.T, value string) []byte {
	t.Helper()
	e := testExperiment()
	e.Agent, e.Model = "gemini", "gemini-3.8-flash"
	e.MaxTotalTokens, e.MaxRunSeconds = 1000, 60
	e.Gemini = &GeminiConfig{Auth: "api-key-env", Reasoning: "low", OutputFormat: "json-schema"}
	data, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	if value != "" {
		data = bytes.Replace(data, []byte(`"gemini":{`), []byte(`"gemini":{"max_output_tokens":`+value+`,`), 1)
	}
	return data
}

func assertGeminiOutputAllowance(t *testing.T, data []byte, want int) {
	t.Helper()
	var settings struct {
		ModelConfigs struct {
			CustomOverrides []struct {
				ModelConfig struct {
					GenerateContentConfig struct {
						MaxOutputTokens int `json:"maxOutputTokens"`
					} `json:"generateContentConfig"`
				} `json:"modelConfig"`
			} `json:"customOverrides"`
		} `json:"modelConfigs"`
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatal(err)
	}
	if len(settings.ModelConfigs.CustomOverrides) != 2 {
		t.Fatal("expected core and model overrides")
	}
	for i, override := range settings.ModelConfigs.CustomOverrides {
		if got := override.ModelConfig.GenerateContentConfig.MaxOutputTokens; got != want {
			t.Fatalf("output allowance override %d = %d; want %d", i, got, want)
		}
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Explicit Gemini output allowance
func TestGeminiOutputAllowance(t *testing.T) {
	// Finite equivalence classes cover omitted/null, both inclusive boundaries,
	// an interior override, and explicit default; no interacting state or history.
	var defaultSettings []byte
	var defaultDigest string
	for _, tc := range []struct {
		value string
		want  int
	}{{"", 8192}, {"null", 8192}, {"8192", 8192}, {"1", 1}, {"32768", 32768}, {"65536", 65536}} {
		t.Run("value="+tc.value, func(t *testing.T) {
			e, err := ParseExperiment(geminiBudgetConfigJSON(t, tc.value))
			if err != nil {
				t.Fatalf("valid output allowance refused: %v", err)
			}
			settings, err := geminiSettings(e)
			if err != nil {
				t.Fatal(err)
			}
			assertGeminiOutputAllowance(t, settings, tc.want)
			raw, err := json.Marshal(e.Gemini)
			if err != nil {
				t.Fatal(err)
			}
			switch tc.value {
			case "":
				defaultSettings, defaultDigest = settings, jsonDigest(e)
				if string(raw) != `{"reasoning":"low","auth":"api-key-env","output_format":"json-schema"}` {
					t.Fatalf("omission changed existing config serialization: %s", raw)
				}
			case "null":
				if !bytes.Equal(settings, defaultSettings) || jsonDigest(e) != defaultDigest || bytes.Contains(raw, []byte("max_output_tokens")) {
					t.Fatal("null changed default settings, serialization or digest")
				}
			case "8192":
				if !bytes.Equal(settings, defaultSettings) || jsonDigest(e) == defaultDigest || !bytes.Contains(raw, []byte(`"max_output_tokens":8192`)) {
					t.Fatal("explicit default must preserve settings but record explicit provenance")
				}
			}
		})
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Explicit Gemini output allowance
func TestGeminiOutputAllowanceRefusal(t *testing.T) {
	for _, value := range []string{"0", "-1", "65537", "1.5", `"32768"`} {
		t.Run(value, func(t *testing.T) {
			data := geminiBudgetConfigJSON(t, value)
			if _, err := ParseExperiment(data); err == nil || !strings.Contains(err.Error(), "max_output_tokens") {
				t.Fatalf("invalid output allowance %s not refused: %v", value, err)
			}
			var e Experiment
			if err := json.Unmarshal(data, &e); err != nil {
				return // Noninteger values are rejected by the typed JSON boundary.
			}
			// Resolution must refuse even when callers bypass ParseExperiment, before
			// command lookup/version probing or any need for a provider credential.
			e.Command = []string{filepath.Join(t.TempDir(), "must-not-exist")}
			t.Setenv("GEMINI_API_KEY", "")
			if _, err := ResolveExperiment(e, "."); err == nil || !strings.Contains(err.Error(), "max_output_tokens") {
				t.Fatalf("invalid output allowance reached command resolution: %v", err)
			}
		})
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Explicit Gemini output allowance
func TestGeminiOutputAllowanceDelivered(t *testing.T) {
	e := fakeGeminiExperiment(t)
	if err := json.Unmarshal([]byte(`{"reasoning":"low","auth":"api-key-env","output_format":"json-schema","max_output_tokens":32768}`), &e.Gemini); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	req := newAdapterRequest(e, Trial{Task: "safe-path", Repetition: 1, Arm: "baseline"})
	_, stdout, stderr, status, err := executeGemini(context.Background(), e, req, dir)
	if err != nil || status != "completed" {
		t.Fatalf("status=%s err=%v stderr=%s stdout=%s", status, err, stderr, stdout)
	}
	settings, err := os.ReadFile(filepath.Join(dir, ".gemini", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	assertGeminiOutputAllowance(t, settings, 32768)
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Explicit Gemini output allowance
func TestGeminiOutputAllowanceStaleSettings(t *testing.T) {
	e := nativeTestExperiment(t, `null`)
	legacy, err := ResolveExperiment(e, ".")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(legacy.BudgetPolicy, "4 model turns/attempt and 8192 output tokens/turn") {
		t.Fatal("omission changed default budget policy")
	}
	if err := json.Unmarshal([]byte(`{"reasoning":"low","auth":"api-key-env","output_format":"json-schema","max_output_tokens":32768}`), &e.Gemini); err != nil {
		t.Fatal(err)
	}
	resolved, err := ResolveExperiment(e, ".")
	if err != nil {
		t.Fatal(err)
	}
	assertGeminiOutputAllowance(t, resolved.NativeSettings, 32768)
	if !strings.Contains(resolved.BudgetPolicy, "4 model turns/attempt and 32768 output tokens/turn") {
		t.Fatal("explicit allowance missing from budget policy")
	}
	if err := revalidateExperiment(context.Background(), resolved); err != nil {
		t.Fatalf("fresh resolved allowance refused: %v", err)
	}
	run := ExperimentRun{Version: 1, Created: time.Now().UTC(), Resolved: resolved, StopReason: "cancelled"}
	run.Digest = runDigest(run)
	if err := validateRun(run); err != nil {
		t.Fatalf("fresh replay allowance refused: %v", err)
	}
	resolved.NativeSettings = bytes.ReplaceAll(resolved.NativeSettings, []byte(`"maxOutputTokens": 32768`), []byte(`"maxOutputTokens": 8192`))
	outputRoot := filepath.Join(t.TempDir(), "must-not-be-created")
	if _, err := RunExperiment(context.Background(), resolved, outputRoot); err == nil || !strings.Contains(err.Error(), "changed before execution") {
		t.Fatalf("stale output allowance reached execution: %v", err)
	}
	if _, err := os.Stat(outputRoot); !os.IsNotExist(err) {
		t.Fatal("stale allowance created an execution output directory")
	}
	run.Resolved = resolved
	run.Digest = runDigest(run)
	if err := validateRun(run); err == nil || !strings.Contains(err.Error(), "settings provenance mismatch") {
		t.Fatalf("stale output allowance accepted during replay: %v", err)
	}
}
