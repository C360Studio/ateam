package bench

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"unicode/utf8"
)

const maxBundleManifestBytes = 64 << 10
const maxBundleFiles = 32
const maxBundleInstructionBytes = 1 << 20
const maxExperimentArtifactBytes = 64 << 20

// BundleReferences selects explicit manifests relative to the experiment file.
type BundleReferences struct {
	Baseline  string `json:"baseline"`
	Candidate string `json:"candidate"`
}

// BundleManifest lists ordered inputs beneath the manifest's own directory.
type BundleManifest struct {
	Version int      `json:"version"`
	Name    string   `json:"name"`
	Files   []string `json:"files"`
}

// BundleComponent retains exact UTF-8 input bytes, including line endings.
type BundleComponent struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	SHA     string `json:"sha256"`
}

// BundleSnapshot records delivered instructions, not native skill execution.
type BundleSnapshot struct {
	ManifestPath    string            `json:"manifest_path"`
	ManifestContent string            `json:"manifest_content"`
	ManifestSHA     string            `json:"manifest_sha256"`
	Name            string            `json:"name"`
	Components      []BundleComponent `json:"components"`
	Instructions    string            `json:"instructions"`
	InstructionsSHA string            `json:"instructions_sha256"`
}

// BundleResolution anchors future source checks while retaining offline evidence.
type BundleResolution struct {
	ConfigDir string         `json:"config_dir"`
	Baseline  BundleSnapshot `json:"baseline"`
	Candidate BundleSnapshot `json:"candidate"`
}

func validateBundlePath(path string) error {
	if path == "." || !fs.ValidPath(path) || strings.ContainsAny(path, "\\:\x00") || !filepath.IsLocal(path) {
		return fmt.Errorf("bundle path %q must be a portable relative path without traversal", path)
	}
	return nil
}

func parseBundleManifest(data []byte) (BundleManifest, error) {
	var m BundleManifest
	if len(data) > maxBundleManifestBytes || !utf8.Valid(data) {
		return m, fmt.Errorf("bundle manifest must be UTF-8 and at most 64 KiB")
	}
	if err := uniqueJSON(data); err != nil {
		return m, fmt.Errorf("bundle manifest: %w", err)
	}
	if err := strictJSON(data, &m); err != nil {
		return m, fmt.Errorf("bundle manifest: %w", err)
	}
	if m.Version != 1 || strings.TrimSpace(m.Name) == "" || len(m.Files) < 1 || len(m.Files) > maxBundleFiles {
		return m, fmt.Errorf("bundle manifest requires version 1, a name and 1..32 files")
	}
	seen := make(map[string]bool, len(m.Files))
	for _, path := range m.Files {
		if err := validateBundlePath(path); err != nil {
			return m, err
		}
		if seen[path] {
			return m, fmt.Errorf("duplicate bundle file %q", path)
		}
		seen[path] = true
	}
	return m, nil
}

// Read through an anchored root after rejecting stable symlinks on every
// component. Root prevents escape during path races; descriptor checks reject
// nonregular files and replacement of the inspected file. This is not a lock
// against adversarial edits to the contents of an already-open regular file.
func readBundleFile(root *os.Root, path string, limit int) ([]byte, error) {
	if err := validateBundlePath(path); err != nil {
		return nil, err
	}
	var inspected os.FileInfo
	parts := strings.Split(path, "/")
	for i := range parts {
		info, err := root.Lstat(filepath.FromSlash(strings.Join(parts[:i+1], "/")))
		if err != nil {
			return nil, fmt.Errorf("inspect bundle input %q: %w", path, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || (i < len(parts)-1 && !info.IsDir()) || (i == len(parts)-1 && !info.Mode().IsRegular()) {
			return nil, fmt.Errorf("bundle input %q requires regular files and directories without symlinks", path)
		}
		inspected = info
	}
	file, err := root.OpenFile(filepath.FromSlash(path), bundleReadFlags, 0)
	if err != nil {
		return nil, fmt.Errorf("open bundle input %q: %w", path, err)
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("stat bundle input %q: %w", path, err)
	}
	if !opened.Mode().IsRegular() || !os.SameFile(inspected, opened) {
		return nil, fmt.Errorf("bundle input %q changed while opening", path)
	}
	data, err := readBounded(file, int64(limit))
	if err != nil {
		return nil, fmt.Errorf("read bundle input %q: %w", path, err)
	}
	if !utf8.Valid(data) {
		return nil, fmt.Errorf("bundle input %q is not UTF-8", path)
	}
	return data, nil
}

func resolveBundle(root *os.Root, manifestPath string) (BundleSnapshot, error) {
	s := BundleSnapshot{ManifestPath: manifestPath}
	data, err := readBundleFile(root, manifestPath, maxBundleManifestBytes)
	if err != nil {
		return s, err
	}
	m, err := parseBundleManifest(data)
	if err != nil {
		return s, err
	}
	s.ManifestContent, s.ManifestSHA, s.Name = string(data), digest(data), m.Name
	inputs, err := root.OpenRoot(filepath.Dir(filepath.FromSlash(manifestPath)))
	if err != nil {
		return s, fmt.Errorf("open bundle directory: %w", err)
	}
	defer inputs.Close()
	remaining := maxBundleInstructionBytes - 2*(len(m.Files)-1)
	for _, path := range m.Files {
		content, err := readBundleFile(inputs, path, remaining)
		if err != nil {
			return s, err
		}
		remaining -= len(content)
		s.Components = append(s.Components, BundleComponent{Path: path, Content: string(content), SHA: digest(content)})
	}
	s.Instructions = composeBundle(s.Components)
	s.InstructionsSHA = digest([]byte(s.Instructions))
	return s, nil
}

func composeBundle(components []BundleComponent) string {
	text := make([]string, len(components))
	for i, component := range components {
		text[i] = component.Content
	}
	return strings.Join(text, "\n\n")
}

func resolveBundles(refs *BundleReferences, base string) (*BundleResolution, error) {
	base, err := filepath.Abs(base)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(base)
	if err != nil {
		return nil, fmt.Errorf("open experiment directory: %w", err)
	}
	defer root.Close()
	r := &BundleResolution{ConfigDir: base}
	r.Baseline, err = resolveBundle(root, refs.Baseline)
	if err != nil {
		return nil, fmt.Errorf("baseline bundle: %w", err)
	}
	r.Candidate, err = resolveBundle(root, refs.Candidate)
	if err != nil {
		return nil, fmt.Errorf("candidate bundle: %w", err)
	}
	return r, nil
}

// validateBundleSnapshot uses retained bytes only, including original manifest
// ordering. Checksums detect inconsistent evidence, not coordinated forgery.
func validateBundleSnapshot(s BundleSnapshot, manifestPath string) error {
	if s.ManifestPath != manifestPath || s.ManifestSHA != digest([]byte(s.ManifestContent)) {
		return fmt.Errorf("bundle manifest provenance mismatch")
	}
	m, err := parseBundleManifest([]byte(s.ManifestContent))
	if err != nil {
		return err
	}
	if s.Name != m.Name || len(s.Components) != len(m.Files) {
		return fmt.Errorf("bundle component provenance mismatch")
	}
	remaining := maxBundleInstructionBytes - 2*(len(m.Files)-1)
	for i, component := range s.Components {
		if component.Path != m.Files[i] || !utf8.ValidString(component.Content) || len(component.Content) > remaining || component.SHA != digest([]byte(component.Content)) {
			return fmt.Errorf("bundle component %d content/order/hash mismatch", i)
		}
		remaining -= len(component.Content)
	}
	if s.Instructions != composeBundle(s.Components) || s.InstructionsSHA != digest([]byte(s.Instructions)) {
		return fmt.Errorf("bundle effective instructions mismatch")
	}
	return nil
}

func validateResolvedExperiment(r ResolvedExperiment) error {
	if r.Experiment.Gemini != nil && r.GeminiVersion != geminiCLIVersion(r.Experiment.Gemini) {
		return fmt.Errorf("Gemini version provenance does not match configured cli_version")
	}
	if err := validateExperiment(r.Original); err != nil {
		return err
	}
	if err := validateExperiment(r.Experiment); err != nil {
		return err
	}
	expected := r.Original
	expected.Command = r.Experiment.Command
	if expected.Bundles == nil {
		if r.Bundles != nil {
			return fmt.Errorf("unexpected bundle evidence on inline experiment")
		}
	} else {
		if r.Bundles == nil || !filepath.IsAbs(r.Bundles.ConfigDir) || filepath.Clean(r.Bundles.ConfigDir) != r.Bundles.ConfigDir {
			return fmt.Errorf("missing bundle evidence or absolute experiment directory")
		}
		if err := validateBundleSnapshot(r.Bundles.Baseline, expected.Bundles.Baseline); err != nil {
			return err
		}
		if err := validateBundleSnapshot(r.Bundles.Candidate, expected.Bundles.Candidate); err != nil {
			return err
		}
		expected.Baseline, expected.Candidate = r.Bundles.Baseline.Instructions, r.Bundles.Candidate.Instructions
		expected.Bundles = nil
	}
	if r.ConfigSHA != jsonDigest(r.Original) || !reflect.DeepEqual(expected, r.Experiment) {
		return fmt.Errorf("original configuration provenance mismatch")
	}
	return boundBundleArtifact(r)
}

// Compute the exact indented JSON size of a run containing all known requests
// and zero-valued outcomes, without allocating a schedule-sized string. Actual
// outputs and diagnostics can still exceed the existing artifact cap later.
func boundBundleArtifact(r ResolvedExperiment) error {
	if r.Bundles == nil {
		return nil
	}
	empty := ExperimentRun{Version: experimentVersion, Resolved: r, Attempts: []Attempt{}, Digest: strings.Repeat("0", 64)}
	data, err := json.MarshalIndent(empty, "", "  ")
	if err != nil {
		return err
	}
	size := len(data)
	schedule := Schedule(r.Experiment)
	if len(schedule) > 0 {
		// Replace [] with [\n + indented attempts + \n  ].
		size += 4
	}
	for i, trial := range schedule {
		data, err := json.MarshalIndent(Attempt{Request: newAdapterRequest(r.Experiment, trial)}, "    ", "  ")
		if err != nil {
			return err
		}
		size += 4 + len(data)
		if i > 0 {
			size += 2 // comma and newline between attempts
		}
		if size > maxExperimentArtifactBytes {
			return fmt.Errorf("bundle snapshot and scheduled requests exceed 64 MiB artifact limit; reduce instruction size or repetitions")
		}
	}
	return nil
}
