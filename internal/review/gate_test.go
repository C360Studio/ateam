package review

import (
	"bytes"
	"cmp"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// gateWorld is one pull request and its repository, described once and served
// by both forges, so a scenario expects the same lines from either.
type gateWorld struct {
	draft, bot bool
	body       string
	files      []string
	commits    []worldCommit // oldest first; the last is head
	comments   []string
	labels     []string
	issues     []worldIssue // open issues
	// required: main requires the Required check; strict: and the up-to-date setting.
	required, strict bool
	// giteaContexts overrides the status checks Gitea's protection lists;
	// giteaChecksOff turns its status checks off.
	giteaContexts    []string
	giteaChecksOff   bool
	giteaUnprotected bool
	giteaRuleHidden  bool
	// giteaOldestFirst serves Gitea's commit list oldest first instead of as git log does.
	giteaOldestFirst bool
	// giteaStatus answers a Gitea path with an HTTP status instead.
	giteaStatus map[string]int
	// giteaTotalSkew adds to the X-Total-Count Gitea sends for a list path.
	giteaTotalSkew map[string]int
	// ghNull answers these gh argv with the JSON literal null.
	ghNull []string
	// proposals serves file content at head; proposalsMissing answers 404.
	// GitHub fails a read of any other path; Gitea answers it 404.
	proposals        map[string]string
	proposalsMissing []string
}

type worldCommit struct {
	sha, message string
	parents      []string
	files        []string
}

type worldIssue struct {
	number   int
	flake    bool
	closedBy []int // pull requests whose closing reference stands
	neutered []int // pull requests whose closing reference was edited away
	noise    int   // timeline entries before the references
	pr       bool  // a pull request carrying the label (Gitea's issue list may return it)
}

const (
	gatePR       = 12
	gateOwner    = "acme"
	gateRepo     = "widgets"
	gateToken    = "test-token"
	gateHTMLRoot = "https://gitea.example.test/acme/widgets"
	shaBase      = "03121ae8066ec58a2ecc5a95c38c4447389eb0a3"
	shaA         = "99d19a4cfa55e59c62dd49f79bd420e653675767"
	shaContent   = "7413d0c05c25db0e9143f7a466a64954855aa979"
	shaMerge     = "0209eb7f6ebaf44e2dd87c401d61a7a48763799b"
	shaMainTip   = "dc854c651fbf82695f4d552badc36340e2f7c472"
	implRevCodex = "implemented-by: claude (opus)\nreviewed-by: codex (gpt-6)\n"
)

func issueURL(n int) string { return fmt.Sprintf("%s/issues/%d", gateHTMLRoot, n) }

func healthyWorld() gateWorld {
	return gateWorld{
		body:     implRevCodex,
		files:    []string{"docs/a.md", "scripts/x.sh"},
		commits:  []worldCommit{{sha: shaA, message: "feat: add script", parents: []string{shaBase}, files: []string{"docs/a.md", "scripts/x.sh"}}},
		comments: []string{"reviewed-by: codex\nReviewed at " + shaA[:7] + "\nVerdict: APPROVE"},
		labels:   []string{"bug", "class:flake"},
		required: true,
		strict:   true,
	}
}

func worldFiles(names []string) []map[string]any {
	out := make([]map[string]any, len(names))
	for i, n := range names {
		out[i] = map[string]any{"filename": n, "status": "modified", "additions": 1, "deletions": 0}
	}
	return out
}

func worldCommitJSON(c worldCommit) map[string]any {
	parents := make([]map[string]any, len(c.parents))
	for i, p := range c.parents {
		parents[i] = map[string]any{"sha": p}
	}
	return map[string]any{"sha": c.sha, "commit": map[string]any{"message": c.message}, "parents": parents}
}

// fakeGH answers each argv from a file pair written by the test.
const fakeGHScript = `#!/bin/sh
for f in "$FAKE_GH_DIR"/*.argv; do
  if [ "$(cat "$f")" = "$*" ]; then
    if [ -f "${f%.argv}.err" ]; then cat "${f%.argv}.err" >&2; exit 1; fi
    cat "${f%.argv}.out"; exit 0
  fi
done
echo "fake gh: unexpected call: $*" >&2
exit 3
`

// runGitHub runs Gate against a fake gh serving w as GitHub would.
func runGitHub(t *testing.T, w gateWorld) (int, string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "gh"), []byte(fakeGHScript), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FAKE_GH_DIR", dir)
	answers := map[string][]byte{}
	rule := map[string]any{"type": "deletion", "ruleset_id": 7}
	if w.required {
		rule = map[string]any{"type": "required_status_checks", "ruleset_id": 7, "parameters": map[string]any{
			"strict_required_status_checks_policy": w.strict, "required_status_checks": []map[string]any{{"context": "Required"}}}}
	}
	answers["api repos/{owner}/{repo}/rules/branches/main"] = mustJSON(t, []any{rule})
	answers["api repos/{owner}/{repo}/rulesets/7"] = []byte(`{"enforcement":"active"}`)
	userType := "User"
	if w.bot {
		userType = "Bot"
	}
	head := w.commits[len(w.commits)-1].sha
	answers[fmt.Sprintf("api repos/{owner}/{repo}/pulls/%d", gatePR)] = mustJSON(t, map[string]any{
		"draft": w.draft, "changed_files": len(w.files), "commits": len(w.commits), "body": w.body,
		"head": map[string]any{"sha": head}, "user": map[string]any{"type": userType}})
	answers[fmt.Sprintf("api --paginate repos/{owner}/{repo}/pulls/%d/files?per_page=100", gatePR)] = mustJSON(t, worldFiles(w.files))
	commits := []map[string]any{}
	for _, c := range w.commits {
		commits = append(commits, worldCommitJSON(c))
		detail := worldCommitJSON(c)
		detail["files"] = worldFiles(c.files)
		answers["api --paginate repos/{owner}/{repo}/commits/"+c.sha+"?per_page=100"] = mustJSON(t, detail)
	}
	answers[fmt.Sprintf("api --paginate repos/{owner}/{repo}/pulls/%d/commits?per_page=100", gatePR)] = mustJSON(t, commits)
	comments := []map[string]any{}
	for _, b := range w.comments {
		comments = append(comments, map[string]any{"body": b})
	}
	answers[fmt.Sprintf("api --paginate repos/{owner}/{repo}/issues/%d/comments?per_page=100", gatePR)] = mustJSON(t, comments)
	labels := []map[string]any{}
	for _, l := range w.labels {
		labels = append(labels, map[string]any{"name": l})
	}
	answers["label list --json name --limit 100"] = mustJSON(t, labels)
	issues, refs := []map[string]any{}, []map[string]any{}
	for _, is := range w.issues {
		if is.flake && !is.pr {
			issues = append(issues, map[string]any{"number": is.number, "url": issueURL(is.number), "title": "flaky"})
		}
		if slices.Contains(is.closedBy, gatePR) {
			refs = append(refs, map[string]any{"number": is.number, "url": issueURL(is.number)})
		}
	}
	answers["issue list --label class:flake --state open --json number,url,title --limit 100"] = mustJSON(t, issues)
	answers[fmt.Sprintf("pr view %d --json closingIssuesReferences", gatePR)] = mustJSON(t, map[string]any{"closingIssuesReferences": refs})
	contents := func(path string) string { return "api repos/{owner}/{repo}/contents/" + path + "?ref=" + head }
	for path, body := range w.proposals {
		answers[contents(path)] = mustJSON(t, map[string]any{"type": "file", "encoding": "base64", "content": base64.StdEncoding.EncodeToString([]byte(body))})
	}
	errs := map[string]string{}
	for _, path := range w.proposalsMissing {
		answers[contents(path)] = nil
		errs[contents(path)] = "gh: Not Found (HTTP 404)"
	}
	for _, argv := range w.ghNull {
		if _, ok := answers[argv]; !ok {
			t.Fatalf("ghNull names no read: %s", argv)
		}
		answers[argv] = []byte("null")
	}
	i := 0
	for argv, out := range answers {
		i++
		base := filepath.Join(dir, strconv.Itoa(i))
		if err := os.WriteFile(base+".argv", []byte(argv), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(base+".out", out, 0o644); err != nil {
			t.Fatal(err)
		}
		if e, ok := errs[argv]; ok {
			if err := os.WriteFile(base+".err", []byte(e), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	var out bytes.Buffer
	code := Gate(context.Background(), &out, ForgeConfig{Provider: "github"}, GateOptions{PR: strconv.Itoa(gatePR), PolicyPath: filepath.Join(dir, "no-policy.json")})
	return code, out.String()
}

// giteaServer serves w as Gitea v1 does: lists paginated with X-Total-Count,
// PR commits newest first, comments in one response, the issue label filter
// not trusted (every open issue is returned), closing references only on the
// referenced issue's timeline.
func giteaServer(t *testing.T, w gateWorld) *httptest.Server {
	t.Helper()
	prefix := "/api/v1/repos/" + gateOwner + "/" + gateRepo
	contexts := w.giteaContexts
	if contexts == nil {
		contexts = []string{"CI / Verify (pull_request)"}
		if w.required {
			contexts = append(contexts, giteaRequired)
		}
	}
	userType := "User"
	if w.bot {
		userType = "Bot"
	}
	commits := make([]any, 0, len(w.commits))
	for _, c := range w.commits {
		commits = append(commits, worldCommitJSON(c))
	}
	if !w.giteaOldestFirst {
		slices.Reverse(commits)
	}
	var files, comments, labels, issues []any
	for _, f := range worldFiles(w.files) {
		files = append(files, f)
	}
	for _, b := range w.comments {
		comments = append(comments, map[string]any{"body": b})
	}
	for _, l := range w.labels {
		labels = append(labels, map[string]any{"name": l})
	}
	timelines := map[string][]any{}
	for _, is := range w.issues {
		ls := []any{map[string]any{"name": "bug"}}
		if is.flake {
			ls = append(ls, map[string]any{"name": "class:flake"})
		}
		var prMeta any
		if is.pr {
			prMeta = map[string]any{"merged": false}
		}
		issues = append(issues, map[string]any{"number": is.number, "html_url": issueURL(is.number), "title": "flaky", "labels": ls, "pull_request": prMeta})
		entries := []any{map[string]any{"type": "comment", "ref_action": "", "ref_issue": nil}}
		for range is.noise {
			entries = append(entries, map[string]any{"type": "label", "ref_action": "", "ref_issue": nil})
		}
		ref := func(action string, pr int, repo string) map[string]any {
			return map[string]any{"type": "pull_ref", "ref_action": action, "ref_issue": map[string]any{
				"number": pr, "pull_request": map[string]any{"merged": false}, "repository": map[string]any{"full_name": repo}}}
		}
		for _, pr := range is.closedBy {
			entries = append(entries, ref("closes", pr, gateOwner+"/"+gateRepo))
		}
		for _, pr := range is.neutered {
			entries = append(entries, ref("neutered", pr, gateOwner+"/"+gateRepo))
		}
		// The same number closing it from another repository never counts.
		entries = append(entries, ref("closes", gatePR, "someone/fork"))
		timelines[fmt.Sprintf("/issues/%d/timeline", is.number)] = entries
	}
	lists := map[string][]any{
		fmt.Sprintf("/pulls/%d/files", gatePR):   files,
		fmt.Sprintf("/pulls/%d/commits", gatePR): commits,
		"/labels":                                labels,
		"/issues":                                issues,
	}
	for p, e := range timelines {
		lists[p] = e
	}
	ruleName := "main"
	if w.giteaRuleHidden {
		ruleName = ""
	}
	objects := map[string]any{
		"/branches/main": map[string]any{"name": "main", "protected": !w.giteaUnprotected, "effective_branch_protection_name": ruleName,
			"enable_status_check": !w.giteaChecksOff, "status_check_contexts": contexts},
		"/branch_protections/main": map[string]any{"rule_name": "main", "enable_status_check": !w.giteaChecksOff, "status_check_contexts": contexts, "block_on_outdated_branch": w.strict},
		fmt.Sprintf("/pulls/%d", gatePR): map[string]any{"draft": w.draft, "changed_files": len(w.files), "body": w.body,
			"head": map[string]any{"sha": w.commits[len(w.commits)-1].sha}, "user": map[string]any{"type": userType}},
		fmt.Sprintf("/issues/%d/comments", gatePR): comments,
	}
	for path, body := range w.proposals {
		objects["/contents/"+path] = map[string]any{"type": "file", "encoding": "base64", "content": base64.StdEncoding.EncodeToString([]byte(body))}
	}
	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "token "+gateToken {
			http.Error(rw, "unauthorized", http.StatusUnauthorized)
			return
		}
		path, ok := strings.CutPrefix(r.URL.Path, prefix)
		if !ok {
			http.NotFound(rw, r)
			return
		}
		if status := w.giteaStatus[path]; status != 0 {
			http.Error(rw, "forbidden", status)
			return
		}
		if strings.HasPrefix(path, "/contents/") && r.URL.Query().Get("ref") != w.commits[len(w.commits)-1].sha {
			http.Error(rw, "contents read not at head", http.StatusBadRequest)
			return
		}
		if obj, ok := objects[path]; ok {
			if err := json.NewEncoder(rw).Encode(obj); err != nil {
				t.Errorf("fake gitea: write %s: %v", path, err)
			}
			return
		}
		items, ok := lists[path]
		if !ok {
			http.NotFound(rw, r)
			return
		}
		if path == fmt.Sprintf("/pulls/%d/commits", gatePR) && (r.URL.Query().Get("files") != "false" || r.URL.Query().Get("verification") != "false") {
			http.Error(rw, "commit list asked for files or verification", http.StatusBadRequest)
			return
		}
		// Gitea reads a missing or invalid page as 1 and caps limit at 50.
		page, limit := queryInt(r, "page", 1), min(queryInt(r, "limit", 50), 50)
		start := min((page-1)*limit, len(items))
		pageItems := items[start:min(start+limit, len(items))]
		total := len(items) + w.giteaTotalSkew[path]
		if strings.HasSuffix(path, "/timeline") {
			// Gitea 28.2 sets the timeline's X-Total-Count to the page length.
			total = len(pageItems)
		}
		rw.Header().Set("X-Total-Count", strconv.Itoa(total))
		if err := json.NewEncoder(rw).Encode(append([]any{}, pageItems...)); err != nil {
			t.Errorf("fake gitea: write %s: %v", path, err)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func queryInt(r *http.Request, key string, def int) int {
	n, err := strconv.Atoi(r.URL.Query().Get(key))
	if err != nil || n < 1 {
		return def
	}
	return n
}

func runGitea(t *testing.T, w gateWorld) (int, string) {
	t.Helper()
	srv := giteaServer(t, w)
	var out bytes.Buffer
	cfg := ForgeConfig{Provider: "gitea", BaseURL: srv.URL, Owner: gateOwner, Repo: gateRepo, Token: gateToken}
	code := Gate(context.Background(), &out, cfg, GateOptions{PR: strconv.Itoa(gatePR), PolicyPath: filepath.Join(t.TempDir(), "no-policy.json")})
	// Messages name API paths, never the forge host (owner ruling 6 on #143).
	if host := strings.TrimPrefix(srv.URL, "http://"); strings.Contains(out.String(), host) {
		t.Errorf("gitea output names the forge host %s\n%s", host, out.String())
	}
	return code, out.String()
}

type forgeWant struct {
	code  int
	lines []string
}

// spec: openspec/specs/harness/spec.md#Requirement: Merge gate and CI workflow enforcement
// spec: openspec/specs/model-review/spec.md#Requirement: Portable enforcement and honest trust boundaries
func TestGateGitea(t *testing.T) {
	// #143: one scenario, the same verdict and lines on GitHub and Gitea, except
	// where an owner ruling makes Gitea stricter. A nil github want is Gitea only.
	contentCommit := worldCommit{sha: shaContent, message: "fix: adjust script", parents: []string{shaA}, files: []string{"scripts/x.sh"}}
	mergeMain := worldCommit{sha: shaMerge, message: "Merge branch 'main' into claude/x", parents: []string{shaA, shaMainTip}, files: []string{"internal/other/main_only.go"}}
	giteaRerun := "review record at " + shaA + " is older than commit "
	manyDocs := make([]string, 120)
	for i := range manyDocs {
		manyDocs[i] = fmt.Sprintf("docs/%03d.md", i)
	}
	for _, tc := range []struct {
		name   string
		world  func(w *gateWorld)
		both   *forgeWant
		github *forgeWant
		gitea  *forgeWant
	}{
		{name: "documents only", world: func(w *gateWorld) {
			w.body, w.files, w.comments = "", []string{"docs/a.md", "openspec/config.yaml"}, nil
		},
			both: &forgeWant{0, []string{"is documents only (2 changed files)", "no open class:flake issue", "merge-check: ok"}}},
		{name: "documents only across pages", world: func(w *gateWorld) { w.body, w.files, w.comments = "", manyDocs, nil },
			both: &forgeWant{0, []string{"is documents only (120 changed files)"}}},
		{name: "markdown under .claude/agents is code", world: func(w *gateWorld) { w.files, w.comments = []string{".claude/agents/dev.md"}, nil },
			both: &forgeWant{1, []string{"code file: .claude/agents/dev.md", "no review record comment names reviewed-by: codex"}}},
		{name: "record at head", world: func(*gateWorld) {},
			both: &forgeWant{0, []string{"reviewing agent: codex (record at " + shaA + ")", "merge-check: ok"}}},
		{name: "record then content commit", world: func(w *gateWorld) { w.commits = append(w.commits, contentCommit) },
			github: &forgeWant{1, []string{"review record at " + shaA + " is older than content commit " + shaContent + "; request re-review"}},
			gitea:  &forgeWant{1, []string{giteaRerun + shaContent + ", and on Gitea every commit after the review record needs a new review"}}},
		{name: "record then content commit, Gitea list oldest first", world: func(w *gateWorld) { w.commits, w.giteaOldestFirst = append(w.commits, contentCommit), true },
			gitea: &forgeWant{1, []string{giteaRerun + shaContent}}},
		{name: "record then merge of main", world: func(w *gateWorld) { w.commits = append(w.commits, mergeMain) },
			github: &forgeWant{0, []string{"reviewing agent: codex (record at " + shaA + ")"}},
			gitea:  &forgeWant{1, []string{giteaRerun + shaMerge + ", and on Gitea"}}},
		{name: "merge of main touching a changed file", world: func(w *gateWorld) {
			touching := mergeMain
			touching.files = []string{"scripts/x.sh"}
			w.commits = append(w.commits, touching)
		}, github: &forgeWant{1, []string{"review record at " + shaA + " is older than content commit " + shaMerge}},
			gitea: &forgeWant{1, []string{giteaRerun + shaMerge}}},
		{name: "record naming two commits", world: func(w *gateWorld) {
			w.commits = append(w.commits, contentCommit)
			w.comments = []string{"reviewed-by: codex\nReviewed at " + shaA + " and reviewed at " + shaContent + "."}
		}, both: &forgeWant{1, []string{"no review record comment names reviewed-by: codex"}}},
		{name: "re-review at the new head", world: func(w *gateWorld) {
			w.commits = append(w.commits, contentCommit)
			w.comments = append(w.comments, "reviewed-by: codex\nRe-reviewed at "+shaContent[:7]+"\nVerdict: APPROVE")
		}, both: &forgeWant{0, []string{"reviewing agent: codex (record at " + shaContent + ")"}}},
		{name: "draft", world: func(w *gateWorld) { w.draft, w.body = true, "implemented-by: claude (opus)\n" },
			both: &forgeWant{0, []string{"no line starts reviewed-by:", "is a draft: a review finding does not fail a draft (1 above)"}}},
		{name: "bot author", world: func(w *gateWorld) { w.bot, w.body = true, "reviewed-by: codex (gpt-6)\n" },
			both: &forgeWant{0, []string{"reviewing agent: codex"}}},
		{name: "open flake closed by the pull request", world: func(w *gateWorld) {
			w.issues = []worldIssue{{number: 40, flake: true, closedBy: []int{gatePR}}, {number: 41}}
		}, both: &forgeWant{0, []string{"closing #40 exempts pull request #12 (" + issueURL(40) + ")", "::warning::pull request #12 is exempt from the known-flake check by closing #40;"}}},
		{name: "open flake closed by another pull request", world: func(w *gateWorld) {
			w.issues = []worldIssue{{number: 40, flake: true, closedBy: []int{7}, neutered: []int{gatePR}}}
		}, both: &forgeWant{1, []string{"FAIL: known flakes are open and not closed by this pull request:", "  #40 " + issueURL(40) + " flaky"}}},
		{name: "closing reference past the first timeline page", world: func(w *gateWorld) {
			w.issues = []worldIssue{{number: 40, flake: true, closedBy: []int{gatePR}, noise: 60}}
		}, both: &forgeWant{0, []string{"closing #40 exempts pull request #12"}}},
		{name: "a labelled pull request is not a flake", world: func(w *gateWorld) {
			w.issues = []worldIssue{{number: 50, flake: true, pr: true}}
		}, both: &forgeWant{0, []string{"no open class:flake issue", "merge-check: ok"}}},
		{name: "file total disagrees with the files read", world: func(w *gateWorld) { w.giteaTotalSkew = map[string]int{fmt.Sprintf("/pulls/%d/files", gatePR): 1} },
			gitea: &forgeWant{2, []string{"unavailable: the read of the files of pull request #12 is incomplete: it has 2 of 3 entries"}}},
		{name: "null issue list", world: func(w *gateWorld) {
			w.ghNull = []string{"issue list --label class:flake --state open --json number,url,title --limit 100"}
		}, github: &forgeWant{2, []string{"unavailable: the read of open class:flake issues returned something other than what was asked for"}}},
		{name: "null label list", world: func(w *gateWorld) { w.ghNull = []string{"label list --json name --limit 100"} },
			github: &forgeWant{2, []string{"unavailable: the read of labels returned something other than what was asked for"}}},
		{name: "null rules", world: func(w *gateWorld) { w.ghNull = []string{"api repos/{owner}/{repo}/rules/branches/main"} },
			github: &forgeWant{2, []string{"unavailable: the read of the rules in force on main returned something other than what was asked for"}}},
		{name: "label missing", world: func(w *gateWorld) {
			w.labels = []string{"bug"}
			w.issues = []worldIssue{{number: 41}}
		}, both: &forgeWant{1, []string{"FAIL: the label class:flake is missing"}}},
		{name: "up-to-date setting off", world: func(w *gateWorld) { w.strict = false },
			github: &forgeWant{1, []string{"strict_required_status_checks_policy=false", "a head behind main could merge", "merge-check: FAILED"}},
			gitea:  &forgeWant{1, []string{"block_on_outdated_branch=false; a head behind main could merge", "merge-check: FAILED"}}},
		{name: "check not required", world: func(w *gateWorld) { w.required = false },
			both: &forgeWant{1, []string{"no rule in force on main requires", "merge-check: FAILED"}}},
		{name: "context listed but status checks off", world: func(w *gateWorld) { w.giteaChecksOff = true },
			gitea: &forgeWant{1, []string{"enable_status_check=false", "merge-check: FAILED"}}},
		{name: "glob context is not the exact check", world: func(w *gateWorld) { w.giteaContexts = []string{"CI / *"} },
			gitea: &forgeWant{1, []string{"no rule in force on main requires " + giteaRequired, "merge-check: FAILED"}}},
		// The job token cannot read the rule (owner ruling on #143: no extra
		// secrets), so the reader-level fields decide and strict is unverified.
		{name: "branch protection unreadable", world: func(w *gateWorld) { w.giteaStatus = map[string]int{"/branch_protections/main": http.StatusForbidden} },
			gitea: &forgeWant{0, []string{"block_on_outdated_branch=unverified", "rule strict=unverified (branch protection needs a repo-admin token",
				"\n::warning::the up-to-date rule is unverified", "merge-check: ok"}}},
		// Live Gitea 28.2: a non-admin token reads protected=true with an empty
		// effective_branch_protection_name, so the rule cannot even be named.
		{name: "rule name hidden from a reader", world: func(w *gateWorld) { w.giteaRuleHidden = true },
			gitea: &forgeWant{0, []string{"name=(hidden from this token)", "rule strict=unverified", "\n::warning::the up-to-date rule is unverified", "merge-check: ok"}}},
		{name: "rule name hidden and the check not required", world: func(w *gateWorld) {
			w.giteaRuleHidden = true
			w.required = false
		},
			gitea: &forgeWant{1, []string{"no rule in force on main requires " + giteaRequired, "merge-check: FAILED"}}},
		{name: "branch protection not found to a reader", world: func(w *gateWorld) { w.giteaStatus = map[string]int{"/branch_protections/main": http.StatusNotFound} },
			gitea: &forgeWant{0, []string{"rule strict=unverified", "merge-check: ok"}}},
		{name: "branch protection unreadable and the check not required", world: func(w *gateWorld) {
			w.required = false
			w.giteaStatus = map[string]int{"/branch_protections/main": http.StatusForbidden}
		},
			gitea: &forgeWant{1, []string{"no rule in force on main requires " + giteaRequired, "merge-check: FAILED"}}},
		{name: "branch protection unreadable and status checks off", world: func(w *gateWorld) {
			w.giteaChecksOff = true
			w.giteaStatus = map[string]int{"/branch_protections/main": http.StatusForbidden}
		},
			gitea: &forgeWant{1, []string{"enable_status_check=false", "merge-check: FAILED"}}},
		{name: "branch protection unreadable and the branch unprotected", world: func(w *gateWorld) {
			w.giteaUnprotected = true
			w.giteaStatus = map[string]int{"/branch_protections/main": http.StatusForbidden}
		},
			gitea: &forgeWant{1, []string{"no branch protection in force on main requires", "merge-check: FAILED"}}},
		{name: "branch protection read fails", world: func(w *gateWorld) { w.giteaStatus = map[string]int{"/branch_protections/main": http.StatusBadGateway} },
			gitea: &forgeWant{2, []string{"unavailable: the read of branch protection main failed", "HTTP 502"}}},
		{name: "branch protection unauthorized", world: func(w *gateWorld) {
			w.giteaStatus = map[string]int{"/branch_protections/main": http.StatusUnauthorized}
		},
			gitea: &forgeWant{2, []string{"unavailable: the read of branch protection main failed", "HTTP 401"}}},
		{name: "comments unreadable", world: func(w *gateWorld) {
			w.giteaStatus = map[string]int{fmt.Sprintf("/issues/%d/comments", gatePR): http.StatusBadGateway}
		},
			gitea: &forgeWant{2, []string{"unavailable: the read of the comments of pull request #12 failed"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := healthyWorld()
			tc.world(&w)
			for _, run := range []struct {
				forge string
				want  *forgeWant
				run   func(*testing.T, gateWorld) (int, string)
			}{{"github", cmp.Or(tc.github, tc.both), runGitHub}, {"gitea", cmp.Or(tc.gitea, tc.both), runGitea}} {
				if run.want == nil {
					continue
				}
				code, out := run.run(t, w)
				if code != run.want.code {
					t.Errorf("%s: exit %d, want %d\n%s", run.forge, code, run.want.code, out)
				}
				for _, l := range run.want.lines {
					if !strings.Contains(out, l) {
						t.Errorf("%s: output lacks %q\n%s", run.forge, l, out)
					}
				}
				// The run line names the forge's Actions; an unverified
				// up-to-date setting always comes with its annotation.
				actions := map[string]string{"github": "(outside GitHub Actions;", "gitea": "(outside Gitea Actions;"}[run.forge]
				if !strings.Contains(out, actions) {
					t.Errorf("%s: run line lacks %q\n%s", run.forge, actions, out)
				}
				if strings.Contains(out, "strict=unverified") != strings.Contains(out, "\n::warning::the up-to-date rule is unverified") {
					t.Errorf("%s: strict=unverified and its ::warning:: line must appear together\n%s", run.forge, out)
				}
			}
		})
	}
}

// spec: openspec/changes/merge-check-verdict-201/specs/harness/spec.md#Requirement: Merge check requires an approving review verdict
func TestGateVerdict(t *testing.T) {
	// #201: the latest record (the last record designating one commit) must
	// carry exactly one unfenced Verdict: APPROVE line.
	contentCommit := worldCommit{sha: shaContent, message: "fix: adjust script", parents: []string{shaA}, files: []string{"scripts/x.sh"}}
	record := func(sha, rest string) string {
		return "reviewed-by: codex (gpt-6, high)\nReviewed at " + sha[:7] + "\n" + rest
	}
	const fix = "the line to write: post a new review record from codex with Verdict: APPROVE naming the current head commit"
	finding := func(what string) []string {
		return []string{"review finding: the latest review record (at " + shaA + ") " + what, fix, "merge-check: FAILED"}
	}
	passAt := func(sha string) []string {
		return []string{"reviewing agent: codex (record at " + sha + ")", "merge-check: ok"}
	}
	for _, tc := range []struct {
		name    string
		world   func(w *gateWorld)
		code    int
		want    []string
		without []string
	}{
		{name: "approving record", world: func(w *gateWorld) { w.comments = []string{record(shaA, "Verdict: APPROVE\nno findings")} },
			code: 0, want: passAt(shaA)},
		{name: "approval in lower case with spaces", world: func(w *gateWorld) { w.comments = []string{record(shaA, "  verdict:   approve  ")} },
			code: 0, want: passAt(shaA)},
		{name: "rejecting record", world: func(w *gateWorld) { w.comments = []string{record(shaA, "Verdict: REQUEST CHANGES\nF1 high")} },
			code: 1, want: finding("says Verdict: REQUEST CHANGES")},
		{name: "rejection then a later approval at the new head", world: func(w *gateWorld) {
			w.commits = append(w.commits, contentCommit)
			w.comments = []string{record(shaA, "Verdict: REQUEST CHANGES"), record(shaContent, "Verdict: APPROVE")}
		}, code: 0, want: passAt(shaContent)},
		{name: "approval then a later rejection", world: func(w *gateWorld) {
			w.comments = []string{record(shaA, "Verdict: APPROVE"), record(shaA, "Verdict: REQUEST CHANGES")}
		}, code: 1, want: finding("says Verdict: REQUEST CHANGES")},
		{name: "missing verdict", world: func(w *gateWorld) { w.comments = []string{record(shaA, "looks fine")} },
			code: 1, want: finding("has no Verdict: line")},
		{name: "two verdict lines", world: func(w *gateWorld) { w.comments = []string{record(shaA, "Verdict: APPROVE\nVerdict: APPROVE")} },
			code: 1, want: finding("has 2 Verdict: lines")},
		{name: "approval only inside a fence", world: func(w *gateWorld) {
			w.comments = []string{record(shaA, "```text\nVerdict: APPROVE\n```")}
		}, code: 1, want: finding("has no Verdict: line")},
		{name: "fenced verdict beside an approval", world: func(w *gateWorld) {
			w.comments = []string{record(shaA, "Verdict: APPROVE\n~~~\nVerdict: REQUEST CHANGES\n~~~")}
		}, code: 0, want: passAt(shaA)},
		{name: "fenced record does not supersede a rejection", world: func(w *gateWorld) {
			w.comments = []string{record(shaA, "Verdict: REQUEST CHANGES"), "```\n" + record(shaA, "Verdict: APPROVE") + "\n```"}
		}, code: 1, want: finding("says Verdict: REQUEST CHANGES")},
		{name: "record only inside a fence", world: func(w *gateWorld) { w.comments = []string{"```\n" + record(shaA, "Verdict: APPROVE") + "\n```"} },
			code: 1, want: []string{"no review record comment names reviewed-by: codex"}, without: []string{"latest review record"}},
		{name: "fenced reviewed-by of another agent", world: func(w *gateWorld) {
			w.comments = []string{record(shaA, "Verdict: APPROVE\n```\nreviewed-by: claude (opus)\n```")}
		}, code: 0, want: passAt(shaA)},
		{name: "fenced Reviewed at of an older commit", world: func(w *gateWorld) {
			w.commits = append(w.commits, contentCommit)
			w.comments = []string{record(shaContent, "Verdict: APPROVE\n```\nReviewed at "+shaA[:7]+"\n```")}
		}, code: 0, want: passAt(shaContent)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := healthyWorld()
			tc.world(&w)
			for forge, run := range map[string]func(*testing.T, gateWorld) (int, string){"github": runGitHub, "gitea": runGitea} {
				code, out := run(t, w)
				if code != tc.code {
					t.Errorf("%s: exit %d, want %d\n%s", forge, code, tc.code, out)
				}
				for _, l := range tc.want {
					if !strings.Contains(out, l) {
						t.Errorf("%s: output lacks %q\n%s", forge, l, out)
					}
				}
				for _, l := range tc.without {
					if strings.Contains(out, l) {
						t.Errorf("%s: output has %q\n%s", forge, l, out)
					}
				}
			}
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Merge check notes an unapproved OpenSpec change
func TestGateApprovalNote(t *testing.T) {
	const (
		fooTasks  = "openspec/changes/foo/tasks.md"
		fooProp   = "openspec/changes/foo/proposal.md"
		note      = "merge-check: note: OpenSpec change foo has no Approved-by; the spec was not human-reviewed"
		anyNote   = "merge-check: note: OpenSpec change"
		archived  = "openspec/changes/archive/2026-10-26-bar/proposal.md"
		unchecked = "merge-check: note: " + fooProp + " was not read, so its Approved-by is unchecked"
	)
	for _, tc := range []struct {
		name    string
		world   func(w *gateWorld)
		forges  []string
		want    []string
		without []string
	}{
		{name: "unapproved change on a code pull request", world: func(w *gateWorld) {
			w.files = []string{fooTasks, fooProp, "scripts/x.sh"}
			w.proposals = map[string]string{fooProp: "# p\n\nApproved-by:\n"}
		}, want: []string{note, "merge-check: ok"}},
		{name: "approved change", world: func(w *gateWorld) {
			w.files = []string{fooTasks, "scripts/x.sh"}
			w.proposals = map[string]string{fooProp: "# p\n\nApproved-by: Owner (2026-10-26)\n"}
		}, want: []string{"merge-check: ok"}, without: []string{anyNote, "merge-check: note:"}},
		{name: "header only inside a fence", world: func(w *gateWorld) {
			w.files = []string{fooTasks, "scripts/x.sh"}
			w.proposals = map[string]string{fooProp: "```\nApproved-by: Owner (2026-10-26)\n```\n"}
		}, want: []string{note}},
		{name: "archived change with no proposal", world: func(w *gateWorld) {
			w.files = []string{"openspec/changes/archive/2026-10-26-bar/tasks.md", "scripts/x.sh"}
			w.proposalsMissing = []string{archived}
		}, want: []string{"merge-check: note: OpenSpec change 2026-10-26-bar has no Approved-by", "merge-check: ok"}},
		{name: "two changes, each noted once", world: func(w *gateWorld) {
			w.files = []string{fooTasks, fooProp, "openspec/changes/zed/tasks.md", "scripts/x.sh"}
			w.proposals = map[string]string{fooProp: "Approved-by:\n", "openspec/changes/zed/proposal.md": "Approved-by:\n"}
		}, want: []string{note, "OpenSpec change zed has no Approved-by"}},
		{name: "files outside a change directory", world: func(w *gateWorld) {
			w.files = []string{"openspec/changes/README.md", "openspec/changes/archive/notes.md", "openspec/specs/harness/spec.md", "scripts/x.sh"}
		}, want: []string{"merge-check: ok"}, without: []string{"merge-check: note:"}},
		{name: "documents-only pull request", world: func(w *gateWorld) {
			w.body, w.files, w.comments = "", []string{fooProp}, nil
			w.proposals = map[string]string{fooProp: "Approved-by:\n"}
		}, want: []string{"is documents only"}, without: []string{anyNote}},
		{name: "read failure on GitHub", forges: []string{"github"}, world: func(w *gateWorld) {
			w.files = []string{fooTasks, "scripts/x.sh"}
		}, want: []string{unchecked, "merge-check: ok"}, without: []string{anyNote}},
		{name: "read failure on Gitea", forges: []string{"gitea"}, world: func(w *gateWorld) {
			w.files = []string{fooTasks, "scripts/x.sh"}
			w.giteaStatus = map[string]int{"/contents/" + fooProp: http.StatusBadGateway}
		}, want: []string{unchecked, "HTTP 502", "merge-check: ok"}, without: []string{anyNote}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := healthyWorld()
			tc.world(&w)
			runs := map[string]func(*testing.T, gateWorld) (int, string){"github": runGitHub, "gitea": runGitea}
			forges := tc.forges
			if forges == nil {
				forges = []string{"github", "gitea"}
			}
			for _, forge := range forges {
				code, out := runs[forge](t, w)
				if code != 0 {
					t.Errorf("%s: exit %d, want 0: a note never changes the verdict\n%s", forge, code, out)
				}
				for _, l := range tc.want {
					if !strings.Contains(out, l) {
						t.Errorf("%s: output lacks %q\n%s", forge, l, out)
					}
				}
				for _, l := range tc.without {
					if strings.Contains(out, l) {
						t.Errorf("%s: output has %q\n%s", forge, l, out)
					}
				}
				if strings.Count(out, note) > 1 {
					t.Errorf("%s: the note for foo appears more than once\n%s", forge, out)
				}
			}
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Merge gate and CI workflow enforcement
// spec: openspec/changes/merge-check-verdict-201/specs/harness/spec.md#Requirement: Merge check requires an approving review verdict
func TestGateRecordBoundaries(t *testing.T) {
	// The boundary checks that replace jq's lookaheads and Unicode \b in
	// scripts/merge-check.sh. Expectations were generated by that script's jq
	// is_record and designated_commit_indices (reviewer codex, commits A then
	// C); the test itself needs no jq.
	commits := []gateCommit{{SHA: shaA}, {SHA: shaContent}}
	for _, tc := range []struct {
		body   string
		record bool
		idxs   []int
	}{
		{"Reviewed at 99d19a4.", false, []int{0}},
		{"Reviewed at 99d19a4x", false, nil},
		{"Reviewed at 99d19a4g", false, nil},
		{"Reviewed at 99d19a4G", false, nil},
		{"Reviewed at `99d19a4`", false, []int{0}},
		{"prereviewed at 99d19a4", false, nil},
		{"éreviewed at 99d19a4", false, nil},
		{"_reviewed at 99d19a4", false, nil},
		{"Ⅰreviewed at 99d19a4", false, nil},
		{"²reviewed at 99d19a4", false, nil},
		{"-reviewed at 99d19a4", false, []int{0}},
		{"xre-reviewed at 99d19a4", false, []int{0}},
		{"Reviewed head commit 7413d0cg", false, nil},
		{"Reviewed head commit 7413d0c-", false, []int{1}},
		{"xreviewed head commit 7413d0c", false, nil},
		{"Reviewed base 99d19a4 → head 7413d0c", false, []int{1}},
		{"aReviewed base 99d19a4 → head 7413d0c", false, nil},
		{"Reviewed base 99d19a4 → head 7413d0cz", false, nil},
		{"Head: 7413d0c", false, []int{1}},
		{"Head: 7413d0cq", false, nil},
		{"x\nreviewed-commit: 99d19a4", false, []int{0}},
		{"x reviewed-commit: 99d19a4", false, nil},
		{"reviewed-by: codex", true, nil},
		{"reviewed-by: codexy", false, nil},
		{"reviewed-by: codex2", false, nil},
		{"reviewed-by: codex.", true, nil},
		{"reviewed-by: codexé", false, nil},
		{"reviewed-by:`codex`", true, nil},
		{"Reviewed-By: Codex", true, nil},
		{"### Independent Cross-Agent Review", true, nil},
		{"### Independent Cross-Agent Reviewer", false, nil},
		{"### Independent Cross-Agent Review_x", false, nil},
		{"### Independent Cross-Agent Review2", false, nil},
		{"### Independent Cross-Agent Reviewé", false, nil},
		{"### Independent Cross-Agent Review (codex)", true, nil},
		{"### Independent Cross-Agent Review (claude)", false, nil},
		{"  ### Independent Cross-Agent Review: codex", true, nil},
		{"### Independent Cross-Agent Review\nreviewed-by: claude", false, nil},
		{"reviewed-by: codex\n### Independent Cross-Agent Review (claude)", true, nil},
		// #201: lines inside ``` and ~~~ fences are not read.
		{"```\nreviewed-by: codex\n```", false, nil},
		{"~~~\n### Independent Cross-Agent Review\n~~~", false, nil},
		{"reviewed-by: codex\n```\nreviewed-by: claude\n```", true, nil},
		{"```\nReviewed at 99d19a4\n```", false, nil},
		{"Reviewed at 7413d0c\n~~~\nReviewed at 99d19a4\n~~~", false, []int{1}},
		{"reviewed-commit: 99d19a4\n```\nHead: 7413d0c\n```", false, []int{0}},
	} {
		if got := gateIsRecord(tc.body, "codex"); got != tc.record {
			t.Errorf("gateIsRecord(%q) = %t, want %t", tc.body, got, tc.record)
		}
		if got := gateDesignated(tc.body, commits); !slices.Equal(got, tc.idxs) {
			t.Errorf("gateDesignated(%q) = %v, want %v", tc.body, got, tc.idxs)
		}
	}
}
