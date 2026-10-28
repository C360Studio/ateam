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
		{name: "gitea workflow without label", files: names(".gitea/workflows/ci.yml"), reported: -1, wantCode: 1, wantOut: ".gitea/workflows/ci.yml"},
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
	// #143: the Gitea twins under .gitea/workflows/ hold to the same.
	for _, forge := range []string{".github", ".gitea"} {
		data, err := os.ReadFile(filepath.Join(repoRoot(t), forge, "workflows", "gate-files.yml"))
		if err != nil {
			t.Fatal(err)
		}
		wf := string(data)
		for _, want := range []string{"pull_request_target:", "labeled, unlabeled", "runs-on: ubuntu-24.04", "persist-credentials: false", "run: scripts/gate-files.sh \"$PR_NUMBER\"", "  contents: read\n  pull-requests: read\n",
			// #250: skipped on a draft except when a label is removed, so revoking
			// gate-change-approved replaces an earlier pass at the same head;
			// pull_request_target is never the event name pull_request, so the
			// guard tests the draft flag. The GitHub twin also renames the skip.
			"    if: ${{ !github.event.pull_request.draft || github.event.action == 'unlabeled' }}\n"} {
			if !strings.Contains(wf, want) {
				t.Fatalf("%s gate-files.yml lacks %q", forge, want)
			}
		}
		if forge == ".github" && !strings.Contains(wf, "    name: ${{ github.event.pull_request.draft && github.event.action != 'unlabeled' && 'Gate files (draft, skipped)' || 'Gate files' }}\n") {
			t.Fatalf("%s gate-files.yml: the job must take a non-required name when skipped on a draft (#250)", forge)
		}
		for _, banned := range []string{"pull_request.head", "head_ref", "ref:", "write", "\n  pull_request:\n"} {
			if strings.Contains(wf, banned) {
				t.Fatalf("%s gate-files.yml contains %q: it must not check out or run pull request code, or write", forge, banned)
			}
		}
		ci, err := os.ReadFile(filepath.Join(repoRoot(t), forge, "workflows", "ci.yml"))
		if err != nil {
			t.Fatal(err)
		}
		job := string(ci)
		start := strings.Index(job, "\n  merge-check:\n")
		end := strings.Index(job, "\n  required:\n")
		if start < 0 || end < start {
			t.Fatalf("%s ci.yml: cannot find the merge-check job before the required job", forge)
		}
		job = job[start:end]
		if !strings.Contains(job, "ref: ${{ github.event.pull_request.base.sha || github.sha }}") || strings.Contains(job, "pull_request.head") {
			t.Fatalf("%s ci.yml merge-check job must check out the base commit, not the pull request:\n%s", forge, job)
		}
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Merge gate and CI workflow enforcement
func TestGiteaWorkflowsTwinGitHub(t *testing.T) {
	// #143: each Gitea workflow is the GitHub one except the merge-check step's
	// credentials (Gitea's CI) or the forge and token (Gitea's gate files), so a
	// change to one that misses the other fails here. #250: the Gitea trigger
	// also lists edited, because leaving Gitea's WIP: title prefix is an edit
	// (GitHub does not take it, or every body edit would rerun Verify), and
	// Gitea keeps plain job names, because Gitea 1.22 does not evaluate name:.
	type swap struct{ github, gitea string }
	for _, tc := range []struct {
		file  string
		swaps []swap
	}{
		{"ci.yml", []swap{
			{"          GH_TOKEN: ${{ github.token }}\n          GH_REPO: ${{ github.repository }}\n",
				"          # Gitea: the job token only, no extra secret. It cannot read the branch\n" +
					"          # protection rule, so the up-to-date setting prints strict=unverified.\n" +
					"          GITEA_TOKEN: ${{ github.token }}\n"},
			{"    types: [opened, synchronize, reopened, ready_for_review]\n",
				"    types: [opened, synchronize, reopened, ready_for_review, edited]\n"},
			{"    name: " + draftName("Verify") + "\n", "    name: Verify\n"},
			{"    name: " + draftName("Merge check") + "\n", "    name: Merge check\n"},
			{"    name: " + draftName("Required") + "\n", "    name: Required\n"},
			{"  # On a draft every job is skipped and renamed, so a draft head carries no check\n  # run named Required (GitHub counts a skipped one as passing, #250). A skipped\n  # job's name is not evaluated: the draft check run shows the raw expression.\n",
				"  # On a draft every job is skipped. Gitea counts a skipped job as success, but\n  # it posts a pending status for every job as soon as a run is created, so\n  # leaving WIP: replaces it at once; Gitea 1.22 does not evaluate job names, so\n  # the GitHub draft rename would break the required context here (#250).\n"},
		}},
		{"gate-files.yml", []swap{
			{"          GH_TOKEN: ${{ github.token }}\n          GH_REPO: ${{ github.repository }}\n",
				"          ATEAM_FORGE: gitea\n          GITEA_TOKEN: ${{ github.token }}\n"},
			{"    types: [opened, synchronize, reopened, ready_for_review, labeled, unlabeled]\n",
				"    types: [opened, synchronize, reopened, ready_for_review, labeled, unlabeled, edited]\n"},
			{"that judges it (#140). It must never check out or run pull request code: the\n",
				"that judges it (#140; the Gitea twin of .github/workflows/gate-files.yml, #143).\n# It must never check out or run pull request code: the\n"},
			{"    # Skipped and renamed on a draft, so a draft head carries no Gate files check\n    # run for GitHub to count as passing (#250). Removing a label still runs it\n    # on a draft, so revoking gate-change-approved replaces an earlier pass at\n    # the same head before the PR can be marked ready again.\n    name: ${{ github.event.pull_request.draft && github.event.action != 'unlabeled' && 'Gate files (draft, skipped)' || 'Gate files' }}\n",
				"    # Skipped on a draft (WIP:); removing a label still runs it, so revoking\n    # gate-change-approved replaces an earlier pass at the same head (#250).\n    # Gitea 1.22 does not evaluate job names, so it keeps the plain name.\n    name: Gate files\n"},
		}},
	} {
		gh, err := os.ReadFile(filepath.Join(repoRoot(t), ".github", "workflows", tc.file))
		if err != nil {
			t.Fatal(err)
		}
		gt, err := os.ReadFile(filepath.Join(repoRoot(t), ".gitea", "workflows", tc.file))
		if err != nil {
			t.Fatal(err)
		}
		for _, line := range strings.Split(string(gh), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "types:") && strings.Contains(line, "edited") {
				t.Errorf(".github/workflows/%s triggers on edited; only the Gitea twin takes it (#250)", tc.file)
			}
		}
		want := string(gh)
		for _, sw := range tc.swaps {
			if !strings.Contains(want, sw.github) {
				t.Errorf(".github/workflows/%s lacks %q", tc.file, sw.github)
			}
			want = strings.Replace(want, sw.github, sw.gitea, 1)
		}
		if string(gt) != want {
			t.Errorf(".gitea/workflows/%s is not .github/workflows/%s with only the Gitea lines changed", tc.file, tc.file)
		}
	}
	gt, err := os.ReadFile(filepath.Join(repoRoot(t), ".gitea", "workflows", "ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if v := ciWorkflowViolations(t, gt, false); len(v) > 0 {
		t.Fatalf(".gitea/workflows/ci.yml violations:\n- %s", strings.Join(v, "\n- "))
	}
}

// fakeGateCurl answers gate-files.sh's Gitea reads from files in $FAKE_GH_DIR:
// pull.txt, files-<page>.txt (an empty list past the last page), labels in pull.txt.
// The token must arrive on stdin as a header, never in argv.
const fakeGateCurl = `#!/bin/sh
d="$FAKE_GH_DIR"
for a in "$@"; do url=$a; done
case "$*" in *gitea-test-token*) echo "fake curl: the token is in argv" >&2; exit 3 ;; esac
case "$*" in *"-H @-"*) headers=$(cat) ;; *) headers= ;; esac
case "$headers" in *"Authorization: token gitea-test-token"*) ;; *) echo "curl: (22) 401" >&2; exit 22 ;; esac
case "$url" in
  "https://gitea.example.test/api/v1/repos/acme/widgets/pulls/7") key=pull ;;
  "https://gitea.example.test/api/v1/repos/acme/widgets/pulls/7/files?page="*"&limit=50")
    p=${url#*page=}; p=${p%%&*}; key=files-$p
    [ -e "$d/$key.txt" ] || { echo '[]'; exit 0; } ;;
  *) echo "fake curl: unexpected call: $*" >&2; exit 3 ;;
esac
if [ -e "$d/$key.fail" ]; then echo "curl: (22) The requested URL returned error: 502" >&2; exit 22; fi
cat "$d/$key.txt"
`

// spec: openspec/specs/harness/spec.md#Requirement: Merge gate and CI workflow enforcement
func TestGateFilesCheckGitea(t *testing.T) {
	// #143: on Gitea the same rule reads the API with curl, every page of the
	// file list, and the labels from the pull request itself.
	many := make([]gateFile, 0, 121)
	for i := range 120 {
		many = append(many, gateFile{Filename: fmt.Sprintf("docs/n%03d.md", i)})
	}
	many = append(many, gateFile{Filename: ".gitea/workflows/ci.yml"})
	for _, tc := range []struct {
		name     string
		files    []gateFile
		reported int // changed_files; -1 means len(files)
		labels   []string
		failRead string
		wantCode int
		wantOut  string
	}{
		{name: "no gate file", files: names("README.md", "internal/state/state.go"), reported: -1, wantOut: "touches no gate file"},
		{name: "gitea workflow without label", files: names(".gitea/workflows/gate-files.yml"), reported: -1, labels: []string{"bug"}, wantCode: 1, wantOut: ".gitea/workflows/gate-files.yml"},
		{name: "gate file on a later page", files: many, reported: -1, wantCode: 1, wantOut: ".gitea/workflows/ci.yml"},
		{name: "listing cut short", files: many[:100], reported: 121, wantCode: 2, wantOut: "incomplete"},
		{name: "gate file with owner label", files: names("internal/review/gate.go"), reported: -1, labels: []string{"gate-change-approved"}, wantOut: "labelled gate-change-approved"},
		{name: "pull read fails", files: names("README.md"), reported: -1, failRead: "pull", wantCode: 2, wantOut: "could not read pull request"},
		{name: "files read fails", files: names("README.md"), reported: -1, failRead: "files-1", wantCode: 2, wantOut: "could not read the changed files"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := t.TempDir()
			reported := tc.reported
			if reported < 0 {
				reported = len(tc.files)
			}
			labels := make([]map[string]string, len(tc.labels))
			for i, l := range tc.labels {
				labels[i] = map[string]string{"name": l}
			}
			pull, err := json.Marshal(map[string]any{"number": 7, "changed_files": reported, "labels": labels})
			if err != nil {
				t.Fatal(err)
			}
			write := func(name, body string) {
				if err := os.WriteFile(filepath.Join(d, name), []byte(body), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			write("pull.txt", string(pull))
			for page := 0; page*50 < len(tc.files); page++ {
				write(fmt.Sprintf("files-%d.txt", page+1), gatePages(t, tc.files[page*50:min(page*50+50, len(tc.files))]))
			}
			if tc.failRead != "" {
				write(tc.failRead+".fail", "")
			}
			root := copyScript(t, "gate-files.sh")
			cmd := exec.Command(filepath.Join(root, "scripts", "gate-files.sh"), "7")
			cmd.Env = fakeBin(t, map[string]string{"curl": fakeGateCurl, "gh": "#!/bin/sh\necho 'gh must not run on Gitea' >&2; exit 3\n"},
				"FAKE_GH_DIR="+d, "ATEAM_FORGE=gitea", "GITHUB_SERVER_URL=https://gitea.example.test/", "GITHUB_REPOSITORY=acme/widgets", "GITEA_TOKEN=gitea-test-token")
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
