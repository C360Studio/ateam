package bench

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func bundleTestExperiment(t *testing.T, baseline, candidate []string) (Experiment, string) {
	t.Helper()
	dir := t.TempDir()
	for _, arm := range []struct {
		name     string
		contents []string
	}{{"baseline", baseline}, {"candidate", candidate}} {
		root := filepath.Join(dir, arm.name)
		if err := os.Mkdir(root, 0700); err != nil {
			t.Fatal(err)
		}
		manifest := BundleManifest{Version: 1, Name: arm.name}
		for i, content := range arm.contents {
			name := fmt.Sprintf("part-%d.txt", i)
			manifest.Files = append(manifest.Files, name)
			if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0600); err != nil {
				t.Fatal(err)
			}
		}
		data, err := json.Marshal(manifest)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "manifest.json"), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	e := testExperiment()
	e.Tasks, e.Repetitions = []string{"safe-path"}, 1
	e.Baseline, e.Candidate = "", ""
	e.Bundles = &BundleReferences{Baseline: "baseline/manifest.json", Candidate: "candidate/manifest.json"}
	return e, dir
}

// The oracle is literal text and a published SHA-256 test value for "alpha";
// filename labels, whitespace normalization or reversed components must differ.
// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Immutable instruction bundle resolution
func TestBundleImmutableResolution(t *testing.T) {
	e, dir := bundleTestExperiment(t, []string{"alpha", "\ufeffline\r\n"}, []string{"beta"})
	r, err := ResolveExperiment(e, dir)
	if err != nil {
		t.Fatal(err)
	}
	if r.Experiment.Bundles != nil || r.Experiment.Baseline != "alpha\n\n\ufeffline\r\n" || r.Experiment.Candidate != "beta" {
		t.Fatalf("bundle instruction bytes changed: %+v", r.Experiment)
	}
	if r.Bundles.Baseline.Components[0].SHA != "8ed3f6ad685b959ead7022518e1af76cd816f8e8ec7ccdda1ed4018e8f2223f8" {
		t.Fatal("component hash differs from independent SHA-256 oracle")
	}
	if r.Bundles.Baseline.Components[1].Content != "\ufeffline\r\n" || !reflect.DeepEqual(r.Original, e) || r.Bundles.ConfigDir != dir {
		t.Fatal("bundle resolution lost original bytes or configuration")
	}
	if err := validateResolvedExperiment(r); err != nil {
		t.Fatal(err)
	}
	e.Bundles.Baseline = "changed-by-caller.json"
	if r.Original.Bundles.Baseline != "baseline/manifest.json" {
		t.Fatal("caller mutation changed retained bundle references")
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Effective treatment comparison
func TestBundleEffectiveTreatmentEquality(t *testing.T) {
	for _, tc := range []struct {
		name                string
		baseline, candidate []string
		equal               bool
	}{
		{"different labels", []string{"same"}, []string{"same"}, true},
		{"different splits", []string{"alpha", "beta"}, []string{"alpha\n\nbeta"}, true},
		{"different order", []string{"alpha", "beta"}, []string{"beta", "alpha"}, false},
		{"preserve trailing newline", []string{"alpha\n"}, []string{"alpha"}, false},
		{"empty text", []string{""}, []string{""}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e, dir := bundleTestExperiment(t, tc.baseline, tc.candidate)
			for _, mode := range []string{"aa", "ab", ""} {
				e.Mode = mode
				_, err := ResolveExperiment(e, dir)
				want := (mode == "aa") == tc.equal
				if (err == nil) != want {
					t.Fatalf("effective equality mode=%q equal=%t accepted=%t", mode, tc.equal, err == nil)
				}
			}
		})
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Immutable instruction bundle resolution
func TestBundleManifestAndReferenceRefusals(t *testing.T) {
	for _, input := range []string{
		`{"version":1,"version":1,"name":"x","files":["a"]}`,
		`{"version":2,"name":"x","files":["a"]}`,
		`{"version":1,"name":" ","files":["a"]}`,
		`{"version":1,"name":"x","files":[]}`,
		`{"version":1,"name":"x","files":["a","a"]}`,
		`{"version":1,"name":"x","files":["a"],"root":".."}`,
		`{"version":1,"name":"x","files":["a"]} {}`,
	} {
		if _, err := parseBundleManifest([]byte(input)); err == nil {
			t.Fatalf("accepted malformed manifest: %s", input)
		}
	}
	for _, path := range []string{"", ".", "..", "../outside", "/absolute", "a/../b", "a/./b", "a//b", "a/", "C:drive", `a\b`, "https://example.test/a", "a\x00b"} {
		manifest, _ := json.Marshal(BundleManifest{Version: 1, Name: "x", Files: []string{path}})
		if _, err := parseBundleManifest(manifest); err == nil {
			t.Fatalf("accepted escaping/nonportable component %q", path)
		}
		e := testExperiment()
		e.Baseline, e.Candidate = "", ""
		e.Bundles = &BundleReferences{Baseline: path, Candidate: "candidate.json"}
		if err := validateExperiment(e); err == nil {
			t.Fatalf("accepted escaping/nonportable manifest reference %q", path)
		}
	}
	e := testExperiment()
	e.Bundles = &BundleReferences{Baseline: "a.json", Candidate: "b.json"}
	if err := validateExperiment(e); err == nil {
		t.Fatal("accepted ambiguous inline and bundle treatments")
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Immutable instruction bundle resolution
func TestBundleResolutionFileRefusals(t *testing.T) {
	for _, kind := range []string{"missing", "directory", "invalid-utf8", "file-symlink", "directory-symlink", "manifest-symlink"} {
		t.Run(kind, func(t *testing.T) {
			e, dir := bundleTestExperiment(t, []string{"alpha"}, []string{"beta"})
			input := filepath.Join(dir, "baseline", "part-0.txt")
			switch kind {
			case "missing":
				if err := os.Remove(input); err != nil {
					t.Fatal(err)
				}
			case "directory":
				if err := os.Remove(input); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(input, 0700); err != nil {
					t.Fatal(err)
				}
			case "invalid-utf8":
				if err := os.WriteFile(input, []byte{0xff}, 0600); err != nil {
					t.Fatal(err)
				}
			case "file-symlink":
				if err := os.Remove(input); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("../candidate/part-0.txt", input); err != nil {
					t.Fatal(err)
				}
			case "directory-symlink":
				if err := os.Rename(filepath.Join(dir, "baseline"), filepath.Join(dir, "renamed")); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("renamed", filepath.Join(dir, "baseline")); err != nil {
					t.Fatal(err)
				}
			case "manifest-symlink":
				path := filepath.Join(dir, "baseline", "manifest.json")
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("../candidate/manifest.json", path); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := ResolveExperiment(e, dir); err == nil {
				t.Fatalf("accepted unsupported bundle input: %s", kind)
			} else if strings.Contains(kind, "symlink") && !strings.Contains(err.Error(), "without symlinks") {
				t.Fatalf("symlink guard not reached for %s: %v", kind, err)
			}
		})
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Immutable instruction bundle resolution
func TestBundleContentAndManifestBoundaries(t *testing.T) {
	for _, count := range []int{32, 33} {
		files := make([]string, count)
		for i := range files {
			files[i] = fmt.Sprintf("part-%d", i)
		}
		data, _ := json.Marshal(BundleManifest{Version: 1, Name: "x", Files: files})
		if _, err := parseBundleManifest(data); (err == nil) != (count == 32) {
			t.Fatalf("manifest file boundary %d: %v", count, err)
		}
	}
	minimal := []byte(`{"version":1,"name":"x","files":["a"]}`)
	for _, size := range []int{maxBundleManifestBytes, maxBundleManifestBytes + 1} {
		data := append(append([]byte{}, minimal...), bytes.Repeat([]byte(" "), size-len(minimal))...)
		if _, err := parseBundleManifest(data); (err == nil) != (size == maxBundleManifestBytes) {
			t.Fatalf("manifest byte boundary %d: %v", size, err)
		}
	}
	for _, extra := range []int{0, 1} {
		e, dir := bundleTestExperiment(t, []string{strings.Repeat("x", maxBundleInstructionBytes-2+extra), ""}, []string{"beta"})
		r, err := ResolveExperiment(e, dir)
		if (err == nil) != (extra == 0) {
			t.Fatalf("effective byte boundary +%d: %v", extra, err)
		}
		if err == nil && len(r.Experiment.Baseline) != maxBundleInstructionBytes {
			t.Fatal("exact effective boundary not reached")
		}
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Immutable instruction bundle resolution
func TestBundlePredictableArtifactLimit(t *testing.T) {
	for _, tc := range []struct {
		name, text string
		want       bool
	}{
		{"ordinary", strings.Repeat("x", 32<<10), true},
		{"JSON escaped", strings.Repeat("<", 32<<10), false},
		{"long schedule", strings.Repeat("x", 128<<10), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e, dir := bundleTestExperiment(t, []string{tc.text}, []string{tc.text + "different"})
			e.Tasks, e.Repetitions = []string{"safe-path", "sql-user", "gitea-webhook"}, 100
			r, err := ResolveExperiment(e, dir)
			if (err == nil) != tc.want {
				t.Fatalf("predictable artifact bound %s accepted=%t: %v", tc.name, err == nil, err)
			}
			if err == nil && len(r.Schedule) != 600 {
				t.Fatal("maximum schedule boundary not reached")
			}
			if err != nil && !strings.Contains(err.Error(), "scheduled requests exceed 64 MiB") {
				t.Fatalf("wrong refusal: %v", err)
			}
		})
	}
}

func bundleLaunchMarker(t *testing.T, e *Experiment, dir string) string {
	t.Helper()
	marker := filepath.Join(t.TempDir(), "launched")
	if err := os.WriteFile(filepath.Join(dir, "adapter.sh"), []byte("printf launch > \"$1\"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	e.Command = []string{"/bin/sh", "./adapter.sh", marker}
	return marker
}

// Source and metadata edits must fail before any adapter execution. Checking
// effective text alone would silently admit a metadata-only manifest change.
// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Bundle replay independence
func TestBundleStaleResolutionNeverLaunches(t *testing.T) {
	for _, change := range []string{"content", "metadata", "missing", "argv", "budget-policy", "runner", "runtime"} {
		t.Run(change, func(t *testing.T) {
			e, dir := bundleTestExperiment(t, []string{"alpha"}, []string{"beta"})
			marker := bundleLaunchMarker(t, &e, dir)
			r, err := ResolveExperiment(e, dir)
			if err != nil {
				t.Fatal(err)
			}
			switch change {
			case "content":
				err = os.WriteFile(filepath.Join(dir, "baseline", "part-0.txt"), []byte("changed"), 0600)
			case "metadata":
				err = os.WriteFile(filepath.Join(dir, "baseline", "manifest.json"), []byte(`{"version":1,"name":"new label","files":["part-0.txt"]}`), 0600)
			case "missing":
				err = os.Remove(filepath.Join(dir, "baseline", "part-0.txt"))
			case "argv":
				r.Experiment.Command = append(r.Experiment.Command, "extra-argument")
			case "budget-policy":
				r.BudgetPolicy = "invented policy"
			case "runner":
				r.RunnerSHA = "different runner"
			case "runtime":
				r.Runtime = "different runtime"
			}
			if err != nil {
				t.Fatal(err)
			}
			if _, err := RunExperiment(context.Background(), r, t.TempDir()); err == nil {
				t.Fatalf("stale bundle resolution accepted: %s", change)
			}
			if _, err := os.Stat(marker); !os.IsNotExist(err) {
				t.Fatalf("adapter launched after stale %s: %v", change, err)
			}
		})
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Bundle replay independence
func TestBundleTamperedSnapshotRefusal(t *testing.T) {
	e, dir := bundleTestExperiment(t, []string{"alpha", "second"}, []string{"beta"})
	r, err := ResolveExperiment(e, dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, mutate := range []func(*ResolvedExperiment){
		func(r *ResolvedExperiment) { r.Bundles.Baseline.Components[0].Content = "changed" },
		func(r *ResolvedExperiment) { r.Bundles.Baseline.Components[0].SHA = "wrong" },
		func(r *ResolvedExperiment) {
			r.Bundles.Baseline.Components[0], r.Bundles.Baseline.Components[1] = r.Bundles.Baseline.Components[1], r.Bundles.Baseline.Components[0]
		},
		func(r *ResolvedExperiment) { r.Bundles.Baseline.ManifestContent += " " },
		func(r *ResolvedExperiment) { r.Bundles.Baseline.ManifestPath = "other.json" },
		func(r *ResolvedExperiment) { r.Bundles.Baseline.Name = "other" },
		func(r *ResolvedExperiment) { r.Bundles.Baseline.Instructions = "changed" },
		func(r *ResolvedExperiment) { r.Bundles.Baseline.InstructionsSHA = "wrong" },
		func(r *ResolvedExperiment) { r.Experiment.Baseline = "changed" },
		func(r *ResolvedExperiment) { r.Experiment.Model = "changed" },
		func(r *ResolvedExperiment) { r.Experiment.Seed++ },
		func(r *ResolvedExperiment) {
			content := strings.Repeat("x", maxBundleInstructionBytes+1)
			r.Bundles.Baseline.Components[0].Content = content
			r.Bundles.Baseline.Components[0].SHA = digest([]byte(content))
			r.Bundles.Baseline.Instructions = content + "\n\nsecond"
			r.Bundles.Baseline.InstructionsSHA = digest([]byte(r.Bundles.Baseline.Instructions))
			r.Experiment.Baseline = r.Bundles.Baseline.Instructions
		},
		func(r *ResolvedExperiment) { r.Bundles.ConfigDir = "relative" },
		func(r *ResolvedExperiment) { r.Bundles = nil },
	} {
		data, _ := json.Marshal(r)
		var changed ResolvedExperiment
		if err := json.Unmarshal(data, &changed); err != nil {
			t.Fatal(err)
		}
		mutate(&changed)
		if err := validateResolvedExperiment(changed); err == nil {
			t.Fatal("accepted internally inconsistent bundle snapshot")
		}
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Bundle replay independence
func TestBundleReplayAfterSourceRemoval(t *testing.T) {
	e, dir := bundleTestExperiment(t, []string{"good"}, []string{"bad"})
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	script := "exec '" + strings.ReplaceAll(executable, "'", "'\\''") + "' -test.run=^TestCommandAdapterHelper$ -- fixture\n"
	if err := os.WriteFile(filepath.Join(dir, "adapter.sh"), []byte(script), 0600); err != nil {
		t.Fatal(err)
	}
	e.Command = []string{"/bin/sh", "./adapter.sh"}
	t.Chdir(dir)
	r, err := ResolveExperiment(e, ".")
	if err != nil {
		t.Fatal(err)
	}
	// Neither the config-relative launcher nor bundle refs depend on the cwd.
	t.Chdir(t.TempDir())
	path, err := RunExperiment(context.Background(), r, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range []string{"baseline", "candidate", "adapter.sh"} {
		if err := os.RemoveAll(filepath.Join(dir, source)); err != nil {
			t.Fatal(err)
		}
	}
	run, err := LoadExperimentRun(path)
	if err != nil {
		t.Fatal(err)
	}
	comparison, err := CompareExperiment(run)
	if err != nil || comparison.Regressions != 1 {
		t.Fatalf("source-independent comparison changed outcomes: %+v %v", comparison, err)
	}
	replay, err := RescoreExperiment(context.Background(), path)
	if err != nil {
		t.Fatalf("replay reread removed sources or adapter: %v", err)
	}
	rescored, err := LoadExperimentRun(replay)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(rescored.Resolved, run.Resolved) || rescored.Attempts[0].Request.Instructions != "good" || rescored.Attempts[1].Request.Instructions != "bad" {
		t.Fatal("replay changed immutable treatments")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("replay modified original artifact")
	}
	// An attacker can recompute the outer checksum; inner provenance still must agree.
	run.Resolved.Bundles.Baseline.Components[0].Content = "tampered"
	run.Digest = runDigest(run)
	if err := validateRun(run); err == nil {
		t.Fatal("outer digest concealed tampered retained component")
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Bundle replay independence
func TestBundleInlineCompatibility(t *testing.T) {
	e := testExperiment()
	r, err := ResolveExperiment(e, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte(`"bundles"`)) {
		t.Fatal("inline artifact gained bundle fields")
	}
	if err := validateResolvedExperiment(r); err != nil {
		t.Fatal(err)
	}
	r.Bundles = &BundleResolution{}
	if err := validateResolvedExperiment(r); err == nil {
		t.Fatal("inline artifact accepted unrelated bundle evidence")
	}
}

// RawMessage indentation changes during artifact serialization, but the native
// settings are the same JSON object and must not become a false stale refusal.
// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Bundle replay independence
func TestBundleNativeResolutionJSONRoundTrip(t *testing.T) {
	bundle, dir := bundleTestExperiment(t, []string{"same"}, []string{"same"})
	e := nativeTestExperiment(t, `{"input_tokens":10,"output_tokens":5}`)
	e.Baseline, e.Candidate, e.Bundles = "", "", bundle.Bundles
	resolved, err := ResolveExperiment(e, dir)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.MarshalIndent(resolved, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	var restored ResolvedExperiment
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(resolved.NativeSettings, restored.NativeSettings) {
		t.Fatal("native settings indentation boundary not reached")
	}
	path, err := RunExperiment(context.Background(), restored, t.TempDir())
	if err != nil {
		t.Fatalf("native bundle JSON round trip falsely stale: %v", err)
	}
	run, err := LoadExperimentRun(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(run.Attempts) != 2 || run.Attempts[0].Status != "passed" || run.Attempts[1].Status != "passed" {
		t.Fatal("native bundle round trip did not execute both fixtures")
	}
	restored.NativeSettings = json.RawMessage(`{}`)
	if path, err := RunExperiment(context.Background(), restored, t.TempDir()); err == nil || path != "" {
		t.Fatal("native settings semantic change survived round-trip normalization")
	}
}

// Reference composition appends literal bytes and the specified separator. It
// checks empty, Unicode/BOM, CRLF and generated arbitrary-byte histories without
// deriving the oracle from production joining or hashing code.
// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Immutable instruction bundle resolution
func FuzzBundleComposition(f *testing.F) {
	f.Add("alpha", "beta", uint8(1))
	f.Add("\ufefffirst\r\n", "last\n", uint8(31))
	f.Add("", "", uint8(2))
	f.Fuzz(func(t *testing.T, first, second string, count uint8) {
		if len(first) > 4096 || len(second) > 4096 {
			return
		}
		n := int(count%32) + 1
		var expected []byte
		components := make([]BundleComponent, n)
		for i := range components {
			text := first
			if i%2 != 0 {
				text = second
			}
			components[i] = BundleComponent{Content: text}
			if i > 0 {
				expected = append(expected, '\n', '\n')
			}
			expected = append(expected, []byte(text)...)
		}
		if composeBundle(components) != string(expected) {
			t.Fatal("bundle composition changed ordered exact bytes")
		}
	})
}
