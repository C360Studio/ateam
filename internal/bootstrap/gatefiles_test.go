package bootstrap

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// fakeGateGH answers the three reads gate-files.sh makes, from files in $FAKE_GH_DIR.
const fakeGateGH = `#!/bin/sh
d="$FAKE_GH_DIR"
case "$*" in
  "api repos/{owner}/{repo}/pulls/7") key=pull ;;
  "api --paginate repos/{owner}/{repo}/pulls/7/files?per_page=100") key=files ;;
  "pr view 7 --json labels --jq "*) key=labels ;;
  *) echo "fake gh: unexpected call: $*" >&2; exit 3 ;;
esac
if [ -e "$d/$key.fail" ]; then echo "HTTP 502 (fake)" >&2; exit 1; fi
cat "$d/$key.txt"
`

type gateFile struct {
	Filename         string `json:"filename"`
	PreviousFilename string `json:"previous_filename,omitempty"`
}

// gatePages renders files as GitHub's paginated listing: one JSON array per page of 100.
func gatePages(t *testing.T, files []gateFile) string {
	t.Helper()
	var b strings.Builder
	for start := 0; start < len(files) || start == 0; start += 100 {
		end := min(start+100, len(files))
		page, err := json.Marshal(files[start:end])
		if err != nil {
			t.Fatal(err)
		}
		b.Write(page)
		if end == len(files) {
			break
		}
	}
	return b.String()
}

func names(paths ...string) []gateFile {
	out := make([]gateFile, len(paths))
	for i, p := range paths {
		out[i] = gateFile{Filename: p}
	}
	return out
}

// spec: openspec/specs/harness/spec.md#Requirement: Merge gate and CI workflow enforcement
func TestGateFilesCheck(t *testing.T) {
	// #140: a pull request touching a gate file fails until it carries the owner label,
	// and an incomplete or malformed file list never reads as "no gate file".
	many := make([]gateFile, 0, 301)
	for i := range 300 {
		many = append(many, gateFile{Filename: fmt.Sprintf("docs/n%03d.md", i)})
	}
	many = append(many, gateFile{Filename: "scripts/merge-check.sh"})
	for _, tc := range []struct {
		name     string
		files    []gateFile
		reported int // changed_files; -1 means len(files)
		raw      string
		labels   string
		failRead string
		wantCode int
		wantOut  string
	}{
		{name: "no gate file", files: names("README.md", "internal/state/state.go"), reported: -1, wantOut: "touches no gate file"},
		{name: "policy file without label", files: names("internal/state/state.go", ".agents/review-policy.json"), reported: -1, labels: "bug\n", wantCode: 1, wantOut: "waits for the owner"},
		{name: "workflow without label", files: names(".github/workflows/ci.yml"), reported: -1, wantCode: 1, wantOut: ".github/workflows/ci.yml"},
		{name: "review package without label", files: names("internal/review/compare.go"), reported: -1, wantCode: 1, wantOut: "internal/review/compare.go"},
		{name: "this script without label", files: names("scripts/gate-files.sh"), reported: -1, wantCode: 1, wantOut: "scripts/gate-files.sh"},
		{name: "renamed away from a gate path", files: []gateFile{{Filename: "scripts/old-check.sh", PreviousFilename: "scripts/merge-check.sh"}}, reported: -1, wantCode: 1, wantOut: "scripts/merge-check.sh"},
		{name: "gate file on a later page", files: many, reported: -1, wantCode: 1, wantOut: "scripts/merge-check.sh"},
		{name: "listing cut short", files: many[:300], reported: 301, wantCode: 2, wantOut: "incomplete"},
		{name: "pages not lists of files", raw: `{"message":"oops"}`, reported: 1, wantCode: 2, wantOut: "not a list of files"},
		{name: "gate file with owner label", files: names("scripts/merge-check.sh"), reported: -1, labels: "bug\ngate-change-approved\n", wantOut: "labelled gate-change-approved"},
		{name: "label name must match exactly", files: names("scripts/merge-check.sh"), reported: -1, labels: "not-gate-change-approved\n", wantCode: 1, wantOut: "waits for the owner"},
		{name: "pull read fails", files: names("README.md"), reported: -1, failRead: "pull", wantCode: 2, wantOut: "could not read pull request"},
		{name: "files read fails", files: names("README.md"), reported: -1, failRead: "files", wantCode: 2, wantOut: "could not read the changed files"},
		{name: "labels read fails", files: names("internal/review/x.go"), reported: -1, failRead: "labels", wantCode: 2, wantOut: "could not read the labels"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := t.TempDir()
			reported := tc.reported
			if reported < 0 {
				reported = len(tc.files)
			}
			pages := tc.raw
			if pages == "" {
				pages = gatePages(t, tc.files)
			}
			for name, body := range map[string]string{
				"pull.txt":   fmt.Sprintf(`{"number":7,"changed_files":%d}`, reported),
				"files.txt":  pages,
				"labels.txt": tc.labels,
			} {
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
	// #140: the gate-files workflow takes its definition from the base branch and never
	// runs PR code, and the merge-check job builds and runs from the base commit.
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
	ci, err := os.ReadFile(filepath.Join(repoRoot(t), ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	job := string(ci)
	start := strings.Index(job, "\n  merge-check:\n")
	end := strings.Index(job, "\n  required:\n")
	if start < 0 || end < start {
		t.Fatal("ci.yml: cannot find the merge-check job before the required job")
	}
	job = job[start:end]
	if !strings.Contains(job, "ref: ${{ github.event.pull_request.base.sha || github.sha }}") || strings.Contains(job, "pull_request.head") {
		t.Fatalf("ci.yml merge-check job must check out the base commit, not the pull request:\n%s", job)
	}
}
