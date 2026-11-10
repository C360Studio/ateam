package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// spec: openspec/specs/harness/spec.md#Requirement: Board state and authenticated API access
func TestOpenSpecAPIListsWorktreeChanges(t *testing.T) {
	tmp, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	root, wt := filepath.Join(tmp, "root"), filepath.Join(tmp, "wt")
	git := func(dir string, args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-c", "user.name=t", "-c", "user.email=t@t"}, args...)...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	git(tmp, "init", "-q", "-b", "main", root)
	git(root, "commit", "-q", "--allow-empty", "-m", "init")
	git(root, "worktree", "add", "-q", "-b", "claude/alpha", wt)
	const proposal = "# Alpha\n\nApproved-by:\n"
	dir := filepath.Join(wt, "openspec", "changes", "alpha")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "proposal.md"), []byte(proposal), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tasks.md"), []byte("## 1. Approval\n\n- [ ] 1.1 Approve. Hold: OPEN-Q proposal awaits owner approval\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	srv, err := NewServer(root)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/openspec", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/openspec = %d: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Changes         []json.RawMessage `json:"changes"`
		WorktreeChanges []struct {
			Name     string `json:"name"`
			Branch   string `json:"branch"`
			Proposal string `json:"proposal"`
			Approval struct {
				State string `json:"state"`
			} `json:"approval"`
		} `json:"worktree_changes"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Changes) != 0 {
		t.Errorf("changes = %d, want 0: root holds no change", len(body.Changes))
	}
	if len(body.WorktreeChanges) != 1 {
		t.Fatalf("worktree_changes = %+v, want alpha once", body.WorktreeChanges)
	}
	got := body.WorktreeChanges[0]
	if got.Name != "alpha" || got.Branch != "claude/alpha" || got.Proposal != proposal || got.Approval.State != "unapproved" {
		t.Errorf("worktree change = %+v, want alpha on claude/alpha, unapproved, with its proposal", got)
	}
}
