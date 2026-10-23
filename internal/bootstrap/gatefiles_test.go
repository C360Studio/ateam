package bootstrap

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// fakeGateGH answers the two reads gate-files.sh makes, from files in $FAKE_GH_DIR.
const fakeGateGH = `#!/bin/sh
d="$FAKE_GH_DIR"
case "$*" in
  "api --paginate repos/{owner}/{repo}/pulls/7/files?per_page=100 --jq "*) key=files ;;
  "pr view 7 --json labels --jq "*) key=labels ;;
  *) echo "fake gh: unexpected call: $*" >&2; exit 3 ;;
esac
if [ -e "$d/$key.fail" ]; then echo "HTTP 502 (fake)" >&2; exit 1; fi
cat "$d/$key.txt"
`

// spec: openspec/specs/harness/spec.md#Requirement: Merge gate and CI workflow enforcement
func TestGateFilesCheck(t *testing.T) {
	// #140: a pull request touching a gate file fails until it carries the owner label.
	for _, tc := range []struct {
		name, files, labels string
		failRead            string
		wantCode            int
		wantOut             string
	}{
		{"no gate file", "README.md\ninternal/state/state.go\n", "", "", 0, "touches no gate file"},
		{"policy file without label", "internal/state/state.go\n.agents/review-policy.json\n", "bug\n", "", 1, "waits for the owner"},
		{"workflow without label", ".github/workflows/ci.yml\n", "", "", 1, ".github/workflows/ci.yml"},
		{"review package without label", "internal/review/compare.go\n", "", "", 1, "internal/review/compare.go"},
		{"this script without label", "scripts/gate-files.sh\n", "", "", 1, "scripts/gate-files.sh"},
		{"renamed away from a gate path", "scripts/old-check.sh\nscripts/merge-check.sh\n", "", "", 1, "scripts/merge-check.sh"},
		{"gate file with owner label", "scripts/merge-check.sh\n", "bug\ngate-change-approved\n", "", 0, "labelled gate-change-approved"},
		{"label name must match exactly", "scripts/merge-check.sh\n", "not-gate-change-approved\n", "", 1, "waits for the owner"},
		{"files read fails", "", "", "files", 2, "could not read the changed files"},
		{"labels read fails", "internal/review/x.go\n", "", "labels", 2, "could not read the labels"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := t.TempDir()
			for name, body := range map[string]string{"files.txt": tc.files, "labels.txt": tc.labels} {
				if err := os.WriteFile(filepath.Join(d, name), []byte(body), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if tc.failRead != "" {
				if err := os.WriteFile(filepath.Join(d, tc.failRead+".fail"), nil, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			root := copyScript(t, "gate-files.sh")
			cmd := exec.Command(filepath.Join(root, "scripts", "gate-files.sh"), "7")
			cmd.Env = fakeBin(t, map[string]string{"gh": fakeGateGH}, "FAKE_GH_DIR="+d)
			out, err := cmd.CombinedOutput()
			code := 0
			if ee, ok := err.(*exec.ExitError); ok {
				code = ee.ExitCode()
			} else if err != nil {
				t.Fatal(err)
			}
			if code != tc.wantCode || !strings.Contains(string(out), tc.wantOut) {
				t.Fatalf("gate-files.sh exit %d, want %d with %q:\n%s", code, tc.wantCode, tc.wantOut, out)
			}
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Merge gate and CI workflow enforcement
func TestGateFilesWorkflowRunsFromBase(t *testing.T) {
	// #140: the workflow must take its definition from the base branch and never run PR code.
	data, err := os.ReadFile(filepath.Join(repoRoot(t), ".github", "workflows", "gate-files.yml"))
	if err != nil {
		t.Fatal(err)
	}
	wf := string(data)
	for _, want := range []string{"pull_request_target:", "labeled, unlabeled", "runs-on: ubuntu-24.04", "persist-credentials: false", "run: scripts/gate-files.sh \"$PR_NUMBER\"", "  contents: read\n  pull-requests: read\n"} {
		if !strings.Contains(wf, want) {
			t.Fatalf("gate-files.yml lacks %q", want)
		}
	}
	for _, banned := range []string{"pull_request.head", "head_ref", "ref:", "write", "\n  pull_request:\n"} {
		if strings.Contains(wf, banned) {
			t.Fatalf("gate-files.yml contains %q: it must not check out or run pull request code, or write", banned)
		}
	}
}
