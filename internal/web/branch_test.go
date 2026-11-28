package web

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// spec: openspec/specs/harness/spec.md#Requirement: Approve an OpenSpec change from the board
func TestApproveRefusesInvalidBranchNames(t *testing.T) {
	branches := []string{
		"", "   ", "../../x", "-x", "a..b", "a b", "/claude/alpha", "claude/alpha/", "claude/alpha.lock", "claude//alpha",
		"claude/a@{b", "a~b", "a^b", "a:b", "a?b", "a*b", "a[b", `a\b`, "a#b", "a\x01b", "a\x7fb", "a\tb", ".hidden", "a/.b",
	}
	for _, branch := range branches {
		t.Run(strings.ReplaceAll(branch, "/", "|"), func(t *testing.T) {
			files := &fakeFiles{tree: approveTree(approveTasks)}
			srv, h := approveServer(t, files)
			body, err := json.Marshal(approveRequest{Change: "alpha", Branch: branch})
			if err != nil {
				t.Fatal(err)
			}
			rec := postApprove(srv, h, string(body), nil)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("approve on branch %q = %d, want 400: %s", branch, rec.Code, rec.Body.String())
			}
			if files.calls != 0 {
				t.Errorf("approve on branch %q made %d forge calls, want none", branch, files.calls)
			}
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Approve an OpenSpec change from the board
func TestValidBranchNameAcceptsGitBranches(t *testing.T) {
	for _, branch := range []string{"main", "claude/topic", "claude/board-host-check", "feature/x.y_z", "release-1.2", "v1.0.0", "a@b", "claude/topic.locked", "feat/ünï", "a/b/c"} {
		if !validBranchName(branch) {
			t.Errorf("validBranchName(%q) = false, want true", branch)
		}
	}
}
