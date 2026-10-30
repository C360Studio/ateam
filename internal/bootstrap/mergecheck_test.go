package bootstrap

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
)

const fakeGH = `#!/bin/sh
d="$FAKE_GH_DIR"
echo "$*" >> "$d/argv.log"
args=" $* "
need() { case "$args" in *" $1 "*) ;; *) echo "fake gh: '$*' lacks '$1'" >&2; exit 3 ;; esac; }
case "$1 $2" in
  "label list")
    key=labels; need "--json name"
    case "$args" in *" --search "*) exit 0 ;; esac ;;
  "issue list") key=issues; need "--label class:flake"; need "--state open"; need "--limit 100" ;;
  "pr view")
    key=pr
    [ "$3" = "$FAKE_GH_PR" ] || { echo "fake gh: pr view of '$3', want '$FAKE_GH_PR'" >&2; exit 3; }
    need "--json closingIssuesReferences" ;;
  *)
    case "$*" in
      "api repos/{owner}/{repo}/rules/branches/main") key=rules ;;
      "api repos/{owner}/{repo}/rulesets/"*) key=ruleset ;;
      "api repos/{owner}/{repo}/pulls/$FAKE_GH_PR") key=pull ;;
      "api --paginate repos/{owner}/{repo}/pulls/$FAKE_GH_PR/files?per_page=100") key=files ;;
      "api --paginate repos/{owner}/{repo}/pulls/$FAKE_GH_PR/commits?per_page=100") key=commits ;;
      "api --paginate repos/{owner}/{repo}/issues/$FAKE_GH_PR/comments?per_page=100") key=comments ;;
      "api --paginate repos/{owner}/{repo}/commits/"*"?per_page=100")
        s=${3%\?per_page=100}
        key="commit-${s##*/}" ;;
      *) echo "fake gh: unexpected call: $*" >&2; exit 3 ;;
    esac ;;
esac
if [ -e "$d/$key.fail" ]; then echo "HTTP 502: Bad Gateway (fake)" >&2; exit 1; fi
cat "$d/$key.json"
`

const (
	flakeRepo = "https://github.com/C360Studio/ateam/issues/"
	otherRepo = "https://github.com/C360Studio/semstreams/issues/"
)

func issueJSON(n int) string {
	return fmt.Sprintf(`{"number":%d,"url":"%s%d","title":"flaky test %d"}`, n, flakeRepo, n, n)
}

func refJSON(url string, n int) string {
	return fmt.Sprintf(`{"number":%d,"url":"%s%d","repository":{"name":"x","owner":{"login":"C360Studio"}}}`, n, url, n)
}

func listJSON(items ...string) string { return "[" + strings.Join(items, ",") + "]" }

type ghState struct {
	labels, issues, pr, rules, ruleset string
	pull, files, commits, comments     string
	commitDetails                      map[string]string
	fail                               []string
	// policy, when set, is written to .agents/review-policy.json.
	policy string
}

var (
	ateamBuild    sync.Once
	ateamBin      string
	ateamBuildErr error
)

// builtAteam builds cmd/ateam once per test binary, so merge-check calls the
// real review compare rather than a stub.
func builtAteam(t *testing.T) string {
	t.Helper()
	ateamBuild.Do(func() {
		dir, err := os.MkdirTemp("", "ateam-mergecheck-")
		if err != nil {
			ateamBuildErr = err
			return
		}
		ateamBin = filepath.Join(dir, "ateam")
		out, err := exec.Command("go", "build", "-o", ateamBin, filepath.Join(repoRoot(t), "cmd", "ateam")).CombinedOutput()
		if err != nil {
			ateamBuildErr = fmt.Errorf("go build ./cmd/ateam: %v\n%s", err, out)
		}
	})
	if ateamBuildErr != nil {
		t.Fatal(ateamBuildErr)
	}
	return ateamBin
}

const (
	requiredRule = `{"type":"required_status_checks","ruleset_id":24272345,"ruleset_source":"C360Studio/ateam",` +
		`"parameters":{"strict_required_status_checks_policy":true,"do_not_enforce_on_create":false,` +
		`"required_status_checks":[{"context":"Required","integration_id":15368}]}}`
	otherRules    = `{"type":"deletion","ruleset_id":24272345},{"type":"non_fast_forward","ruleset_id":24272345}`
	baseCommitSHA = "03121ae8066ec58a2ecc5a95c38c4447389eb0a3"
	headCommitSHA = "99d19a4cfa55e59c62dd49f79bd420e653675767"
)

func prCommitJSON(sha, message string, parents ...string) string {
	ps := make([]string, len(parents))
	for i, p := range parents {
		ps[i] = fmt.Sprintf(`{"sha":%q}`, p)
	}
	msg, err := json.Marshal(message)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf(`{"sha":%q,"commit":{"message":%s},"parents":%s}`, sha, msg, listJSON(ps...))
}

func commitDetailJSON(sha, message string, parents []string, files ...string) string {
	ps := make([]string, len(parents))
	for i, p := range parents {
		ps[i] = fmt.Sprintf(`{"sha":%q}`, p)
	}
	msg, err := json.Marshal(message)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf(`{"sha":%q,"commit":{"message":%s},"parents":%s,"files":%s}`,
		sha, msg, listJSON(ps...), filesJSON(files...))
}

func commentJSON(body string) string {
	b, err := json.Marshal(body)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf(`{"body":%s}`, b)
}

func healthyGH() ghState {
	return ghState{
		labels:   listJSON(`{"name":"bug"}`, `{"name":"class:flake"}`),
		issues:   listJSON(),
		pr:       `{"closingIssuesReferences":[]}`,
		rules:    listJSON(otherRules, requiredRule),
		ruleset:  `{"id":24272345,"name":"main","enforcement":"active"}`,
		pull:     pullJSON(false, "User", "", 1),
		files:    filesJSON("docs/a.md"),
		commits:  listJSON(prCommitJSON(headCommitSHA, "feat: change", baseCommitSHA)),
		comments: listJSON(),
	}
}

func pullJSON(draft bool, userType, body string, changedFiles int) string {
	return pullCommitsJSON(draft, userType, body, changedFiles, 1, headCommitSHA)
}

func pullCommitsJSON(draft bool, userType, body string, changedFiles, commits int, headSHA string) string {
	b, err := json.Marshal(body)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf(`{"number":12,"state":"open","draft":%t,"user":{"login":"x","type":%q},"body":%s,"changed_files":%d,"commits":%d,"head":{"sha":%q},"base":{"ref":"main"}}`,
		draft, userType, b, changedFiles, commits, headSHA)
}

func filesJSON(names ...string) string {
	entries := make([]string, len(names))
	for i, n := range names {
		entries[i] = fmt.Sprintf(`{"filename":%q,"status":"modified","additions":1,"deletions":0}`, n)
	}
	return listJSON(entries...)
}

type mergeRun struct {
	out    string
	status int
	argv   string
}

func fakeBin(t *testing.T, tools map[string]string, extra ...string) []string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range tools {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	var env []string
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "PATH=") {
			kv = "PATH=" + dir + ":" + strings.TrimPrefix(kv, "PATH=")
		}
		env = append(env, kv)
	}
	return append(env, extra...)
}

func copyScript(t *testing.T, name string) string {
	t.Helper()
	src, err := os.ReadFile(filepath.Join(repoRoot(t), "scripts", name))
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "scripts", name), src, 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

var forgeEnv = []string{"GITHUB_ACTIONS", "GITHUB_EVENT_NAME", "GITHUB_SERVER_URL", "GITHUB_REPOSITORY",
	"ATEAM_FORGE", "GITEA_URL", "GITEA_TOKEN", "ATEAM_GITEA_TOKEN"}

func runMergeCheck(t *testing.T, st ghState, pr string, actionsEnv ...string) mergeRun {
	t.Helper()
	root := copyScript(t, "merge-check.sh")
	dir := t.TempDir()
	for key, body := range map[string]string{
		"labels":   st.labels,
		"issues":   st.issues,
		"pr":       st.pr,
		"rules":    st.rules,
		"ruleset":  st.ruleset,
		"pull":     st.pull,
		"files":    st.files,
		"commits":  st.commits,
		"comments": st.comments,
	} {
		if err := os.WriteFile(filepath.Join(dir, key+".json"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for sha, body := range st.commitDetails {
		if err := os.WriteFile(filepath.Join(dir, "commit-"+sha+".json"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, key := range st.fail {
		if err := os.WriteFile(filepath.Join(dir, key+".fail"), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	tools := map[string]string{"gh": fakeGH}
	extra := []string{"FAKE_GH_DIR=" + dir, "FAKE_GH_PR=" + pr}
	if st.policy != "" {
		if err := os.MkdirAll(filepath.Join(root, ".agents"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, ".agents", "review-policy.json"), []byte(st.policy), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	extra = append(extra, "ATEAM="+builtAteam(t))
	var env []string
	for _, kv := range fakeBin(t, tools, extra...) {
		// The forge comes from these too, and a Gitea runner sets
		// GITHUB_SERVER_URL to its own host: the fixtures are GitHub's.
		if slices.ContainsFunc(forgeEnv, func(k string) bool { return strings.HasPrefix(kv, k+"=") }) {
			continue
		}
		env = append(env, kv)
	}
	args := []string{filepath.Join(root, "scripts", "merge-check.sh")}
	if pr != "" {
		args = append(args, pr)
	}
	cmd := exec.Command("bash", args...)
	cmd.Dir = root
	cmd.Env = append(env, actionsEnv...)
	out, err := cmd.CombinedOutput()
	r := mergeRun{out: string(out)}
	var exit *exec.ExitError
	switch {
	case errors.As(err, &exit):
		r.status = exit.ExitCode()
	case err != nil:
		t.Fatalf("run merge-check.sh: %v", err)
	}
	argv, _ := os.ReadFile(filepath.Join(dir, "argv.log"))
	r.argv = string(argv)
	return r
}

func (r mergeRun) requirePass(t *testing.T, fragments ...string) {
	t.Helper()
	if r.status != 0 {
		t.Fatalf("exit %d, want 0\n%s", r.status, r.out)
	}
	r.requireOutput(t, fragments...)
}

func (r mergeRun) requireFail(t *testing.T, fragments ...string) {
	t.Helper()
	if r.status == 0 {
		t.Fatalf("exit 0, want non-zero\n%s", r.out)
	}
	r.requireOutput(t, fragments...)
}

func (r mergeRun) requireOutput(t *testing.T, fragments ...string) {
	t.Helper()
	for _, f := range fragments {
		if !strings.Contains(r.out, f) {
			t.Errorf("output does not contain %q:\n%s", f, r.out)
		}
	}
}

func (r mergeRun) requireReadPullAndFiles(t *testing.T) {
	t.Helper()
	for _, call := range []string{
		"api repos/{owner}/{repo}/pulls/12\n",
		"api --paginate repos/{owner}/{repo}/pulls/12/files?per_page=100\n",
	} {
		if !strings.Contains(r.argv, call) {
			t.Errorf("the run did not call gh %q:\n%s", strings.TrimSpace(call), r.argv)
		}
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Merge gate and CI workflow enforcement
func TestMergeCheckKnownFlake(t *testing.T) {
	open40 := healthyGH()
	open40.issues = listJSON(issueJSON(40))
	open40And52 := healthyGH()
	open40And52.issues = listJSON(issueJSON(40), issueJSON(52))
	closes := func(st ghState, refs ...string) ghState {
		st.pr = `{"closingIssuesReferences":` + listJSON(refs...) + `}`
		return st
	}

	t.Run("open flake and an unrelated pull request", func(t *testing.T) {
		runMergeCheck(t, closes(open40, refJSON(flakeRepo, 7)), "12").requireFail(t, "#40")
	})
	t.Run("pull request that closes the only open flake", func(t *testing.T) {
		r := runMergeCheck(t, closes(open40, refJSON(flakeRepo, 40)), "12")
		r.requirePass(t, "closing #40 exempts pull request #12", "::warning::")
		if !strings.Contains(r.out[strings.Index(r.out, "::warning::"):], "#40") {
			t.Errorf("the ::warning:: line does not name #40:\n%s", r.out)
		}
	})
	t.Run("two open flakes and a pull request that closes one", func(t *testing.T) {
		r := runMergeCheck(t, closes(open40And52, refJSON(flakeRepo, 40)), "12")
		r.requireFail(t, "#52")
		if strings.Contains(r.out, "\n  #40 ") {
			t.Errorf("#40 is closed by the pull request but reported as not closed:\n%s", r.out)
		}
	})
	t.Run("two open flakes and a pull request that closes both", func(t *testing.T) {
		runMergeCheck(t, closes(open40And52, refJSON(flakeRepo, 40), refJSON(flakeRepo, 52)), "12").
			requirePass(t, "closing #40 exempts", "closing #52 exempts")
	})
	t.Run("same number in another repository", func(t *testing.T) {
		runMergeCheck(t, closes(open40, refJSON(otherRepo, 40)), "12").requireFail(t, "#40")
	})
	t.Run("no open flake", func(t *testing.T) {
		r := runMergeCheck(t, healthyGH(), "12")
		r.requirePass(t, "no open class:flake issue")
		r.requireReadPullAndFiles(t)
	})
	t.Run("label missing", func(t *testing.T) {
		st := open40
		st.labels = listJSON(`{"name":"bug"}`)
		runMergeCheck(t, st, "12").requireFail(t, "label class:flake is missing")
	})
	t.Run("push run", func(t *testing.T) {
		r := runMergeCheck(t, open40, "", "GITHUB_ACTIONS=true", "GITHUB_EVENT_NAME=push")
		r.requirePass(t, "push run", "known-flake check does not apply")
		if strings.Contains(r.argv, "issue list") || strings.Contains(r.argv, "pr view") || strings.Contains(r.argv, "/pulls/") {
			t.Errorf("a push run read issues, a pull request or its files:\n%s", r.argv)
		}
		if !strings.Contains(r.argv, "rules/branches/main") {
			t.Errorf("a push run did not read the rules in force on main:\n%s", r.argv)
		}
	})
}

// spec: openspec/specs/harness/spec.md#Requirement: Merge gate and CI workflow enforcement
func TestMergeCheckUpToDateRule(t *testing.T) {
	t.Run("setting turned off in GitHub", func(t *testing.T) {
		st := healthyGH()
		st.rules = listJSON(otherRules, strings.Replace(requiredRule, `"strict_required_status_checks_policy":true`, `"strict_required_status_checks_policy":false`, 1))
		runMergeCheck(t, st, "12").requireFail(t, "strict_required_status_checks_policy=false", "ruleset_id=24272345")
	})
	t.Run("no rule requires the check", func(t *testing.T) {
		st := healthyGH()
		st.rules = listJSON(otherRules, strings.Replace(requiredRule, `"context":"Required"`, `"context":"Verify"`, 1))
		runMergeCheck(t, st, "12").requireFail(t, "no rule in force on main requires Required")
	})
	t.Run("ruleset not active", func(t *testing.T) {
		for _, enforcement := range []string{"evaluate", "disabled"} {
			st := healthyGH()
			st.ruleset = `{"id":24272345,"name":"main","enforcement":"` + enforcement + `"}`
			r := runMergeCheck(t, st, "12")
			r.requireFail(t, "enforcement="+enforcement)
			if !strings.Contains(r.argv, "rulesets/24272345") {
				t.Errorf("did not read the ruleset the rule names:\n%s", r.argv)
			}
		}
	})
}

func reviewGH(draft bool, author, body string, files ...string) ghState {
	st := healthyGH()
	st.pull = pullJSON(draft, author, body, len(files))
	st.files = filesJSON(files...)
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "reviewed-by:") {
			st.comments = listJSON(commentJSON(line + "\nImplementation reviewed at " + headCommitSHA[:7] + "\nVerdict: APPROVE"))
			break
		}
	}
	return st
}

const codeFilePrefix = "merge-check: code file: "

func (r mergeRun) requireCodeNames(t *testing.T, want ...string) {
	t.Helper()
	var got []string
	for _, line := range strings.Split(r.out, "\n") {
		if name, ok := strings.CutPrefix(line, codeFilePrefix); ok {
			got = append(got, name)
		}
	}
	slices.Sort(got)
	want = slices.Clone(want)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("code names printed %q, want %q\n%s", got, want, r.out)
	}
}

func (r mergeRun) requireNotInOutput(t *testing.T, fragments ...string) {
	t.Helper()
	for _, f := range fragments {
		if strings.Contains(r.out, f) {
			t.Errorf("output contains %q:\n%s", f, r.out)
		}
	}
}

// tiersPolicyJSON is the ADR 0002 table, with cross_provider as given.
func tiersPolicyJSON(crossProvider bool) string {
	return fmt.Sprintf(`{"version":1,
"model_tiers":{"frontier":["claude:opus","claude:fable","codex:gpt-6-astra"],"workhorse":["claude:sonnet","gemini:gemini-4-argon"],"light":["claude:haiku"]},
"effort_bands":{"claude":{"low":["low"],"mid":["medium"],"high":["high","xhigh","max"]},
"codex":{"low":["low"],"mid":["medium"],"high":["high","xhigh","max"]},
"gemini":{"low":["low"],"mid":["medium"],"high":["high"]}},
"cross_provider":%t}`, crossProvider)
}

// spec: openspec/specs/harness/spec.md#Requirement: Merge gate and CI workflow enforcement
func TestMergeCheckReview(t *testing.T) {
	const implClaude = "Summary.\n\nimplemented-by: claude (opus)\n"

	t.Run("documents only", func(t *testing.T) {
		r := runMergeCheck(t, reviewGH(false, "User", "", "docs/a.md", "openspec/config.yaml"), "12")
		r.requirePass(t, "documents only")
		r.requireCodeNames(t)
	})
	t.Run("openspec markdown and json only", func(t *testing.T) {
		r := runMergeCheck(t, reviewGH(false, "User", "",
			"openspec/changes/mercury-runtime-qualification/tasks.md",
			"openspec/changes/mercury-runtime-qualification/offline/stack.json",
			"openspec/changes/mercury-runtime-qualification/offline/observations.json",
			"openspec/changes/mercury-runtime-qualification/offline/config.yml"), "12")
		r.requirePass(t, "documents only")
		r.requireCodeNames(t)
	})
	t.Run("openspec script and lockfile are code", func(t *testing.T) {
		r := runMergeCheck(t, reviewGH(false, "User", implClaude,
			"openspec/changes/mercury-runtime-qualification/tasks.md",
			"openspec/changes/mercury-runtime-qualification/offline/stack.json",
			"openspec/changes/mercury-runtime-qualification/offline/reproduce.py",
			"openspec/changes/mercury-runtime-qualification/offline/selected-wheels.lock"), "12")
		r.requireFail(t, "is a code pull request", "no line starts reviewed-by:")
		r.requireCodeNames(t,
			"openspec/changes/mercury-runtime-qualification/offline/reproduce.py",
			"openspec/changes/mercury-runtime-qualification/offline/selected-wheels.lock")
	})
	t.Run("one code file among documents", func(t *testing.T) {
		r := runMergeCheck(t, reviewGH(false, "User", implClaude, "docs/a.md", "scripts/x.sh"), "12")
		r.requireFail(t, "is a code pull request", "no line starts reviewed-by:")
		r.requireCodeNames(t, "scripts/x.sh")
	})
	t.Run("the other agent is named", func(t *testing.T) {
		for _, tc := range []struct{ body, reviewer string }{
			{implClaude + "reviewed-by: codex (gpt-5.4) — APPROVE\n", "codex"},
			{"Summary.\n\nimplemented-by: gemini (pro)\nreviewed-by: claude (opus)\n", "claude"},
		} {
			r := runMergeCheck(t, reviewGH(false, "User", tc.body, "docs/a.md", "scripts/x.sh"), "12")
			r.requirePass(t, "reviewing agent: "+tc.reviewer)
			r.requireNotInOutput(t, "a description edit reruns this check")
		}
	})
	t.Run("model named on both lines", func(t *testing.T) {
		for _, tc := range []struct{ body, says string }{
			{"implemented-by: claude\nreviewed-by: codex (gpt-6)\n", "the implemented-by: line must name the implementing model"},
			{"implemented-by: claude ( )\nreviewed-by: codex (gpt-6)\n", "the implemented-by: line must name the implementing model"},
			{implClaude + "reviewed-by: codex\n", "the reviewed-by: line must name the reviewing model"},
			{implClaude + "reviewed-by: codex ()\n", "the reviewed-by: line must name the reviewing model"},
			// The model is the group directly after the agent, not any parenthesised text on the line.
			{"implemented-by: (author) claude\nreviewed-by: codex (gpt-6)\n", "the implemented-by: line must name the implementing model"},
			{"implemented-by: claude () (implementation)\nreviewed-by: codex (gpt-6)\n", "the implemented-by: line must name the implementing model"},
			{implClaude + "reviewed-by: (independent) codex\n", "the reviewed-by: line must name the reviewing model"},
			{implClaude + "reviewed-by: codex () (independent review)\n", "the reviewed-by: line must name the reviewing model"},
		} {
			runMergeCheck(t, reviewGH(false, "User", tc.body, "scripts/x.sh"), "12").requireFail(t, tc.says)
		}
		for _, body := range []string{
			"implemented-by: claude (sonnet, opus)\nreviewed-by: codex (gpt-6)\n",
			"implemented-by: `Claude`(opus)\nreviewed-by: codex (gpt-6) — APPROVE (no findings)\n",
		} {
			runMergeCheck(t, reviewGH(false, "User", body, "scripts/x.sh"), "12").requirePass(t, "reviewing agent: codex")
		}
	})
	t.Run("same agent implements and reviews", func(t *testing.T) {
		// Without a policy, or with cross_provider off, one agent may review its own (ADR 0002 rule 4).
		for _, policy := range []string{"", tiersPolicyJSON(false)} {
			st := reviewGH(false, "User", "implemented-by: claude (opus, high)\nreviewed-by: Claude (opus, high)\n", "scripts/x.sh")
			st.policy = policy
			runMergeCheck(t, st, "12").requirePass(t, "reviewing agent: claude")
		}
		for _, tc := range []struct{ body, says string }{
			{"implemented-by: claude (opus, high)\nreviewed-by: Claude (opus, high)\n", "another agent (codex or gemini) reviews"},
			{"implemented-by: gemini (gemini-4-argon, high)\nreviewed-by: Gemini (gemini-4-argon, high)\n", "another agent (claude or codex) reviews"},
		} {
			st := reviewGH(false, "User", tc.body, "scripts/x.sh")
			st.policy = tiersPolicyJSON(true)
			runMergeCheck(t, st, "12").requireFail(t, tc.says)
		}
	})
	t.Run("reviewer strength against the policy", func(t *testing.T) {
		for _, tc := range []struct{ body, says string }{
			{"implemented-by: claude (opus, high)\nreviewed-by: codex (GPT-6 Astra, xhigh)\n", "strength: pass"},
			{"implemented-by: claude (sonnet, medium, opus, high)\nreviewed-by: gemini (gemini-4-argon, high)\n", ""},
			// Owner ruling on #125: unranked passes only against unranked.
			{"implemented-by: claude (opus-9, high)\nreviewed-by: codex (gpt-7, low)\n", "strength: note: unranked: codex (gpt-7, low)"},
		} {
			st := reviewGH(false, "User", tc.body, "scripts/x.sh")
			st.policy = tiersPolicyJSON(false)
			r := runMergeCheck(t, st, "12")
			if tc.says == "" {
				r.requireFail(t, "reviewer gemini (gemini-4-argon, high) is workhorse tier, weaker than implementer claude (opus, high) at frontier tier")
				continue
			}
			r.requirePass(t, tc.says, "reviewing agent: codex")
		}
		for _, tc := range []struct{ body, says string }{
			{"implemented-by: claude (opus, high)\nreviewed-by: claude (fable, medium)\n", "frontier tier at mid band, weaker than implementer claude (opus, high) at high band"},
			{"implemented-by: claude (opus, high)\nreviewed-by: codex (gpt-7, low)\n", "unranked: reviewer codex (gpt-7, low) is not in model_tiers"},
			{"implemented-by: claude (opus, unknown)\nreviewed-by: codex (GPT-6, unknown)\n", "names no effort"},
			{"implemented-by: claude (opus)\nreviewed-by: codex (gpt-6-astra, high)\n", "claude (opus) names no effort"},
			{"implemented-by: claude (opus, high)\nreviewed-by: codex (gpt-6-astra)\n", "codex (gpt-6-astra) names no effort"},
			// The #108 multi-model form without efforts reads opus as an effort; it must not pass unranked.
			{"implemented-by: claude (sonnet, opus)\nreviewed-by: claude (sonnet, low)\n", `claude (sonnet, opus) names model "opus" where the effort goes`},
			{"implemented-by: claude (,)\nreviewed-by: codex (,)\n", "the reviewed-by: group exactly one"},
			// From the Gemini review of e3a209f: these exited 2 (unavailable) instead of a finding.
			{"implemented-by: claude (claude:opus, high)\nreviewed-by: codex (gpt-6-astra, high)\n", "with no colon in a model or effort"},
			{"implemented-by: claude (opus, high)\nreviewed-by: codex (gpt-6-astra, effort: high)\n", "with no colon in a model or effort"},
			{"implemented-by: claude (opus, high)\nreviewed-by: codex (gpt-6-astra, high, gpt-6-sol, high)\n", "the reviewed-by: group exactly one"},
		} {
			st := reviewGH(false, "User", tc.body, "scripts/x.sh")
			st.policy = tiersPolicyJSON(false)
			runMergeCheck(t, st, "12").requireFail(t, "review finding: ", tc.says)
		}
		// Without a policy, effort stays optional and compare is not called.
		r := runMergeCheck(t, reviewGH(false, "User", "implemented-by: claude (opus)\nreviewed-by: claude (haiku)\n", "scripts/x.sh"), "12")
		r.requirePass(t, "reviewing agent: claude")
		r.requireNotInOutput(t, "strength:")
		// A policy with no usable binary fails closed.
		st := reviewGH(false, "User", "implemented-by: claude (opus, high)\nreviewed-by: codex (gpt-6-astra, high)\n", "scripts/x.sh")
		st.policy = `{"version":2}`
		if r := runMergeCheck(t, st, "12"); r.status != 2 || !strings.Contains(r.out, "unavailable: ateam review compare exited 2") {
			t.Fatalf("invalid policy: exit %d, want 2 naming compare\n%s", r.status, r.out)
		}
	})
	t.Run("draft", func(t *testing.T) {
		runMergeCheck(t, reviewGH(true, "User", implClaude, "docs/a.md", "scripts/x.sh"), "12").
			requirePass(t, "no line starts reviewed-by:", "does not fail a draft")
	})

	const (
		implClaudeRevCodex = implClaude + "reviewed-by: codex (gpt-5.4)\n"
		mergeMainSHA       = "0209eb7f6ebaf44e2dd87c401d61a7a48763799b"
		docsCommitSHA      = "d7bb79680494e20514edf94b035352db7e72114b"
		contentCommitSHA   = "7413d0c05c25db0e9143f7a466a64954855aa979"
		mainTipSHA         = "dc854c651fbf82695f4d552badc36340e2f7c472"
	)

	t.Run("record at head", func(t *testing.T) {
		for _, comment := range []string{
			"**Review record:** reviewed-by: codex (OpenAI). Reviewed at `99d19a4`.\nVerdict: APPROVE",
			"### Independent Cross-Agent Review (`codex`)\n\nReviewed head commit `" + headCommitSHA + "`.\nVerdict: APPROVE",
			"Reviewed base `" + baseCommitSHA + "` → head `" + headCommitSHA + "`.\n\nreviewed-by: codex (OpenAI)\nVerdict: APPROVE",
		} {
			st := reviewGH(false, "User", implClaudeRevCodex, "docs/a.md", "scripts/x.sh")
			st.comments = listJSON(commentJSON(comment))
			r := runMergeCheck(t, st, "12")
			r.requirePass(t, "reviewing agent: codex (record at "+headCommitSHA+")")
		}
	})

	t.Run("record then merge-of-main", func(t *testing.T) {
		st := reviewGH(false, "User", implClaudeRevCodex, "docs/a.md", "scripts/x.sh")
		st.pull = pullCommitsJSON(false, "User", implClaudeRevCodex, 2, 2, mergeMainSHA)
		st.commits = listJSON(
			prCommitJSON(headCommitSHA, "feat: add script", baseCommitSHA),
			prCommitJSON(mergeMainSHA, "Merge remote-tracking branch 'origin/main' into claude/feat", headCommitSHA, mainTipSHA),
		)
		st.commitDetails = map[string]string{
			mergeMainSHA: commitDetailJSON(mergeMainSHA, "Merge remote-tracking branch 'origin/main' into claude/feat",
				[]string{headCommitSHA, mainTipSHA}, "internal/other/main_only.go", "README.md"),
		}
		r := runMergeCheck(t, st, "12")
		r.requirePass(t, "reviewing agent: codex (record at "+headCommitSHA+")")

		// A merge commit that touches one of the PR's changed files requires re-review.
		stTouch := st
		stTouch.commitDetails = map[string]string{
			mergeMainSHA: commitDetailJSON(mergeMainSHA, "Merge branch 'main' into claude/feat",
				[]string{headCommitSHA, mainTipSHA}, "scripts/x.sh"),
		}
		runMergeCheck(t, stTouch, "12").requireFail(t,
			"review record at "+headCommitSHA+" is older than content commit "+mergeMainSHA+"; request re-review")
	})

	t.Run("record then docs-only", func(t *testing.T) {
		st := reviewGH(false, "User", implClaudeRevCodex, "docs/a.md", "scripts/x.sh")
		st.pull = pullCommitsJSON(false, "User", implClaudeRevCodex, 2, 3, docsCommitSHA)
		st.commits = listJSON(
			prCommitJSON(headCommitSHA, "feat: add script", baseCommitSHA),
			prCommitJSON(mergeMainSHA, "Merge remote-tracking branch 'origin/main' into claude/feat", headCommitSHA, mainTipSHA),
			prCommitJSON(docsCommitSHA, "docs: clarify notes", mergeMainSHA),
		)
		st.commitDetails = map[string]string{
			mergeMainSHA: commitDetailJSON(mergeMainSHA, "Merge remote-tracking branch 'origin/main' into claude/feat",
				[]string{headCommitSHA, mainTipSHA}, "internal/other/main_only.go"),
			docsCommitSHA: commitDetailJSON(docsCommitSHA, "docs: clarify notes",
				[]string{mergeMainSHA}, "docs/a.md", "openspec/changes/foo/tasks.md", "openspec/changes/foo/offline/stack.json"),
		}
		r := runMergeCheck(t, st, "12")
		r.requirePass(t, "reviewing agent: codex (record at "+headCommitSHA+")")
	})

	t.Run("record then merge-of-main with a file list over the Linux argv limit", func(t *testing.T) {
		// GitHub's files payload carries each file's patch; a 200 KiB list must not be
		// handed to jq as one argv string (Linux MAX_ARG_STRLEN is 128 KiB, #122).
		st := reviewGH(false, "User", implClaudeRevCodex, "docs/a.md", "scripts/x.sh")
		st.files = strings.Replace(st.files, `"filename":"scripts/x.sh",`,
			`"filename":"scripts/x.sh","patch":"`+strings.Repeat("+x", 100<<10)+`",`, 1)
		if len(st.files) <= 128<<10 {
			t.Fatalf("files payload is %d bytes; the case needs more than 128 KiB", len(st.files))
		}
		st.pull = pullCommitsJSON(false, "User", implClaudeRevCodex, 2, 2, mergeMainSHA)
		st.commits = listJSON(
			prCommitJSON(headCommitSHA, "feat: add script", baseCommitSHA),
			prCommitJSON(mergeMainSHA, "Merge branch 'main' into claude/feat", headCommitSHA, mainTipSHA),
		)
		st.commitDetails = map[string]string{
			mergeMainSHA: commitDetailJSON(mergeMainSHA, "Merge branch 'main' into claude/feat",
				[]string{headCommitSHA, mainTipSHA}, "internal/other/main_only.go"),
		}
		runMergeCheck(t, st, "12").requirePass(t, "reviewing agent: codex (record at "+headCommitSHA+")")
	})

	t.Run("record with a commit list over the Linux argv limit", func(t *testing.T) {
		// Full commit objects run to about 4 KB each, so a branch with a few dozen
		// commits must not reach the record filter as one argv string (#122).
		st := reviewGH(false, "User", implClaudeRevCodex, "docs/a.md", "scripts/x.sh")
		const filler = 40
		commits := make([]string, 0, filler+1)
		parent := baseCommitSHA
		for i := 0; i < filler; i++ {
			sha := fmt.Sprintf("%040x", i+1)
			commits = append(commits, prCommitJSON(sha, "chore: filler "+strings.Repeat("x", 4<<10), parent))
			parent = sha
		}
		commits = append(commits, prCommitJSON(headCommitSHA, "feat: add script", parent))
		st.commits = listJSON(commits...)
		if len(st.commits) <= 128<<10 {
			t.Fatalf("commits payload is %d bytes; the case needs more than 128 KiB", len(st.commits))
		}
		st.pull = pullCommitsJSON(false, "User", implClaudeRevCodex, 2, filler+1, headCommitSHA)
		runMergeCheck(t, st, "12").requirePass(t, "reviewing agent: codex (record at "+headCommitSHA+")")
	})

	t.Run("record then content commit", func(t *testing.T) {
		st := reviewGH(false, "User", implClaudeRevCodex, "docs/a.md", "scripts/x.sh")
		st.pull = pullCommitsJSON(false, "User", implClaudeRevCodex, 2, 2, contentCommitSHA)
		st.commits = listJSON(
			prCommitJSON(headCommitSHA, "feat: add script", baseCommitSHA),
			prCommitJSON(contentCommitSHA, "fix: adjust script", headCommitSHA),
		)
		st.commitDetails = map[string]string{
			contentCommitSHA: commitDetailJSON(contentCommitSHA, "fix: adjust script",
				[]string{headCommitSHA}, "scripts/x.sh"),
		}
		runMergeCheck(t, st, "12").requireFail(t,
			"review record at "+headCommitSHA+" is older than content commit "+contentCommitSHA+"; request re-review")

		// A comment mentioning the unreviewed head SHA alongside the reviewed SHA must not advance the review boundary.
		for _, comment := range []string{
			"reviewed-by: codex\nReviewed at " + headCommitSHA + ".\nCommit " + contentCommitSHA + " remains unreviewed; request re-review.\nVerdict: APPROVE",
			"reviewed-by: codex\nReviewed at " + headCommitSHA + "; see https://github.com/C360Studio/ateam/blob/" + contentCommitSHA + "/scripts/x.sh\nVerdict: APPROVE",
			"reviewed-by: codex\nReviewed at " + headCommitSHA + " while looking at " + contentCommitSHA + " as unreviewed.\nVerdict: APPROVE",
		} {
			stUnreviewed := st
			stUnreviewed.comments = listJSON(commentJSON(comment))
			runMergeCheck(t, stUnreviewed, "12").requireFail(t,
				"review record at "+headCommitSHA+" is older than content commit "+contentCommitSHA+"; request re-review")
		}

		// Ambiguous multi-commit or undesignated records fail closed.
		for _, comment := range []string{
			"reviewed-by: codex\nReviewed at " + headCommitSHA + " and reviewed at " + contentCommitSHA + ".",
			"reviewed-by: codex\nReviewed at deadbeef and reviewed at " + contentCommitSHA + ".",
			"reviewed-by: codex\nCommits " + headCommitSHA + " and " + contentCommitSHA + " touched scripts/x.sh.",
		} {
			stAmbiguous := st
			stAmbiguous.comments = listJSON(commentJSON(comment))
			runMergeCheck(t, stAmbiguous, "12").requireFail(t,
				"no review record comment names reviewed-by: codex")
		}

		// A later review record at contentCommitSHA (including both full and short SHA of the same commit) resolves the finding.
		for _, reReviewComment := range []string{
			"reviewed-by: codex\nRe-reviewed at " + contentCommitSHA[:7] + "\nVerdict: APPROVE",
			"reviewed-by: codex\nHead: " + contentCommitSHA + "\nReviewed at " + contentCommitSHA[:7] + "\nVerdict: APPROVE",
		} {
			st.comments = listJSON(
				commentJSON("reviewed-by: codex\nReviewed at "+headCommitSHA[:7]+"\nVerdict: APPROVE"),
				commentJSON(reReviewComment),
			)
			runMergeCheck(t, st, "12").requirePass(t, "reviewing agent: codex (record at "+contentCommitSHA+")")
		}
	})

	t.Run("paginated commit files and 3000-file ceiling", func(t *testing.T) {
		docPage := make([]string, 300)
		otherCodePage := make([]string, 300)
		for i := range 300 {
			docPage[i] = fmt.Sprintf("docs/%03d.md", i)
			otherCodePage[i] = fmt.Sprintf("internal/other/%03d.go", i)
		}

		// Page 2 contains a code file in a post-review non-merge commit.
		st := reviewGH(false, "User", implClaudeRevCodex, "docs/a.md", "scripts/x.sh")
		st.pull = pullCommitsJSON(false, "User", implClaudeRevCodex, 2, 2, contentCommitSHA)
		st.commits = listJSON(
			prCommitJSON(headCommitSHA, "feat: add script", baseCommitSHA),
			prCommitJSON(contentCommitSHA, "docs: bulk update", headCommitSHA),
		)
		st.commitDetails = map[string]string{
			contentCommitSHA: commitDetailJSON(contentCommitSHA, "docs: bulk update", []string{headCommitSHA}, docPage...) + "\n" +
				commitDetailJSON(contentCommitSHA, "docs: bulk update", []string{headCommitSHA}, "scripts/x.sh"),
		}
		runMergeCheck(t, st, "12").requireFail(t,
			"review record at "+headCommitSHA+" is older than content commit "+contentCommitSHA+"; request re-review")

		// Page 2 contains a PR-changed file overlap in a post-review merge of main.
		stMerge := reviewGH(false, "User", implClaudeRevCodex, "docs/a.md", "scripts/x.sh")
		stMerge.pull = pullCommitsJSON(false, "User", implClaudeRevCodex, 2, 2, mergeMainSHA)
		stMerge.commits = listJSON(
			prCommitJSON(headCommitSHA, "feat: add script", baseCommitSHA),
			prCommitJSON(mergeMainSHA, "Merge remote-tracking branch 'origin/main' into claude/feat", headCommitSHA, mainTipSHA),
		)
		stMerge.commitDetails = map[string]string{
			mergeMainSHA: commitDetailJSON(mergeMainSHA, "Merge remote-tracking branch 'origin/main' into claude/feat",
				[]string{headCommitSHA, mainTipSHA}, otherCodePage...) + "\n" +
				commitDetailJSON(mergeMainSHA, "Merge remote-tracking branch 'origin/main' into claude/feat",
					[]string{headCommitSHA, mainTipSHA}, "scripts/x.sh"),
		}
		runMergeCheck(t, stMerge, "12").requireFail(t,
			"review record at "+headCommitSHA+" is older than content commit "+mergeMainSHA+"; request re-review")

		// A single 300-file page (truncated first page) or a 3,000-file ceiling response fails closed (exit 2).
		stTruncFirstPage := st
		stTruncFirstPage.commitDetails = map[string]string{
			contentCommitSHA: commitDetailJSON(contentCommitSHA, "docs: bulk update", []string{headCommitSHA}, docPage...),
		}
		rTrunc := runMergeCheck(t, stTruncFirstPage, "12")
		if rTrunc.status != 2 {
			t.Fatalf("truncated 300-file commit page exit %d, want 2\n%s", rTrunc.status, rTrunc.out)
		}
		rTrunc.requireOutput(t, "unavailable: the file list of commit "+contentCommitSHA)

		var tenDocPages []string
		for range 10 {
			tenDocPages = append(tenDocPages, commitDetailJSON(contentCommitSHA, "docs: bulk update", []string{headCommitSHA}, docPage...))
		}
		stCeiling := st
		stCeiling.commitDetails = map[string]string{
			contentCommitSHA: strings.Join(tenDocPages, "\n"),
		}
		rCeiling := runMergeCheck(t, stCeiling, "12")
		if rCeiling.status != 2 {
			t.Fatalf("3000-file ceiling commit exit %d, want 2\n%s", rCeiling.status, rCeiling.out)
		}
		rCeiling.requireOutput(t, "unavailable: the file list of commit "+contentCommitSHA)
	})

	t.Run("incomplete or capped PR commit history", func(t *testing.T) {
		var commits250 []string
		prev := baseCommitSHA
		for i := 1; i <= 250; i++ {
			sha := fmt.Sprintf("%040x", i)
			commits250 = append(commits250, prCommitJSON(sha, "feat: step", prev))
			prev = sha
		}
		sha250 := fmt.Sprintf("%040x", 250)
		sha251 := fmt.Sprintf("%040x", 251)

		// PR reports 251 commits and head #251, but pulls/12/commits stops at #250 with review at #250.
		stCapped := reviewGH(false, "User", implClaudeRevCodex, "docs/a.md", "scripts/x.sh")
		stCapped.pull = pullCommitsJSON(false, "User", implClaudeRevCodex, 2, 251, sha251)
		stCapped.commits = listJSON(commits250...)
		stCapped.comments = listJSON(commentJSON("reviewed-by: codex\nReviewed at " + sha250 + "\nVerdict: APPROVE"))
		rCapped := runMergeCheck(t, stCapped, "12")
		if rCapped.status != 2 {
			t.Fatalf("capped 250-commit history exit %d, want 2\n%s", rCapped.status, rCapped.out)
		}
		rCapped.requireOutput(t, "unavailable: the commit list of pull request #12 is incomplete or does not end at head: it has 250 of 251 commits")

		// Including #251 in the commit list fails the review gate at #251 (exit 1).
		stFull := stCapped
		stFull.commits = listJSON(append(commits250, prCommitJSON(sha251, "fix: unreviewed code change", sha250))...)
		stFull.commitDetails = map[string]string{
			sha251: commitDetailJSON(sha251, "fix: unreviewed code change", []string{sha250}, "scripts/x.sh"),
		}
		runMergeCheck(t, stFull, "12").requireFail(t,
			"review record at "+sha250+" is older than content commit "+sha251+"; request re-review")
	})

	t.Run("no record", func(t *testing.T) {
		for _, comments := range [][]string{
			nil,
			{commentJSON("LGTM at " + headCommitSHA)},
			{commentJSON("reviewed-by: gemini\nReviewed at " + headCommitSHA)},
			{commentJSON("reviewed-by: codex\nReviewed at deadbeef")},
		} {
			st := reviewGH(false, "User", implClaudeRevCodex, "docs/a.md", "scripts/x.sh")
			st.comments = listJSON(comments...)
			runMergeCheck(t, st, "12").requireFail(t, "no review record comment names reviewed-by: codex")
		}
	})
}

var (
	requiredNeeds         = []string{"verify"}
	mergeCheckPermissions = map[string]string{"contents": "read", "issues": "read", "pull-requests": "read"}
	pullRequestTypes      = []string{"opened", "synchronize", "reopened", "ready_for_review"}
)

const (
	verifyTimeoutMinutes = 15
	runnerLabel          = "ubuntu-24.04"
	// draftGuard skips a pull-request job on a draft and keeps every push run (#250).
	draftGuard = "github.event_name != 'pull_request' || !github.event.pull_request.draft"
)

// ciJobNames are the required-check names each ci.yml job reports when it runs.
var ciJobNames = map[string]string{"verify": "Verify", "required": "Required"}

// draftName is the job name the merge-gate spec requires: a skipped draft job
// must not carry its required-check name, or GitHub counts the skip as passing (#250).
func draftName(name string) string {
	return fmt.Sprintf("${{ github.event.pull_request.draft && '%s (draft, skipped)' || '%s' }}", name, name)
}

func draftViolations(wf ciWorkflow) []string {
	var out []string
	for _, id := range slices.Sorted(maps.Keys(ciJobNames)) {
		job, ok := wf.Jobs[id]
		if !ok {
			continue
		}
		want := draftName(ciJobNames[id])
		if got := strings.TrimSpace(fmt.Sprint(job.Name)); got != want {
			out = append(out, fmt.Sprintf("ci.yml: job %s is named %v; the merge-gate spec requires name: %s so a skipped draft job leaves no %s check run at the head", id, job.Name, want, ciJobNames[id]))
		}
		if id == "required" {
			continue // requiredStepViolations checks its always() form
		}
		if got := strings.TrimSpace(fmt.Sprint(job.If)); got != draftGuard {
			out = append(out, fmt.Sprintf("ci.yml: job %s runs under if: %v; the merge-gate spec requires if: %s so drafts skip it and push to main still runs it", id, job.If, draftGuard))
		}
	}
	return out
}

// spec: openspec/specs/harness/spec.md#Requirement: Merge gate and CI workflow enforcement
func TestCIWorkflowPinned(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRoot(t), ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if v := ciWorkflowViolations(t, data); len(v) > 0 {
		t.Fatalf("ci.yml violations:\n- %s", strings.Join(v, "\n- "))
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Merge gate and CI workflow enforcement
func TestCIWorkflowPinnedSensitivity(t *testing.T) {
	const good = `on:
  push:
    branches: [main]
  pull_request:
    types: [opened, synchronize, reopened, ready_for_review]
concurrency:
  group: ci-${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: ${{ github.event_name == 'pull_request' && github.event.action != 'ready_for_review' }}
permissions:
  contents: read
jobs:
  verify:
    name: ${{ github.event.pull_request.draft && 'Verify (draft, skipped)' || 'Verify' }}
    if: github.event_name != 'pull_request' || !github.event.pull_request.draft
    runs-on: ubuntu-24.04
    timeout-minutes: 15
    steps: [{run: task verify}]
  required:
    name: ${{ github.event.pull_request.draft && 'Required (draft, skipped)' || 'Required' }}
    runs-on: ubuntu-24.04
    needs: [verify]
    if: always() && (github.event_name != 'pull_request' || !github.event.pull_request.draft)
    steps:
      - name: Require every needed job to succeed
        env:
          RESULTS: ${{ join(needs.*.result, ' ') }}
        run: |
          echo "needed job results: ${RESULTS:-<none>}"
          if [ -z "$RESULTS" ]; then echo "no needed jobs reported"; exit 1; fi
          for r in $RESULTS; do
            if [ "$r" != "success" ]; then echo "required check did not succeed: $r"; exit 1; fi
          done
`
	if v := ciWorkflowViolations(t, []byte(good)); len(v) > 0 {
		t.Fatalf("clean fixture violations:\n- %s", strings.Join(v, "\n- "))
	}

	plant := func(t *testing.T, old, repl string) []byte {
		t.Helper()
		if !strings.Contains(good, old) {
			t.Fatalf("fixture lacks %q", old)
		}
		return []byte(strings.Replace(good, old, repl, 1))
	}
	for _, tc := range []struct {
		name, old, repl string
		wants           []string
	}{
		{"required needs nothing", "needs: [verify]", "needs: []",
			[]string{"job required", "verify is missing"}},
		{"merge-check back in ci.yml", "  required:\n", "  merge-check: {runs-on: ubuntu-24.04, steps: [{run: scripts/merge-check.sh}]}\n  required:\n",
			[]string{"job merge-check is in ci.yml", "merge-check.yml"}},
		{"write permission on a job", "    timeout-minutes: 15\n", "    timeout-minutes: 15\n    permissions:\n      issues: write\n",
			[]string{"job verify", "issues: write"}},
		{"verify limit changed", "timeout-minutes: 15", "timeout-minutes: 30",
			[]string{"job verify", "30", "15 minutes"}},
		{"write-all on the workflow", "permissions:\n  contents: read\njobs:", "permissions: write-all\njobs:",
			[]string{"workflow", "write-all"}},
		{"required without if: always()", "    if: always() && (github.event_name != 'pull_request' || !github.event.pull_request.draft)\n", "",
			[]string{"job required", "if: always() &&"}},
		{"required skips drafts without always()", "    if: always() && (github.event_name != 'pull_request' || !github.event.pull_request.draft)\n", "    if: github.event_name != 'pull_request' || !github.event.pull_request.draft\n",
			[]string{"job required", "if: always() &&"}},
		{"verify runs on drafts", "    if: github.event_name != 'pull_request' || !github.event.pull_request.draft\n    runs-on: ubuntu-24.04\n    timeout-minutes: 15\n", "    runs-on: ubuntu-24.04\n    timeout-minutes: 15\n",
			[]string{"job verify", "drafts skip it"}},
		{"verify guard skips push to main", "    if: github.event_name != 'pull_request' || !github.event.pull_request.draft\n    runs-on: ubuntu-24.04\n    timeout-minutes: 15\n", "    if: ${{ !github.event.pull_request.draft }}\n    runs-on: ubuntu-24.04\n    timeout-minutes: 15\n",
			[]string{"job verify", "push to main"}},
		{"required keeps its name on a draft", "    name: ${{ github.event.pull_request.draft && 'Required (draft, skipped)' || 'Required' }}\n", "    name: Required\n",
			[]string{"job required", "no Required check run"}},
		{"step exits 0 for skipped", `if [ "$r" != "success" ]`, `if [ "$r" != "success" ] && [ "$r" != "skipped" ]`,
			[]string{"job required", "exits 0", "skipped"}},
		{"step reads verify alone", "${{ join(needs.*.result, ' ') }}", "${{ needs.verify.result }}",
			[]string{"job required", "does not take its results from needs.*.result"}},
		{"ready type missing", "types: [opened, synchronize, reopened, ready_for_review]", "types: [opened, synchronize, reopened]",
			[]string{"pull_request", "ready_for_review"}},
		{"ready_for_review run cancels the same head", "  cancel-in-progress: ${{ github.event_name == 'pull_request' && github.event.action != 'ready_for_review' }}\n", "  cancel-in-progress: ${{ github.event_name == 'pull_request' }}\n",
			[]string{"cancel-in-progress", "ready_for_review", "failed Required"}},
		{"cancel-in-progress unconditionally", "  cancel-in-progress: ${{ github.event_name == 'pull_request' && github.event.action != 'ready_for_review' }}\n", "  cancel-in-progress: true\n",
			[]string{"cancel-in-progress", "ready_for_review"}},
		{"a fourth job on the moving label", "jobs:\n", "jobs:\n  lint: {runs-on: ubuntu-latest, steps: [{run: echo}]}\n",
			[]string{"job lint", "ubuntu-latest", "ubuntu-24.04"}},
		{"verify on a list of labels", "    if: github.event_name != 'pull_request' || !github.event.pull_request.draft\n    runs-on: ubuntu-24.04\n    timeout-minutes: 15\n", "    if: github.event_name != 'pull_request' || !github.event.pull_request.draft\n    runs-on: [self-hosted, ubuntu-24.04]\n    timeout-minutes: 15\n",
			[]string{"job verify", "[self-hosted ubuntu-24.04]"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requireViolation(t, ciWorkflowViolations(t, plant(t, tc.old, tc.repl)), tc.wants...)
		})
	}
}

// mergeCheckTypes are the pull_request activity types merge-check.yml runs on:
// edited carries the reviewer's reviewed-by: description edit (#255).
var mergeCheckTypes = []string{"opened", "synchronize", "reopened", "ready_for_review", "edited"}

// mergeCheckIf and mergeCheckName skip the job on a draft (#250) and while the
// description has no reviewed-by: (#262), each under a name that is not a
// required context, so neither state posts a failed or a passing Merge check.
const (
	mergeCheckIf   = "${{ !github.event.pull_request.draft && contains(github.event.pull_request.body, 'reviewed-by:') }}"
	mergeCheckName = "${{ github.event.pull_request.draft && 'Merge check (draft, skipped)' || !contains(github.event.pull_request.body, 'reviewed-by:') && 'Merge check (awaiting review record, skipped)' || 'Merge check' }}"
)

// mergeCheckWorkflowViolations checks merge-check.yml against the merge-gate
// spec (#255): PR-state triggers, runs queued and never cancelled, skipped and
// renamed on a draft, read-only, built from the base commit, and told every
// context main must require.
func mergeCheckWorkflowViolations(data []byte, checks string) []string {
	wf, err := parseCIWorkflowYAML(data)
	if err != nil {
		return []string{fmt.Sprintf("merge-check.yml: %v", err)}
	}
	violations := pullRequestTriggerViolations(wf.On, mergeCheckTypes)
	if o, ok := wf.On.(map[string]any); !ok || len(o) != 1 {
		violations = append(violations, fmt.Sprintf("merge-check.yml: triggers on %v; the merge-gate spec allows only pull_request", wf.On))
	}
	c, _ := wf.Concurrency.(map[string]any)
	if g := fmt.Sprint(c["group"]); !strings.Contains(g, "github.event.pull_request.number") {
		violations = append(violations, fmt.Sprintf("merge-check.yml: concurrency group %q is not per pull request", g))
	}
	if got := fmt.Sprint(c["cancel-in-progress"]); got != "false" {
		violations = append(violations, fmt.Sprintf("merge-check.yml: cancel-in-progress is %s; the merge-gate spec requires false, so no cancelled Merge check sits beside a pass at one head (#87)", got))
	}
	violations = append(violations, writeGrants("workflow", wf.Permissions)...)
	if len(wf.Jobs) != 1 {
		violations = append(violations, fmt.Sprintf("merge-check.yml: has %d jobs; the merge-gate spec requires the one job merge-check", len(wf.Jobs)))
	}
	job, ok := wf.Jobs["merge-check"]
	if !ok {
		return append(violations, "merge-check.yml: no job merge-check")
	}
	if got := strings.TrimSpace(fmt.Sprint(job.Name)); got != mergeCheckName {
		violations = append(violations, fmt.Sprintf("merge-check.yml: job merge-check is named %v; the merge-gate spec requires %s so a skipped draft or awaiting job leaves no Merge check check run", job.Name, mergeCheckName))
	}
	if got := strings.TrimSpace(fmt.Sprint(job.If)); got != mergeCheckIf {
		violations = append(violations, fmt.Sprintf("merge-check.yml: job merge-check runs under if: %v; the merge-gate spec requires drafts and descriptions without reviewed-by: skip it", job.If))
	}
	if label, ok := job.RunsOn.(string); !ok || label != runnerLabel {
		violations = append(violations, fmt.Sprintf("merge-check.yml: job merge-check runs on %v; the merge-gate spec requires %s", job.RunsOn, runnerLabel))
	}
	violations = append(violations, mergeCheckPermissionViolations(job.Permissions)...)
	ran := false
	for _, st := range job.Steps {
		if st.Run != `scripts/merge-check.sh "$PR_NUMBER"` {
			continue
		}
		ran = true
		if got := fmt.Sprint(st.Env["ATEAM_REQUIRED_CHECKS"]); got != checks {
			violations = append(violations, fmt.Sprintf("merge-check.yml: ATEAM_REQUIRED_CHECKS is %q; the merge-gate spec requires %q so a ruleset dropping one fails the check", got, checks))
		}
	}
	if !ran {
		violations = append(violations, "merge-check.yml: no step runs scripts/merge-check.sh \"$PR_NUMBER\"")
	}
	return violations
}

// spec: openspec/specs/harness/spec.md#Requirement: Merge check runs on pull-request state events
func TestMergeCheckWorkflowPinned(t *testing.T) {
	for forge, checks := range map[string]string{
		".github": "Required,Merge check",
		".gitea":  "CI / Required (pull_request),Merge check / Merge check (pull_request)",
	} {
		data, err := os.ReadFile(filepath.Join(repoRoot(t), forge, "workflows", "merge-check.yml"))
		if err != nil {
			t.Fatal(err)
		}
		if v := mergeCheckWorkflowViolations(data, checks); len(v) > 0 {
			t.Errorf("%s merge-check.yml violations:\n- %s", forge, strings.Join(v, "\n- "))
		}
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Merge check runs on pull-request state events
func TestMergeCheckWorkflowPinnedSensitivity(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRoot(t), ".github", "workflows", "merge-check.yml"))
	if err != nil {
		t.Fatal(err)
	}
	good := string(data)
	const checks = "Required,Merge check"
	for _, tc := range []struct {
		name, old, repl string
		wants           []string
	}{
		{"edited dropped", "ready_for_review, edited]", "ready_for_review]", []string{"edited is missing"}},
		{"a push trigger", "on:\n", "on:\n  push:\n    branches: [main]\n", []string{"allows only pull_request"}},
		{"runs cancel", "cancel-in-progress: false", "cancel-in-progress: true", []string{"cancel-in-progress is true"}},
		{"group per ref", "merge-check-${{ github.event.pull_request.number }}", "merge-check-${{ github.ref }}", []string{"not per pull request"}},
		{"draft guard dropped", "    if: " + mergeCheckIf + "\n", "    if: ${{ contains(github.event.pull_request.body, 'reviewed-by:') }}\n", []string{"drafts and descriptions"}},
		{"awaiting guard dropped", "    if: " + mergeCheckIf + "\n", "    if: ${{ !github.event.pull_request.draft }}\n", []string{"drafts and descriptions"}},
		{"name kept on a draft", "    name: " + mergeCheckName + "\n", "    name: Merge check\n", []string{"is named Merge check"}},
		{"name kept while awaiting", "    name: " + mergeCheckName + "\n", "    name: " + draftName("Merge check") + "\n", []string{"awaiting job leaves no"}},
		{"a check dropped from the list", "ATEAM_REQUIRED_CHECKS: Required,Merge check", "ATEAM_REQUIRED_CHECKS: Required", []string{`ATEAM_REQUIRED_CHECKS is "Required"`}},
		{"write permission", "      pull-requests: read\n", "      pull-requests: write\n", []string{"pull-requests: write"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !strings.Contains(good, tc.old) {
				t.Fatalf("merge-check.yml lacks %q", tc.old)
			}
			requireViolation(t, mergeCheckWorkflowViolations([]byte(strings.Replace(good, tc.old, tc.repl, 1)), checks), tc.wants...)
		})
	}
}

type ciWorkflow struct {
	On          any
	Permissions any
	Concurrency any
	Defaults    ciDefaults
	Jobs        map[string]ciJob
}

type ciJob struct {
	Name            any
	RunsOn          any
	Needs           any
	If              any
	ContinueOnError any
	Permissions     any
	TimeoutMinutes  any
	Env             map[string]any
	Defaults        ciDefaults
	Steps           []ciStep
}

type ciDefaults struct {
	Run struct {
		Shell string
	}
}

type ciStep struct {
	Name            string
	If              any
	ContinueOnError any
	Uses            string
	Shell           string
	Env             map[string]any
	Run             string
}

// ciWorkflowViolations checks ci.yml against the merge-gate spec. On both
// forges a draft job is skipped under a non-required name (#250, #257).
func ciWorkflowViolations(t *testing.T, data []byte) []string {
	wf, err := parseCIWorkflowYAML(data)
	if err != nil {
		return []string{fmt.Sprintf("ci.yml: %v", err)}
	}
	violations := pullRequestTriggerViolations(wf.On, pullRequestTypes)
	violations = append(violations, concurrencyViolations(wf.Concurrency)...)
	violations = append(violations, draftViolations(wf)...)
	violations = append(violations, writeGrants("workflow", wf.Permissions)...)
	names := make([]string, 0, len(wf.Jobs))
	for name := range wf.Jobs {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		job := wf.Jobs[name]
		violations = append(violations, writeGrants("job "+name, job.Permissions)...)
		if job.RunsOn == nil {
			violations = append(violations, fmt.Sprintf("ci.yml: job %s has no runs-on; the merge-gate spec (\"Pinned runner image\") requires runs-on: %s", name, runnerLabel))
		} else if label, ok := job.RunsOn.(string); !ok || label != runnerLabel {
			violations = append(violations, fmt.Sprintf("ci.yml: job %s runs on %v; the merge-gate spec (\"Pinned runner image\") requires the one label %s", name, job.RunsOn, runnerLabel))
		}
	}

	if _, ok := wf.Jobs["merge-check"]; ok {
		violations = append(violations, "ci.yml: job merge-check is in ci.yml; the merge-gate spec runs it from merge-check.yml, so a pull-request state event does not rerun Verify (#255)")
	}

	if req, ok := wf.Jobs["required"]; !ok {
		violations = append(violations, "ci.yml: no job required; the merge-gate spec requires it")
	} else {
		var needs []string
		switch n := req.Needs.(type) {
		case string:
			needs = []string{n}
		case []any:
			for _, v := range n {
				needs = append(needs, fmt.Sprint(v))
			}
		}
		for _, want := range requiredNeeds {
			if !slices.Contains(needs, want) {
				violations = append(violations, fmt.Sprintf("ci.yml: job required needs %v; %s is missing, and the merge-gate spec requires it to need %v", needs, want, requiredNeeds))
			}
		}
		for _, d := range []struct{ where, shell string }{{"workflow", wf.Defaults.Run.Shell}, {"job required", req.Defaults.Run.Shell}} {
			if d.shell != "" && d.shell != "bash" {
				violations = append(violations, fmt.Sprintf("ci.yml: %s sets defaults.run.shell %q; the test runs required's step as GitHub runs bash", d.where, d.shell))
			}
		}
		violations = append(violations, requiredStepViolations(t, req, needs)...)
	}

	if v, ok := wf.Jobs["verify"]; !ok {
		violations = append(violations, "ci.yml: no job verify; the merge-gate spec requires it")
	} else if limit, ok := v.TimeoutMinutes.(int); !ok || limit != verifyTimeoutMinutes {
		violations = append(violations, fmt.Sprintf("ci.yml: job verify has timeout-minutes %v; the merge-gate spec requires %d minutes", v.TimeoutMinutes, verifyTimeoutMinutes))
	}
	return violations
}

// cancelInProgressExpr is the only cancel-in-progress value the merge-gate spec
// accepts: superseded pull-request runs are cancelled, but the ready_for_review
// run reuses the head SHA, and cancelling the run already in flight leaves a
// failed Required on a head that was never broken (#87).
const cancelInProgressExpr = "${{ github.event_name == 'pull_request' && github.event.action != 'ready_for_review' }}"

func concurrencyViolations(raw any) []string {
	if raw == nil {
		return nil
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return []string{fmt.Sprintf("ci.yml: concurrency is %v; the merge-gate spec requires a group with cancel-in-progress %s", raw, cancelInProgressExpr)}
	}
	cancel, ok := m["cancel-in-progress"]
	if !ok {
		return nil
	}
	if got := strings.TrimSpace(fmt.Sprint(cancel)); got != cancelInProgressExpr && got != "false" {
		return []string{fmt.Sprintf("ci.yml: concurrency sets cancel-in-progress %v; the merge-gate spec requires %s so a ready_for_review run never cancels the run for the same head and leaves a failed Required on it", cancel, cancelInProgressExpr)}
	}
	return nil
}

func pullRequestTriggerViolations(on any, pullRequestTypes []string) []string {
	var trigger any
	found := false
	switch o := on.(type) {
	case string:
		found = o == "pull_request"
	case []any:
		found = slices.Contains(o, any("pull_request"))
	case map[string]any:
		trigger, found = o["pull_request"]
	}
	if !found {
		return []string{fmt.Sprintf("ci.yml: no pull_request trigger; the merge-gate spec requires one listing the activity types %v", pullRequestTypes)}
	}
	var types []string
	if m, ok := trigger.(map[string]any); ok {
		switch ty := m["types"].(type) {
		case string:
			types = []string{ty}
		case []any:
			for _, v := range ty {
				types = append(types, fmt.Sprint(v))
			}
		}
	}
	var violations []string
	for _, want := range pullRequestTypes {
		if !slices.Contains(types, want) {
			violations = append(violations, fmt.Sprintf("ci.yml: the pull_request trigger lists activity types %v; %s is missing, and the merge-gate spec requires %v", types, want, pullRequestTypes))
		}
	}
	return violations
}

func writeGrants(where string, perms any) []string {
	switch p := perms.(type) {
	case string:
		if p != "read-all" {
			return []string{fmt.Sprintf("ci.yml: %s is granted permissions: %s; the merge-gate spec allows no write", where, p)}
		}
	case map[string]any:
		var out []string
		for _, scope := range slices.Sorted(maps.Keys(p)) {
			if level := fmt.Sprint(p[scope]); level == "write" {
				out = append(out, fmt.Sprintf("ci.yml: %s is granted %s: %s; the merge-gate spec allows no write", where, scope, level))
			}
		}
		return out
	}
	return nil
}

func mergeCheckPermissionViolations(perms any) []string {
	p, ok := perms.(map[string]any)
	if !ok {
		return []string{fmt.Sprintf("ci.yml: job merge-check has no permissions of its own (%v); the merge-gate spec requires exactly %v", perms, mergeCheckPermissions)}
	}
	var out []string
	for _, scope := range slices.Sorted(maps.Keys(p)) {
		level := fmt.Sprint(p[scope])
		if mergeCheckPermissions[scope] != level {
			out = append(out, fmt.Sprintf("ci.yml: job merge-check is granted %s: %s; the merge-gate spec allows only %v", scope, level, mergeCheckPermissions))
		}
	}
	for _, scope := range slices.Sorted(maps.Keys(mergeCheckPermissions)) {
		if _, ok := p[scope]; !ok {
			out = append(out, fmt.Sprintf("ci.yml: job merge-check is missing %s: %s; the merge-gate spec requires exactly %v", scope, mergeCheckPermissions[scope], mergeCheckPermissions))
		}
	}
	return out
}

func requiredResultSets() []map[string]string {
	sets := []map[string]string{{}}
	for _, j := range requiredNeeds {
		sets[0][j] = "success"
	}
	for _, bad := range []string{"failure", "cancelled", "skipped"} {
		for _, j := range requiredNeeds {
			set := maps.Clone(sets[0])
			set[j] = bad
			sets = append(sets, set)
		}
	}
	return append(sets, nil)
}

var ghExpression = regexp.MustCompile(`\$\{\{\s*(.*?)\s*\}\}`)

var (
	joinNeedsResults = regexp.MustCompile(`^join\(\s*needs\.\*\.result\s*(?:,\s*'([^']*)'\s*)?\)$`)
	oneNeedResult    = regexp.MustCompile(`^needs\.([A-Za-z0-9_-]+)\.result$`)
)

func evalNeedsExpression(expr string, needs []string, results map[string]string) (string, bool) {
	if m := joinNeedsResults.FindStringSubmatch(expr); m != nil {
		sep := ","
		if strings.Contains(expr, "'") {
			sep = m[1]
		}
		var rs []string
		for _, j := range needs {
			if r, ok := results[j]; ok {
				rs = append(rs, r)
			}
		}
		return strings.Join(rs, sep), true
	}
	if m := oneNeedResult.FindStringSubmatch(expr); m != nil {
		return results[m[1]], true
	}
	return "", false
}

func requiredStepViolations(t *testing.T, req ciJob, needs []string) []string {
	var out []string
	want := "always() && (" + draftGuard + ")"
	if req.If == nil || strings.TrimSpace(fmt.Sprint(req.If)) != want {
		out = append(out, fmt.Sprintf("ci.yml: job required runs under if: %v; the merge-gate spec requires if: %s, or a failed or cancelled needed job leaves Required skipped, which GitHub reports as success", req.If, want))
	}
	if req.ContinueOnError != nil {
		out = append(out, fmt.Sprintf("ci.yml: job required sets continue-on-error: %v; the merge-gate spec requires it to fail", req.ContinueOnError))
	}
	if len(req.Steps) == 0 {
		return append(out, "ci.yml: job required has no steps; the merge-gate spec requires a step that reads the results of its needed jobs")
	}

	var texts []string
	for _, v := range req.Env {
		texts = append(texts, fmt.Sprint(v))
	}
	runnable := true
	for i, st := range req.Steps {
		where := fmt.Sprintf("ci.yml: job required, step %d (%s)", i+1, st.Name)
		switch {
		case st.If != nil:
			out = append(out, fmt.Sprintf("%s has the condition %v; the merge-gate spec requires the step to run on every result", where, st.If))
		case st.ContinueOnError != nil:
			out = append(out, fmt.Sprintf("%s sets continue-on-error: %v; the merge-gate spec requires the step's failure to fail Required", where, st.ContinueOnError))
		case st.Uses != "" || st.Run == "":
			out = append(out, fmt.Sprintf("%s is not a run step (uses %q); the test runs the step's script as the workflow writes it", where, st.Uses))
			runnable = false
		case st.Shell != "" && st.Shell != "bash":
			out = append(out, fmt.Sprintf("%s runs under shell %q; the test runs it as GitHub runs bash", where, st.Shell))
			runnable = false
		}
		texts = append(texts, st.Run)
		for _, v := range st.Env {
			texts = append(texts, fmt.Sprint(v))
		}
	}
	readsAll := false
	for _, text := range texts {
		for _, m := range ghExpression.FindAllStringSubmatch(text, -1) {
			if joinNeedsResults.MatchString(m[1]) {
				readsAll = true
			}
			if _, ok := evalNeedsExpression(m[1], needs, nil); !ok {
				out = append(out, fmt.Sprintf("ci.yml: job required carries the expression ${{ %s }}, which the test cannot evaluate; the merge-gate spec requires its step to read join(needs.*.result, ...)", m[1]))
				runnable = false
			}
		}
	}
	if !readsAll {
		out = append(out, "ci.yml: job required: its step does not take its results from needs.*.result; the merge-gate spec requires it to read the result of every job it needs")
	}
	if !runnable {
		return out
	}

	for _, results := range requiredResultSets() {
		label := "no results"
		if results != nil {
			var parts []string
			for _, j := range requiredNeeds {
				parts = append(parts, j+"="+results[j])
			}
			label = strings.Join(parts, " ")
		}
		subst := func(text string) string {
			return ghExpression.ReplaceAllStringFunc(text, func(e string) string {
				v, _ := evalNeedsExpression(ghExpression.FindStringSubmatch(e)[1], needs, results)
				return v
			})
		}
		ok, output, err := runRequiredSteps(t, req, subst)
		if err != nil {
			return append(out, fmt.Sprintf("ci.yml: job required: running its step for %s: %v", label, err))
		}
		allSuccess := results != nil && !slices.ContainsFunc(requiredNeeds, func(j string) bool { return results[j] != "success" })
		switch {
		case allSuccess && !ok:
			out = append(out, fmt.Sprintf("ci.yml: job required: its step exits non-zero when every needed job succeeded (%s): %s", label, output))
		case !allSuccess && ok:
			out = append(out, fmt.Sprintf("ci.yml: job required: its step exits 0 for %s; the merge-gate spec requires it to fail unless every needed job succeeded", label))
		}
	}
	return out
}

func runRequiredSteps(t *testing.T, req ciJob, subst func(string) string) (bool, string, error) {
	for _, st := range req.Steps {
		env := []string{"PATH=" + os.Getenv("PATH")}
		for _, m := range []map[string]any{req.Env, st.Env} {
			for _, k := range slices.Sorted(maps.Keys(m)) {
				env = append(env, k+"="+subst(fmt.Sprint(m[k])))
			}
		}
		cmd := exec.Command("bash", "--noprofile", "--norc", "-e", "-c", subst(st.Run))
		cmd.Env = env
		cmd.Dir = t.TempDir()
		output, err := cmd.CombinedOutput()
		var exit *exec.ExitError
		switch {
		case errors.As(err, &exit):
			return false, strings.TrimSpace(string(output)), nil
		case err != nil:
			return false, "", err
		}
	}
	return true, "", nil
}

// Minimal stdlib YAML parser for GitHub Actions workflow structures (avoids external yaml dependency).
func parseCIWorkflowYAML(data []byte) (ciWorkflow, error) {
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	root, _ := parseYAMLBlockMap(lines, 0, 0)
	var wf ciWorkflow
	wf.On = root["on"]
	wf.Permissions = root["permissions"]
	wf.Concurrency = root["concurrency"]
	wf.Defaults = decodeCIDefaults(root["defaults"])
	if jobsMap, ok := root["jobs"].(map[string]any); ok {
		wf.Jobs = make(map[string]ciJob, len(jobsMap))
		for name, rawJob := range jobsMap {
			wf.Jobs[name] = decodeCIJob(rawJob)
		}
	}
	return wf, nil
}

func decodeCIDefaults(raw any) ciDefaults {
	var d ciDefaults
	if m, ok := raw.(map[string]any); ok {
		if runMap, ok := m["run"].(map[string]any); ok {
			if sh, ok := runMap["shell"]; ok {
				d.Run.Shell = fmt.Sprint(sh)
			}
		}
	}
	return d
}

func decodeCIJob(raw any) ciJob {
	var j ciJob
	m, ok := raw.(map[string]any)
	if !ok {
		return j
	}
	j.Name = m["name"]
	j.RunsOn = m["runs-on"]
	j.Needs = m["needs"]
	j.If = m["if"]
	j.ContinueOnError = m["continue-on-error"]
	j.Permissions = m["permissions"]
	j.TimeoutMinutes = m["timeout-minutes"]
	if envMap, ok := m["env"].(map[string]any); ok {
		j.Env = envMap
	}
	j.Defaults = decodeCIDefaults(m["defaults"])
	if stepsSlice, ok := m["steps"].([]any); ok {
		for _, item := range stepsSlice {
			if sm, ok := item.(map[string]any); ok {
				var st ciStep
				if v, ok := sm["name"]; ok {
					st.Name = fmt.Sprint(v)
				}
				st.If = sm["if"]
				st.ContinueOnError = sm["continue-on-error"]
				if v, ok := sm["uses"]; ok {
					st.Uses = fmt.Sprint(v)
				}
				if v, ok := sm["shell"]; ok {
					st.Shell = fmt.Sprint(v)
				}
				if envMap, ok := sm["env"].(map[string]any); ok {
					st.Env = envMap
				}
				if v, ok := sm["run"]; ok {
					st.Run = fmt.Sprint(v)
				}
				j.Steps = append(j.Steps, st)
			}
		}
	}
	return j
}

func stripYAMLComment(s string) string {
	inSingle := false
	inDouble := false
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		case '#':
			if !inSingle && !inDouble && (i == 0 || s[i-1] == ' ' || s[i-1] == '\t') {
				return strings.TrimRight(s[:i], " \t")
			}
		}
	}
	return strings.TrimRight(s, " \t")
}

func lineIndent(s string) int {
	n := 0
	for n < len(s) && s[n] == ' ' {
		n++
	}
	return n
}

func nextSignificantLine(lines []string, idx int) (int, int, string) {
	for i := idx; i < len(lines); i++ {
		cleaned := stripYAMLComment(lines[i])
		if strings.TrimSpace(cleaned) == "" {
			continue
		}
		return i, lineIndent(cleaned), strings.TrimSpace(cleaned)
	}
	return len(lines), -1, ""
}

func parseYAMLBlockMap(lines []string, startIdx, minIndent int) (map[string]any, int) {
	out := make(map[string]any)
	idx := startIdx
	for {
		nextIdx, indent, trimmed := nextSignificantLine(lines, idx)
		if nextIdx >= len(lines) || indent < minIndent || strings.HasPrefix(trimmed, "- ") {
			return out, nextIdx
		}
		key, valPart, hasColon := splitYAMLKeyValue(trimmed)
		if !hasColon {
			idx = nextIdx + 1
			continue
		}
		if valPart == "|" {
			var blockLines []string
			blockIndent := -1
			i := nextIdx + 1
			for ; i < len(lines); i++ {
				raw := lines[i]
				if strings.TrimSpace(raw) == "" {
					if blockIndent >= 0 {
						blockLines = append(blockLines, "")
					}
					continue
				}
				ind := lineIndent(raw)
				if ind <= indent {
					break
				}
				if blockIndent < 0 {
					blockIndent = ind
				}
				if len(raw) >= blockIndent {
					blockLines = append(blockLines, raw[blockIndent:])
				} else {
					blockLines = append(blockLines, strings.TrimLeft(raw, " "))
				}
			}
			out[key] = strings.Join(blockLines, "\n") + "\n"
			idx = i
			continue
		}
		if valPart != "" {
			out[key] = parseYAMLValue(valPart)
			idx = nextIdx + 1
			continue
		}
		childIdx, childIndent, childTrimmed := nextSignificantLine(lines, nextIdx+1)
		if childIdx >= len(lines) || childIndent <= indent {
			out[key] = nil
			idx = nextIdx + 1
			continue
		}
		if strings.HasPrefix(childTrimmed, "- ") {
			seq, afterSeq := parseYAMLBlockSeq(lines, childIdx, childIndent)
			out[key] = seq
			idx = afterSeq
		} else {
			subMap, afterMap := parseYAMLBlockMap(lines, childIdx, childIndent)
			out[key] = subMap
			idx = afterMap
		}
	}
}

func parseYAMLBlockSeq(lines []string, startIdx, seqIndent int) ([]any, int) {
	var out []any
	idx := startIdx
	for {
		nextIdx, indent, trimmed := nextSignificantLine(lines, idx)
		if nextIdx >= len(lines) || indent < seqIndent || !strings.HasPrefix(trimmed, "- ") {
			return out, nextIdx
		}
		rest := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
		itemKey, itemVal, hasColon := splitYAMLKeyValue(rest)
		if !hasColon {
			out = append(out, parseYAMLValue(rest))
			idx = nextIdx + 1
			continue
		}
		synthLines := slices.Clone(lines)
		prefixSpaces := strings.Repeat(" ", indent+2)
		synthLines[nextIdx] = prefixSpaces + itemKey + ": " + itemVal
		itemMap, afterItem := parseYAMLBlockMap(synthLines, nextIdx, indent+2)
		out = append(out, itemMap)
		idx = afterItem
	}
}

func splitYAMLKeyValue(s string) (string, string, bool) {
	depth := 0
	inSingle := false
	inDouble := false
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		case '[', '{':
			if !inSingle && !inDouble {
				depth++
			}
		case ']', '}':
			if !inSingle && !inDouble && depth > 0 {
				depth--
			}
		case ':':
			if !inSingle && !inDouble && depth == 0 && (i+1 == len(s) || s[i+1] == ' ' || s[i+1] == '\t') {
				return strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+1:]), true
			}
		}
	}
	return "", "", false
}

func splitFlowItems(s string) []string {
	var items []string
	depth := 0
	inSingle := false
	inDouble := false
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		case '[', '{', '(':
			if !inSingle && !inDouble {
				depth++
			}
		case ']', '}', ')':
			if !inSingle && !inDouble && depth > 0 {
				depth--
			}
		case ',':
			if !inSingle && !inDouble && depth == 0 {
				if part := strings.TrimSpace(s[start:i]); part != "" {
					items = append(items, part)
				}
				start = i + 1
			}
		}
	}
	if last := strings.TrimSpace(s[start:]); last != "" {
		items = append(items, last)
	}
	return items
}

func parseYAMLValue(s string) any {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		inner := strings.TrimSpace(s[1 : len(s)-1])
		if inner == "" {
			return []any{}
		}
		parts := splitFlowItems(inner)
		out := make([]any, 0, len(parts))
		for _, p := range parts {
			out = append(out, parseYAMLValue(p))
		}
		return out
	}
	if strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}") && !strings.HasPrefix(s, "${{") {
		inner := strings.TrimSpace(s[1 : len(s)-1])
		out := make(map[string]any)
		if inner == "" {
			return out
		}
		for _, p := range splitFlowItems(inner) {
			if k, v, ok := splitYAMLKeyValue(p); ok {
				out[k] = parseYAMLValue(v)
			}
		}
		return out
	}
	if (strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'")) || (strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"")) {
		return s[1 : len(s)-1]
	}
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	if s == "true" {
		return true
	}
	if s == "false" {
		return false
	}
	return s
}
