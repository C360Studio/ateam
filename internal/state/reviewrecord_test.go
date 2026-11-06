package state

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/c360studio/ateam/internal/review"
)

const reviewRecordBody = "reviewed-by: codex (gpt-6-astra, xhigh)\nReviewed at 0123456\nVerdict: APPROVE\nfindings"

// reviewRepo is a repository with a linked worktree on claude/pr-7 and none on claude/pr-8.
func reviewRepo(t *testing.T) string {
	t.Helper()
	root, wt := t.TempDir(), filepath.Join(t.TempDir(), "pr-7")
	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "--allow-empty", "-m", "init"},
		{"worktree", "add", "-q", "-b", "claude/pr-7", wt},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	return root
}

func reviewOf(cards []BoardCard, id string) (*review.RecordComment, bool) {
	for _, c := range cards {
		if c.ID == id {
			return c.Review, true
		}
	}
	return nil, false
}

// spec: openspec/changes/review-launch-295/specs/harness/spec.md#Requirement: Review record in the forge cache and pickup
func TestGitHubSyncReadsReviewRecordForWorktreePRs(t *testing.T) {
	root := reviewRepo(t)
	comments, _ := json.Marshal(map[string]any{"comments": []map[string]string{{"body": "Reviewed at 89abcde by nobody"}, {"body": "reviewed-by: gemini (gemini-4-argon, high)\nReviewed at 89abcde\nVerdict: REQUEST CHANGES"}, {"body": reviewRecordBody}, {"body": "thanks!"}}})
	log := installForgeCLI(t, "gh", "case \"$*\" in\n"+
		"'pr list --state open'*) printf '%s' '[{\"number\":7,\"title\":\"PR 7\",\"state\":\"OPEN\",\"headRefName\":\"claude/pr-7\"},{\"number\":8,\"title\":\"PR 8\",\"state\":\"OPEN\",\"headRefName\":\"claude/pr-8\"}]';;\n"+
		"'pr view 7 --json comments'*) printf '%s' '"+string(comments)+"';;\n"+
		"'pr view'*) exit 1;;\n"+
		"*) printf '[]';;\nesac\n")
	cards, err := SyncForgeBoard(context.Background(), root, ForgeConfig{Provider: ForgeGitHub})
	if err != nil {
		t.Fatal(err)
	}
	want := review.RecordComment{SHA: "0123456", Verdict: "APPROVE", Agent: "codex", Model: "gpt-6-astra", Effort: "xhigh"}
	if got, ok := reviewOf(cards, "GH-PR-7"); !ok || got == nil || *got != want {
		t.Fatalf("PR #7 review = %+v (card found %t), want the newest record %+v", got, ok, want)
	}
	if got, ok := reviewOf(cards, "GH-PR-8"); !ok || got != nil {
		t.Fatalf("PR #8 has no local worktree: review = %+v (card found %t), want none", got, ok)
	}
	data, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "pr view 8") {
		t.Fatalf("sync fetched comments for a PR with no local worktree:\n%s", data)
	}
}

// spec: openspec/changes/review-launch-295/specs/harness/spec.md#Requirement: Review record in the forge cache and pickup
func TestGitHubSyncReviewBudgetExhausted(t *testing.T) {
	saved := ghCommentTimeout
	ghCommentTimeout = 200 * time.Millisecond
	t.Cleanup(func() { ghCommentTimeout = saved })
	root := reviewRepo(t)
	installForgeCLI(t, "gh", "case \"$*\" in\n"+
		"'pr list --state open'*) printf '%s' '[{\"number\":7,\"title\":\"PR 7\",\"state\":\"OPEN\",\"headRefName\":\"claude/pr-7\"}]';;\n"+
		"'pr view 7'*) sleep 3; printf '%s' '{\"comments\":[{\"body\":\"reviewed-by: codex (x, high)\\nReviewed at 0123456\"}]}';;\n"+
		"*) printf '[]';;\nesac\n")
	cards, err := SyncForgeBoard(context.Background(), root, ForgeConfig{Provider: ForgeGitHub})
	if err != nil {
		t.Fatalf("sync failed when the review comments call ran out of budget: %v", err)
	}
	if got, ok := reviewOf(cards, "GH-PR-7"); !ok || got != nil {
		t.Fatalf("PR #7 review = %+v (card found %t), want none after the budget ran out", got, ok)
	}
}

// spec: openspec/changes/review-launch-295/specs/harness/spec.md#Requirement: Review record in the forge cache and pickup
func TestGiteaSyncReadsReviewRecordForWorktreePRs(t *testing.T) {
	root := reviewRepo(t)
	var mu sync.Mutex
	var fetched []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		switch {
		case strings.HasSuffix(r.URL.Path, "/comments"):
			mu.Lock()
			fetched = append(fetched, r.URL.Path)
			mu.Unlock()
			_ = json.NewEncoder(w).Encode([]map[string]string{{"body": "first"}, {"body": reviewRecordBody}})
		case strings.HasSuffix(r.URL.Path, "/pulls") && q.Get("state") == "open" && q.Get("page") == "1":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"number": 7, "title": "PR 7", "state": "open", "head": map[string]string{"ref": "claude/pr-7"}},
				{"number": 8, "title": "PR 8", "state": "open", "head": map[string]string{"ref": "claude/pr-8"}},
			})
		default:
			_ = json.NewEncoder(w).Encode([]map[string]any{})
		}
	}))
	t.Cleanup(srv.Close)
	cards, err := SyncForgeBoard(context.Background(), root, ForgeConfig{Provider: ForgeGitea, BaseURL: srv.URL, Owner: "o", Repo: "r"})
	if err != nil {
		t.Fatal(err)
	}
	want := review.RecordComment{SHA: "0123456", Verdict: "APPROVE", Agent: "codex", Model: "gpt-6-astra", Effort: "xhigh"}
	if got, ok := reviewOf(cards, "GT-PR-7"); !ok || got == nil || *got != want {
		t.Fatalf("PR #7 review = %+v (card found %t), want %+v", got, ok, want)
	}
	if got, _ := reviewOf(cards, "GT-PR-8"); got != nil {
		t.Fatalf("PR #8 has no local worktree: review = %+v", got)
	}
	if len(fetched) != 1 || fetched[0] != "/api/v1/repos/o/r/issues/7/comments" {
		t.Fatalf("comment fetches = %v, want only PR #7's issue comments", fetched)
	}
}

// spec: openspec/changes/review-launch-295/specs/harness/spec.md#Requirement: Review record in the forge cache and pickup
func TestPickupWorktreeReviewRecord(t *testing.T) {
	const head = "0123456789abcdef0123456789abcdef01234567"
	rec := &review.RecordComment{SHA: "0123456", Verdict: "APPROVE", Agent: "codex", Model: "gpt-6-astra", Effort: "xhigh"}
	for _, tc := range []struct {
		name   string
		review *review.RecordComment
		head   string
		suffix string
	}{
		{"present", rec, head, "open PR #7); review record at 0123456, APPROVE, codex (gpt-6-astra, xhigh)"},
		{"stale", rec, "fedcba9876543210fedcba9876543210fedcba98", "open PR #7); review record at 0123456, APPROVE, codex (gpt-6-astra, xhigh), stale"},
		{"absent", nil, head, "open PR #7)"},
		{"no-verdict-anonymous", &review.RecordComment{SHA: "0123456"}, head, "open PR #7); review record at 0123456, no verdict, unnamed agent"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st := defaultState()
			st.Git.Worktrees = []Worktree{{Path: "/tmp/pr-7", Head: tc.head, Branch: "claude/pr-7", AheadCount: 1}}
			st.Cards = []BoardCard{{ID: "GH-PR-7", Number: 7, Type: "pr", Column: "in-review", Branch: "claude/pr-7", Forge: "github", Review: tc.review}}
			var line string
			for _, l := range strings.Split(BuildPickupSummary(st), "\n") {
				if strings.HasPrefix(l, "  - claude/pr-7 ") {
					line = l
				}
			}
			if !strings.HasSuffix(line, tc.suffix) || (tc.review == nil && strings.Contains(line, "review record")) {
				t.Fatalf("worktree line = %q, want it to end with %q", line, tc.suffix)
			}
		})
	}
}
