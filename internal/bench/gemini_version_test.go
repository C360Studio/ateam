package bench

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Versioned Gemini best-effort accounting
func TestGeminiStableVersionSelection(t *testing.T) {
	cfg := nativeTestExperiment(t, `{"input_tokens":10,"output_tokens":5}`)
	cfg.Gemini.CLIVersion = "0.64.0"
	script, err := os.ReadFile(cfg.Command[0])
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(cfg.Command[0], []byte(strings.ReplaceAll(string(script), "0.43.0", "0.64.0")), 0700); err != nil {
		t.Fatal(err)
	}
	resolved, err := ResolveExperiment(cfg, ".")
	if err != nil {
		t.Fatalf("stable CLI selection refused: %v", err)
	}
	if resolved.GeminiVersion != "0.64.0" || !strings.Contains(resolved.BudgetPolicy, "best-effort") {
		t.Fatalf("stable version/accounting policy missing: %+v", resolved)
	}
	var settings map[string]json.RawMessage
	if json.Unmarshal(resolved.NativeSettings, &settings) != nil || !strings.Contains(string(settings["privacy"]), `"usageStatisticsEnabled": false`) {
		t.Fatal("stable CLI usage statistics not explicitly disabled")
	}
	resolved.GeminiVersion = "0.43.0"
	if err := validateResolvedExperiment(resolved); err == nil || !strings.Contains(err.Error(), "Gemini version") {
		t.Fatalf("retained version mismatch accepted: %v", err)
	}
	current, err := os.ReadFile(cfg.Command[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg.Command[0], []byte(strings.ReplaceAll(string(current), "0.64.0", "0.43.0")), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveExperiment(cfg, "."); err == nil || !strings.Contains(err.Error(), "CLI 0.64.0 is required") {
		t.Fatalf("mismatched installed CLI accepted: %v", err)
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Versioned Gemini best-effort accounting
func TestGeminiLegacyExecutionRefusal(t *testing.T) {
	for _, version := range []string{"", "0.43.0"} {
		t.Run("version="+version, func(t *testing.T) {
			cfg := testExperiment()
			cfg.MaxTotalTokens, cfg.MaxRunSeconds = 1000, 60
			cfg.Agent = "gemini"
			cfg.Model = "gemini-3.8-flash"
			cfg.Command = []string{"/nonexistent/no-version-probe"}
			cfg.Gemini = &GeminiConfig{CLIVersion: version, Auth: "api-key-env", Reasoning: "low", OutputFormat: "json-schema"}
			if err := validateExperiment(cfg); err != nil {
				t.Fatalf("legacy configuration cannot be replayed: %v", err)
			}
			if _, err := ResolveExperiment(cfg, "."); err == nil || !strings.Contains(err.Error(), "cli_version") {
				t.Fatalf("legacy execution reached command lookup: %v", err)
			}
			t.Setenv("GEMINI_API_KEY", "")
			_, _, _, _, err := executeGemini(context.Background(), cfg, AdapterRequest{}, t.TempDir())
			if err == nil || !strings.Contains(err.Error(), "cli_version") {
				t.Fatalf("legacy execution reached credentials: %v", err)
			}
		})
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Versioned Gemini best-effort accounting
func TestGeminiVersionSchema(t *testing.T) {
	for _, version := range []string{"", "0.43.0", "0.64.0", "0.65.0", "latest", " 0.64.0"} {
		cfg := testExperiment()
		cfg.MaxTotalTokens, cfg.MaxRunSeconds = 1000, 60
		cfg.Agent = "gemini"
		cfg.Model = "gemini-3.8-flash"
		cfg.Gemini = &GeminiConfig{CLIVersion: version, Auth: "api-key-env", Reasoning: "low", OutputFormat: "json-schema"}
		err := validateExperiment(cfg)
		valid := version == "" || version == "0.43.0" || version == "0.64.0"
		if (err == nil) != valid {
			t.Fatalf("version %q valid=%t error=%v", version, valid, err)
		}
		if version == "" {
			data, _ := json.Marshal(cfg.Gemini)
			if strings.Contains(string(data), "cli_version") {
				t.Fatal("omitted historical version changed serialization")
			}
		}
	}
}
