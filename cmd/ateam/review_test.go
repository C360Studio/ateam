package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/c360studio/ateam/internal/review"
	"github.com/c360studio/ateam/internal/state"
)

func cliReviewPolicy() review.Policy {
	return review.Policy{Version: 1, Models: []review.Model{{ID: "work", Provider: "codex", StrongerThan: []string{}}, {ID: "audit", Provider: "codex", StrongerThan: []string{"work"}}}, Routes: review.Routes{Developer: review.Route{Model: "work", Effort: "medium"}, Reviewer: review.Route{Model: "audit", Effort: "high"}}, Owners: []string{"owner"}, ReviewAuthors: []string{"review-account"}, RequiredChecks: []string{"task verify"}}
}

// spec: openspec/specs/model-review/spec.md#Requirement: Explicit setup and portable routing
// spec: openspec/specs/model-review/spec.md#Requirement: Stronger independent reviewer
func TestReviewCLISetupAndRouting(t *testing.T) {
	root := t.TempDir()
	p := cliReviewPolicy()
	b, _ := json.Marshal(p)
	file := filepath.Join(t.TempDir(), "policy.json")
	if err := os.WriteFile(file, b, 0600); err != nil {
		t.Fatal(err)
	}
	if code := runReview(root, []string{"setup", "--policy", file}); code != 0 {
		t.Fatalf("setup=%d", code)
	}
	if code := runReview(root, []string{"route"}); code != 0 {
		t.Fatalf("route=%d", code)
	}
	for _, args := range [][]string{{}, {"unknown"}, {"route", "--developer-models", "audit"}, {"route", "--developer-models", "work,unknown"}, {"route", "unexpected"}, {"check", "--pr", "1"}, {"setup", "--policy", file, "unexpected"}} {
		if code := runReview(root, args); code == 0 {
			t.Fatalf("invalid command accepted: %v", args)
		}
	}
}

// spec: openspec/specs/model-review/spec.md#Requirement: Current and independently attributed review evidence
// spec: openspec/specs/model-review/spec.md#Requirement: Stronger independent reviewer
// spec: openspec/specs/model-review/spec.md#Requirement: Explicit owner exception at the model ceiling
// spec: openspec/specs/model-review/spec.md#Requirement: Portable enforcement and honest trust boundaries
// spec: openspec/specs/model-review/spec.md#Requirement: Explicit setup and portable routing
func TestReviewCLICurrentForgeEvidence(t *testing.T) {
	for _, scenario := range []string{"pass", "untrusted-author", "stale-head", "moving-head", "same-tier", "exception", "wrong-owner", "candidate-policy", "edited-record", "moving-record", "moving-exception"} {
		t.Run(scenario, func(t *testing.T) {
			root := t.TempDir()
			git := func(args ...string) string {
				t.Helper()
				cmd := exec.Command("git", args...)
				cmd.Dir = root
				out, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("git %v: %v %s", args, err, out)
				}
				return strings.TrimSpace(string(out))
			}
			git("init", "-q")
			git("config", "user.name", "Fixture")
			git("config", "user.email", "fixture@example.invalid")
			p := cliReviewPolicy()
			data, _ := json.Marshal(p)
			if err := os.Mkdir(filepath.Join(root, ".agents"), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, ".agents", "review-policy.json"), data, 0600); err != nil {
				t.Fatal(err)
			}
			git("add", ".")
			git("commit", "-qm", "trusted policy")
			base := git("rev-parse", "HEAD")
			if scenario == "candidate-policy" {
				modified := p
				modified.ReviewAuthors = []string{"attacker"}
				b, _ := json.Marshal(modified)
				if err := os.WriteFile(filepath.Join(root, ".agents", "review-policy.json"), b, 0600); err != nil {
					t.Fatal(err)
				}
				git("add", ".")
			}
			git("commit", "--allow-empty", "-qm", "implementation")
			head := git("rev-parse", "HEAD")
			record := review.Record{Version: 1, Base: base, Head: head, PolicyDigest: review.PolicyDigest(data), Implementers: []review.Run{{ID: "dev-session", Model: "work", Provider: "codex", Effort: "medium"}}, Reviewer: review.Run{ID: "review-session", Model: "audit", Provider: "codex", Effort: "high"}, ReadOnly: true, Verdict: "approve", Findings: []review.Finding{}, Checks: []review.Check{{Name: "task verify", Commit: head, Status: "passed", Source: "reviewer"}}, Limitations: []string{"Runtime identities are attested."}}
			author := "review-account"
			exceptionAuthor := "owner"
			reads := 0
			recordReads, exceptionReads := 0, 0
			if scenario == "untrusted-author" || scenario == "candidate-policy" {
				author = "attacker"
			}
			if scenario == "stale-head" {
				record.Head = base
			}
			if scenario == "same-tier" || scenario == "exception" || scenario == "wrong-owner" || scenario == "moving-exception" {
				record.Reviewer.Model = "work"
			}
			if scenario == "wrong-owner" {
				exceptionAuthor = "attacker"
			}
			var baseURL string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/api/v1/repos/acme/project":
					_ = json.NewEncoder(w).Encode(map[string]string{"full_name": "acme/project", "default_branch": "main"})
				case "/api/v1/repos/acme/project/branches/main":
					_ = json.NewEncoder(w).Encode(map[string]any{"name": "main", "commit": map[string]string{"id": base}})
				case "/api/v1/repos/acme/project/pulls/15":
					reads++
					currentHead := head
					if scenario == "moving-head" && reads > 1 {
						currentHead = base
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"number": 15, "state": "open", "merged": false, "draft": true, "user": map[string]string{"login": "dev-account"}, "base": map[string]any{"sha": base, "ref": "main", "repo": map[string]string{"full_name": "acme/project"}}, "head": map[string]string{"sha": currentHead}})
				case "/api/v1/repos/acme/project/issues/comments/1":
					recordReads++
					updated := "2026-10-19T01:00:00Z"
					if scenario == "edited-record" {
						updated = "2026-10-19T01:00:01Z"
					}
					record.Repository = "gitea:" + baseURL + "/acme/project"
					b, _ := json.Marshal(record)
					if scenario == "moving-record" && recordReads > 1 {
						b = append(b, byte(32))
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"id": 1, "issue_url": "", "pull_request_url": baseURL + "/acme/project/pulls/15", "created_at": "2026-10-19T01:00:00Z", "updated_at": updated, "body": "ateam-review\n" + string(b), "user": map[string]string{"login": author}})
				case "/api/v1/repos/acme/project/issues/comments/2":
					exceptionReads++
					e := review.Exception{Version: 1, Repository: "gitea:" + baseURL + "/acme/project", Head: head, ReviewerModel: "work", PolicyDigest: review.PolicyDigest(data), Reason: "Explicit test owner exception"}
					b, _ := json.Marshal(e)
					if scenario == "moving-exception" && exceptionReads > 1 {
						b = append(b, byte(32))
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"id": 2, "issue_url": "", "pull_request_url": baseURL + "/acme/project/pulls/15", "created_at": "2026-10-19T01:00:00Z", "updated_at": "2026-10-19T01:00:00Z", "body": "ateam-review-exception\n" + string(b), "user": map[string]string{"login": exceptionAuthor}})
				default:
					http.NotFound(w, r)
				}
			}))
			defer srv.Close()
			baseURL = srv.URL
			git("remote", "add", "origin", srv.URL+"/acme/project.git")
			t.Setenv("ATEAM_FORGE", "gitea")
			t.Setenv("GITEA_URL", srv.URL)
			t.Setenv("GITEA_TOKEN", "")
			t.Setenv("ATEAM_TOKEN_READ", "")
			t.Setenv("ATEAM_TOKEN_WRITE", "")
			args := []string{"--pr", "15", "--record-comment", "1"}
			if scenario == "exception" || scenario == "wrong-owner" || scenario == "moving-exception" {
				args = append(args, "--exception-comment", "2")
			}
			err := reviewCheck(root, args)
			wantPass := scenario == "pass" || scenario == "exception"
			if (err == nil) != wantPass {
				t.Fatalf("scenario %s: error=%v, want pass=%v", scenario, err, wantPass)
			}
		})
	}
}

// spec: openspec/specs/model-review/spec.md#Requirement: Current and independently attributed review evidence
// spec: openspec/specs/model-review/spec.md#Requirement: Portable enforcement and honest trust boundaries
func TestReviewCommentAndFileBoundaries(t *testing.T) {
	for _, body := range []string{"{}", "ateam-review {}", "```json\n{}\n```", "ateam-review-exception\n{}"} {
		if _, err := reviewCommentPayload(body, "ateam-review"); err == nil {
			t.Fatalf("ambiguous comment accepted: %q", body)
		}
	}
	b, err := reviewCommentPayload("ateam-review\n{\"version\":1}", "ateam-review")
	if err != nil || string(b) != "{\"version\":1}" {
		t.Fatalf("payload=%q %v", b, err)
	}
	p := filepath.Join(t.TempDir(), "oversized.json")
	if err := os.WriteFile(p, make([]byte, (1<<20)+1), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readReviewFile(p); err == nil {
		t.Fatal("oversized policy accepted")
	}
}

// spec: openspec/specs/model-review/spec.md#Requirement: Portable enforcement and honest trust boundaries
func TestReviewTokenDoesNotReadDefaultAccountForCustomAPI(t *testing.T) {
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("ATEAM_TOKEN_READ", "")
	t.Setenv("ATEAM_TOKEN_WRITE", "")
	t.Setenv("PATH", t.TempDir())
	token, err := reviewToken(context.Background(), state.ForgeConfig{Provider: state.ForgeGitHub, BaseURL: "https://custom.example.invalid"})
	if err != nil || token != "" {
		t.Fatal(fmt.Sprintf("unexpected credential fallback: %q %v", token, err))
	}
}

// spec: openspec/specs/model-review/spec.md#Requirement: Portable enforcement and honest trust boundaries
// spec: openspec/specs/harness/spec.md#Requirement: Forge token roles
func TestReviewTokenEnvironment(t *testing.T) {
	for _, tc := range []struct {
		name                                            string
		provider                                        state.ForgeProvider
		gitea, alternate, gh, github, read, write, want string
	}{
		{name: "gitea read role first", provider: state.ForgeGitea, gitea: "primary", read: "fake-read", write: "fake-write", want: "fake-read"},
		{name: "github read role first", provider: state.ForgeGitHub, gh: "primary", read: "fake-read", write: "fake-write", want: "fake-read"},
		{name: "write role alone serves review reads", provider: state.ForgeGitHub, gh: "primary", write: " fake-write ", want: "fake-write"},
		{name: "gitea fallback", provider: state.ForgeGitea, alternate: " fallback \n", want: "fallback"},
		{name: "gitea whitespace fallback", provider: state.ForgeGitea, gitea: " \t", alternate: " fallback ", want: "fallback"},
		{name: "gitea primary", provider: state.ForgeGitea, gitea: " primary \n", alternate: "fallback", want: "primary"},
		{name: "gh primary", provider: state.ForgeGitHub, gh: " primary \n", github: "fallback", want: "primary"},
		{name: "github fallback", provider: state.ForgeGitHub, gh: " \t", github: " fallback \n", want: "fallback"},
		{name: "no token", provider: state.ForgeGitea},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("GITEA_TOKEN", tc.gitea)
			t.Setenv("ATEAM_GITEA_TOKEN", tc.alternate)
			t.Setenv("GH_TOKEN", tc.gh)
			t.Setenv("GITHUB_TOKEN", tc.github)
			t.Setenv("ATEAM_TOKEN_READ", tc.read)
			t.Setenv("ATEAM_TOKEN_WRITE", tc.write)
			got, err := reviewToken(context.Background(), state.ForgeConfig{Provider: tc.provider, BaseURL: "https://custom.example.invalid"})
			if err != nil || got != tc.want {
				t.Fatalf("token normalization/fallback mismatch: got %q want %q, err=%v", got, tc.want, err)
			}
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Forge token roles
func TestForgeTokenWarnings(t *testing.T) {
	for _, tc := range []struct {
		name, read, write string
		warn              bool
	}{
		{name: "neither set", warn: true},
		{name: "read only", read: "fake-read", warn: true},
		{name: "write only", write: "fake-write", warn: true},
		{name: "same token in both", read: "fake-same", write: " fake-same\n", warn: true},
		{name: "separate tokens", read: "fake-read", write: "fake-write"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("ATEAM_TOKEN_READ", tc.read)
			t.Setenv("ATEAM_TOKEN_WRITE", tc.write)
			got := forgeTokenWarnings()
			if !tc.warn {
				if len(got) != 0 {
					t.Fatalf("separate tokens warned: %q", got)
				}
				return
			}
			want := "forge reads and writes share one token; set ATEAM_TOKEN_READ and ATEAM_TOKEN_WRITE to separate tokens scoped to each role"
			if len(got) != 1 || got[0] != want {
				t.Fatalf("warnings = %q, want [%q]", got, want)
			}
		})
	}
}

// spec: openspec/specs/model-review/spec.md#Requirement: Model tiers and effort bands policy
func TestReviewRouteTiersOnlyFailsClosed(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(t.TempDir(), "policy.json")
	if err := os.WriteFile(file, []byte(`{"version":1,"model_tiers":{"frontier":["claude:opus"],"workhorse":["codex:gpt-6-sol"],"light":["gemini:gemini-3.8-flash"]},"effort_bands":{"claude":{"high":["max"]},"codex":{"high":["xhigh"]},"gemini":{"high":["high"]}},"cross_provider":false}`), 0600); err != nil {
		t.Fatal(err)
	}
	if code := runReview(root, []string{"setup", "--policy", file}); code != 0 {
		t.Fatalf("tiers-only setup=%d", code)
	}
	if err := reviewRoute(root, nil); err == nil || !strings.Contains(err.Error(), "model tiers only") {
		t.Fatalf("route must fail closed naming the tiers-only policy, got %v", err)
	}
}

// spec: openspec/specs/model-review/spec.md#Requirement: Not-weaker reviewer by tier and effort band
func TestReviewCompareExitCodes(t *testing.T) {
	root := t.TempDir()
	pass := []string{"compare", "--implementer", "claude:opus:high", "--reviewer", "codex:gpt-6-astra:xhigh"}
	if code := runReview(root, pass); code != 0 {
		t.Fatalf("no policy: compare=%d, want 0 (skipped)", code)
	}
	policy := `{"version":1,"model_tiers":{"frontier":["claude:opus","codex:gpt-6-astra"],"workhorse":["claude:sonnet"]},
"effort_bands":{"claude":{"low":["low"],"mid":["medium"],"high":["high"]},"codex":{"mid":["medium"],"high":["xhigh"]},"gemini":{"high":["high"]}}}`
	if err := os.MkdirAll(filepath.Join(root, ".agents"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".agents", "review-policy.json"), []byte(policy), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		args []string
		want int
	}{
		{pass, 0},
		{[]string{"compare", "--implementer", "claude:opus:high", "--reviewer", "claude:sonnet:high"}, 1},
		{[]string{"compare", "--implementer", "claude:opus:high", "--reviewer", "codex:gpt-6-astra:medium"}, 1},
		{[]string{"compare", "--implementer", "claude:opus", "--reviewer", "codex:gpt-6-astra:xhigh"}, 2},
		{[]string{"compare", "--implementer", "claude:opus:", "--reviewer", "codex:gpt-6-astra:xhigh"}, 1},
		{[]string{"compare", "--reviewer", "codex:gpt-6-astra:xhigh"}, 2},
		{[]string{"compare", "--implementer", "claude:opus:high", "--reviewer", "codex:a:high", "--reviewer", "codex:b:high"}, 2},
	} {
		if code := runReview(root, tc.args); code != tc.want {
			t.Errorf("%v: exit %d, want %d", tc.args, code, tc.want)
		}
	}
	if err := os.WriteFile(filepath.Join(root, ".agents", "review-policy.json"), []byte(`{"version":2}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if code := runReview(root, pass); code != 2 {
		t.Fatalf("invalid policy: compare=%d, want 2", code)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Merge gate and CI workflow enforcement
func TestGateForgeSelection(t *testing.T) {
	// #143: ATEAM_FORGE wins; under Actions GITHUB_SERVER_URL picks the forge
	// (Gitea Actions sets it too); otherwise origin; with no origin, gh's GitHub.
	gitRepo := func(t *testing.T, origin string) string {
		t.Helper()
		dir := t.TempDir()
		for _, args := range [][]string{{"init", "-q"}, {"remote", "add", "origin", origin}} {
			if out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput(); err != nil {
				t.Fatalf("git %v: %v\n%s", args, err, out)
			}
		}
		return dir
	}
	for _, tc := range []struct {
		name       string
		env        map[string]string
		actions    bool
		origin     string
		want       review.ForgeConfig
		wantSource string
		wantErr    string
	}{
		{name: "github actions", actions: true, env: map[string]string{"GITHUB_SERVER_URL": "https://github.com/", "GITHUB_REPOSITORY": "acme/widgets"},
			want: review.ForgeConfig{Provider: "github", BaseURL: "https://github.com", Owner: "acme", Repo: "widgets"}, wantSource: "GITHUB_SERVER_URL"},
		{name: "gitea actions", actions: true, env: map[string]string{"GITHUB_SERVER_URL": "https://gitea.example.test", "GITHUB_REPOSITORY": "acme/widgets"},
			want: review.ForgeConfig{Provider: "gitea", BaseURL: "https://gitea.example.test", Owner: "acme", Repo: "widgets"}, wantSource: "GITHUB_SERVER_URL"},
		{name: "explicit forge under actions", actions: true, env: map[string]string{"ATEAM_FORGE": "github", "GITHUB_SERVER_URL": "https://gitea.example.test", "GITHUB_REPOSITORY": "acme/widgets"},
			want: review.ForgeConfig{Provider: "github", BaseURL: "https://gitea.example.test", Owner: "acme", Repo: "widgets"}, wantSource: "ATEAM_FORGE"},
		{name: "malformed repository", actions: true, env: map[string]string{"GITHUB_SERVER_URL": "https://gitea.example.test", "GITHUB_REPOSITORY": "widgets"},
			wantErr: "GITHUB_REPOSITORY is not owner/repo"},
		{name: "unknown explicit forge", env: map[string]string{"ATEAM_FORGE": "gitlab"}, wantErr: "neither github nor gitea"},
		{name: "server url outside actions is not read", env: map[string]string{"GITHUB_SERVER_URL": "https://gitea.example.test", "GITHUB_REPOSITORY": "acme/widgets"},
			origin: "git@github.com:acme/widgets.git",
			want:   review.ForgeConfig{Provider: "github", BaseURL: "https://api.github.com", Owner: "acme", Repo: "widgets"}, wantSource: "origin"},
		{name: "gitea origin", origin: "https://gitea.example.test/acme/widgets.git",
			want: review.ForgeConfig{Provider: "gitea", BaseURL: "https://gitea.example.test", Owner: "acme", Repo: "widgets"}, wantSource: "origin"},
		{name: "no origin", want: review.ForgeConfig{Provider: "github"}, wantSource: "default: no origin forge"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, k := range []string{"ATEAM_FORGE", "GITEA_URL", "GITHUB_API_URL", "GITHUB_SERVER_URL", "GITHUB_REPOSITORY"} {
				t.Setenv(k, tc.env[k])
			}
			root := t.TempDir()
			if tc.origin != "" {
				root = gitRepo(t, tc.origin)
			}
			got, source, err := gateForge(context.Background(), root, tc.actions)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil || got != tc.want || source != tc.wantSource {
				t.Fatalf("gateForge = %+v, %q, %v; want %+v, %q", got, source, err, tc.want, tc.wantSource)
			}
		})
	}
}

// spec: openspec/specs/model-review/spec.md#Requirement: Reviewer run from a review record
func TestReviewCompareReviewerRecord(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "openspec", "changes", "archive", "2026-10-26-example")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	recordPath := filepath.Join(dir, "review-implementation.md")
	approvalPath := filepath.Join(dir, "approval.md")
	// write puts a synthetic REVIEW-002 record in place and, when approved, an
	// approval.md row holding the record's SHA-256 (PolicyDigest is plain SHA-256 hex).
	write := func(verdict, reviewer string, approved bool, edit func(string) string) {
		t.Helper()
		record := "## Verdict\n\n- **Stage**: implementation\n- **Verdict**: " + verdict + "\n- **Reviewer**: " + reviewer + "\n- **Date**: 2026-10-26\n- **Round**: 1\n\n## Manifest\n\n```text\ndeleted  example.go\n```\n\n## Findings\n\n| ID | Severity | Location | Rule | Finding | Status |\n|----|----------|----------|------|---------|--------|\n"
		if edit != nil {
			record = edit(record)
		}
		if err := os.WriteFile(recordPath, []byte(record), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(approvalPath); err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
		if approved {
			row := fmt.Sprintf("| Stage | Approver | Date | Review digest |\n|-------|----------|------|---------------|\n| implementation | person@example.com | 2026-10-26 | %s |\n", review.PolicyDigest([]byte(record)))
			if err := os.WriteFile(approvalPath, []byte(row), 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
	args := []string{"compare", "--implementer", "claude:sonnet:high", "--reviewer-record", recordPath}

	write("FAIL", "claude (opus, high)", true, nil)
	if code := runReview(root, args); code != 1 {
		t.Errorf("no policy, FAIL verdict: exit %d, want 1", code)
	}
	if err := os.MkdirAll(filepath.Join(root, ".agents"), 0o755); err != nil {
		t.Fatal(err)
	}
	policy := `{"version":1,"review_record":"aip","model_tiers":{"frontier":["claude:opus"],"workhorse":["claude:sonnet"]},"effort_bands":{"claude":{"mid":["medium"],"high":["high"]},"codex":{"high":["high"]},"gemini":{"high":["high"]}}}`
	if err := os.WriteFile(filepath.Join(root, ".agents", "review-policy.json"), []byte(policy), 0o600); err != nil {
		t.Fatal(err)
	}
	type testCase struct {
		name     string
		verdict  string
		reviewer string
		approved bool
		args     []string
		want     int
		edit     func(string) string
	}
	cases := []testCase{
		{"approved pass, stronger reviewer", "PASS", "claude (opus, high)", true, args, 0, nil},
		{"approved pass, weaker reviewer", "PASS", "claude (sonnet, medium)", true, args, 1, nil},
		{"fail verdict", "FAIL", "claude (opus, high)", true, args, 1, nil},
		{"no approval.md", "PASS", "claude (opus, high)", false, args, 1, nil},
		{"reviewer names no agent", "PASS", "example-reviewer (claude-opus-5-5)", true, args, 1, nil},
		{"record and reviewer both", "PASS", "claude (opus, high)", true, append(slices.Clone(args), "--reviewer", "claude:opus:high"), 2, nil},
		{"record missing", "PASS", "claude (opus, high)", true, []string{"compare", "--implementer", "claude:sonnet:high", "--reviewer-record", filepath.Join(dir, "missing.md")}, 2, nil},
		{"fenced verdict only", "PASS", "claude (opus, high)", true, args, 1, func(r string) string { return "~~~\n" + r + "~~~\n" }},
		{"fenced example in findings", "PASS", "claude (opus, high)", true, args, 0, func(r string) string {
			return r + "\n```text\n## Verdict\n- **Verdict**: FAIL\n```\n"
		}},
	}
	// A blank duplicate of Stage, Verdict or Reviewer, before or after the populated one, is a finding.
	for _, field := range []string{"Stage", "Verdict", "Reviewer"} {
		populated, blank := "- **"+field+"**: ", "- **"+field+"**:\n"
		cases = append(cases,
			testCase{"blank " + field + " before", "PASS", "claude (opus, high)", true, args, 1, func(r string) string { return strings.Replace(r, populated, blank+populated, 1) }},
			testCase{"blank " + field + " after", "PASS", "claude (opus, high)", true, args, 1, func(r string) string {
				i := strings.Index(r, populated)
				eol := i + strings.Index(r[i:], "\n") + 1
				return r[:eol] + blank + r[eol:]
			}},
		)
	}
	for _, tc := range cases {
		write(tc.verdict, tc.reviewer, tc.approved, tc.edit)
		if code := runReview(root, tc.args); code != tc.want {
			t.Errorf("%s: exit %d, want %d", tc.name, code, tc.want)
		}
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Merge check runs on pull-request state events
func TestRequiredChecks(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want []string
	}{
		{"", nil},
		{" , ", nil},
		{"Required", []string{"Required"}},
		{"Required, Merge check ,", []string{"Required", "Merge check"}},
		{"CI / Required (pull_request),Merge check / Merge check (pull_request)", []string{"CI / Required (pull_request)", "Merge check / Merge check (pull_request)"}},
	} {
		if got := requiredChecks(tc.in); !slices.Equal(got, tc.want) {
			t.Errorf("requiredChecks(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
