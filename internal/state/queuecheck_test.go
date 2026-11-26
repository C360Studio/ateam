package state

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func withLabels(labels ...string) func(*BoardCard) {
	return func(c *BoardCard) { c.Labels = append(c.Labels, labels...) }
}

func dependsOn(nums ...int) func(*BoardCard) {
	return func(c *BoardCard) { c.DependsOn = nums }
}

// Each case's want is written by hand from the rule in #416 design D2, never derived from computeQueue. Every
// milestone in these cases is ordered and holds a few open issues, so only the rule under test fires.
// spec: harness / Queue health checks
func TestComputeQueueRules(t *testing.T) {
	const desc = "Order: #1 -> #2 -> #3"
	calm := &MilestoneSnapshot{Items: []MilestoneCount{{Title: "A", OpenIssues: 3, ClosedIssues: 1}}}
	unread := func(c *BoardCard) { c.CommentsUnread = true }
	closed := func(c *BoardCard) { c.Column = "done" }
	decided := func(c *BoardCard) { c.Decision = "Decision: ship it" }
	cases := []struct {
		name  string
		cards []BoardCard
		ms    *MilestoneSnapshot
		want  []QueueFinding
	}{
		{
			name:  "Q1 hit: every blocker closed",
			cards: []BoardCard{issue(1, "A", desc, withLabels("status:blocked"), dependsOn(5)), issue(5, "A", desc, closed)},
			ms:    calm,
			want:  []QueueFinding{{Rule: "Q1", Issue: 1, Message: "status:blocked, but every Depends on: issue (#5) is closed"}},
		},
		{
			name:  "Q1 miss: one blocker still open",
			cards: []BoardCard{issue(1, "A", desc, withLabels("status:blocked"), dependsOn(5, 2)), issue(2, "A", desc)},
			ms:    calm,
		},
		{
			name:  "Q1 miss: the blocker is an open PR",
			cards: []BoardCard{issue(1, "A", desc, withLabels("status:blocked"), dependsOn(9)), pr(9)},
			ms:    calm,
		},
		{
			name:  "Q2 hit: blocked with no Depends on line",
			cards: []BoardCard{issue(1, "A", desc, withLabels("status:blocked"))},
			ms:    calm,
			want:  []QueueFinding{{Rule: "Q2", Issue: 1, Message: "status:blocked, but names no readable blocker (no Depends on: #n line)"}},
		},
		{
			name:  "Q2 miss: not blocked and no Depends on",
			cards: []BoardCard{issue(1, "A", desc)},
			ms:    calm,
		},
		{
			name:  "unread comments replace Q1 and Q2",
			cards: []BoardCard{issue(1, "A", desc, withLabels("status:blocked"), unread)},
			ms:    calm,
			want:  []QueueFinding{{Rule: "unread", Issue: 1, Message: "status:blocked, but its comments were not read, so its blocker is unknown"}},
		},
		{
			name:  "Q3 hit: needs-decision with a recorded decision",
			cards: []BoardCard{issue(1, "A", desc, withLabels("status:needs-decision"), decided)},
			ms:    calm,
			want:  []QueueFinding{{Rule: "Q3", Issue: 1, Message: "status:needs-decision, but a decision is recorded: Decision: ship it"}},
		},
		{
			name:  "Q3 miss: needs-decision with none recorded",
			cards: []BoardCard{issue(1, "A", desc, withLabels("status:needs-decision"))},
			ms:    calm,
		},
		{
			name:  "Q4 hit: no milestone, and missing from Order",
			cards: []BoardCard{issue(1, "A", desc), issue(7, "", ""), issue(8, "A", desc)},
			ms:    calm,
			want: []QueueFinding{
				{Rule: "Q4", Issue: 7, Message: "unplaced: no milestone, or missing from its milestone's Order: line"},
				{Rule: "Q4", Issue: 8, Message: "unplaced: no milestone, or missing from its milestone's Order: line"},
			},
		},
		{
			name:  "Q5 hit: over the threshold",
			cards: []BoardCard{issue(1, "A", desc)},
			ms:    &MilestoneSnapshot{Items: []MilestoneCount{{Title: "A", OpenIssues: 16}}},
			want:  []QueueFinding{{Rule: "Q5", Milestone: "A", Message: "16 open issues, over queue.max_milestone_issues 15"}},
		},
		{
			name:  "Q5 miss: exactly at the threshold",
			cards: []BoardCard{issue(1, "A", desc)},
			ms:    &MilestoneSnapshot{Items: []MilestoneCount{{Title: "A", OpenIssues: 15}}},
		},
		{
			name:  "Q6 hit: open with no open issues",
			cards: []BoardCard{issue(1, "A", desc)},
			ms:    &MilestoneSnapshot{Items: []MilestoneCount{{Title: "A", OpenIssues: 1}, {Title: "Done", OpenIssues: 0, ClosedIssues: 4}}},
			want:  []QueueFinding{{Rule: "Q6", Milestone: "Done", Message: "open, with no open issues left: finished, not closed"}},
		},
		{
			name:  "closed issues are never findings",
			cards: []BoardCard{issue(1, "A", desc, withLabels("status:blocked"), closed), issue(7, "", "", closed)},
			ms:    calm,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := computeQueue(tc.cards, "github", tc.ms, DefaultQueueMaxMilestoneIssues)
			if got.Unavailable != "" || !reflect.DeepEqual(got.Findings, tc.want) {
				t.Errorf("computeQueue findings =\n%+v (unavailable %q)\nwant\n%+v", got.Findings, got.Unavailable, tc.want)
			}
		})
	}
}

// spec: harness / Queue health checks
func TestQueueUnavailable(t *testing.T) {
	if got := (&State{Forge: ForgeConfig{Provider: ForgeGitHub}}).ComputeQueue(); got.Unavailable != NextNotSynced || got.Field() != "unavailable" {
		t.Errorf("unsynced queue = %+v field %q, want not synced and unavailable", got, got.Field())
	}
	cards := []BoardCard{issue(7, "", "")}
	// Never read: the issue rules still run, but the count is not shown as if Q5 and Q6 were clean.
	if got := computeQueue(cards, "github", nil, 15); got.Unavailable != "milestones not read" || got.Field() != "unavailable" || len(got.Findings) != 1 {
		t.Errorf("no milestone snapshot = %+v field %q, want unavailable with the Q4 finding kept", got, got.Field())
	}
	st := &State{}
	st.SetForgeMilestones(ForgeGitHub, []MilestoneCount{{Title: "A", OpenIssues: 0}}, nil)
	st.SetForgeMilestones(ForgeGitHub, nil, errors.New("gh api milestones: exit status 1"))
	if got := computeQueue(cards, "github", st.ForgeMilestones[ForgeGitHub], 15); got.Unavailable != "milestones not read: gh api milestones: exit status 1" {
		t.Errorf("an unsuccessful read kept the older list: %+v", got)
	}
}

// spec: harness / Queue health checks
func TestQueueSurfaces(t *testing.T) {
	st := &State{Forge: ForgeConfig{Provider: ForgeGitHub}, QueueMaxMilestoneIssues: 2}
	st.MergeForgeCards([]BoardCard{issue(1, "A", "Order: #1"), issue(7, "", "")}, ForgeGitHub)
	if got := BuildPickupSummary(st); !strings.Contains(got, "| next=#1 | queue=unavailable | build=") {
		t.Fatalf("summary without milestones:\n%s", got)
	}
	st.SetForgeMilestones(ForgeGitHub, []MilestoneCount{{Title: "A", OpenIssues: 3}}, nil)
	if got := BuildPickupSummary(st); !strings.Contains(got, "| next=#1 | queue=2 | build=") {
		t.Fatalf("summary with a Q4 and a Q5 (threshold 2) finding:\n%s", got)
	}
	res := st.ComputeQueue()
	if got, want := res.Format(), "Q4 #7 unplaced: no milestone, or missing from its milestone's Order: line\nQ5 A 3 open issues, over queue.max_milestone_issues 2\n"; got != want {
		t.Errorf("Format =\n%q\nwant\n%q", got, want)
	}
	if got, want := res.DoctorLine(), "queue: 2 finding(s); run ateam state queue-check"; got != want {
		t.Errorf("DoctorLine = %q, want %q", got, want)
	}
	if got, want := (QueueResult{}).Format(), "queue: no findings\n"; got != want {
		t.Errorf("empty Format = %q, want %q", got, want)
	}
	if got, want := (QueueResult{Unavailable: "not synced"}).DoctorLine(), "queue: unavailable (not synced); run ateam state queue-check"; got != want {
		t.Errorf("unavailable DoctorLine = %q, want %q", got, want)
	}
}

// spec: harness / Queue health checks
func TestQueueSettings(t *testing.T) {
	cases := []struct {
		json    string
		want    int
		wantErr string
	}{
		{json: `{"version":1}`, want: 15},
		{json: `{"version":1,"queue":{"max_milestone_issues":8}}`, want: 8},
		{json: `{"version":1,"queue":{"max_milestone_issues":0}}`, wantErr: "queue.max_milestone_issues must be a positive integer, got 0"},
		{json: `{"version":1,"queue":{"max_milestone_issues":-3}}`, wantErr: "queue.max_milestone_issues must be a positive integer, got -3"},
		{json: `{"version":1,"queue":{"max_milestone_issues":"9"}}`, wantErr: "cannot unmarshal string"},
		{json: `{"version":1,"queue":{"max_open":9}}`, wantErr: `unknown field "max_open"`},
	}
	for _, tc := range cases {
		s, err := parseSettings([]byte(tc.json))
		if tc.wantErr != "" {
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("parseSettings(%s) error = %v, want %q", tc.json, err, tc.wantErr)
			}
			continue
		}
		if err != nil || s.queueMaxMilestoneIssues() != tc.want {
			t.Errorf("parseSettings(%s) = %d, %v; want %d", tc.json, s.queueMaxMilestoneIssues(), err, tc.want)
		}
	}
	// Refresh copies the committed value into the cache, so the hook applies it without reading the file.
	root := t.TempDir()
	if err := os.MkdirAll(root+"/.agents", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(root+"/"+SettingsPath, []byte(`{"version":1,"queue":{"max_milestone_issues":4}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	st, err := Refresh(context.Background(), root, false)
	if err != nil {
		t.Fatal(err)
	}
	if st.QueueMaxMilestoneIssues != 4 {
		t.Errorf("Refresh cached queue_max_milestone_issues = %d, want 4", st.QueueMaxMilestoneIssues)
	}
}

// spec: harness / Queue health checks
func TestGitHubMilestonesRead(t *testing.T) {
	log := installForgeCLI(t, "gh", `case "$*" in
api*milestones*) printf '%s\n%s\n' '{"title":"A","open_issues":3,"closed_issues":1}' '{"title":"Done","open_issues":0,"closed_issues":4}'; exit 0;;
esac
exit 1
`)
	got, err := SyncForgeMilestones(context.Background(), t.TempDir(), ForgeConfig{Provider: ForgeGitHub, Owner: "o", Repo: "r"})
	if err != nil {
		t.Fatal(err)
	}
	want := []MilestoneCount{{Title: "A", OpenIssues: 3, ClosedIssues: 1}, {Title: "Done", OpenIssues: 0, ClosedIssues: 4}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("milestones = %+v, want %+v", got, want)
	}
	_, calls := readListCalls(t, log)
	if len(calls) != 1 || !strings.Contains(calls[0], "api --paginate repos/o/r/milestones?state=open&per_page=100") {
		t.Errorf("gh calls = %q, want one paginated open-milestones read of o/r", calls)
	}
	installForgeCLI(t, "gh", "exit 1\n")
	if _, err := SyncForgeMilestones(context.Background(), t.TempDir(), ForgeConfig{Provider: ForgeGitHub, Owner: "o", Repo: "r"}); err == nil {
		t.Error("a gh failure returned no error, so the queue would read it as zero milestones")
	}
}

// spec: harness / Queue health checks
func TestGiteaMilestonesRead(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/repos/c360/ateam/milestones" || r.URL.Query().Get("state") != "open" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		switch r.URL.Query().Get("page") {
		case "1":
			_ = json.NewEncoder(w).Encode([]map[string]any{{"title": "A", "open_issues": 2, "closed_issues": 5}})
		case "2":
			_ = json.NewEncoder(w).Encode([]map[string]any{{"title": "B", "open_issues": 0, "closed_issues": 1}})
		default:
			_ = json.NewEncoder(w).Encode([]map[string]any{})
		}
	}))
	t.Cleanup(srv.Close)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cfg := ForgeConfig{Provider: ForgeGitea, BaseURL: srv.URL, Owner: "c360", Repo: "ateam"}
	got, err := SyncForgeMilestones(ctx, t.TempDir(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	want := []MilestoneCount{{Title: "A", OpenIssues: 2, ClosedIssues: 5}, {Title: "B", OpenIssues: 0, ClosedIssues: 1}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("milestones = %+v, want %+v (both pages)", got, want)
	}
	if _, err := SyncForgeMilestones(ctx, t.TempDir(), ForgeConfig{Provider: ForgeGitea}); err == nil {
		t.Error("the tea fallback returned milestones, want unavailable")
	}
	srv.Close()
	if _, err := SyncForgeMilestones(ctx, t.TempDir(), cfg); err == nil {
		t.Error("an unreachable Gitea returned no error, so the queue would read it as zero milestones")
	}
}
