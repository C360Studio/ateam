package state

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// spec: openspec/specs/harness/spec.md#Requirement: Prune finished worktrees the caller owns
func TestPruneVerdict(t *testing.T) {
	clean, dirty := false, true
	st := defaultState()
	st.Git.Branch, st.Git.HeadSHA = "claude/here", "9999999"
	st.Cards = []BoardCard{
		{ID: "GH-PR-1", Number: 1, Type: "pr", Column: "done", Merged: true, Branch: "claude/merged", HeadSHA: "aaaaaaa"},
		{ID: "GH-PR-2", Number: 2, Type: "pr", Column: "done", Merged: true, Branch: "claude/ahead", HeadSHA: "bbbbbbb"},
		{ID: "GH-PR-3", Number: 3, Type: "pr", Column: "in-review", Branch: "claude/open", HeadSHA: "ccccccc"},
		{ID: "GH-PR-4", Number: 4, Type: "pr", Column: "done", Merged: true, Branch: "codex/merged", HeadSHA: "ddddddd"},
		{ID: "GH-PR-5", Number: 5, Type: "pr", Column: "done", Merged: false, Branch: "claude/closed", HeadSHA: "eeeeeee"},
		{ID: "GH-PR-6", Number: 6, Type: "pr", Column: "done", Merged: true, Branch: "feature/x", HeadSHA: "fffffff"},
		{ID: "GT-PR-7", Number: 7, Type: "pr", Forge: "gitea", Column: "done", Merged: true, Branch: "claude/gitea", HeadSHA: "1111111"},
		{ID: "GH-PR-8", Number: 8, Type: "pr", Column: "done", Merged: false, Branch: "codex/x", HeadSHA: "2222222"},
		{ID: "GH-PR-9", Number: 9, Type: "pr", Column: "in-review", Branch: "codex/y", HeadSHA: "3333333"},
		{ID: "GH-PR-10", Number: 10, Type: "pr", Column: "done", Merged: true, Branch: "claude/here", HeadSHA: "9999999"},
	}
	cases := []struct {
		name   string
		wt     Worktree
		remove bool
		reason string
	}{
		{"merged via, clean, owned", Worktree{Path: "/r/a", Branch: "claude/merged", Head: "aaaaaaa", Dirty: &clean, AheadCount: 1}, true, "merged via #1"},
		{"gitea merged via", Worktree{Path: "/r/g", Branch: "claude/gitea", Head: "1111111", Dirty: &clean, AheadCount: 2}, true, "merged via #7"},
		{"ancestor of main", Worktree{Path: "/r/b", Branch: "claude/landed", Head: "5555555", Dirty: &clean, MergedIntoMain: true}, true, "merged (ancestor of main)"},
		{"dirty", Worktree{Path: "/r/a", Branch: "claude/merged", Head: "aaaaaaa", Dirty: &dirty, AheadCount: 1}, false, "dirty"},
		{"dirty unknown", Worktree{Path: "/r/a", Branch: "claude/merged", Head: "aaaaaaa", AheadCount: 1}, false, "dirty state unknown"},
		{"ahead of merged", Worktree{Path: "/r/c", Branch: "claude/ahead", Head: "0000000", Dirty: &clean, AheadCount: 3}, false, "ahead of merged #2"},
		{"open PR", Worktree{Path: "/r/d", Branch: "claude/open", Head: "ccccccc", Dirty: &clean, AheadCount: 1}, false, "open PR #3"},
		{"closed without merge", Worktree{Path: "/r/e", Branch: "claude/closed", Head: "eeeeeee", Dirty: &clean, AheadCount: 1}, false, "closed without merge (#5)"},
		{"no PR", Worktree{Path: "/r/f", Branch: "claude/wip", Head: "4444444", Dirty: &clean, AheadCount: 1}, false, "no PR"},
		{"another agent", Worktree{Path: "/r/h", Branch: "codex/merged", Head: "ddddddd", Dirty: &clean, AheadCount: 1}, false, "owned by codex"},
		{"no agent prefix", Worktree{Path: "/r/i", Branch: "feature/x", Head: "fffffff", Dirty: &clean, AheadCount: 1}, false, "no agent prefix"},
		{"current checkout", Worktree{Path: "/r/j", Branch: "claude/here", Head: "9999999", Dirty: &clean, AheadCount: 1}, false, "current checkout"},
		{"current checkout by path", Worktree{Path: "/r/me", Branch: "claude/merged", Head: "aaaaaaa", Dirty: &clean, AheadCount: 1}, false, "current checkout"},
		{"default branch", Worktree{Path: "/r", Branch: "main", Head: "9999999", Dirty: &clean}, false, "default branch"},
		{"review of merged PR", Worktree{Path: "/r/.worktrees/claude-review-pr-6", Branch: "detached", Head: "fffffff", Dirty: &clean}, true, "review of closed PR #6"},
		{"review of closed PR with suffix", Worktree{Path: "/r/.worktrees/claude-review-pr-8-d850", Branch: "", Head: "2222222", Dirty: &clean}, true, "review of closed PR #8"},
		{"review of open PR", Worktree{Path: "/r/.worktrees/claude-review-pr-9", Branch: "detached", Head: "3333333", Dirty: &clean}, false, "review of open PR #9"},
		{"review of uncached PR", Worktree{Path: "/r/.worktrees/claude-review-pr-99", Branch: "detached", Head: "3333333", Dirty: &clean}, false, "head is not PR #99's head in the forge cache"},
		{"review with a commit past the PR head", Worktree{Path: "/r/.worktrees/claude-review-pr-6", Branch: "detached", Head: "7777777", Dirty: &clean}, false, "head is not PR #6's head in the forge cache"},
		{"another agent's review", Worktree{Path: "/r/.worktrees/codex-review-pr-6", Branch: "detached", Head: "fffffff", Dirty: &clean}, false, "owned by codex"},
		{"dirty review", Worktree{Path: "/r/.worktrees/claude-review-pr-6", Branch: "detached", Head: "fffffff", Dirty: &dirty}, false, "dirty"},
		{"plain detached", Worktree{Path: "/r/.worktrees/scratch", Branch: "detached", Head: "fffffff", Dirty: &clean}, false, "detached"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			remove, reason := st.PruneVerdict(tc.wt, "claude", "/r/me")
			if remove != tc.remove || reason != tc.reason {
				t.Fatalf("PruneVerdict = (%v, %q), want (%v, %q)", remove, reason, tc.remove, tc.reason)
			}
		})
	}
	if remove, reason := st.PruneVerdict(cases[0].wt, "", ""); remove || reason != "owned by claude" {
		t.Fatalf("unresolved caller: PruneVerdict = (%v, %q), want (false, %q)", remove, reason, "owned by claude")
	}

	st.CurrentAgent = "claude"
	st.Git.Worktrees = []Worktree{cases[0].wt, cases[3].wt}
	summary := BuildPickupSummary(st)
	for _, want := range []string{
		"- claude/merged (head=aaaaaaa, dirty=false, ahead=1, merged via #1, prune candidate)\n",
		"- claude/merged (head=aaaaaaa, dirty=true, ahead=1, merged via #1)\n",
	} {
		if !strings.Contains(summary, want) {
			t.Fatalf("pickup summary missing %q:\n%s", want, summary)
		}
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Prune finished worktrees the caller owns
func TestPruneRemovesOnlyOwnedFinishedWorktrees(t *testing.T) {
	clearAgentSessionEnv(t)
	for k, v := range map[string]string{"GIT_AUTHOR_NAME": "t", "GIT_AUTHOR_EMAIL": "t@t", "GIT_COMMITTER_NAME": "t", "GIT_COMMITTER_EMAIL": "t@t"} {
		t.Setenv(k, v)
	}
	base := t.TempDir()
	origin, root, other := filepath.Join(base, "origin.git"), filepath.Join(base, "repo"), filepath.Join(base, "other")
	git := func(dir string, args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v in %s: %v: %s", args, dir, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git(base, "init", "-q", "--bare", "-b", "main", origin)
	git(base, "init", "-q", "-b", "main", root)
	git(root, "commit", "-q", "--allow-empty", "-m", "init")
	git(root, "remote", "add", "origin", origin)
	git(root, "push", "-q", "origin", "main")

	wtPath := func(name string) string { return filepath.Join(root, ".worktrees", name) }
	branchWorktree := func(name, branch string) string {
		git(root, "worktree", "add", "-q", "-b", branch, wtPath(name), "main")
		git(wtPath(name), "commit", "-q", "--allow-empty", "-m", branch)
		git(wtPath(name), "push", "-q", "origin", branch)
		return git(wtPath(name), "rev-parse", "HEAD")
	}
	mergedHead := branchWorktree("claude-merged", "claude/merged")
	dirtyHead := branchWorktree("claude-dirty", "claude/dirty")
	codexHead := branchWorktree("codex-merged", "codex/merged")
	aheadHead := branchWorktree("claude-ahead", "claude/ahead")
	movedHead := branchWorktree("claude-moved", "claude/moved")
	giteaHead := branchWorktree("claude-gitea", "claude/gitea")
	git(wtPath("claude-ahead"), "commit", "-q", "--allow-empty", "-m", "after merge")
	if err := os.WriteFile(filepath.Join(wtPath("claude-dirty"), "wip.txt"), []byte("wip"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Someone else pushed to claude/moved after the PR merged; the remote branch must survive.
	git(base, "clone", "-q", "-b", "claude/moved", origin, other)
	git(other, "commit", "-q", "--allow-empty", "-m", "late push")
	git(other, "push", "-q", "origin", "claude/moved")
	git(root, "worktree", "add", "-q", "--detach", wtPath("claude-review-pr-7-d850"), "main")
	reviewedHead := git(root, "rev-parse", "main")

	ctx := context.Background()
	st, err := Refresh(ctx, root, false)
	if err != nil {
		t.Fatal(err)
	}
	st.Cards = []BoardCard{
		{ID: "GH-PR-1", Number: 1, Type: "pr", Column: "done", Merged: true, Branch: "claude/merged", HeadSHA: mergedHead},
		{ID: "GH-PR-2", Number: 2, Type: "pr", Column: "done", Merged: true, Branch: "claude/dirty", HeadSHA: dirtyHead},
		{ID: "GH-PR-3", Number: 3, Type: "pr", Column: "done", Merged: true, Branch: "codex/merged", HeadSHA: codexHead},
		{ID: "GH-PR-4", Number: 4, Type: "pr", Column: "done", Merged: true, Branch: "claude/ahead", HeadSHA: aheadHead},
		{ID: "GH-PR-5", Number: 5, Type: "pr", Column: "done", Merged: true, Branch: "claude/moved", HeadSHA: movedHead},
		{ID: "GT-PR-6", Number: 6, Type: "pr", Forge: "gitea", Column: "done", Merged: true, Branch: "claude/gitea", HeadSHA: giteaHead},
		{ID: "GH-PR-7", Number: 7, Type: "pr", Column: "done", Merged: true, Branch: "codex/reviewed", HeadSHA: reviewedHead},
	}
	st.ReconcileWorktrees()

	worktrees := func() string { return git(root, "worktree", "list", "--porcelain") }
	remoteRefs := func() string { return git(root, "ls-remote", "origin") }
	beforeWT, beforeRemote := worktrees(), remoteRefs()
	var dry bytes.Buffer
	if Prune(ctx, root, st, "claude", true, &dry) {
		t.Fatalf("dry run reported a failure:\n%s", dry.String())
	}
	if worktrees() != beforeWT || remoteRefs() != beforeRemote {
		t.Fatalf("dry run changed worktrees or remote refs:\n%s", dry.String())
	}
	if !strings.Contains(dry.String(), "would remove claude/merged: merged via #1\n") {
		t.Fatalf("dry run output:\n%s", dry.String())
	}

	var out bytes.Buffer
	if Prune(ctx, root, st, "claude", false, &out) {
		t.Fatalf("prune reported a failure:\n%s", out.String())
	}
	for _, want := range []string{
		"kept main: default branch\n",
		"removed claude/merged: merged via #1; deleted origin/claude/merged\n",
		"removed claude/gitea: merged via #6; deleted origin/claude/gitea\n",
		"removed claude/moved: merged via #5; kept origin/claude/moved, which moved to ",
		"removed claude-review-pr-7-d850: review of closed PR #7\n",
		"kept claude/dirty: dirty\n",
		"kept codex/merged: owned by codex\n",
		"kept claude/ahead: ahead of merged #4\n",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("prune output missing %q:\n%s", want, out.String())
		}
	}
	exists := func(p string) bool { _, err := os.Stat(p); return err == nil }
	for name, want := range map[string]bool{"claude-merged": false, "claude-gitea": false, "claude-moved": false, "claude-review-pr-7-d850": false, "claude-dirty": true, "codex-merged": true, "claude-ahead": true} {
		if exists(wtPath(name)) != want {
			t.Fatalf("worktree %s exists=%v, want %v", name, !want, want)
		}
	}
	locals := git(root, "branch", "--format=%(refname:short)")
	remotes := remoteRefs()
	for branch, want := range map[string]bool{"claude/merged": false, "claude/gitea": false, "claude/moved": false, "claude/dirty": true, "codex/merged": true, "claude/ahead": true} {
		if got := strings.Contains(locals+"\n", branch+"\n"); got != want {
			t.Fatalf("local branch %s present=%v, want %v:\n%s", branch, got, want, locals)
		}
	}
	for branch, want := range map[string]bool{"claude/merged": false, "claude/gitea": false, "claude/moved": true, "claude/dirty": true, "codex/merged": true} {
		if got := strings.Contains(remotes, "refs/heads/"+branch+"\n") || strings.HasSuffix(remotes, "refs/heads/"+branch); got != want {
			t.Fatalf("remote branch %s present=%v, want %v:\n%s", branch, got, want, remotes)
		}
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Prune finished worktrees the caller owns
func TestPruneVerdictReviewPRNumberOnTwoForges(t *testing.T) {
	// The same PR number on GitHub (open) and Gitea (closed) must not let the closed one authorize removal (#246 review).
	clean := false
	wt := Worktree{Path: "/r/claude-review-pr-1", Branch: "detached", Head: "aaaaaaa", Dirty: &clean}
	gitea := BoardCard{ID: "GT-PR-1", Number: 1, Type: "pr", Forge: "gitea", Column: "done", Merged: true, HeadSHA: "bbbbbbb"}
	github := BoardCard{ID: "GH-PR-1", Number: 1, Type: "pr", Forge: "github", Column: "in-review", HeadSHA: "aaaaaaa"}
	for name, cards := range map[string][]BoardCard{"gitea first": {gitea, github}, "github first": {github, gitea}} {
		st := defaultState()
		st.Cards = cards
		if remove, reason := st.PruneVerdict(wt, "claude", ""); remove || reason != "review of open PR #1" {
			t.Fatalf("%s: PruneVerdict = (%v, %q), want (false, %q)", name, remove, reason, "review of open PR #1")
		}
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Prune finished worktrees the caller owns
func TestPrunePreservesWorkAddedAfterRefresh(t *testing.T) {
	// Each scenario adds work between the refresh that approved a removal and the git step that would
	// lose it; the work must survive (#246 review).
	realGit, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []string{"local commit after refresh", "branch moved after worktree removal", "remote push after ls-remote", "detached commit past the PR head"} {
		t.Run(scenario, func(t *testing.T) {
			clearAgentSessionEnv(t)
			for k, v := range map[string]string{"GIT_AUTHOR_NAME": "t", "GIT_AUTHOR_EMAIL": "t@t", "GIT_COMMITTER_NAME": "t", "GIT_COMMITTER_EMAIL": "t@t"} {
				t.Setenv(k, v)
			}
			git := func(dir string, args ...string) string {
				t.Helper()
				cmd := exec.Command(realGit, args...)
				cmd.Dir = dir
				out, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("git %v: %v: %s", args, err, out)
				}
				return strings.TrimSpace(string(out))
			}
			base := t.TempDir()
			root, origin, other := filepath.Join(base, "repo"), filepath.Join(base, "origin.git"), filepath.Join(base, "other")
			wt := filepath.Join(base, "claude-review-pr-1")
			git(base, "init", "-q", "--bare", "-b", "main", origin)
			git(base, "init", "-q", "-b", "main", root)
			git(root, "commit", "-q", "--allow-empty", "-m", "init")
			git(root, "remote", "add", "origin", origin)
			git(root, "push", "-q", "origin", "main")
			git(root, "worktree", "add", "-q", "-b", "claude/topic", wt, "main")
			git(wt, "commit", "-q", "--allow-empty", "-m", "reviewed")
			prHead := git(wt, "rev-parse", "HEAD")
			git(wt, "push", "-q", "origin", "claude/topic")
			if scenario == "detached commit past the PR head" {
				git(wt, "switch", "-q", "--detach")
				git(wt, "commit", "-q", "--allow-empty", "-m", "unpushed review work")
			}
			st, err := Refresh(context.Background(), root, false)
			if err != nil {
				t.Fatal(err)
			}
			st.Cards = []BoardCard{{ID: "GH-PR-1", Number: 1, Type: "pr", Column: "done", Merged: true, Branch: "claude/topic", HeadSHA: prHead}}
			st.ReconcileWorktrees()

			// A git wrapper on PATH injects the concurrent step at the exact point under test.
			wrap := func(trigger, inject string) {
				bin := filepath.Join(base, "bin")
				script := "#!/bin/sh\nif [ \"$1\" = \"" + trigger + "\" ]; then\n  \"$REAL_GIT\" \"$@\" > \"$SNAP\" 2>&1; rc=$?\n  " + inject + " >/dev/null 2>&1 || exit 99\n  cat \"$SNAP\"; exit $rc\nfi\nexec \"$REAL_GIT\" \"$@\"\n"
				if err := os.MkdirAll(bin, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0o755); err != nil {
					t.Fatal(err)
				}
				t.Setenv("REAL_GIT", realGit)
				t.Setenv("SNAP", filepath.Join(base, "snap"))
				t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
			}
			switch scenario {
			case "local commit after refresh":
				git(wt, "commit", "-q", "--allow-empty", "-m", "unpushed work after refresh")
			case "branch moved after worktree removal":
				git(base, "clone", "-q", "-b", "claude/topic", origin, other)
				git(other, "commit", "-q", "--allow-empty", "-m", "later work")
				git(root, "fetch", "-q", other, "claude/topic")
				wrap("worktree", `"$REAL_GIT" -C "`+root+`" update-ref refs/heads/claude/topic FETCH_HEAD`)
			case "remote push after ls-remote":
				git(base, "clone", "-q", "-b", "claude/topic", origin, other)
				git(other, "commit", "-q", "--allow-empty", "-m", "concurrent remote work")
				wrap("ls-remote", `"$REAL_GIT" -C "`+other+`" push -q origin claude/topic`)
			}

			var out bytes.Buffer
			failed := Prune(context.Background(), root, st, "claude", false, &out)
			got := out.String()
			switch scenario {
			case "local commit after refresh", "detached commit past the PR head":
				if _, err := os.Stat(wt); err != nil || failed {
					t.Fatalf("worktree holding unpushed work was removed (failed=%v, stat %v):\n%s", failed, err, got)
				}
			case "branch moved after worktree removal":
				if git(root, "branch", "--list", "claude/topic") == "" || !strings.Contains(got, "kept branch claude/topic, which moved since refresh") {
					t.Fatalf("moved local branch was deleted:\n%s", got)
				}
			case "remote push after ls-remote":
				if git(root, "ls-remote", "origin", "refs/heads/claude/topic") == "" || !strings.Contains(got, "kept origin/claude/topic, which moved to ") {
					t.Fatalf("remote branch pushed after ls-remote was deleted (failed=%v):\n%s", failed, got)
				}
			}
		})
	}
}
