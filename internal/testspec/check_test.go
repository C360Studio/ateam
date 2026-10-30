package testspec

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

const citation = "// spec: openspec/specs/demo/spec.md#Requirement: Works\n"

func fixture(t *testing.T, source, spec string) string {
	t.Helper()
	root := t.TempDir()
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	for path, text := range map[string]string{"sample_test.go": source, "openspec/specs/demo/spec.md": spec} {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(text), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// spec: openspec/specs/test-attribution/spec.md#Requirement: Attached test citations
func TestGoAttribution(t *testing.T) {
	for _, tc := range []struct{ name, source, want string }{
		{"missing", "package p\nfunc TestWorks(t *testing.T) {}", "sample_test.go:2: TestWorks: missing spec citation"},
		{"valid", "package p\n" + citation + "func TestWorks(t *testing.T) {}", ""},
		{"gap", "package p\n" + citation + "\nfunc TestWorks(t *testing.T) {}", "missing spec citation"},
		{"malformed sibling", "package p\n" + citation + "// spec: bad\nfunc TestWorks(t *testing.T) {}", "malformed spec citation"},
		{"fixture", "package p\nvar fixture = `func TestFake(t *testing.T) {}`", ""},
		{"main and helper", "package p\nfunc TestMain(m *testing.M) {}\nfunc Testhelper() {}", ""},
		{"fuzz", "package p\nfunc FuzzWorks(f *testing.F) {}", "FuzzWorks: missing spec citation"},
		{"non executable example", "package p\nfunc ExampleWorks() {}", ""},
		{"executable example", "package p\nfunc ExampleWorks() {\n// Output: ok\n}", "ExampleWorks: missing spec citation"},
		{"whole file example", "package p\nvar value = 1\nfunc Example() {\n// Output: ok\n}", "Example: missing spec citation"},
		{"parse error", "package p\nfunc TestBroken(", "parse Go"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := fixture(t, tc.source, "### Requirement: Works\n")
			ds, err := Check(context.Background(), root, "")
			if err != nil {
				t.Fatal(err)
			}
			got := Format(ds)
			if tc.want == "" && got != "" || tc.want != "" && !strings.Contains(got, tc.want) {
				t.Fatalf("attribution verdict: got %q want %q", got, tc.want)
			}
		})
	}
}

// spec: openspec/specs/test-attribution/spec.md#Requirement: Safe requirement resolution
func TestReferenceRefusals(t *testing.T) {
	for _, tc := range []struct{ name, ref, spec, want string }{
		{"missing", "openspec/specs/demo/spec.md#Requirement: Absent", "### Requirement: Works\n", "requirement resolves 0 times"},
		{"duplicate", "openspec/specs/demo/spec.md#Requirement: Works", "### Requirement: Works\n### Requirement: Works\n", "requirement resolves 2 times"},
		{"traversal", "openspec/specs/../demo/spec.md#Requirement: Works", "", "unsafe spec path"},
		{"absolute", "/tmp/spec.md#Requirement: Works", "", "unsafe spec path"},
		{"arbitrary markdown", "README.md#Requirement: Works", "", "expected canonical or in-flight spec.md"},
		{"long fence", "openspec/specs/demo/spec.md#Requirement: Works", "````md\n```\n### Requirement: Works\n````\n", "requirement resolves 0 times"},
		{"fenced heading", "openspec/specs/demo/spec.md#Requirement: Works", "```md\n### Requirement: Works\n```\n", "requirement resolves 0 times"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := fixture(t, "package p\n// spec: "+tc.ref+"\nfunc TestWorks(t *testing.T) {}", tc.spec)
			ds, err := Check(context.Background(), root, "")
			if err != nil {
				t.Fatal(err)
			}
			if got := Format(ds); !strings.Contains(got, tc.want) {
				t.Fatalf("reference refusal: got %q want %q", got, tc.want)
			}
		})
	}
	t.Run("symlink escape", func(t *testing.T) {
		root := fixture(t, "package p\n"+citation+"func TestWorks(t *testing.T) {}", "### Requirement: Works\n")
		target := filepath.Join(t.TempDir(), "spec.md")
		if err := os.WriteFile(target, []byte("### Requirement: Works\n"), 0644); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(root, "openspec/specs/demo/spec.md")
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, path); err != nil {
			t.Fatal(err)
		}
		ds, err := Check(context.Background(), root, "")
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(Format(ds), "escapes repository") {
			t.Fatalf("escape accepted: %s", Format(ds))
		}
	})
}

// spec: openspec/specs/test-attribution/spec.md#Requirement: Deterministic discovery and verification
func TestDiscoveryAndDeterminism(t *testing.T) {
	root := fixture(t, "package p\nfunc TestZ(t *testing.T) {}\n", "### Requirement: Works\n")
	for path, source := range map[string]string{
		"a_test.go":          "//go:build never\n\npackage p\nfunc TestA(t *testing.T) {}\n",
		"testdata/b_test.go": "package p\nfunc TestB(t *testing.T) {}\n",
		"vendor/no_test.go":  "not Go", "node_modules/no.test.ts": "not TypeScript", ".worktrees/no_test.go": "not Go",
	} {
		p := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(source), 0644); err != nil {
			t.Fatal(err)
		}
	}
	want := "a_test.go:4: TestA: missing spec citation\nsample_test.go:2: TestZ: missing spec citation\ntestdata/b_test.go:2: TestB: missing spec citation\n"
	for range 2 {
		ds, err := Check(context.Background(), root, "")
		if err != nil {
			t.Fatal(err)
		}
		if got := Format(ds); got != want {
			t.Fatalf("discovery: got %q want %q", got, want)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Check(ctx, root, ""); err == nil {
		t.Fatal("cancellation ignored")
	}
}

// spec: openspec/specs/test-attribution/spec.md#Requirement: Safe requirement resolution
func FuzzRequirementResolution(f *testing.F) {
	for _, s := range []string{"Works", "Boundary # with spaces", "", "\n", "é"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, title string) {
		// The accepted title grammar is a single nonblank UTF-8 line. Invalid titles
		// are separately covered by explicit refusal fixtures.
		if !utf8.ValidString(title) || strings.ContainsAny(title, "\r\n") || strings.TrimSpace(title) == "" || strings.TrimSpace(title) != title {
			return
		}
		root := fixture(t, "package p\n", "### Requirement: "+title+"\n")
		ref := "openspec/specs/demo/spec.md#Requirement: " + title
		if err := resolve(root, ref); err != nil {
			t.Fatalf("exact title rejected: %v", err)
		}
		if err := resolve(root, ref+" missing"); err == nil {
			t.Fatal("unresolved title accepted")
		}
	})
}

// spec: openspec/specs/test-attribution/spec.md#Requirement: Attached test citations
// spec: openspec/specs/test-attribution/spec.md#Requirement: Deterministic discovery and verification
func TestVitestIntegration(t *testing.T) {
	root := fixture(t, "package p\n", "### Requirement: Works\n")
	if err := os.MkdirAll(filepath.Join(root, "ui"), 0755); err != nil {
		t.Fatal(err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	modules := filepath.Clean(filepath.Join(cwd, "../../ui/node_modules"))
	if err := os.Symlink(modules, filepath.Join(root, "ui/node_modules")); err != nil {
		t.Fatal(err)
	}
	source := "import {it as check} from 'vitest';\n" + citation + "check('valid',()=>{});\ncheck('missing',()=>{});\n"
	if err := os.WriteFile(filepath.Join(root, "ui/sample.test.ts"), []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	ds, err := Check(context.Background(), root, "")
	if err != nil {
		t.Fatal(err)
	}
	if got := Format(ds); got != "ui/sample.test.ts:4: missing: missing spec citation\n" {
		t.Fatalf("Vitest integration: %q", got)
	}
	if err := os.Remove(filepath.Join(root, "ui/node_modules")); err != nil {
		t.Fatal(err)
	}
	if _, err := Check(context.Background(), root, ""); err == nil {
		t.Fatal("missing TypeScript installation accepted")
	}
}

const aipSpecRef = "// spec: openspec/specs/test-attribution/spec.md#Requirement: AIP scenario directive\n"

func writeFiles(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for path, text := range files {
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(text), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func checkAIP(t *testing.T, directive string, files map[string]string) string {
	t.Helper()
	root := fixture(t, "package p\n"+directive+"func TestWorks(t *testing.T) {}", "### Requirement: Works\n")
	writeFiles(t, root, files)
	ds, err := Check(context.Background(), root, "")
	if err != nil {
		t.Fatal(err)
	}
	return Format(ds)
}

// spec: openspec/specs/test-attribution/spec.md#Requirement: AIP scenario directive
func TestAIPDirective(t *testing.T) {
	const canonical = "openspec/specs/dev/spec.md"
	const scenarios = "#### Scenario: DEV-001.1 first\n#### Scenario: DEV-001.2\n"
	const unknown = "sample_test.go:2: TestWorks: unknown scenario ID: "
	for _, tc := range []struct {
		name, directive, want string
		files                 map[string]string
	}{
		{"two known IDs", "//aip:verifies DEV-001.1 DEV-001.2\n", "", map[string]string{canonical: scenarios}},
		{"tab separated", "//aip:verifies\tDEV-001.1\n", "", map[string]string{canonical: scenarios}},
		{"in-flight change", "//aip:verifies NEW-1\n", "", map[string]string{"openspec/changes/c/specs/dev/spec.md": "#### Scenario: NEW-1 x\n"}},
		{"archived only", "//aip:verifies OLD-1\n", unknown + "OLD-1\n", map[string]string{"openspec/changes/archive/2026-01-01-c/specs/dev/spec.md": "#### Scenario: OLD-1 x\n"}},
		{"change directory named archive", "//aip:verifies OLD-2\n", unknown + "OLD-2\n", map[string]string{"openspec/changes/archive/specs/dev/spec.md": "#### Scenario: OLD-2 x\n"}},
		{"unknown", "//aip:verifies DEV-001.1 NOPE-9\n", unknown + "NOPE-9\n", map[string]string{canonical: scenarios}},
		{"empty", "//aip:verifies\n", "sample_test.go:2: TestWorks: aip:verifies names no scenario IDs\n", nil},
		{"spaced form", "// aip:verifies DEV-001.1\n", "sample_test.go:2: TestWorks: write //aip:verifies without a space\n", map[string]string{canonical: scenarios}},
		{"fenced header", "//aip:verifies FENCED-1\n", unknown + "FENCED-1\n", map[string]string{canonical: "```\n#### Scenario: FENCED-1 x\n```\n"}},
		{"prefix boundary", "//aip:verifies DEV-001.1\n", unknown + "DEV-001.1\n", map[string]string{canonical: "#### Scenario: DEV-001.10 x\n"}},
		{"duplicate across files", "//aip:verifies DEV-001.1\n", "", map[string]string{canonical: scenarios, "openspec/specs/other/spec.md": scenarios}},
		{"both forms", citation + "//aip:verifies DEV-001.1\n", "", map[string]string{canonical: scenarios}},
		{"glued suffix", "//aip:verifiesX\n", "sample_test.go:3: TestWorks: missing spec citation\n", nil},
		{"spec citation survives a bad aip ID", citation + "//aip:verifies NOPE-9\n", "sample_test.go:3: TestWorks: unknown scenario ID: NOPE-9\n", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := checkAIP(t, tc.directive, tc.files); got != tc.want {
				t.Fatalf("aip verdict: got %q want %q", got, tc.want)
			}
		})
	}
}

// spec: openspec/specs/test-attribution/spec.md#Requirement: AIP scenario directive
func TestAIPRootWithGlobMetacharacters(t *testing.T) {
	root := filepath.Join(t.TempDir(), "proj[1")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	writeFiles(t, root, map[string]string{
		"sample_test.go":             "package p\n//aip:verifies BR-1\nfunc TestWorks(t *testing.T) {}",
		"openspec/specs/dev/spec.md": "#### Scenario: BR-1 x\n",
	})
	ds, err := Check(context.Background(), root, "")
	if err != nil {
		t.Fatal(err)
	}
	if got := Format(ds); got != "" {
		t.Fatalf("root with [: got %q want no diagnostics", got)
	}
}

// spec: openspec/specs/test-attribution/spec.md#Requirement: AIP scenario directive
func TestAIPSpecFileRefusals(t *testing.T) {
	root := fixture(t, "package p\n//aip:verifies A-1\nfunc TestWorks(t *testing.T) {}", "### Requirement: Works\n")
	outside := t.TempDir()
	writeFiles(t, root, map[string]string{"openspec/specs/bad/spec.md": "#### Scenario: A-1 \xff\n"})
	writeFiles(t, outside, map[string]string{"spec.md": "#### Scenario: A-1 x\n"})
	if err := os.MkdirAll(filepath.Join(root, "openspec/specs/link"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "spec.md"), filepath.Join(root, "openspec/specs/link/spec.md")); err != nil {
		t.Fatal(err)
	}
	ds, err := Check(context.Background(), root, "")
	if err != nil {
		t.Fatal(err)
	}
	got := Format(ds)
	for _, want := range []string{
		"openspec/specs/bad/spec.md: spec is not UTF-8",
		"openspec/specs/link/spec.md: spec path escapes repository",
		"unknown scenario ID: A-1",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %q", want, got)
		}
	}
}
