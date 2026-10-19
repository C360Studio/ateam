package state

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// spec: openspec/specs/harness/spec.md#Requirement: Durable shared state and five-field handoff
func TestHandoffPickupRoundTrip(t *testing.T) {
	dir := t.TempDir()
	st, err := Load(dir)
	if err != nil {
		t.Fatalf("Load empty dir: %v", err)
	}
	if st.Mode != ModeFull {
		t.Fatalf("default mode = %q, want %q", st.Mode, ModeFull)
	}
	if len(st.Cards) != 0 {
		t.Fatalf("fresh state Cards count = %d, want 0 (no fabricated board cards)", len(st.Cards))
	}

	h := Handoff{
		Goal:          "PR #12: implement universal hooks and Gitea sync",
		Addresses:     "branch gemini/hooks, SHA abc1234",
		Constraints:   "Hold: none; write ownership transferred to codex",
		DoneWhen:      "task verify passes on clean tree",
		ChecksToRun:   "task test:unit && task spec:check",
		Agent:         "gemini",
		Branch:        "gemini/hooks",
		CheckpointSHA: "abc1234",
		UpdatedAt:     time.Now().UTC(),
	}
	st.LastHandoff = &h
	st.Mode = ModeUltra
	if err := Save(dir, st); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load saved: %v", err)
	}
	if loaded.Mode != ModeUltra {
		t.Fatalf("loaded mode = %q, want %q", loaded.Mode, ModeUltra)
	}
	if loaded.LastHandoff == nil || loaded.LastHandoff.Goal != h.Goal {
		t.Fatalf("loaded handoff mismatch: %+v", loaded.LastHandoff)
	}

	block := FormatFiveFieldPrompt(*loaded.LastHandoff)
	for _, want := range []string{
		"Goal: PR #12: implement universal hooks and Gitea sync",
		"Addresses: branch gemini/hooks, SHA abc1234",
		"Constraints: Hold: none; write ownership transferred to codex",
		"Done when: task verify passes on clean tree",
		"Checks to run: task test:unit && task spec:check",
	} {
		if !strings.Contains(block, want) {
			t.Errorf("FormatFiveFieldPrompt missing %q in:\n%s", want, block)
		}
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Inline shortcut inventory
func TestScanShortcuts(t *testing.T) {
	dir := t.TempDir()
	src := `package sample

// shortcut: in-memory cache only, upgrade to NATS KV when multi-node is enabled
func compute() int {
	return 42
}
`
	if err := os.WriteFile(filepath.Join(dir, "sample.go"), []byte(src), 0o644); err != nil {
		t.Fatalf("write sample.go: %v", err)
	}
	items, err := ScanShortcuts(dir)
	if err != nil {
		t.Fatalf("ScanShortcuts: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("got %d shortcuts, want 1", len(items))
	}
	if items[0].File != "sample.go" || items[0].Line != 3 {
		t.Errorf("unexpected location: %+v", items[0])
	}
	if items[0].Limit != "in-memory cache only" || items[0].RevisitTrigger != "upgrade to NATS KV when multi-node is enabled" {
		t.Errorf("unexpected parsed shortcut: %+v", items[0])
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Durable shared state and five-field handoff
func TestConcurrentUpdateAndCorruptCachePreservation(t *testing.T) {
	dir := t.TempDir()

	// Probe 2: Concurrent / sequential state.Update calls must preserve both handoff and mode without clobbering.
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, err := Update(dir, func(st *State) error {
			st.LastHandoff = &Handoff{Goal: "Preserve handoff across concurrent updates"}
			return nil
		})
		if err != nil {
			t.Errorf("Update handoff: %v", err)
		}
	}()
	go func() {
		defer wg.Done()
		_, err := Update(dir, func(st *State) error {
			st.Mode = ModeUltra
			return nil
		})
		if err != nil {
			t.Errorf("Update mode: %v", err)
		}
	}()
	wg.Wait()

	st, err := Load(dir)
	if err != nil {
		t.Fatalf("Load after concurrent Update: %v", err)
	}
	if st.Mode != ModeUltra {
		t.Fatalf("st.Mode = %q, want %q", st.Mode, ModeUltra)
	}
	if st.LastHandoff == nil || st.LastHandoff.Goal != "Preserve handoff across concurrent updates" {
		t.Fatalf("LastHandoff lost after concurrent Update: %+v", st.LastHandoff)
	}

	// Probe 3: Malformed cache must not be overwritten by Refresh.
	corruptRaw := []byte(`{"mode":"ultra","last_handoff":{"goal":"partialcorrupt`)
	if err := os.WriteFile(CacheFilePath(dir), corruptRaw, 0o644); err != nil {
		t.Fatalf("write corrupt cache: %v", err)
	}
	if _, err := Refresh(context.Background(), dir, false); err == nil {
		t.Fatal("expected Refresh to fail on malformed state.json, got nil")
	}
	afterBytes, err := os.ReadFile(CacheFilePath(dir))
	if err != nil {
		t.Fatalf("read cache after failed Refresh: %v", err)
	}
	if string(afterBytes) != string(corruptRaw) {
		t.Fatalf("Refresh overwrote corrupt cache: got %q, want %q", string(afterBytes), string(corruptRaw))
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Fast Git identity across worktrees
func TestReadFastGitHeadLinkedWorktree(t *testing.T) {
	base := t.TempDir()
	mainGit := filepath.Join(base, "main-repo", ".git")
	wtGitDir := filepath.Join(mainGit, "worktrees", "wt-gemini")
	wtRoot := filepath.Join(base, "wt-gemini")

	if err := os.MkdirAll(filepath.Join(mainGit, "refs", "heads", "gemini"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(wtGitDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(wtRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	// Linked worktree .git file pointing to wtGitDir, and commondir pointing to ../..
	if err := os.WriteFile(filepath.Join(wtRoot, ".git"), []byte("gitdir: ../main-repo/.git/worktrees/wt-gemini\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wtGitDir, "commondir"), []byte("../..\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wtGitDir, "HEAD"), []byte("ref: refs/heads/gemini/worktree-fix\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mainGit, "refs", "heads", "gemini", "worktree-fix"), []byte("deadbeefcafe1234567890\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	br, sha := ReadFastGitHead(wtRoot)
	if br != "gemini/worktree-fix" || sha != "deadbee" {
		t.Fatalf("ReadFastGitHead linked worktree loose ref = (%q, %q), want (\"gemini/worktree-fix\", \"deadbee\")", br, sha)
	}

	// Also verify packed-refs fallback via commondir.
	_ = os.Remove(filepath.Join(mainGit, "refs", "heads", "gemini", "worktree-fix"))
	packed := "# pack-refs with: peeled fully-peeled sorted\n9876543210abcdef1234567890abcdef12345678 refs/heads/gemini/worktree-fix\n"
	if err := os.WriteFile(filepath.Join(mainGit, "packed-refs"), []byte(packed), 0o644); err != nil {
		t.Fatal(err)
	}
	br, sha = ReadFastGitHead(wtRoot)
	if br != "gemini/worktree-fix" || sha != "9876543" {
		t.Fatalf("ReadFastGitHead linked worktree packed-ref = (%q, %q), want (\"gemini/worktree-fix\", \"9876543\")", br, sha)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Durable shared state and five-field handoff
// spec: openspec/specs/harness/spec.md#Requirement: Dual-forge GitHub and Gitea state sync
func TestPostPRHandoffCommentAndDecisionsInPickup(t *testing.T) {
	var postedComment string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/pulls"):
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{
					"number": 12,
					"title":  "Hardening fixes",
					"state":  "open",
					"draft":  true,
					"head":   map[string]string{"ref": "gemini/hardening"},
				},
			})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/issues/12/comments"):
			raw, _ := io.ReadAll(r.Body)
			var body struct {
				Body string `json:"body"`
			}
			_ = json.Unmarshal(raw, &body)
			postedComment = body.Body
			w.WriteHeader(http.StatusCreated)
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/issues"):
			_ = json.NewEncoder(w).Encode([]map[string]any{
				{
					"number": 7,
					"title":  "Auth token policy",
					"state":  "open",
					"body":   "Decision: require ATEAM_API_TOKEN for all non-loopback reads and writes",
					"labels": []map[string]string{{"name": "decision:approved"}},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	cfg := ForgeConfig{
		Provider: ForgeGitea,
		BaseURL:  ts.URL,
		Owner:    "c360",
		Repo:     "ateam",
	}
	h := Handoff{
		Goal:        "Land PR #12 review fixes",
		Addresses:   "PR #12",
		Constraints: "none",
		DoneWhen:    "task verify green",
		ChecksToRun: "task verify",
	}
	wantPrompt := strings.TrimSpace(FormatFiveFieldPrompt(h))
	posted, err := PostPRHandoffComment(context.Background(), t.TempDir(), cfg, "gemini/hardening", wantPrompt)
	if err != nil || !posted {
		t.Fatalf("PostPRHandoffComment = (%v, %v), want (true, nil)", posted, err)
	}
	if strings.TrimSpace(postedComment) != wantPrompt {
		t.Fatalf("posted PR comment = %q, want %q", postedComment, wantPrompt)
	}

	cards, err := SyncForgeBoard(context.Background(), t.TempDir(), cfg)
	if err != nil {
		t.Fatalf("SyncForgeBoard: %v", err)
	}
	st := defaultState()
	st.MergeForgeCards(cards, ForgeGitea)
	summary := BuildPickupSummary(st)
	if !strings.Contains(summary, "Recent decisions:") || !strings.Contains(summary, "Decision: require ATEAM_API_TOKEN") {
		t.Fatalf("BuildPickupSummary missing decision summary:\n%s", summary)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Optional herdr pane identity in the writer lease
func TestWriterLeaseHerdrIdentity(t *testing.T) {
	tests := []struct {
		name          string
		env           map[string]string
		wantPane      string
		wantWorkspace string
	}{
		{
			name:          "inside herdr records pane and workspace",
			env:           map[string]string{"HERDR_ENV": "1", "HERDR_PANE_ID": "p_7", "HERDR_WORKSPACE_ID": "w_2"},
			wantPane:      "p_7",
			wantWorkspace: "w_2",
		},
		{
			name: "outside herdr records nothing",
			env:  map[string]string{"HERDR_ENV": "", "HERDR_PANE_ID": "", "HERDR_WORKSPACE_ID": ""},
		},
		{
			name: "stray pane id without HERDR_ENV is ignored",
			env:  map[string]string{"HERDR_ENV": "", "HERDR_PANE_ID": "p_7", "HERDR_WORKSPACE_ID": "w_2"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			dir := t.TempDir()
			if _, err := WriteWriterLease(dir, "claude", "s1"); err != nil {
				t.Fatalf("WriteWriterLease: %v", err)
			}
			raw, err := os.ReadFile(WriterLeasePath(dir))
			if err != nil {
				t.Fatal(err)
			}
			got, err := ReadWriterLease(dir)
			if err != nil {
				t.Fatalf("ReadWriterLease: %v", err)
			}
			if got.HerdrPaneID != tc.wantPane || got.HerdrWorkspaceID != tc.wantWorkspace {
				t.Fatalf("lease herdr ids = (%q, %q), want (%q, %q)", got.HerdrPaneID, got.HerdrWorkspaceID, tc.wantPane, tc.wantWorkspace)
			}
			if tc.wantPane == "" && strings.Contains(string(raw), "herdr_") {
				t.Fatalf("writer.json should omit herdr fields outside herdr, got:\n%s", raw)
			}
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Optional herdr pane identity in the writer lease
func TestPickupWarningNamesHerdrPane(t *testing.T) {
	t.Setenv("ATEAM_SESSION_ID", "")
	base := WriterLease{Agent: "codex", SessionID: "other", PID: 42, LastSeen: time.Now().UTC()}

	withPane := base
	withPane.HerdrPaneID = "p_9"
	out := BuildPickupSummary(&State{Writer: &withPane, CurrentSessionID: "mine"})
	for _, want := range []string{
		"Writer lease: agent=codex session=other pid=42 last_seen=",
		" ago (herdr pane p_9)\nWARNING: another writer may be active (agent=codex, session=other, pid=42, last_seen=",
		" ago) (herdr pane p_9)",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("pickup summary missing %q:\n%s", want, out)
		}
	}

	out = BuildPickupSummary(&State{Writer: &base, CurrentSessionID: "mine"})
	if !strings.Contains(out, "WARNING: another writer may be active") || strings.Contains(out, "herdr pane") {
		t.Fatalf("pickup summary without pane should warn but not name a pane:\n%s", out)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Board state and authenticated API access
func TestResolveGitUser(t *testing.T) {
	cases := []struct {
		gitName   string
		gitEmail  string
		osUser    string
		wantName  string
		wantEmail string
	}{
		{"Coby B", "coby@c360.dev", "coby", "Coby B", "coby@c360.dev"},
		{"", "alice@example.com", "local-alice", "alice", "alice@example.com"},
		{"", "", "coby", "coby", ""},
		{"  ", "  ", "  ", "Local User", ""},
	}
	for _, tc := range cases {
		gotName, gotEmail := resolveGitUser(tc.gitName, tc.gitEmail, tc.osUser)
		if gotName != tc.wantName || gotEmail != tc.wantEmail {
			t.Fatalf("resolveGitUser(%q, %q, %q) = (%q, %q), want (%q, %q)",
				tc.gitName, tc.gitEmail, tc.osUser, gotName, gotEmail, tc.wantName, tc.wantEmail)
		}
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Board state and authenticated API access
func TestParseSpecMarkdownAndScanOpenSpecSpecs(t *testing.T) {
	dir := t.TempDir()
	specDir := filepath.Join(dir, "openspec", "specs", "harness")
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw := `# harness Specification

## Purpose

Defines the behavioral contract for ateam hooks and MCP budget.

## Requirements

### Requirement: Lifecycle hook normalization under 5ms

ateam hook SHALL normalize events in under 5ms.

#### Scenario: Mode switch

- **WHEN** ateam hook receives /mode ultra
- **THEN** it updates state.json to ultra
`
	if err := os.WriteFile(filepath.Join(specDir, "spec.md"), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	specs, err := ScanOpenSpecSpecs(dir)
	if err != nil {
		t.Fatalf("ScanOpenSpecSpecs: %v", err)
	}
	if len(specs) != 1 {
		t.Fatalf("len(specs) = %d, want 1", len(specs))
	}
	sp := specs[0]
	if sp.Name != "harness" || !strings.Contains(sp.Purpose, "behavioral contract") {
		t.Fatalf("unexpected spec header: %+v", sp)
	}
	if len(sp.Requirements) != 1 {
		t.Fatalf("len(Requirements) = %d, want 1", len(sp.Requirements))
	}
	req := sp.Requirements[0]
	if req.ID != "harness:1" || req.Name != "Lifecycle hook normalization under 5ms" {
		t.Fatalf("unexpected requirement: %+v", req)
	}
	if len(req.Scenarios) != 1 || req.Scenarios[0].Name != "Mode switch" || len(req.Scenarios[0].Steps) != 2 {
		t.Fatalf("unexpected scenarios: %+v", req.Scenarios)
	}
}
