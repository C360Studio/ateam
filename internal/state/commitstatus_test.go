package state

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/c360studio/ateam/internal/review"
)

const (
	headFull = "abc1234def567890abc1234def567890abc12345"
	prURL    = "https://github.com/o/r/pull/7"
)

// statusesFor keeps only ctx's status from a chooser result.
func statusesFor(all []CommitStatus, ctx string) []CommitStatus {
	var out []CommitStatus
	for _, s := range all {
		if s.Context == ctx {
			out = append(out, s)
		}
	}
	return out
}

// spec: openspec/specs/harness/spec.md#Requirement: Informational commit statuses on the PR head
func TestChooseStatusesD3Table(t *testing.T) {
	// Each row is one row of the D3 table in design.md; expected states and descriptions are copied from it.
	at := time.Date(2026, 11, 7, 10, 42, 59, 0, time.FixedZone("PST", -8*3600)) // 18:42 UTC
	approved := touchedChange{ID: "feat-1", Proposal: "openspec/changes/feat-1/proposal.md", ApprovedBy: "Ann Owner (2026-11-07)"}
	rec := func(sha, verdict string) *review.RecordComment {
		return &review.RecordComment{SHA: sha, Verdict: verdict, Agent: "codex", Model: "gpt-6-astra", Effort: "xhigh"}
	}
	for _, tc := range []struct {
		name string
		ctx  string
		in   statusInputs
		want []CommitStatus
	}{
		{name: "proposal: no touched change", ctx: StatusContextProposal, in: statusInputs{}},
		{name: "proposal: empty Approved-by", ctx: StatusContextProposal,
			in: statusInputs{Changes: []touchedChange{{ID: "feat-1", Proposal: "openspec/changes/feat-1/proposal.md"}}},
			want: []CommitStatus{{Context: "ateam/proposal", State: "pending", Description: "awaiting owner approval (OPEN-Q): feat-1",
				TargetURL: "https://github.com/o/r/blob/" + headFull + "/openspec/changes/feat-1/proposal.md"}}},
		{name: "proposal: approved but OPEN-Q hold open", ctx: StatusContextProposal,
			in: statusInputs{Changes: []touchedChange{{ID: "feat-1", Proposal: "openspec/changes/feat-1/proposal.md", ApprovedBy: "Ann", OpenQ: true}}},
			want: []CommitStatus{{Context: "ateam/proposal", State: "pending", Description: "awaiting owner approval (OPEN-Q): feat-1",
				TargetURL: "https://github.com/o/r/blob/" + headFull + "/openspec/changes/feat-1/proposal.md"}}},
		{name: "proposal: all approved", ctx: StatusContextProposal, in: statusInputs{Changes: []touchedChange{approved}},
			want: []CommitStatus{{Context: "ateam/proposal", State: "success", Description: "approved by Ann Owner (2026-11-07): feat-1",
				TargetURL: "https://github.com/o/r/blob/" + headFull + "/openspec/changes/feat-1/proposal.md"}}},
		{name: "proposal: Gitea link", ctx: StatusContextProposal,
			in: statusInputs{Provider: ForgeGitea, PRURL: "https://forge.example/o/r/pulls/7", Changes: []touchedChange{approved}},
			want: []CommitStatus{{Context: "ateam/proposal", State: "success", Description: "approved by Ann Owner (2026-11-07): feat-1",
				TargetURL: "https://forge.example/o/r/src/commit/" + headFull + "/openspec/changes/feat-1/proposal.md"}}},
		{name: "handoff: no slot", ctx: StatusContextHandoff, in: statusInputs{}},
		{name: "handoff: checkpoint is not the head", ctx: StatusContextHandoff,
			in:   statusInputs{Handoff: &Handoff{CheckpointSHA: "0123456", UpdatedAt: at}},
			want: []CommitStatus{{Context: "ateam/handoff", State: "pending", Description: "handoff stale: at 0123456, head abc1234", TargetURL: prURL}}},
		{name: "handoff: stale wins over dirty", ctx: StatusContextHandoff,
			in:   statusInputs{Handoff: &Handoff{CheckpointSHA: "0123456", Dirty: true, UpdatedAt: at}},
			want: []CommitStatus{{Context: "ateam/handoff", State: "pending", Description: "handoff stale: at 0123456, head abc1234", TargetURL: prURL}}},
		{name: "handoff: dirty", ctx: StatusContextHandoff,
			in:   statusInputs{Handoff: &Handoff{CheckpointSHA: "abc1234d", Dirty: true, UpdatedAt: at}, HandoffURL: prURL + "#issuecomment-9"},
			want: []CommitStatus{{Context: "ateam/handoff", State: "pending", Description: "handoff at abc1234 has uncommitted changes", TargetURL: prURL + "#issuecomment-9"}}},
		{name: "handoff: fresh", ctx: StatusContextHandoff,
			in:   statusInputs{Handoff: &Handoff{CheckpointSHA: "abc1234", UpdatedAt: at}, HandoffURL: prURL + "#issuecomment-9"},
			want: []CommitStatus{{Context: "ateam/handoff", State: "success", Description: "handoff at abc1234, 2026-11-07T18:42Z", TargetURL: prURL + "#issuecomment-9"}}},
		{name: "record: none", ctx: StatusContextRecord, in: statusInputs{}},
		{name: "record: behind head", ctx: StatusContextRecord, in: statusInputs{Record: rec("0123456", "APPROVE")},
			want: []CommitStatus{{Context: "ateam/review-record", State: "pending", Description: "record at 0123456 is behind head abc1234", TargetURL: prURL}}},
		{name: "record: behind head wins over REQUEST CHANGES", ctx: StatusContextRecord, in: statusInputs{Record: rec("0123456", "REQUEST CHANGES")},
			want: []CommitStatus{{Context: "ateam/review-record", State: "pending", Description: "record at 0123456 is behind head abc1234", TargetURL: prURL}}},
		{name: "record: APPROVE", ctx: StatusContextRecord, in: statusInputs{Record: rec("abc1234", "APPROVE"), RecordURL: prURL + "#issuecomment-5"},
			want: []CommitStatus{{Context: "ateam/review-record", State: "success", Description: "APPROVE self-reported by codex (gpt-6-astra, xhigh) at abc1234", TargetURL: prURL + "#issuecomment-5"}}},
		{name: "record: REQUEST CHANGES", ctx: StatusContextRecord, in: statusInputs{Record: rec(headFull, "REQUEST CHANGES")},
			want: []CommitStatus{{Context: "ateam/review-record", State: "failure", Description: "REQUEST CHANGES self-reported by codex (gpt-6-astra, xhigh) at abc1234", TargetURL: prURL}}},
		{name: "record: other verdict", ctx: StatusContextRecord, in: statusInputs{Record: rec("abc1234", "other")},
			want: []CommitStatus{{Context: "ateam/review-record", State: "pending", Description: "record at abc1234 has no single verdict", TargetURL: prURL}}},
		{name: "record: no verdict", ctx: StatusContextRecord, in: statusInputs{Record: rec("abc1234", "")},
			want: []CommitStatus{{Context: "ateam/review-record", State: "pending", Description: "record at abc1234 has no single verdict", TargetURL: prURL}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.in.HeadSHA = headFull
			if tc.in.PRURL == "" {
				tc.in.PRURL = prURL
			}
			if got := statusesFor(chooseStatuses(tc.in), tc.ctx); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("%s = %+v, want %+v", tc.ctx, got, tc.want)
			}
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Informational commit statuses on the PR head
func TestCleanDescription(t *testing.T) {
	long := strings.Repeat("a", 139) + "é" // 141 bytes: the é straddles byte 140
	for _, tc := range []struct{ name, in, want string }{
		{name: "short text unchanged", in: "handoff at abc1234", want: "handoff at abc1234"},
		{name: "control characters dropped", in: "APPROVE by co\x1b[31mdex\n\r\u2028x", want: "APPROVE by co[31mdexx"},
		{name: "exactly 140 bytes kept", in: strings.Repeat("b", 140), want: strings.Repeat("b", 140)},
		{name: "cut before a straddling rune", in: long, want: strings.Repeat("a", 139)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := cleanDescription(tc.in); got != tc.want {
				t.Fatalf("cleanDescription(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@t"}, args...)...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func writeRepoFile(t *testing.T, dir, rel, body string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

const (
	approvedProposal = "# Change\n\nApproved-by: Ann Owner (2026-11-07)\n"
	emptyProposal    = "# Change\n\nApproved-by:\n"
	doneTasks        = "## 1. Approval\n\n- [x] 1.1 Owner approves (#1).\n"
	openQTasks       = "## 1. Approval\n\n- [ ] 1.1 Owner approves (#1). Hold: OPEN-Q proposal awaits owner approval\n"
)

// spec: openspec/specs/harness/spec.md#Requirement: Informational commit statuses on the PR head
func TestTouchedChangesAtHead(t *testing.T) {
	repo := t.TempDir()
	gitIn(t, repo, "init", "-q", "-b", "main")
	// A change already on main is not one the PR touches.
	writeRepoFile(t, repo, "openspec/changes/on-main/proposal.md", emptyProposal)
	gitIn(t, repo, "add", "-A")
	gitIn(t, repo, "commit", "-q", "-m", "init")
	gitIn(t, repo, "update-ref", "refs/remotes/origin/main", "main")

	branch := func(name string, files map[string]string) string {
		gitIn(t, repo, "checkout", "-q", "-b", name, "main")
		for rel, body := range files {
			writeRepoFile(t, repo, rel, body)
		}
		gitIn(t, repo, "add", "-A")
		gitIn(t, repo, "commit", "-q", "-m", name)
		sha := gitIn(t, repo, "rev-parse", "HEAD")
		gitIn(t, repo, "checkout", "-q", "main")
		return sha
	}
	approvedHead := branch("approved", map[string]string{
		"openspec/changes/feat-a/proposal.md": approvedProposal, "openspec/changes/feat-a/tasks.md": doneTasks, "main.go": "package main\n"})
	openQHead := branch("openq", map[string]string{
		"openspec/changes/feat-b/proposal.md": approvedProposal, "openspec/changes/feat-b/tasks.md": openQTasks})
	archivedHead := branch("archived", map[string]string{
		"openspec/changes/archive/2026-11-01-feat-c/proposal.md": approvedProposal, "openspec/changes/archive/2026-11-01-feat-c/tasks.md": doneTasks})
	twoHead := branch("two", map[string]string{
		"openspec/changes/feat-a/proposal.md": approvedProposal, "openspec/changes/feat-a/tasks.md": doneTasks,
		"openspec/changes/feat-d/proposal.md": emptyProposal, "openspec/changes/feat-d/tasks.md": doneTasks})
	noneHead := branch("none", map[string]string{"main.go": "package main\n", "openspec/specs/harness/spec.md": "# spec\n"})

	pending := func(desc, file, sha string) []CommitStatus {
		return []CommitStatus{{Context: "ateam/proposal", State: "pending", Description: desc, TargetURL: "https://github.com/o/r/blob/" + sha + "/" + file}}
	}
	success := func(desc, file, sha string) []CommitStatus {
		return []CommitStatus{{Context: "ateam/proposal", State: "success", Description: desc, TargetURL: "https://github.com/o/r/blob/" + sha + "/" + file}}
	}
	for _, tc := range []struct {
		name, head string
		want       []CommitStatus
	}{
		{name: "approved change", head: approvedHead,
			want: success("approved by Ann Owner (2026-11-07): feat-a", "openspec/changes/feat-a/proposal.md", approvedHead)},
		{name: "OPEN-Q change", head: openQHead,
			want: pending("awaiting owner approval (OPEN-Q): feat-b", "openspec/changes/feat-b/proposal.md", openQHead)},
		{name: "archived change", head: archivedHead,
			want: success("approved by Ann Owner (2026-11-07): 2026-11-01-feat-c", "openspec/changes/archive/2026-11-01-feat-c/proposal.md", archivedHead)},
		{name: "two changes, one unapproved", head: twoHead,
			want: pending("awaiting owner approval (OPEN-Q): feat-d", "openspec/changes/feat-d/proposal.md", twoHead)},
		{name: "no change touched", head: noneHead},
		{name: "head object missing locally", head: "0123456789abcdef0123456789abcdef01234567"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			changes := touchedChanges(context.Background(), repo, "origin/main", tc.head)
			got := statusesFor(chooseStatuses(statusInputs{Provider: ForgeGitHub, HeadSHA: tc.head, PRURL: prURL, Changes: changes}), StatusContextProposal)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("proposal status = %+v (changes %+v), want %+v", got, changes, tc.want)
			}
		})
	}
}

// fakeGHStatuses installs a gh that logs each call as GH_TOKEN|arg|arg..., answers the combined-status read from
// $FAKE_STATUS_JSON, and exits $FAKE_POST_EXIT (default 0) on a POST after printing a body that must never leak.
func fakeGHStatuses(t *testing.T, extra string) string {
	t.Helper()
	dir := t.TempDir()
	log := filepath.Join(dir, "calls")
	t.Setenv("FORGE_CALL_LOG", log)
	t.Setenv("GH_TOKEN", "") // the logged token is the role token or nothing, never the developer's own
	writeFake(t, filepath.Join(dir, "gh"), "{ printf '%s' \"$GH_TOKEN\"; for a in \"$@\"; do printf '|%s' \"$a\"; done; printf '\\n'; } >> \"$FORGE_CALL_LOG\"\n"+
		extra+
		"case \"$*\" in\n"+
		"'api -X POST '*) echo 'response body secret-body' >&2; exit \"${FAKE_POST_EXIT:-0}\";;\n"+
		"'api repos/'*'/status?per_page=100') cat \"$FAKE_STATUS_JSON\";;\n"+
		"*) printf '[]';;\nesac\n")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return log
}

func setStatusJSON(t *testing.T, body string) {
	t.Helper()
	p := filepath.Join(t.TempDir(), "status.json")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAKE_STATUS_JSON", p)
}

func readCalls(t *testing.T, log string) [][]string {
	t.Helper()
	data, err := os.ReadFile(log)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatal(err)
	}
	var calls [][]string
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line != "" {
			calls = append(calls, strings.Split(line, "|"))
		}
	}
	return calls
}

var threeStatuses = []CommitStatus{
	{Context: "ateam/proposal", State: "success", Description: "approved by Ann: feat-a", TargetURL: "https://github.com/o/r/blob/x/p.md"},
	{Context: "ateam/handoff", State: "success", Description: "handoff at abc1234, 2026-11-07T18:42Z", TargetURL: prURL},
	{Context: "ateam/review-record", State: "failure", Description: "REQUEST CHANGES by codex at abc1234", TargetURL: prURL},
}

// The latest ateam/handoff matches; ateam/proposal differs only in its description; ateam/review-record is absent.
const latestStatuses = `{"state":"pending","statuses":[` +
	`{"context":"ateam/handoff","state":"success","description":"handoff at abc1234, 2026-11-07T18:42Z"},` +
	`{"context":"ateam/proposal","state":"success","description":"approved by Ann: feat-old"},` +
	`{"context":"ci/build","state":"success","description":"ok"}]}`

// spec: openspec/specs/harness/spec.md#Requirement: Informational commit statuses on the PR head
// spec: openspec/specs/harness/spec.md#Requirement: Forge token roles
func TestPublishCommitStatusesGitHub(t *testing.T) {
	t.Setenv("ATEAM_TOKEN_READ", "fake-read")
	t.Setenv("ATEAM_TOKEN_WRITE", "fake-write")
	log := fakeGHStatuses(t, "")
	setStatusJSON(t, latestStatuses)
	res, err := PublishCommitStatuses(context.Background(), t.TempDir(), ForgeConfig{Provider: ForgeGitHub, Owner: "o", Repo: "r"}, headFull, threeStatuses)
	if err != nil {
		t.Fatal(err)
	}
	if want := (StatusResult{Posted: 2, Unchanged: 1}); res != want {
		t.Fatalf("result = %+v, want %+v", res, want)
	}
	want := [][]string{
		{"fake-read", "api", "repos/o/r/commits/" + headFull + "/status?per_page=100"},
		{"fake-write", "api", "-X", "POST", "repos/o/r/statuses/" + headFull, "-f", "state=success", "-f", "context=ateam/proposal",
			"-f", "description=approved by Ann: feat-a", "-f", "target_url=https://github.com/o/r/blob/x/p.md"},
		{"fake-write", "api", "-X", "POST", "repos/o/r/statuses/" + headFull, "-f", "state=failure", "-f", "context=ateam/review-record",
			"-f", "description=REQUEST CHANGES by codex at abc1234", "-f", "target_url=" + prURL},
	}
	if got := readCalls(t, log); !reflect.DeepEqual(got, want) {
		t.Fatalf("gh calls =\n%q\nwant\n%q", got, want)
	}

	// A second run against statuses that now all match posts nothing.
	if err := os.Remove(log); err != nil {
		t.Fatal(err)
	}
	setStatusJSON(t, `{"statuses":[`+
		`{"context":"ateam/review-record","state":"failure","description":"REQUEST CHANGES by codex at abc1234"},`+
		`{"context":"ateam/proposal","state":"success","description":"approved by Ann: feat-a"},`+
		`{"context":"ateam/handoff","state":"success","description":"handoff at abc1234, 2026-11-07T18:42Z"}]}`)
	res, err = PublishCommitStatuses(context.Background(), t.TempDir(), ForgeConfig{Provider: ForgeGitHub, Owner: "o", Repo: "r"}, headFull, threeStatuses)
	if err != nil || res != (StatusResult{Unchanged: 3}) {
		t.Fatalf("second run = %+v, %v; want unchanged 3", res, err)
	}
	if calls := readCalls(t, log); len(calls) != 1 {
		t.Fatalf("second run made %d gh calls, want only the read: %q", len(calls), calls)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Informational commit statuses on the PR head
func TestPublishCommitStatusesGitHubWriteError(t *testing.T) {
	t.Setenv("ATEAM_TOKEN_WRITE", "fake-write-secret")
	t.Setenv("FAKE_POST_EXIT", "4")
	fakeGHStatuses(t, "")
	setStatusJSON(t, `{"statuses":[]}`)
	res, err := PublishCommitStatuses(context.Background(), t.TempDir(), ForgeConfig{Provider: ForgeGitHub, Owner: "o", Repo: "r"}, headFull, threeStatuses[:2])
	if want := "ateam/proposal on abc1234: gh exit 4"; err == nil || err.Error() != want {
		t.Fatalf("error = %v, want %q", err, want)
	}
	if res != (StatusResult{Errors: 2}) {
		t.Fatalf("result = %+v, want errors 2", res)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Informational commit statuses on the PR head
// spec: openspec/specs/harness/spec.md#Requirement: Forge token roles
func TestPublishCommitStatusesGitea(t *testing.T) {
	t.Setenv("ATEAM_TOKEN_READ", "fake-read")
	t.Setenv("ATEAM_TOKEN_WRITE", "fake-write")
	var mu sync.Mutex
	type call struct {
		method, path, auth string
		body               map[string]string
	}
	var calls []call
	postCode := http.StatusCreated
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]string
		_ = json.Unmarshal(raw, &body)
		mu.Lock()
		calls = append(calls, call{r.Method, r.URL.Path, r.Header.Get("Authorization"), body})
		code := postCode
		mu.Unlock()
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repos/o/r/commits/"+headFull+"/status":
			// Gitea names each status's state "status".
			_, _ = w.Write([]byte(`{"state":"pending","statuses":[` +
				`{"context":"ateam/handoff","status":"success","description":"handoff at abc1234, 2026-11-07T18:42Z"},` +
				`{"context":"ateam/proposal","status":"pending","description":"approved by Ann: feat-a"}]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/repos/o/r/statuses/"+headFull:
			w.WriteHeader(code)
			_, _ = w.Write([]byte(`{"message":"secret-body"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	cfg := ForgeConfig{Provider: ForgeGitea, BaseURL: srv.URL, Owner: "o", Repo: "r"}
	res, err := PublishCommitStatuses(context.Background(), t.TempDir(), cfg, headFull, threeStatuses)
	if err != nil {
		t.Fatal(err)
	}
	if want := (StatusResult{Posted: 2, Unchanged: 1}); res != want {
		t.Fatalf("result = %+v, want %+v", res, want)
	}
	want := []call{
		{method: "GET", path: "/api/v1/repos/o/r/commits/" + headFull + "/status", auth: "token fake-read"},
		{method: "POST", path: "/api/v1/repos/o/r/statuses/" + headFull, auth: "token fake-write",
			body: map[string]string{"context": "ateam/proposal", "state": "success", "description": "approved by Ann: feat-a", "target_url": "https://github.com/o/r/blob/x/p.md"}},
		{method: "POST", path: "/api/v1/repos/o/r/statuses/" + headFull, auth: "token fake-write",
			body: map[string]string{"context": "ateam/review-record", "state": "failure", "description": "REQUEST CHANGES by codex at abc1234", "target_url": prURL}},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls =\n%+v\nwant\n%+v", calls, want)
	}

	mu.Lock()
	postCode = http.StatusForbidden
	mu.Unlock()
	_, err = PublishCommitStatuses(context.Background(), t.TempDir(), cfg, headFull, threeStatuses)
	if want := "ateam/proposal on abc1234: HTTP 403"; err == nil || err.Error() != want {
		t.Fatalf("write error = %v, want %q (no body, no token)", err, want)
	}
}

// statusRepo is a repository on main with origin/main, and a linked worktree on claude/pr-7 one commit ahead; it
// returns the root, the worktree, and the worktree's full head.
func statusRepo(t *testing.T) (root, wt, head string) {
	t.Helper()
	root, wt = t.TempDir(), filepath.Join(t.TempDir(), "pr-7")
	gitIn(t, root, "init", "-q", "-b", "main")
	gitIn(t, root, "commit", "-q", "--allow-empty", "-m", "init")
	gitIn(t, root, "update-ref", "refs/remotes/origin/main", "main")
	gitIn(t, root, "worktree", "add", "-q", "-b", "claude/pr-7", wt)
	gitIn(t, wt, "commit", "-q", "--allow-empty", "-m", "work")
	return root, wt, gitIn(t, wt, "rev-parse", "HEAD")
}

// spec: openspec/specs/harness/spec.md#Requirement: Informational commit statuses on the PR head
func TestSyncCommitStatusesWorktreePRsOnly(t *testing.T) {
	t.Setenv("ATEAM_TOKEN_READ", "")
	t.Setenv("ATEAM_TOKEN_WRITE", "")
	root, wt, head := statusRepo(t)
	at := time.Date(2026, 11, 7, 18, 42, 0, 0, time.UTC)
	if err := Save(wt, &State{Handoffs: map[string]*Handoff{
		"claude@claude/pr-7": {Branch: "claude/pr-7", CheckpointSHA: head[:7], UpdatedAt: at},
		"codex@claude/pr-7":  {Branch: "claude/pr-7", CheckpointSHA: "0123456", UpdatedAt: at.Add(-time.Hour)},
	}}); err != nil {
		t.Fatal(err)
	}
	log := fakeGHStatuses(t, "")
	setStatusJSON(t, `{"statuses":[]}`)
	cards := []BoardCard{
		{ID: "GH-PR-7", Number: 7, Type: "pr", Column: "in-progress", Branch: "claude/pr-7", HeadSHA: head, URL: prURL, HandoffURL: prURL + "#issuecomment-9"},
		{ID: "GH-PR-8", Number: 8, Type: "pr", Column: "in-progress", Branch: "claude/pr-8", HeadSHA: head, URL: prURL},
		{ID: "GH-PR-6", Number: 6, Type: "pr", Column: "done", Merged: true, Branch: "claude/pr-7", HeadSHA: head, URL: prURL},
		{ID: "GH-5", Number: 5, Type: "issue", Column: "backlog", Branch: "claude/pr-7"},
	}
	res, err := SyncCommitStatuses(context.Background(), root, ForgeConfig{Provider: ForgeGitHub, Owner: "o", Repo: "r"}, "origin/main", cards)
	if err != nil || res != (StatusResult{Posted: 1}) {
		t.Fatalf("sync statuses = %+v, %v; want posted 1 (the newest handoff, no change, no record)", res, err)
	}
	want := [][]string{
		{"", "api", "repos/o/r/commits/" + head + "/status?per_page=100"},
		{"", "api", "-X", "POST", "repos/o/r/statuses/" + head, "-f", "state=success", "-f", "context=ateam/handoff",
			"-f", "description=handoff at " + head[:7] + ", 2026-11-07T18:42Z", "-f", "target_url=" + prURL + "#issuecomment-9"},
	}
	if got := readCalls(t, log); !reflect.DeepEqual(got, want) {
		t.Fatalf("gh calls =\n%q\nwant\n%q", got, want)
	}
	if st, err := Load(root); err != nil || st.StatusWriteError != "" {
		t.Fatalf("status_write_error = %q (%v) after a clean sync, want empty", st.StatusWriteError, err)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Informational commit statuses on the PR head
func TestStatusWriteErrorRecordedAndCleared(t *testing.T) {
	root, wt, head := statusRepo(t)
	if err := Save(wt, &State{Handoffs: map[string]*Handoff{"claude@claude/pr-7": {Branch: "claude/pr-7", CheckpointSHA: head}}}); err != nil {
		t.Fatal(err)
	}
	fakeGHStatuses(t, "")
	setStatusJSON(t, `{"statuses":[]}`)
	t.Setenv("ATEAM_TOKEN_WRITE", "fake-write-secret")
	cards := []BoardCard{{ID: "GH-PR-7", Number: 7, Type: "pr", Column: "in-progress", Branch: "claude/pr-7", HeadSHA: head, URL: prURL}}
	cfg := ForgeConfig{Provider: ForgeGitHub, Owner: "o", Repo: "r"}

	t.Setenv("FAKE_POST_EXIT", "1")
	if _, err := SyncCommitStatuses(context.Background(), root, cfg, "origin/main", cards); err == nil {
		t.Fatal("sync statuses reported no error for a rejected write")
	}
	raw, err := os.ReadFile(CacheFilePath(root))
	if err != nil {
		t.Fatal(err)
	}
	if want := `"status_write_error": "ateam/handoff on ` + head[:7] + `: gh exit 1"`; !strings.Contains(string(raw), want) {
		t.Fatalf("cache lacks %s:\n%s", want, raw)
	}
	if strings.Contains(string(raw), "secret") {
		t.Fatalf("cache holds a token or response body:\n%s", raw)
	}

	t.Setenv("FAKE_POST_EXIT", "0")
	if _, err := SyncCommitStatuses(context.Background(), root, cfg, "origin/main", cards); err != nil {
		t.Fatal(err)
	}
	if st, err := Load(root); err != nil || st.StatusWriteError != "" {
		t.Fatalf("status_write_error = %q (%v) after a clean publication, want cleared", st.StatusWriteError, err)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Informational commit statuses on the PR head
func TestBestEffortSyncWritesNoStatus(t *testing.T) {
	// Pickup, prune, board, MCP and the UI all sync through RefreshWithBestEffortSync; none may write a status.
	root, wt, head := statusRepo(t)
	gitIn(t, root, "remote", "add", "origin", "https://github.com/o/r.git")
	if err := Save(wt, &State{Handoffs: map[string]*Handoff{"claude@claude/pr-7": {Branch: "claude/pr-7", CheckpointSHA: head}}}); err != nil {
		t.Fatal(err)
	}
	log := fakeGHStatuses(t, "case \"$*\" in 'pr list --state open'*) printf '%s' '[{\"number\":7,\"title\":\"t\",\"state\":\"OPEN\",\"headRefName\":\"claude/pr-7\",\"headRefOid\":\""+head+"\"}]'; exit 0;; esac\n")
	setStatusJSON(t, `{"statuses":[]}`)
	st, err := RefreshWithBestEffortSync(context.Background(), root, 0)
	if err != nil {
		t.Fatal(err)
	}
	// Activation: the sync ran and saw the worktree PR.
	if !st.hasCardsForProvider(ForgeGitHub) {
		t.Fatalf("best-effort sync made no GitHub cards (error %q); the test proves nothing", st.BoardSyncError)
	}
	for _, c := range readCalls(t, log) {
		if len(c) > 1 && c[1] == "api" {
			t.Fatalf("best-effort sync called gh api: %q", c)
		}
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Informational commit statuses on the PR head
func TestSyncKeepsCommentURLsOffCache(t *testing.T) {
	// A record and a handoff block reach the card with their links; the cached state.json never holds them.
	handoffBody := strings.TrimSpace(FormatFiveFieldPrompt(Handoff{Goal: "g", ChecksToRun: "task verify"}))
	for _, tc := range []struct {
		name  string
		cards func(t *testing.T, root string) []BoardCard
	}{
		{name: "github", cards: func(t *testing.T, root string) []BoardCard {
			comments, _ := json.Marshal(map[string]any{"comments": []map[string]string{
				{"body": handoffBody, "url": "https://github.com/o/r/pull/7#issuecomment-1"},
				{"body": reviewRecordBody, "url": "https://github.com/o/r/pull/7#issuecomment-2"},
				{"body": handoffBody, "url": "https://github.com/o/r/pull/7#issuecomment-3"},
				{"body": "thanks", "url": "https://github.com/o/r/pull/7#issuecomment-4"},
			}})
			installForgeCLI(t, "gh", "case \"$*\" in\n"+
				"'pr list --state open'*) printf '%s' '[{\"number\":7,\"title\":\"PR 7\",\"state\":\"OPEN\",\"headRefName\":\"claude/pr-7\"}]';;\n"+
				"'pr view 7 --json comments'*) printf '%s' '"+string(comments)+"';;\n"+
				"*) printf '[]';;\nesac\n")
			cards, err := SyncForgeBoard(context.Background(), root, ForgeConfig{Provider: ForgeGitHub})
			if err != nil {
				t.Fatal(err)
			}
			return cards
		}},
		{name: "gitea", cards: func(t *testing.T, root string) []BoardCard {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				q := r.URL.Query()
				switch {
				case strings.HasSuffix(r.URL.Path, "/issues/7/comments"):
					_ = json.NewEncoder(w).Encode([]map[string]string{
						{"body": handoffBody, "html_url": "https://github.com/o/r/pull/7#issuecomment-1"},
						{"body": reviewRecordBody, "html_url": "https://github.com/o/r/pull/7#issuecomment-2"},
						{"body": handoffBody, "html_url": "https://github.com/o/r/pull/7#issuecomment-3"},
					})
				case strings.HasSuffix(r.URL.Path, "/pulls") && q.Get("state") == "open" && q.Get("page") == "1":
					_ = json.NewEncoder(w).Encode([]map[string]any{{"number": 7, "title": "PR 7", "state": "open", "head": map[string]string{"ref": "claude/pr-7"}}})
				default:
					_, _ = w.Write([]byte(`[]`))
				}
			}))
			t.Cleanup(srv.Close)
			cards, err := SyncForgeBoard(context.Background(), root, ForgeConfig{Provider: ForgeGitea, BaseURL: srv.URL, Owner: "o", Repo: "r"})
			if err != nil {
				t.Fatal(err)
			}
			return cards
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := reviewRepo(t)
			cards := tc.cards(t, root)
			var card *BoardCard
			for i := range cards {
				if cards[i].Number == 7 && isPRCard(cards[i]) {
					card = &cards[i]
				}
			}
			if card == nil || card.Review == nil {
				t.Fatalf("PR #7 card or record missing: %+v", cards)
			}
			if card.ReviewURL != "https://github.com/o/r/pull/7#issuecomment-2" || card.HandoffURL != "https://github.com/o/r/pull/7#issuecomment-3" {
				t.Fatalf("record URL %q, handoff URL %q; want comments 2 and 3", card.ReviewURL, card.HandoffURL)
			}
			with := defaultState()
			with.MergeForgeCards(cards, ForgeGitHub)
			without := defaultState()
			stripped := append([]BoardCard(nil), cards...)
			for i := range stripped {
				stripped[i].ReviewURL, stripped[i].HandoffURL = "", ""
			}
			without.MergeForgeCards(stripped, ForgeGitHub)
			without.UpdatedAt, without.BoardSyncedAt, without.ForgeSyncedAt = with.UpdatedAt, with.BoardSyncedAt, with.ForgeSyncedAt
			a, _ := json.MarshalIndent(with, "", "  ")
			b, _ := json.MarshalIndent(without, "", "  ")
			if string(a) != string(b) || strings.Contains(string(a), "issuecomment") {
				t.Fatalf("cached state differs with comment URLs:\n%s\nwithout:\n%s", a, b)
			}
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Informational commit statuses on the PR head
func TestPostPRHandoffCommentReturnsPRHead(t *testing.T) {
	t.Run("github", func(t *testing.T) {
		installForgeCLI(t, "gh", "case \"$*\" in\n"+
			"'pr comment'*) printf '%s\\n' 'https://github.com/o/r/pull/7#issuecomment-42';;\n"+
			"'pr view claude/x --json number,headRefOid,url'*) printf '%s' '{\"number\":7,\"headRefOid\":\""+headFull+"\",\"url\":\"https://github.com/o/r/pull/7\"}';;\n"+
			"*) exit 1;;\nesac\n")
		post, err := PostPRHandoffComment(context.Background(), t.TempDir(), ForgeConfig{Provider: ForgeGitHub}, "claude/x", "Goal: g")
		want := HandoffPost{Posted: true, Number: 7, HeadSHA: headFull, PRURL: "https://github.com/o/r/pull/7", CommentURL: "https://github.com/o/r/pull/7#issuecomment-42"}
		if err != nil || post != want {
			t.Fatalf("post = %+v, %v; want %+v", post, err, want)
		}
	})
	t.Run("gitea", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repos/o/r/pulls" && r.URL.Query().Get("page") == "1":
				_, _ = w.Write([]byte(`[{"number":7,"title":"t","state":"open","html_url":"https://forge.example/o/r/pulls/7","head":{"ref":"claude/x","sha":"` + headFull + `"}}]`))
			case r.Method == http.MethodPost && r.URL.Path == "/api/v1/repos/o/r/issues/7/comments":
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"id":42,"html_url":"https://forge.example/o/r/pulls/7#issuecomment-42","body":"Goal: g"}`))
			default:
				_, _ = w.Write([]byte(`[]`))
			}
		}))
		defer srv.Close()
		post, err := PostPRHandoffComment(context.Background(), t.TempDir(), ForgeConfig{Provider: ForgeGitea, BaseURL: srv.URL, Owner: "o", Repo: "r"}, "claude/x", "Goal: g")
		want := HandoffPost{Posted: true, Number: 7, HeadSHA: headFull, PRURL: "https://forge.example/o/r/pulls/7", CommentURL: "https://forge.example/o/r/pulls/7#issuecomment-42"}
		if err != nil || post != want {
			t.Fatalf("post = %+v, %v; want %+v", post, err, want)
		}
	})
}
