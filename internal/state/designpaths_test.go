package state

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const designPolicyFixture = `{"version":1,
"model_tiers":{"frontier":["claude:opus"]},
"effort_bands":{"claude":{"high":["high"]},"codex":{"high":["high"]},"gemini":{"high":["high"]}},
"cross_provider":false,
"design_paths":["src/arming.rs","src/actuator/**"]}`

// designRepo makes a repository whose main branch holds the policy and src/, then checks out a feature branch.
func designRepo(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".agents/review-policy.json", designPolicyFixture)
	write("src/arming.rs", "fn arm() {}\n")
	write("src/actuator/out.rs", "fn out() {}\n")
	write("src/actuator.rs", "fn other() {}\n")
	write("docs/readme.md", "hi\n")
	for _, args := range [][]string{
		{"init", "-q", "-b", "main"}, {"config", "user.email", "t@example.com"}, {"config", "user.name", "t"},
		{"add", "."}, {"commit", "-qm", "base"}, {"checkout", "-qb", "feature"},
	} {
		designGit(t, root, args...)
	}
	return root
}

func designGit(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func designWrite(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Design-track trigger paths
func TestCheckDesignPaths(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name    string
		edit    func(t *testing.T, root string)
		failing []string // changed paths the check must name, in order
	}{
		{"exact glob without design", func(t *testing.T, root string) {
			designWrite(t, root, "src/arming.rs", "fn arm() { disarm() }\n")
		}, []string{"src/arming.rs"}},
		{"subtree glob, committed and untracked", func(t *testing.T, root string) {
			designWrite(t, root, "src/actuator/out.rs", "fn out() { 1 }\n")
			designGit(t, root, "commit", "-qam", "edit")
			designWrite(t, root, "src/actuator/new/deep.rs", "fn deep() {}\n")
		}, []string{"src/actuator/new/deep.rs", "src/actuator/out.rs"}},
		{"deleted trigger file", func(t *testing.T, root string) {
			designGit(t, root, "rm", "-q", "src/arming.rs")
		}, []string{"src/arming.rs"}},
		{"in-flight design satisfies", func(t *testing.T, root string) {
			designWrite(t, root, "src/arming.rs", "fn arm() { disarm() }\n")
			designWrite(t, root, "openspec/changes/arm-1/design.md", "# Design\n")
		}, nil},
		{"archived design satisfies", func(t *testing.T, root string) {
			designWrite(t, root, "src/actuator/out.rs", "fn out() { 2 }\n")
			designWrite(t, root, "openspec/changes/archive/2026-11-24-arm-1/design.md", "# Design\n")
			designGit(t, root, "add", ".")
			designGit(t, root, "commit", "-qm", "with design")
		}, nil},
		{"sibling of a subtree glob and other paths pass", func(t *testing.T, root string) {
			designWrite(t, root, "src/actuator.rs", "fn other() { 3 }\n")
			designWrite(t, root, "docs/readme.md", "changed\n")
		}, nil},
		{"proposal alone is not a design", func(t *testing.T, root string) {
			designWrite(t, root, "src/arming.rs", "fn arm() { 4 }\n")
			designWrite(t, root, "openspec/changes/arm-1/proposal.md", "# P\n")
		}, []string{"src/arming.rs"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := designRepo(t)
			tc.edit(t, root)
			report, failures, err := CheckDesignPaths(ctx, root, "main")
			if err != nil {
				t.Fatalf("CheckDesignPaths: %v", err)
			}
			if len(failures) != len(tc.failing) {
				t.Fatalf("failures = %q, want one each for %q", failures, tc.failing)
			}
			for i, p := range tc.failing {
				if !strings.HasPrefix(failures[i], "design paths: "+p+" matches design_paths ") || !strings.Contains(failures[i], "design.md") {
					t.Errorf("failure %d = %q, want it to name %s, its glob and design.md", i, failures[i], p)
				}
			}
			if tc.failing == nil && !strings.HasPrefix(report, "design paths: ok") {
				t.Errorf("report = %q, want an ok line", report)
			}
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Design-track trigger paths
func TestCheckDesignPathsBase(t *testing.T) {
	ctx := context.Background()
	root := designRepo(t)
	designWrite(t, root, "src/arming.rs", "fn arm() { 5 }\n")

	report, failures, err := CheckDesignPaths(ctx, root, "")
	if err != nil || failures != nil || !strings.HasPrefix(report, "design paths: not checked") {
		t.Fatalf("no origin: report=%q failures=%q err=%v, want a visible not-checked line and no failure", report, failures, err)
	}
	if _, _, err := CheckDesignPaths(ctx, root, "origin/nope"); err == nil {
		t.Fatal("an explicit base that does not resolve must fail, not skip")
	}
	designGit(t, root, "update-ref", "refs/remotes/origin/main", "main")
	if _, failures, err := CheckDesignPaths(ctx, root, ""); err != nil || len(failures) != 1 {
		t.Fatalf("origin/main present: failures=%q err=%v, want the default base found and one failure", failures, err)
	}
	if err := os.WriteFile(filepath.Join(root, ".agents", "review-policy.json"), []byte(strings.Replace(designPolicyFixture, `,
"design_paths":["src/arming.rs","src/actuator/**"]`, "", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	if report, failures, err := CheckDesignPaths(ctx, root, "main"); report != "" || failures != nil || err != nil {
		t.Fatalf("no design_paths: report=%q failures=%q err=%v, want nothing", report, failures, err)
	}
}
