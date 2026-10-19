package bench

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
)

const bundleLeftInstructions = "  Gentle α guidance.\r\n\n\nSkill-like text: \"verify\" \\ paths.\n"
const bundleRightInstructions = "  Rigorous β guidance.\n\n\nSkill-like text: \"inspect\" \\ paths.\n"
const bundleDeliveryTask = "Write package safepath with func Resolve(base, rel string) (string, error). Return the absolute lexical path for rel within base, including base itself. Reject paths escaping base and sibling-prefix traversal. Do not resolve symlinks; filesystem/symlink security is outside this task. Return one Go source file."

func resolvedBundleDeliveryExperiment(t *testing.T) Experiment {
	t.Helper()
	dir := t.TempDir()
	for name, content := range map[string]string{
		"left-one.txt": "  Gentle α guidance.\r\n", "left-two.txt": "Skill-like text: \"verify\" \\ paths.\n",
		"right-one.txt": "  Rigorous β guidance.\n", "right-two.txt": "Skill-like text: \"inspect\" \\ paths.\n",
		"left.json":  `{"version":1,"name":"excluded-label-left","files":["left-one.txt","left-two.txt"]}`,
		"right.json": `{"version":1,"name":"excluded-label-right","files":["right-one.txt","right-two.txt"]}`,
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	e := testExperiment()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	e.Command = []string{executable, "-test.run=^TestBundleGenericTransportHelper$", "--", "bundle-delivery"}
	e.Baseline, e.Candidate, e.Mode = "", "", "ab"
	e.Tasks, e.Repetitions = []string{"safe-path"}, 1
	encoded, _ := json.Marshal(e)
	var config map[string]any
	if err := json.Unmarshal(encoded, &config); err != nil {
		t.Fatal(err)
	}
	config["bundles"] = map[string]string{"baseline": "left.json", "candidate": "right.json"}
	encoded, _ = json.Marshal(config)
	parsed, err := ParseExperiment(encoded)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := ResolveExperiment(parsed, dir)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Experiment.Baseline != bundleLeftInstructions || resolved.Experiment.Candidate != bundleRightInstructions {
		t.Fatal("bundle resolution changed ordered instruction bytes")
	}
	if resolved.Experiment.Model != "none" || resolved.Experiment.TimeoutSeconds != 5 || resolved.Experiment.Seed != 42 || resolved.Experiment.Mode != "ab" || resolved.Experiment.Repetitions != 1 || !reflect.DeepEqual(resolved.Experiment.Tasks, []string{"safe-path"}) {
		t.Fatal("bundle resolution changed fixed execution controls")
	}
	return resolved.Experiment
}

// Both arms must deliver their literal ordered bytes; trimming, swapping arms or
// including bundle labels instead of the selected treatment must fail this oracle.
// spec: openspec/changes/benchmark-instruction-bundles/specs/benchmark-experiments/spec.md#Requirement: Immutable instruction bundle resolution
// spec: openspec/changes/benchmark-instruction-bundles/specs/benchmark-experiments/spec.md#Requirement: Instruction evidence is not execution evidence
func TestBundleDeliveryToGenericTransport(t *testing.T) {
	e := resolvedBundleDeliveryExperiment(t)
	for _, arm := range []struct{ name, instructions string }{{"baseline", bundleLeftInstructions}, {"candidate", bundleRightInstructions}} {
		t.Run(arm.name, func(t *testing.T) {
			dir := t.TempDir()
			request := newAdapterRequest(e, Trial{Task: "safe-path", Repetition: 1, Arm: arm.name})
			_, stdout, stderr, status, err := executeAdapter(context.Background(), e, request, dir)
			if err != nil || status != "completed" {
				t.Fatalf("generic transport: status=%s error=%v stdout=%s stderr=%s", status, err, stdout, stderr)
			}
			data, err := os.ReadFile(filepath.Join(dir, "bundle-delivered-request.json"))
			if err != nil {
				t.Fatal(err)
			}
			var received AdapterRequest
			if err := strictJSON(data, &received); err != nil {
				t.Fatal(err)
			}
			want := AdapterRequest{Version: 1, Trial: Trial{Task: "safe-path", Repetition: 1, Arm: arm.name}, Prompt: bundleDeliveryTask, Instructions: arm.instructions}
			if !reflect.DeepEqual(received, want) {
				t.Fatal("generic transport changed instruction bytes or leaked treatment metadata")
			}
		})
	}
}

// The resolved instruction pair is shared by all transports. Native fake configs
// retain their existing fixture models; this tests delivery, not native resolution.
// spec: openspec/changes/benchmark-instruction-bundles/specs/benchmark-experiments/spec.md#Requirement: Instruction evidence is not execution evidence
// spec: openspec/changes/benchmark-instruction-bundles/specs/benchmark-experiments/spec.md#Requirement: Effective treatment comparison
func TestBundleDeliveryToNativeTransports(t *testing.T) {
	resolved := resolvedBundleDeliveryExperiment(t)
	for _, provider := range []struct {
		name, model string
		setup       func(*testing.T) Experiment
		observed    []string
	}{
		{"codex", "gpt-6-luna", fakeCodexExperiment, []string{"agentMessage"}},
		{"claude", "explicit-test-model", fakeClaudeExperiment, nil},
		{"gemini", "explicit-test-model", fakeGeminiExperiment, []string{}},
	} {
		t.Run(provider.name, func(t *testing.T) {
			e := provider.setup(t)
			e.Baseline, e.Candidate = resolved.Baseline, resolved.Candidate
			before, err := json.Marshal(e)
			if err != nil {
				t.Fatal(err)
			}
			for _, arm := range []struct{ name, instructions string }{{"baseline", bundleLeftInstructions}, {"candidate", bundleRightInstructions}} {
				t.Run(arm.name, func(t *testing.T) {
					dir := t.TempDir()
					request := newAdapterRequest(e, Trial{Task: "safe-path", Repetition: 1, Arm: arm.name})
					response, stdout, stderr, status, err := executeAdapter(context.Background(), e, request, dir)
					if err != nil || status != "completed" {
						t.Fatalf("native transport: status=%s error=%v stdout=%s stderr=%s", status, err, stdout, stderr)
					}
					data, err := os.ReadFile(filepath.Join(dir, "bundle-delivered-prompt.txt"))
					if err != nil {
						t.Fatal(err)
					}
					want := "Instructions:\n" + arm.instructions + "\n\nTask:\n" + bundleDeliveryTask + "\n\nReturn a JSON object with source containing the complete Go source file."
					if string(data) != want {
						t.Fatal("native transport changed instruction bytes or leaked treatment metadata")
					}
					if response.Native == nil || !reflect.DeepEqual(response.Native.ObservedItemTypes, provider.observed) || !reflect.DeepEqual(response.Native.CompletedItemTypes, provider.observed) {
						t.Fatal("instruction delivery changed native execution evidence")
					}
					assertBundleFixedNativeControls(t, e, provider.model, dir)
				})
			}
			after, _ := json.Marshal(e)
			if string(before) != string(after) {
				t.Fatal("native attempts changed shared experiment configuration")
			}
		})
	}
}

func assertBundleFixedNativeControls(t *testing.T, e Experiment, model, dir string) {
	t.Helper()
	if e.Model != model || (e.Codex != nil && e.Codex.Reasoning != "low") || (e.Claude != nil && e.Claude.Reasoning != "low") || (e.Gemini != nil && e.Gemini.Reasoning != "low") {
		t.Fatal("native model or reasoning changed between treatments")
	}
	// Codex and Claude helpers validate the delivered effort. Gemini's settings
	// carry the effort, so inspect what the fake CLI received on each attempt.
	if e.Gemini == nil {
		return
	}
	data, err := os.ReadFile(filepath.Join(dir, ".gemini", "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var settings struct {
		ModelConfigs struct {
			Overrides []struct {
				Config struct {
					Model      string `json:"model"`
					Generation struct {
						Thinking struct {
							Level string `json:"thinkingLevel"`
						} `json:"thinkingConfig"`
					} `json:"generateContentConfig"`
				} `json:"modelConfig"`
			} `json:"customOverrides"`
		} `json:"modelConfigs"`
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatal(err)
	}
	if len(settings.ModelConfigs.Overrides) != 2 {
		t.Fatal("Gemini fixed model overrides missing")
	}
	for _, override := range settings.ModelConfigs.Overrides {
		if override.Config.Model != "explicit-test-model" || override.Config.Generation.Thinking.Level != "LOW" {
			t.Fatal("Gemini delivered model or reasoning changed between treatments")
		}
	}
}

// spec: openspec/changes/benchmark-instruction-bundles/specs/benchmark-experiments/spec.md#Requirement: Instruction evidence is not execution evidence
func TestBundleGenericTransportHelper(_ *testing.T) {
	if !slices.Contains(os.Args, "bundle-delivery") {
		return
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil || os.WriteFile("bundle-delivered-request.json", data, 0600) != nil {
		os.Exit(2)
	}
	if err := json.NewEncoder(os.Stdout).Encode(AdapterResponse{Version: 1, Source: "package safepath"}); err != nil {
		os.Exit(3)
	}
	os.Exit(0)
}
