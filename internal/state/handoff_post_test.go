package state

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const handoffPostBody = "Goal: g\nAddresses: none\nConstraints: none\nDone when: none\nChecks to run: task verify\n"

// handoffPostCases are the PR comment threads a handoff post may meet before it sends (#320). Each comment's link
// is its index; existing names the comment whose link a skipped post returns.
var handoffPostCases = []struct {
	name     string
	comments []string // oldest first; nil with failRead means the comment read fails
	failRead bool
	existing int // index of the matching comment, or -1 when the post must send
}{
	{name: "no comments", existing: -1},
	{name: "newest handoff has the same body", comments: []string{"Goal: old\nChecks to run: x", handoffPostBody}, existing: 1},
	{name: "same body with CRLF and padding", comments: []string{"\n" + strings.ReplaceAll(handoffPostBody, "\n", "\r\n") + "  "}, existing: 0},
	{name: "a later non-handoff comment does not hide the match", comments: []string{handoffPostBody, "reviewed-by: codex (m, high)\nVerdict: APPROVE"}, existing: 0},
	{name: "newest handoff differs", comments: []string{"Goal: other\nChecks to run: task verify"}, existing: -1},
	{name: "an older equal handoff before a different one", comments: []string{handoffPostBody, "Goal: B\nChecks to run: task verify"}, existing: -1},
	{name: "comment read fails", failRead: true, existing: -1},
}

func handoffComments(t *testing.T, bodies []string) []forgeComment {
	t.Helper()
	out := make([]forgeComment, len(bodies))
	for i, b := range bodies {
		out[i] = forgeComment{Body: b, HTMLURL: "https://forge.example/c/" + string(rune('0'+i))}
	}
	return out
}

// spec: openspec/specs/harness/spec.md#Requirement: Durable shared state and five-field handoff
func TestPostPRHandoffCommentSkipsARepeatOfTheNewestHandoff(t *testing.T) {
	for _, tc := range handoffPostCases {
		t.Run("github/"+tc.name, func(t *testing.T) {
			raw, _ := json.Marshal(map[string]any{"comments": handoffComments(t, tc.comments)})
			t.Setenv("FAKE_COMMENTS", string(raw))
			if tc.failRead {
				t.Setenv("FAKE_COMMENTS", "fail")
			}
			log := installForgeCLI(t, "gh", "case \"$*\" in\n"+
				"'pr view claude/x --json comments') [ \"$FAKE_COMMENTS\" = fail ] && exit 1; printf '%s' \"$FAKE_COMMENTS\";;\n"+
				"'pr comment'*) printf '%s\\n' 'https://forge.example/new';;\n"+
				"'pr view claude/x --json number,headRefOid,url') printf '%s' '{\"number\":7,\"headRefOid\":\""+headFull+"\",\"url\":\"https://forge.example/pr/7\"}';;\n"+
				"*) exit 1;;\nesac\n")
			post, err := PostPRHandoffComment(context.Background(), t.TempDir(), ForgeConfig{Provider: ForgeGitHub}, "claude/x", handoffPostBody)
			data, _ := os.ReadFile(log)
			sent := strings.Count(string(data), "pr comment claude/x")
			checkHandoffPost(t, post, err, sent, tc.existing, "https://forge.example/new")
			// The PR head is read either way, so commit statuses still run on a repeat.
			if post.Number != 7 || post.HeadSHA != headFull {
				t.Fatalf("post = %+v, want PR #7 at its head", post)
			}
		})
		t.Run("gitea/"+tc.name, func(t *testing.T) {
			var sent atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repos/o/r/pulls" && r.URL.Query().Get("page") == "1":
					_, _ = w.Write([]byte(`[{"number":7,"state":"open","html_url":"https://forge.example/pr/7","head":{"ref":"claude/x","sha":"` + headFull + `"}}]`))
				case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repos/o/r/issues/7/comments":
					if tc.failRead {
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					_ = json.NewEncoder(w).Encode(handoffComments(t, tc.comments))
				case r.Method == http.MethodPost && r.URL.Path == "/api/v1/repos/o/r/issues/7/comments":
					sent.Add(1)
					w.WriteHeader(http.StatusCreated)
					_, _ = w.Write([]byte(`{"html_url":"https://forge.example/new"}`))
				default:
					_, _ = w.Write([]byte(`[]`))
				}
			}))
			defer srv.Close()
			post, err := PostPRHandoffComment(context.Background(), t.TempDir(), ForgeConfig{Provider: ForgeGitea, BaseURL: srv.URL, Owner: "o", Repo: "r"}, "claude/x", handoffPostBody)
			checkHandoffPost(t, post, err, int(sent.Load()), tc.existing, "https://forge.example/new")
		})
	}
}

func checkHandoffPost(t *testing.T, post HandoffPost, err error, sent, existing int, newURL string) {
	t.Helper()
	if err != nil || !post.Posted {
		t.Fatalf("post = %+v, %v; want posted", post, err)
	}
	if existing < 0 {
		if sent != 1 || post.Existing || post.CommentURL != newURL {
			t.Fatalf("sent %d, post = %+v; want one send linking %s", sent, post, newURL)
		}
		return
	}
	want := "https://forge.example/c/" + string(rune('0'+existing))
	if sent != 0 || !post.Existing || post.CommentURL != want {
		t.Fatalf("sent %d, post = %+v; want no send, Existing, linking %s", sent, post, want)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Durable shared state and five-field handoff
func TestPostPRHandoffCommentReportsATimedOutSendAsUncertain(t *testing.T) {
	old := handoffPostTimeout
	handoffPostTimeout = 300 * time.Millisecond
	t.Cleanup(func() { handoffPostTimeout = old })

	// Only a send that started and then ran out of time is uncertain; a deadline or cancel that stopped the
	// send from starting sent nothing (#330 review).
	cases := []struct {
		name      string
		parent    string // "", "expired", "cancelled", or "deadline" (300ms, spent by a hanging comment read)
		readHang  bool
		sendHang  bool
		rejected  bool
		uncertain bool
		sent      bool
	}{
		{name: "send outlives the deadline", sendHang: true, uncertain: true, sent: true},
		{name: "send rejected", rejected: true, sent: true},
		{name: "parent already expired", parent: "expired"},
		{name: "parent already cancelled", parent: "cancelled"},
		{name: "parent deadline spent by the comment read", parent: "deadline", readHang: true},
	}
	parentCtx := func(t *testing.T, kind string) context.Context {
		t.Helper()
		switch kind {
		case "expired":
			ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
			t.Cleanup(cancel)
			return ctx
		case "cancelled":
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			return ctx
		case "deadline":
			ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
			t.Cleanup(cancel)
			return ctx
		}
		return context.Background()
	}
	check := func(t *testing.T, err error, sent bool, uncertain, wantSent bool) {
		t.Helper()
		if err == nil || errors.Is(err, ErrPostUncertain) != uncertain || sent != wantSent {
			t.Fatalf("err = %v, sent = %v; want an error with uncertain=%v, sent=%v", err, sent, uncertain, wantSent)
		}
	}
	for _, tc := range cases {
		t.Run("github/"+tc.name, func(t *testing.T) {
			read, send := "exit 1", "printf '%s\\n' https://forge.example/new"
			if tc.readHang {
				read = "exec sleep 5"
			}
			if tc.sendHang {
				send = "exec sleep 5"
			}
			if tc.rejected {
				send = "echo 'HTTP 422' >&2; exit 1"
			}
			log := installForgeCLI(t, "gh", "case \"$*\" in\n'pr view claude/x --json comments') "+read+";;\n'pr comment'*) "+send+";;\n*) exit 1;;\nesac\n")
			_, err := PostPRHandoffComment(parentCtx(t, tc.parent), t.TempDir(), ForgeConfig{Provider: ForgeGitHub}, "claude/x", handoffPostBody)
			data, _ := os.ReadFile(log)
			check(t, err, strings.Contains(string(data), "pr comment claude/x"), tc.uncertain, tc.sent)
		})
		t.Run("gitea/"+tc.name, func(t *testing.T) {
			var sent atomic.Bool
			hang := func(r *http.Request) {
				select {
				case <-r.Context().Done():
				case <-time.After(5 * time.Second):
				}
			}
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repos/o/r/pulls" && r.URL.Query().Get("page") == "1":
					_, _ = w.Write([]byte(`[{"number":7,"state":"open","head":{"ref":"claude/x","sha":"` + headFull + `"}}]`))
				case r.Method == http.MethodGet && r.URL.Path == "/api/v1/repos/o/r/issues/7/comments" && tc.readHang:
					hang(r)
				case r.Method == http.MethodPost:
					sent.Store(true)
					if tc.sendHang {
						hang(r)
						return
					}
					w.WriteHeader(http.StatusUnprocessableEntity)
				default:
					_, _ = w.Write([]byte(`[]`))
				}
			}))
			defer srv.Close()
			_, err := PostPRHandoffComment(parentCtx(t, tc.parent), t.TempDir(), ForgeConfig{Provider: ForgeGitea, BaseURL: srv.URL, Owner: "o", Repo: "r"}, "claude/x", handoffPostBody)
			check(t, err, sent.Load(), tc.uncertain, tc.sent)
		})
	}
}
