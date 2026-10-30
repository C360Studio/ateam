package testspec

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const legacyGo = `package p

import "testing"

func TestLegacyA(t *testing.T) {
	_ = 1
}

func TestLegacyB(t *testing.T) {}

// spec: openspec/specs/demo/spec.md#Requirement: Old
func TestLegacyC(t *testing.T) {}
`

const legacyVitest = `import { it } from 'vitest';
it('legacy', () => {
  const x = 1;
});
it('other', () => {});
`

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false"}, args...)...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// brownfield returns a repository whose main branch holds uncited legacy tests,
// checked out on a feature branch with no changes yet.
func brownfield(t *testing.T) string {
	t.Helper()
	root := fixture(t, "", "### Requirement: Works\n\n### Requirement: Old\n")
	if err := os.Remove(filepath.Join(root, "sample_test.go")); err != nil {
		t.Fatal(err)
	}
	writeFiles(t, root, map[string]string{"legacy_test.go": legacyGo, "ui/legacy.test.ts": legacyVitest, ".gitignore": "node_modules\n"})
	git(t, root, "init", "-q", "-b", "main")
	git(t, root, "add", ".")
	git(t, root, "commit", "-q", "-m", "legacy")
	git(t, root, "checkout", "-q", "-b", "feature")
	return root
}

func linkTypeScript(t *testing.T, root string) {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Clean(filepath.Join(cwd, "../../ui/node_modules")), filepath.Join(root, "ui/node_modules")); err != nil {
		t.Fatal(err)
	}
}

// spec: openspec/changes/test-spec-base-237/specs/test-attribution/spec.md#Requirement: Diff-scoped attribution
func TestBaseScope(t *testing.T) {
	cited := "package p\n\nimport \"testing\"\n\n" + citation + "func TestNew(t *testing.T) {}\n"
	uncited := "package p\n\nimport \"testing\"\n\nfunc TestNew(t *testing.T) {}\n"
	for _, tc := range []struct {
		name   string
		change func(t *testing.T, root string)
		want   string // exact Format output; "" means the gate passes
	}{
		{"cited new test passes over uncited legacy", func(t *testing.T, root string) {
			writeFiles(t, root, map[string]string{"new_test.go": cited})
			git(t, root, "add", ".")
			git(t, root, "commit", "-q", "-m", "new")
		}, ""},
		{"committed uncited new test fails", func(t *testing.T, root string) {
			writeFiles(t, root, map[string]string{"new_test.go": uncited})
			git(t, root, "add", ".")
			git(t, root, "commit", "-q", "-m", "new")
		}, "new_test.go:5: TestNew: missing spec citation\n"},
		{"untracked uncited test file fails", func(t *testing.T, root string) {
			writeFiles(t, root, map[string]string{"new_test.go": uncited})
		}, "new_test.go:5: TestNew: missing spec citation\n"},
		{"uncommitted edit inside a legacy body fails only that test", func(t *testing.T, root string) {
			writeFiles(t, root, map[string]string{"legacy_test.go": strings.Replace(legacyGo, "_ = 1", "_ = 2", 1)})
		}, "legacy_test.go:5: TestLegacyA: missing spec citation\n"},
		{"removing a legacy citation fails", func(t *testing.T, root string) {
			writeFiles(t, root, map[string]string{"legacy_test.go": strings.Replace(legacyGo, "// spec: openspec/specs/demo/spec.md#Requirement: Old\n", "", 1)})
		}, "legacy_test.go:11: TestLegacyC: missing spec citation\n"},
		{"edited Vitest legacy body fails only that test", func(t *testing.T, root string) {
			writeFiles(t, root, map[string]string{"ui/legacy.test.ts": strings.Replace(legacyVitest, "const x = 1", "const x = 2", 1)})
		}, "ui/legacy.test.ts:2: legacy: missing spec citation\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := brownfield(t)
			linkTypeScript(t, root)
			tc.change(t, root)
			ds, err := Check(context.Background(), root, "main")
			if err != nil {
				t.Fatal(err)
			}
			if got := Format(ds); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}

	t.Run("dangling citation on an untouched legacy test fails", func(t *testing.T) {
		root := brownfield(t)
		linkTypeScript(t, root)
		writeFiles(t, root, map[string]string{"openspec/specs/demo/spec.md": "### Requirement: Works\n"})
		ds, err := Check(context.Background(), root, "main")
		if err != nil {
			t.Fatal(err)
		}
		if len(ds) != 1 || ds[0].File != "legacy_test.go" || ds[0].Line != 11 || !strings.HasPrefix(ds[0].Message, "TestLegacyC: ") || strings.Contains(ds[0].Message, "missing spec citation") {
			t.Fatalf("want one unresolved citation on TestLegacyC, got %q", Format(ds))
		}
	})

	t.Run("full scope still reports every legacy test", func(t *testing.T) {
		root := brownfield(t)
		linkTypeScript(t, root)
		ds, err := Check(context.Background(), root, "")
		if err != nil {
			t.Fatal(err)
		}
		want := "legacy_test.go:5: TestLegacyA: missing spec citation\nlegacy_test.go:9: TestLegacyB: missing spec citation\n" +
			"ui/legacy.test.ts:2: legacy: missing spec citation\nui/legacy.test.ts:5: other: missing spec citation\n"
		if got := Format(ds); got != want {
			t.Fatalf("got %q want %q", got, want)
		}
	})
}

// spec: openspec/changes/test-spec-base-237/specs/test-attribution/spec.md#Requirement: Diff-scoped attribution
func TestBaseRefusals(t *testing.T) {
	root := brownfield(t)
	shallow := filepath.Join(t.TempDir(), "shallow")
	git(t, root, "commit", "-q", "--allow-empty", "-m", "feature")
	git(t, root, "checkout", "-q", "main")
	git(t, root, "commit", "-q", "--allow-empty", "-m", "main moves")
	git(t, root, "checkout", "-q", "feature")
	git(t, filepath.Dir(shallow), "clone", "-q", "--depth", "1", "--no-single-branch", "--branch", "feature", "file://"+root, shallow)
	for name, tc := range map[string]struct {
		root, base, want string
	}{
		"unknown ref":         {root, "no-such-branch", "does not resolve"},
		"option-shaped ref":   {root, "--output=x", "cannot start with '-'"},
		"shallow clone":       {shallow, "origin/main", "no merge base"},
		"not a git work tree": {fixture(t, "package p\n", "### Requirement: Works\n"), "main", "does not resolve"},
	} {
		t.Run(name, func(t *testing.T) {
			ds, err := Check(context.Background(), tc.root, tc.base)
			if err == nil || !strings.Contains(err.Error(), tc.want) || ds != nil {
				t.Fatalf("want error containing %q and no diagnostics, got %v, %q", tc.want, err, Format(ds))
			}
		})
	}
}

// spec: openspec/changes/test-spec-base-237/specs/test-attribution/spec.md#Requirement: Diff-scoped attribution
func TestParseHunks(t *testing.T) {
	patch := "diff --git a/x_test.go b/x_test.go\n--- a/x_test.go\n+++ b/x_test.go\n" +
		"@@ -3 +3 @@ func A\n-a\n+b\n" + // one line, no count
		"@@ -7,2 +7,0 @@\n-c\n-d\n" + // deletion-only: marks the line after the deletion point
		"@@ -20,0 +18,2 @@\n+e\n+f\n" + // pure addition
		"diff --git a/gone_test.go b/gone_test.go\n--- a/gone_test.go\n+++ /dev/null\n@@ -1,3 +0,0 @@\n-x\n-y\n-z\n"
	got, err := parseHunks(patch)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]lineSet{"x_test.go": {lines: map[int]bool{3: true, 8: true, 18: true, 19: true}}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for _, bad := range []string{
		"+++ b/x_test.go\n@@ -1 +x,2 @@\n",
		"+++ b/x_test.go\n@@ -1\n",
		"+++ \"b/tab\\tname_test.go\"\n",
	} {
		if _, err := parseHunks(bad); err == nil {
			t.Fatalf("accepted %q", bad)
		}
	}
}
