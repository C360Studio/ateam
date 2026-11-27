package state

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/c360studio/ateam/internal/review"
)

func rejectAt(sha string) string {
	return "reviewed-by: codex (gpt-6-astra, xhigh)\nReviewed at " + sha + "\nVerdict: REQUEST CHANGES\nfindings"
}

// spec: model-review / Owner decision after the third rejecting review
func TestReviewRounds(t *testing.T) {
	approve := "reviewed-by: codex (gpt-6-astra, xhigh)\nReviewed at 3333333\nVerdict: APPROVE"
	noVerdict := "reviewed-by: codex (gpt-6-astra, xhigh)\nReviewed at 4444444\nlooks fine"
	for _, tc := range []struct {
		name    string
		bodies  []string
		shas    []string
		allowed int
	}{
		{"none", []string{"thanks", "Reviewed at 1111111 by nobody"}, nil, 0},
		// Only REQUEST CHANGES records count, oldest first; an approval or a record with no verdict between them does not.
		{"rejections only", []string{rejectAt("1111111"), approve, rejectAt("2222222"), noVerdict, rejectAt("5555555")}, []string{"1111111", "2222222", "5555555"}, 0},
		{"continue line", []string{rejectAt("1111111"), "Owner: go on.\nreview-rounds-continue: 3\n"}, []string{"1111111"}, 3},
		{"newest line wins", []string{"review-rounds-continue: 4", "  Review-Rounds-Continue:  3  "}, nil, 3},
		{"fenced line ignored", []string{"```\nreview-rounds-continue: 3\n```"}, nil, 0},
		{"tilde fence ignored", []string{"~~~text\nreview-rounds-continue: 3\n~~~"}, nil, 0},
		{"not a number", []string{"review-rounds-continue: three", "review-rounds-continue: 0", "review-rounds-continue: -3"}, nil, 0},
		{"mid-line mention ignored", []string{"post review-rounds-continue: 3 to go on"}, nil, 0},
		// A reviewer's record never lifts the stop on its own rounds.
		{"line inside a record ignored", []string{rejectAt("1111111") + "\nreview-rounds-continue: 1"}, []string{"1111111"}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var comments []forgeComment
			for i, b := range tc.bodies {
				comments = append(comments, forgeComment{Body: b, URL: "https://example.test/c/" + string(rune('a'+i))})
			}
			rounds, allowed := reviewRounds(comments)
			var shas []string
			for _, r := range rounds {
				if r.Verdict != "REQUEST CHANGES" || r.Agent != "codex" || !strings.HasPrefix(r.URL, "https://example.test/c/") {
					t.Fatalf("round = %+v, want a codex REQUEST CHANGES record with its link", r)
				}
				shas = append(shas, r.SHA)
			}
			if !reflect.DeepEqual(shas, tc.shas) || allowed != tc.allowed {
				t.Fatalf("rejections %v, continue %d; want %v, %d", shas, allowed, tc.shas, tc.allowed)
			}
		})
	}
}

// Gitea serves one page of comments on an unpaged call, so a long PR's third rejection sits on a later page.
//
// spec: model-review / Owner decision after the third rejecting review
func TestGiteaSyncPagesPRCommentsForRejections(t *testing.T) {
	root := reviewRepo(t)
	pages := map[string][]map[string]string{
		"":  {{"body": rejectAt("1111111")}, {"body": rejectAt("2222222")}},
		"1": {{"body": rejectAt("1111111")}, {"body": rejectAt("2222222")}},
		"2": {{"body": rejectAt("5555555"), "html_url": "https://gitea.test/o/r/pulls/7#issuecomment-3"}},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		switch {
		case strings.HasSuffix(r.URL.Path, "/issues/7/comments"):
			page := pages[q.Get("page")]
			if page == nil {
				page = []map[string]string{}
			}
			_ = json.NewEncoder(w).Encode(page)
		case strings.HasSuffix(r.URL.Path, "/pulls") && q.Get("state") == "open" && q.Get("page") == "1":
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"number": 7, "title": "PR 7", "state": "open", "comments": 3, "head": map[string]string{"ref": "claude/pr-7", "sha": "fedcba9"}},
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
	for _, c := range cards {
		if c.ID != "GT-PR-7" {
			continue
		}
		want := review.RecordComment{SHA: "5555555", Verdict: "REQUEST CHANGES", Agent: "codex", Model: "gpt-6-astra", Effort: "xhigh"}
		if len(c.Rejections) != 3 || c.Rejections[2].RecordComment != want || c.Rejections[2].URL != "https://gitea.test/o/r/pulls/7#issuecomment-3" {
			t.Fatalf("rejections = %+v, want three, the last %+v from page 2", c.Rejections, want)
		}
		if c.Review == nil || c.Review.SHA != "5555555" {
			t.Fatalf("newest record = %+v, want the one on page 2", c.Review)
		}
		return
	}
	t.Fatalf("no card for PR #7 in %+v", cards)
}
