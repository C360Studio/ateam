package state

import (
	"context"
	"encoding/json"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

// issue and pr build open cached cards on the GitHub forge; the oracle in each case below is the expected answer
// written by hand from the scenario, not derived from computeNext.
func issue(n int, milestone, desc string, opts ...func(*BoardCard)) BoardCard {
	c := BoardCard{ID: "GH-" + itoa(n), Number: n, Title: "issue " + itoa(n), Column: "backlog", Type: "issue", Forge: "github", Milestone: milestone, MilestoneDescription: desc}
	for _, o := range opts {
		o(&c)
	}
	return c
}

func pr(n int, claims ...int) BoardCard {
	return BoardCard{ID: "GH-PR-" + itoa(n), Number: n, Title: "pr", Column: "in-progress", Type: "pr", Forge: "github", Claims: claims}
}

func itoa(n int) string { b, _ := json.Marshal(n); return string(b) }

func closed(c *BoardCard)             { c.Column = "done" }
func label(l string) func(*BoardCard) { return func(c *BoardCard) { c.Labels = append(c.Labels, l) } }
func deps(n ...int) func(*BoardCard)  { return func(c *BoardCard) { c.DependsOn = n } }
func unread(c *BoardCard)             { c.CommentsUnread = true }
func otherForge(c *BoardCard)         { c.Forge = "gitea" }

const descA = "First things. Order: #1 -> #2 -> #3"

func nextCases() []struct {
	name  string
	cards []BoardCard
	want  NextResult
} {
	return []struct {
		name  string
		cards []BoardCard
		want  NextResult
	}{
		{
			name:  "claimed and dependent issues are skipped",
			cards: []BoardCard{issue(1, "A", descA), issue(2, "A", descA, deps(1)), issue(3, "A", descA), pr(9, 1)},
			want: NextResult{Milestone: "A", Number: 3, Title: "issue 3", Skipped: []NextSkip{
				{Number: 1, Reason: "claimed by #9"}, {Number: 2, Reason: "waits on #1"}}},
		},
		{
			name:  "a closed dependency does not hold an issue back",
			cards: []BoardCard{issue(2, "A", descA, deps(1)), issue(1, "A", descA, closed)},
			want:  NextResult{Milestone: "A", Number: 2, Title: "issue 2", Skipped: []NextSkip{{Number: 1, Reason: "closed"}}},
		},
		{
			name: "decision and blocked labels are skipped",
			cards: []BoardCard{issue(1, "A", descA, label("status:needs-decision")), issue(2, "A", descA, label("status:blocked")),
				issue(3, "A", descA)},
			want: NextResult{Milestone: "A", Number: 3, Title: "issue 3", Skipped: []NextSkip{
				{Number: 1, Reason: "status:needs-decision"}, {Number: 2, Reason: "status:blocked"}}},
		},
		{
			name:  "an ordered issue moved to another milestone is skipped",
			cards: []BoardCard{issue(1, "A", descA, closed), issue(2, "B", ""), issue(3, "A", descA)},
			want: NextResult{Milestone: "A", Number: 3, Title: "issue 3", Skipped: []NextSkip{
				{Number: 1, Reason: "closed"}, {Number: 2, Reason: "in milestone B"}}, NoOrder: []string{"B"}},
		},
		{
			// Review of PR #419: A keeps its description through unplaced #9 after its last ordered issue moved to B.
			name:  "moving the last ordered issue finishes its old milestone",
			cards: []BoardCard{issue(9, "A", "Order: #1"), issue(1, "B", "After: A\nOrder: #1")},
			want:  NextResult{Milestone: "B", Number: 1, Title: "issue 1", Unplaced: []int{9}},
		},
		{
			name:  "nothing startable stops in the milestone",
			cards: []BoardCard{issue(1, "A", descA, label("status:blocked")), issue(4, "B", "After: A\nOrder: #4")},
			want: NextResult{Milestone: "A", None: "all of A skipped", Skipped: []NextSkip{
				{Number: 1, Reason: "status:blocked"}, {Number: 2, Reason: "closed"}, {Number: 3, Reason: "closed"}}},
		},
		{
			name:  "After: orders milestones",
			cards: []BoardCard{issue(4, "B", "After: A\nOrder: #4"), issue(2, "A", descA)},
			want:  NextResult{Milestone: "A", Number: 2, Title: "issue 2", Skipped: []NextSkip{{Number: 1, Reason: "closed"}}},
		},
		{
			name:  "a finished milestone drops out",
			cards: []BoardCard{issue(4, "B", "After: A\nOrder: #4"), issue(1, "A", descA, closed), issue(5, "A", descA)},
			want:  NextResult{Milestone: "B", Number: 4, Title: "issue 4", Unplaced: []int{5}},
		},
		{
			name:  "two roots are ambiguous",
			cards: []BoardCard{issue(1, "A", descA), issue(4, "B", "Order: #4")},
			want:  NextResult{None: NextAmbiguous},
		},
		{
			name:  "After: loops are a cycle",
			cards: []BoardCard{issue(1, "A", "After: B\n"+descA), issue(4, "B", "After: A\nOrder: #4")},
			want:  NextResult{None: NextCycle},
		},
		{
			name:  "a milestone after itself is a cycle",
			cards: []BoardCard{issue(1, "A", "After: a\n"+descA)},
			want:  NextResult{None: NextCycle},
		},
		{
			name:  "no Order: line anywhere",
			cards: []BoardCard{issue(1, "A", "scheduled after v0.4"), issue(2, "", "")},
			want:  NextResult{None: NextNoOrdered, Unplaced: []int{2}, NoOrder: []string{"A"}},
		},
		{
			name:  "unread comments name nothing",
			cards: []BoardCard{issue(1, "A", descA, unread)},
			want:  NextResult{Milestone: "A", None: NextDepsUnread},
		},
		{
			name:  "unplaced issues are listed and never named",
			cards: []BoardCard{issue(7, "A", descA), issue(8, "", ""), issue(2, "A", descA)},
			want:  NextResult{Milestone: "A", Number: 2, Title: "issue 2", Skipped: []NextSkip{{Number: 1, Reason: "closed"}}, Unplaced: []int{7, 8}},
		},
		{
			name:  "other forges are ignored",
			cards: []BoardCard{issue(1, "A", descA, otherForge), issue(2, "A", descA)},
			want:  NextResult{Milestone: "A", Number: 2, Title: "issue 2", Skipped: []NextSkip{{Number: 1, Reason: "closed"}}},
		},
	}
}

// spec: harness / Shared computed next issue
func TestComputeNextReasons(t *testing.T) {
	for _, tc := range nextCases() {
		t.Run(tc.name, func(t *testing.T) {
			if got := computeNext(tc.cards, "github"); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("computeNext =\n%+v\nwant\n%+v", got, tc.want)
			}
		})
	}
}

// spec: harness / Shared computed next issue
func TestComputeNextIgnoresCardOrder(t *testing.T) {
	// Two sessions reading one sync must get one answer whatever order the cache holds the cards in.
	rng := rand.New(rand.NewSource(418))
	for _, tc := range nextCases() {
		want := computeNext(tc.cards, "github")
		for range 50 {
			shuffled := append([]BoardCard(nil), tc.cards...)
			rng.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
			if got := computeNext(shuffled, "github"); !reflect.DeepEqual(got, want) {
				t.Fatalf("%s: shuffled result %+v, want %+v", tc.name, got, want)
			}
		}
	}
}

// spec: harness / Shared computed next issue
func TestNextParsers(t *testing.T) {
	if got, ok := parseOrder("Optional work, ahead of v0.4.7. Order: #418 -> #416 -> #287"); !ok || !reflect.DeepEqual(got, []int{418, 416, 287}) {
		t.Errorf("mid-line Order: = %v, %v", got, ok)
	}
	if _, ok := parseOrder("in order: the first two"); ok {
		t.Error("prose 'order:' without an issue list read as an Order: line")
	}
	if got := parseAfter("Some prose.\nAfter: v0.4.6 Queue health and coordinator\nOrder: #1"); got != "v0.4.6 Queue health and coordinator" {
		t.Errorf("After: = %q", got)
	}
	if got := parseDependsOn("body without one", "Depends on: #418 (shared next). Related: #411.", "- **Depends on:** #5, #6"); !reflect.DeepEqual(got, []int{418, 5, 6}) {
		t.Errorf("Depends on: = %v, want [418 5 6]", got)
	}
	if got := parseDependsOn("This depends on: #3 in prose"); got != nil {
		t.Errorf("mid-line prose read as Depends on: %v", got)
	}
	if got := parseClaims("Closes #418\nfixes #2, resolved #3 and Addresses: #4; see #5"); !reflect.DeepEqual(got, []int{418, 2, 3, 4}) {
		t.Errorf("claims = %v, want [418 2 3 4]", got)
	}
}

// spec: harness / Shared computed next issue
func TestPickupLineShowsNext(t *testing.T) {
	st := &State{Forge: ForgeConfig{Provider: ForgeGitHub}}
	if got := BuildPickupSummary(st); !strings.Contains(got, "| holds=0 | next=none (not synced) | queue=unavailable | build=") {
		t.Fatalf("unsynced summary:\n%s", got)
	}
	st.MergeForgeCards([]BoardCard{issue(1, "A", descA), issue(4, "B", "Order: #4")}, ForgeGitHub)
	if got := BuildPickupSummary(st); !strings.Contains(got, "| next=none (milestone order ambiguous) |") {
		t.Fatalf("ambiguous summary:\n%s", got)
	}
	st.MergeForgeCards([]BoardCard{issue(1, "A", descA), issue(4, "B", "After: A\nOrder: #4")}, ForgeGitHub)
	if got := BuildPickupSummary(st); !strings.Contains(got, "| next=#1 |") {
		t.Fatalf("ordered summary:\n%s", got)
	}
	if got := st.ComputeNext().Format(); got != "milestone: A\nnext: #1 issue 1\n" {
		t.Fatalf("Format = %q", got)
	}
}

// spec: harness / Shared computed next issue
func TestGitHubSyncFillsNextFields(t *testing.T) {
	installForgeCLI(t, "gh", `case "$*" in
*"issue list --state open"*comments*) printf '[{"number":2,"title":"two","state":"OPEN","body":"","milestone":{"title":"A","description":"Order: #1 -> #2"},"comments":[{"body":"Depends on: #1"}]}]'; exit 0;;
*"pr list --state open"*) printf '[{"number":9,"title":"p","state":"OPEN","isDraft":true,"headRefName":"claude/x","body":"Closes #1"}]'; exit 0;;
esac
printf '[]'
`)
	cards, err := SyncForgeBoard(context.Background(), t.TempDir(), ForgeConfig{Provider: ForgeGitHub, Owner: "o", Repo: "r"})
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]BoardCard{}
	for _, c := range cards {
		byID[c.ID] = c
	}
	if c := byID["GH-2"]; c.MilestoneDescription != "Order: #1 -> #2" || !reflect.DeepEqual(c.DependsOn, []int{1}) || c.CommentsUnread {
		t.Errorf("issue card = %+v", c)
	}
	if c := byID["GH-PR-9"]; !reflect.DeepEqual(c.Claims, []int{1}) {
		t.Errorf("PR card claims = %v", c.Claims)
	}
}

// spec: harness / Shared computed next issue
func TestGitHubSyncWithoutCommentsMarksDependenciesUnread(t *testing.T) {
	installForgeCLI(t, "gh", `case "$*" in
*"issue list --state open"*comments*) exit 1;;
*"issue list --state open"*) printf '[{"number":2,"title":"two","state":"OPEN","body":"","milestone":{"title":"A","description":"Order: #2"}}]'; exit 0;;
esac
printf '[]'
`)
	cards, err := SyncForgeBoard(context.Background(), t.TempDir(), ForgeConfig{Provider: ForgeGitHub, Owner: "o", Repo: "r"})
	if err != nil {
		t.Fatal(err)
	}
	if len(cards) != 1 || !cards[0].CommentsUnread {
		t.Fatalf("cards = %+v, want issue #2 marked comments unread", cards)
	}
	if got := computeNext(cards, "github"); got.None != NextDepsUnread {
		t.Fatalf("next = %+v, want %q", got, NextDepsUnread)
	}
}

// spec: harness / Shared computed next issue
func TestGiteaSyncFillsNextFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		switch {
		case r.URL.Path == "/api/v1/repos/c360/ateam/issues/2/comments":
			_ = json.NewEncoder(w).Encode([]map[string]any{{"body": "Depends on: #1"}})
			return
		case r.URL.Path == "/api/v1/repos/c360/ateam/issues/3/comments":
			w.WriteHeader(http.StatusInternalServerError)
			return
		case strings.HasSuffix(r.URL.Path, "/issues") && q.Get("state") == "open" && q.Get("page") == "1":
			ms := map[string]any{"title": "A", "description": "Order: #2 -> #3"}
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"number": 2, "title": "two", "state": "open", "comments": 1, "milestone": ms},
				{"number": 3, "title": "three", "state": "open", "comments": 2, "milestone": ms},
			})
			return
		case strings.HasSuffix(r.URL.Path, "/pulls") && q.Get("state") == "open" && q.Get("page") == "1":
			_ = json.NewEncoder(w).Encode([]map[string]any{{"number": 9, "title": "p", "state": "open", "body": "Addresses #3", "head": map[string]any{"ref": "codex/x"}}})
			return
		}
		_ = json.NewEncoder(w).Encode([]map[string]any{})
	}))
	t.Cleanup(srv.Close)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cards, err := SyncForgeBoard(ctx, t.TempDir(), ForgeConfig{Provider: ForgeGitea, BaseURL: srv.URL, Owner: "c360", Repo: "ateam"})
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]BoardCard{}
	for _, c := range cards {
		byID[c.ID] = c
	}
	if c := byID["GT-2"]; c.MilestoneDescription != "Order: #2 -> #3" || !reflect.DeepEqual(c.DependsOn, []int{1}) || c.CommentsUnread {
		t.Errorf("issue #2 card = %+v", c)
	}
	if c := byID["GT-3"]; !c.CommentsUnread {
		t.Errorf("issue #3 with a failed comment fetch not marked unread: %+v", c)
	}
	if c := byID["GT-PR-9"]; !reflect.DeepEqual(c.Claims, []int{3}) {
		t.Errorf("PR claims = %v", c.Claims)
	}
}

// spec: harness / Shared computed next issue
func TestGiteaDependencyCommentsArePagedOrMarkedUnread(t *testing.T) {
	// Review of PR #419: a Depends on: line past the first comment page must be read, and a short or failed read
	// must leave the issue unread rather than look dependency-free.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		page := q.Get("page")
		switch {
		case r.URL.Path == "/api/v1/repos/c360/ateam/issues/2/comments" && page == "1":
			_ = json.NewEncoder(w).Encode([]map[string]any{{"body": "a"}, {"body": "b"}})
		case r.URL.Path == "/api/v1/repos/c360/ateam/issues/2/comments" && page == "2":
			_ = json.NewEncoder(w).Encode([]map[string]any{{"body": "Depends on: #1"}})
		case r.URL.Path == "/api/v1/repos/c360/ateam/issues/3/comments" && page == "1":
			_ = json.NewEncoder(w).Encode([]map[string]any{{"body": "a"}})
		case r.URL.Path == "/api/v1/repos/c360/ateam/issues/4/comments" && page == "1":
			_ = json.NewEncoder(w).Encode([]map[string]any{{"body": "a"}})
		case r.URL.Path == "/api/v1/repos/c360/ateam/issues/4/comments" && page == "2":
			w.WriteHeader(http.StatusInternalServerError)
		case strings.HasSuffix(r.URL.Path, "/issues") && q.Get("state") == "open" && page == "1":
			ms := map[string]any{"title": "A", "description": "Order: #2 -> #3 -> #4"}
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{"number": 2, "title": "two", "state": "open", "comments": 3, "milestone": ms},
				{"number": 3, "title": "three", "state": "open", "comments": 2, "milestone": ms},
				{"number": 4, "title": "four", "state": "open", "comments": 2, "milestone": ms},
			})
		default:
			_ = json.NewEncoder(w).Encode([]map[string]any{})
		}
	}))
	t.Cleanup(srv.Close)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cards, err := SyncForgeBoard(ctx, t.TempDir(), ForgeConfig{Provider: ForgeGitea, BaseURL: srv.URL, Owner: "c360", Repo: "ateam"})
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]BoardCard{}
	for _, c := range cards {
		byID[c.ID] = c
	}
	if c := byID["GT-2"]; !reflect.DeepEqual(c.DependsOn, []int{1}) || c.CommentsUnread {
		t.Errorf("issue #2 (dependency on page 2) = DependsOn %v unread %v, want [1] false", c.DependsOn, c.CommentsUnread)
	}
	if c := byID["GT-3"]; !c.CommentsUnread {
		t.Errorf("issue #3 (pages ran out before the count) unread = false, want true")
	}
	if c := byID["GT-4"]; !c.CommentsUnread {
		t.Errorf("issue #4 (page 2 failed) unread = false, want true")
	}
}
