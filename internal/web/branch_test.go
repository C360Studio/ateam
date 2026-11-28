package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"os/exec"
	"strings"
	"testing"
)

// Generated-check decision: examples plus a git oracle, no fuzz target. The rules are per character and per
// slash-separated component, and each is exercised at its boundary below (first, middle and last component, each
// forbidden character); git check-ref-format is the independent oracle, so a fuzzer would only repeat the same
// comparison on random strings.

// rejectedBranchNames are names validBranchName must refuse; the approve handler must answer 400 for each without a
// forge call.
var rejectedBranchNames = []string{
	"", "   ", "../../x", "-x", "-", "a..b", "a b", "/claude/alpha", "claude/alpha/", "claude/alpha.lock", "claude//alpha",
	"claude/a@{b", "a~b", "a^b", "a:b", "a?b", "a*b", "a[b", `a\b`, "a#b", "a\x01b", "a\x7fb", "a\tb", ".hidden", "a/.b",
	".", ".lock", "a.lock", "a.lock/b", "a/b.lock/c", "feature.", "a/b.", "a./b", "@", "HEAD",
}

// acceptedBranchNames are names validBranchName must accept.
var acceptedBranchNames = []string{
	"main", "claude/topic", "claude/board-host-check", "feature/x.y_z", "release-1.2", "v1.0.0", "a@b", "claude/topic.locked",
	"feat/ünï", "a/b/c", "claude/topic.v2", "a/b.c/d", "a/HEAD", "head", "@/x", "x/@", "a{b", "a}b", "a/-b", "refs/heads/x",
}

// spec: openspec/specs/harness/spec.md#Requirement: Approve an OpenSpec change from the board
func TestApproveRefusesInvalidBranchNames(t *testing.T) {
	for _, branch := range rejectedBranchNames {
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
	for _, branch := range acceptedBranchNames {
		if !validBranchName(branch) {
			t.Errorf("validBranchName(%q) = false, want true", branch)
		}
	}
}

// The correct behavior: validBranchName refuses what `git check-ref-format --branch` refuses. The wrong behavior to
// catch: a rule applied to the whole string that git applies to each slash-separated component ("a.lock/b"), or a
// rule left out ("feature."). Names the board refuses although git accepts them are listed with the reason; any other
// difference fails.
//
// spec: openspec/specs/harness/spec.md#Requirement: Approve an OpenSpec change from the board
func TestValidBranchNameAgreesWithGit(t *testing.T) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not on PATH")
	}
	stricter := map[string]string{
		"a#b":  "# starts a URL fragment, so it would end the forge path",
		"a./b": "git refuses only a trailing dot on the whole name; the board refuses a part ending in a dot too",
		"@":    "git accepts @ as a branch name, but it is git's shorthand for HEAD",
	}
	gitAccepts := func(name string) bool {
		out, err := exec.Command(gitPath, "check-ref-format", "--branch", name).CombinedOutput()
		var exit *exec.ExitError
		switch {
		case err == nil:
			return true
		case errors.As(err, &exit) && exit.ExitCode() == 128: // "is not a valid branch name"
			return false
		default:
			t.Fatalf("git check-ref-format --branch %q: %v: %s", name, err, out)
			return false
		}
	}
	var accepted, refused int
	for _, name := range append(append([]string{}, rejectedBranchNames...), acceptedBranchNames...) {
		got, git := validBranchName(name), gitAccepts(name)
		if git {
			accepted++
		} else {
			refused++
		}
		if reason, ok := stricter[name]; ok {
			if !git || got {
				t.Errorf("%q is listed as stricter than git (%s), but git accepts = %v and validBranchName = %v", name, reason, git, got)
			}
			continue
		}
		if got != git {
			t.Errorf("validBranchName(%q) = %v, git check-ref-format --branch accepts = %v", name, got, git)
		}
	}
	// Both verdicts must occur, or every name was refused (or accepted) and the comparison proved nothing.
	if accepted == 0 || refused == 0 {
		t.Fatalf("git accepted %d and refused %d names, want both", accepted, refused)
	}
}
