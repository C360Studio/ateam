package state

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
		{"no agent prefix", Worktree{Path: "/r/i", Branch: "feature/x", Head: "fffffff", Dirty: &clean, AheadCount: 1}, false, "no agent prefix (host-created branch)"},
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
			remove, reason := st.PruneVerdict(tc.wt, "claude", "/r/me", false)
			if remove != tc.remove || reason != tc.reason {
				t.Fatalf("PruneVerdict = (%v, %q), want (%v, %q)", remove, reason, tc.remove, tc.reason)
			}
		})
	}
	if remove, reason := st.PruneVerdict(cases[0].wt, "", "", false); remove || reason != "owned by claude" {
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
func TestPruneKeepsTheCoordinatorWorktree(t *testing.T) {
	clean := false
	st := defaultState()
	st.Git.Branch, st.Git.HeadSHA = "claude/here", "9999999"
	merged := []BoardCard{{ID: "GH-PR-1", Number: 1, Type: "pr", Column: "done", Merged: true, Branch: "claude/coordinator", HeadSHA: "aaaaaaa"}}
	// main has moved past the coordinator's head, so git alone reads it merged (#445).
	ancestor := Worktree{Path: "/r/.worktrees/claude-coordinator", Branch: "claude/coordinator", Head: "5555555", Dirty: &clean, MergedIntoMain: true}
	viaPR := Worktree{Path: "/r/.worktrees/claude-coordinator", Branch: "claude/coordinator", Head: "aaaaaaa", Dirty: &clean, AheadCount: 1}
	lookalike := Worktree{Path: "/r/.worktrees/claude-coordinator-docs", Branch: "claude/coordinator-docs", Head: "5555555", Dirty: &clean, MergedIntoMain: true}
	const kept = "coordinator worktree; pass --force once the run has ended"
	for _, tc := range []struct {
		name   string
		wt     Worktree
		cards  []BoardCard
		force  bool
		remove bool
		reason string
	}{
		{"ancestor of main", ancestor, nil, false, false, kept},
		{"merged via a PR", viaPR, merged, false, false, kept},
		{"ancestor of main with --force", ancestor, nil, true, true, "merged (ancestor of main)"},
		{"merged via a PR with --force", viaPR, merged, true, true, "merged via #1"},
		{"a topic that only starts with coordinator", lookalike, nil, false, true, "merged (ancestor of main)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st.Cards = tc.cards
			remove, reason := st.PruneVerdict(tc.wt, "claude", "/r/me", tc.force)
			if remove != tc.remove || reason != tc.reason {
				t.Fatalf("PruneVerdict = (%v, %q), want (%v, %q)", remove, reason, tc.remove, tc.reason)
			}
		})
	}

	st.CurrentAgent, st.Cards = "claude", nil
	st.Git.Worktrees = []Worktree{ancestor}
	summary := BuildPickupSummary(st)
	if want := "- claude/coordinator (head=5555555, dirty=false, ahead=0, merged (ancestor of main))\n"; !strings.Contains(summary, want) {
		t.Fatalf("pickup summary missing %q (the label stays, the prune hint goes):\n%s", want, summary)
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
	if Prune(ctx, root, st, "claude", true, false, &dry) {
		t.Fatalf("dry run reported a failure:\n%s", dry.String())
	}
	if worktrees() != beforeWT || remoteRefs() != beforeRemote {
		t.Fatalf("dry run changed worktrees or remote refs:\n%s", dry.String())
	}
	if !strings.Contains(dry.String(), "would remove claude/merged: merged via #1\n") {
		t.Fatalf("dry run output:\n%s", dry.String())
	}

	var out bytes.Buffer
	if Prune(ctx, root, st, "claude", false, false, &out) {
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
	if strings.Contains(locals, "ateam-prune/") {
		t.Fatalf("a detached checkout's pin branch outlived its removal:\n%s", locals)
	}
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
		if remove, reason := st.PruneVerdict(wt, "claude", "", false); remove || reason != "review of open PR #1" {
			t.Fatalf("%s: PruneVerdict = (%v, %q), want (false, %q)", name, remove, reason, "review of open PR #1")
		}
		// The review checked out aaaaaaa, then the still-open GitHub PR advanced past it while the Gitea PR closed at it.
		st.Cards = []BoardCard{gitea, github}
		st.Cards[0].HeadSHA, st.Cards[1].HeadSHA = "aaaaaaa", "ccccccc"
		if name == "github first" {
			st.Cards[0], st.Cards[1] = st.Cards[1], st.Cards[0]
		}
		if remove, reason := st.PruneVerdict(wt, "claude", "", false); remove || reason != "review of open PR #1" {
			t.Fatalf("%s, open PR advanced: PruneVerdict = (%v, %q), want (false, %q)", name, remove, reason, "review of open PR #1")
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
	for _, scenario := range []string{"local commit after refresh", "branch moved after worktree removal", "remote push after ls-remote", "detached commit past the PR head", "detached commit after the live check", "local ref lock error"} {
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
			if scenario == "detached commit after the live check" {
				git(wt, "switch", "-q", "--detach")
			}
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
			wrap := func(trigger, inject string, before bool) {
				bin := filepath.Join(base, "bin")
				script := "#!/bin/sh\nif [ \"$1\" = \"" + trigger + "\" ]; then\n  \"$REAL_GIT\" \"$@\" > \"$SNAP\" 2>&1; rc=$?\n  { " + inject + "; } >/dev/null 2>&1 || exit 99\n  cat \"$SNAP\"; exit $rc\nfi\nexec \"$REAL_GIT\" \"$@\"\n"
				if before {
					script = "#!/bin/sh\nif [ \"$1\" = \"" + trigger + "\" ]; then\n  { " + inject + "; } >/dev/null 2>&1 || exit 99\nfi\nexec \"$REAL_GIT\" \"$@\"\n"
				}
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
				wrap("worktree", `"$REAL_GIT" -C "`+root+`" update-ref refs/heads/claude/topic FETCH_HEAD`, false)
			case "remote push after ls-remote":
				git(base, "clone", "-q", "-b", "claude/topic", origin, other)
				git(other, "commit", "-q", "--allow-empty", "-m", "concurrent remote work")
				wrap("ls-remote", `"$REAL_GIT" -C "`+other+`" push -q origin claude/topic`, false)
			case "detached commit after the live check":
				// Lands after every check prune makes, just before git worktree remove runs.
				wrap("worktree", `"$REAL_GIT" -C "`+wt+`" commit -q --allow-empty -m late && "$REAL_GIT" -C "`+wt+`" rev-parse HEAD > "`+filepath.Join(base, "late")+`"`, true)
			case "local ref lock error":
				if err := os.WriteFile(filepath.Join(root, ".git", "refs", "heads", "claude", "topic.lock"), nil, 0o644); err != nil {
					t.Fatal(err)
				}
			}

			var out bytes.Buffer
			failed := Prune(context.Background(), root, st, "claude", false, false, &out)
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
			case "detached commit after the live check":
				late, err := os.ReadFile(filepath.Join(base, "late"))
				if err != nil {
					t.Fatalf("injection not reached: %v\n%s", err, got)
				}
				if git(root, "for-each-ref", "--contains", strings.TrimSpace(string(late))) == "" {
					t.Fatalf("commit made after the live check is no longer referenced:\n%s", got)
				}
			case "local ref lock error":
				if !failed || !strings.Contains(got, "failed claude/topic: worktree removed; git update-ref -d refs/heads/claude/topic: ") {
					t.Fatalf("a locked ref was reported as moved, not failed (failed=%v):\n%s", failed, got)
				}
			case "remote push after ls-remote":
				if git(root, "ls-remote", "origin", "refs/heads/claude/topic") == "" || !strings.Contains(got, "kept origin/claude/topic, which moved to ") {
					t.Fatalf("remote branch pushed after ls-remote was deleted (failed=%v):\n%s", failed, got)
				}
			}
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Prune finished worktrees the caller owns
func TestPruneKeepsWorktreeAnotherSessionOccupies(t *testing.T) {
	// #263: every Claude session shares the claude/ prefix, so ownership alone lets one session remove the
	// finished worktree another is still standing in.
	clearAgentSessionEnv(t)
	now := time.Now()
	alive := os.Getpid()
	type lease struct {
		session  string
		pid      int
		lastSeen time.Duration
	}
	type handoff struct {
		session string
		age     time.Duration
	}
	const handoffReason = "session other handed off 5m ago; prune after 30m or pass --force"
	cases := []struct {
		name     string
		caller   string
		lease    *lease
		handoff  *handoff
		rootSlot bool   // the caller checkout's cache also holds the slot, with an older handoff
		badCache string // "malformed" or "dir": the worktree's state.json cannot be read as JSON
		prefix   bool   // reason ends in an error text that names a temp path, so match its start
		force    bool
		remove   bool
		reason   string
	}{
		{name: "nothing recorded", caller: "me", remove: true, reason: "merged via #1"},
		{name: "live pid lease from another session", caller: "me", lease: &lease{"other", alive, 2 * time.Hour}, reason: "session other holds the lease (2h ago)"},
		{name: "recent lease with no live pid", caller: "me", lease: &lease{"other", 0, 3 * time.Minute}, reason: "session other holds the lease (3m ago)"},
		{name: "expired lease with no live pid", caller: "me", lease: &lease{"other", 0, 2 * time.Hour}, remove: true, reason: "merged via #1"},
		{name: "live lease without a session id", caller: "me", lease: &lease{"", alive, time.Minute}, reason: "an unknown session holds the lease (1m ago)"},
		{name: "fresh handoff from another session", caller: "me", handoff: &handoff{"other", 5 * time.Minute}, reason: handoffReason},
		{name: "fresh handoff whose slot the caller's cache also holds", caller: "me", handoff: &handoff{"other", 5 * time.Minute}, rootSlot: true, reason: handoffReason},
		{name: "old handoff from another session", caller: "me", handoff: &handoff{"other", 2 * time.Hour}, remove: true, reason: "merged via #1"},
		{name: "caller's own live lease", caller: "me", lease: &lease{"me", alive, time.Minute}, remove: true, reason: "merged via #1"},
		{name: "caller's own fresh handoff", caller: "me", handoff: &handoff{"me", 5 * time.Minute}, remove: true, reason: "merged via #1"},
		{name: "unknown caller keeps a lease", lease: &lease{"me", alive, time.Minute}, reason: "session me holds the lease (1m ago)"},
		{name: "unknown caller keeps a handoff", handoff: &handoff{"me", 5 * time.Minute}, reason: "session me handed off 5m ago; prune after 30m or pass --force"},
		{name: "force past a live lease", caller: "me", lease: &lease{"other", alive, 2 * time.Hour}, force: true, remove: true, reason: "merged via #1"},
		{name: "force past a fresh handoff", caller: "me", handoff: &handoff{"other", 5 * time.Minute}, force: true, remove: true, reason: "merged via #1"},
		// A deletion guard fails closed: a cache that may hold a fresh handoff but cannot be read keeps the worktree.
		{name: "malformed handoff cache", caller: "me", badCache: "malformed", reason: "handoff cache unreadable: ", prefix: true},
		{name: "unreadable handoff cache", caller: "me", badCache: "dir", reason: "handoff cache unreadable: ", prefix: true},
		{name: "force past an unreadable handoff cache", caller: "me", badCache: "dir", force: true, remove: true, reason: "merged via #1"},
	}
	clean, dirty := false, true
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.MkdirAll(filepath.Join(dir, ".ateam"), 0o755); err != nil {
				t.Fatal(err)
			}
			write := func(path string, v any) {
				t.Helper()
				data, err := json.Marshal(v)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, data, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			st := defaultState()
			st.Git.Branch = "main"
			st.CurrentAgent, st.CurrentSessionID = "claude", tc.caller
			st.Cards = []BoardCard{{ID: "GH-PR-1", Number: 1, Type: "pr", Column: "done", Merged: true, Branch: "claude/merged", HeadSHA: "aaaaaaa"}}
			if tc.lease != nil {
				write(WriterLeasePath(dir), WriterLease{Agent: "claude", SessionID: tc.lease.session, PID: tc.lease.pid, LastSeen: now.Add(-tc.lease.lastSeen)})
			}
			if tc.handoff != nil {
				h := &Handoff{Goal: "finish #1", Agent: "claude", SessionID: tc.handoff.session, Branch: "claude/merged", UpdatedAt: now.Add(-tc.handoff.age)}
				write(CacheFilePath(dir), map[string]any{"handoffs": map[string]*Handoff{"claude@claude/merged": h}})
				if tc.rootSlot {
					old := *h
					old.UpdatedAt = now.Add(-2 * time.Hour)
					st.Handoffs = map[string]*Handoff{"claude@claude/merged": &old}
				}
			}
			switch tc.badCache {
			case "malformed":
				if err := os.WriteFile(CacheFilePath(dir), []byte(`{"handoffs": {"claude@claude/merged": `), 0o644); err != nil {
					t.Fatal(err)
				}
			case "dir":
				if err := os.MkdirAll(CacheFilePath(dir), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			matches := func(got string) bool {
				if tc.prefix {
					return strings.HasPrefix(got, tc.reason)
				}
				return got == tc.reason
			}
			wt := Worktree{Path: dir, Branch: "claude/merged", Head: "aaaaaaa", Dirty: &clean, AheadCount: 1}
			remove, reason := st.PruneVerdict(wt, "claude", "/r/me", tc.force)
			if remove != tc.remove || !matches(reason) {
				t.Fatalf("PruneVerdict = (%v, %q), want (%v, %q)", remove, reason, tc.remove, tc.reason)
			}
			// Occupancy never overrides the dirty rule, with or without force.
			dirtyWT := wt
			dirtyWT.Dirty = &dirty
			if remove, reason := st.PruneVerdict(dirtyWT, "claude", "/r/me", true); remove || reason != "dirty" {
				t.Fatalf("forced dirty: PruneVerdict = (%v, %q), want (false, %q)", remove, reason, "dirty")
			}

			// Prune's dry run and pickup's label apply the same verdict; dry run with force shows what force removes.
			st.Git.Worktrees = []Worktree{wt}
			var out bytes.Buffer
			if Prune(context.Background(), "/r/me", st, "claude", true, tc.force, &out) {
				t.Fatalf("dry run reported a failure:\n%s", out.String())
			}
			wantLine := "kept claude/merged: " + tc.reason + "\n"
			if tc.remove {
				wantLine = "would remove claude/merged: " + tc.reason + "\n"
			}
			if tc.prefix {
				wantLine = strings.TrimSuffix(wantLine, "\n")
			}
			if (tc.prefix && !strings.HasPrefix(out.String(), wantLine)) || (!tc.prefix && out.String() != wantLine) {
				t.Fatalf("dry run printed %q, want %q", out.String(), wantLine)
			}
			if !tc.force {
				candidate := strings.Contains(BuildPickupSummary(st), "merged via #1, prune candidate)")
				if candidate != tc.remove {
					t.Fatalf("pickup prune candidate = %v, want %v", candidate, tc.remove)
				}
			}
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Prune finished worktrees the caller owns
func TestPruneKeepsReviewCheckoutAnotherSessionOccupies(t *testing.T) {
	// #321: a closed PR's review checkout skipped the occupancy check, so one Claude session's prune removed
	// the review checkout another, idle session was still standing in.
	clearAgentSessionEnv(t)
	now := time.Now()
	cases := []struct {
		name    string
		lease   string // session of a live writer lease in the checkout
		handoff string // session of a 5m-old handoff recorded on HEAD in the checkout
		force   bool
		remove  bool
		reason  string
	}{
		{name: "nothing recorded", remove: true, reason: "review of closed PR #6"},
		{name: "another session's live lease", lease: "other", reason: "session other holds the lease (1m ago)"},
		{name: "another session's fresh handoff", handoff: "other", reason: "session other handed off 5m ago; prune after 30m or pass --force"},
		{name: "caller's own live lease", lease: "me", remove: true, reason: "review of closed PR #6"},
		{name: "force past another session's lease", lease: "other", force: true, remove: true, reason: "review of closed PR #6"},
	}
	clean := false
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "claude-review-pr-6")
			if err := os.MkdirAll(filepath.Join(dir, ".ateam"), 0o755); err != nil {
				t.Fatal(err)
			}
			write := func(path string, v any) {
				t.Helper()
				data, err := json.Marshal(v)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, data, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if tc.lease != "" {
				write(WriterLeasePath(dir), WriterLease{Agent: "claude", SessionID: tc.lease, PID: os.Getpid(), LastSeen: now.Add(-time.Minute)})
			}
			if tc.handoff != "" {
				h := &Handoff{Goal: "review #6", Agent: "claude", SessionID: tc.handoff, Branch: "HEAD", UpdatedAt: now.Add(-5 * time.Minute)}
				write(CacheFilePath(dir), map[string]any{"handoffs": map[string]*Handoff{"claude@HEAD": h}})
			}
			st := defaultState()
			st.Git.Branch = "main"
			st.CurrentAgent, st.CurrentSessionID = "claude", "me"
			st.Cards = []BoardCard{{ID: "GH-PR-6", Number: 6, Type: "pr", Column: "done", Merged: true, Branch: "claude/x", HeadSHA: "fffffff"}}
			wt := Worktree{Path: dir, Branch: "", Head: "fffffff", Dirty: &clean}
			remove, reason := st.PruneVerdict(wt, "claude", "/r/me", tc.force)
			if remove != tc.remove || reason != tc.reason {
				t.Fatalf("PruneVerdict = (%v, %q), want (%v, %q)", remove, reason, tc.remove, tc.reason)
			}
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Dual-forge GitHub and Gitea state sync
// spec: openspec/specs/harness/spec.md#Requirement: Prune finished worktrees the caller owns
func TestPruneAndPickupStaleCacheAndWorktreeFallback(t *testing.T) {
	clearAgentSessionEnv(t)
	now := time.Now().UTC()
	clean := false

	t.Run("cache predating branch head reports PR unknown instead of no PR", func(t *testing.T) {
		st := defaultState()
		st.Forge = ForgeConfig{Provider: ForgeGitHub, Owner: "c360", Repo: "ateam"}
		st.BoardSyncedAt = now.Add(-15 * time.Minute)
		st.BoardSyncError = "gh issue list failed: signal: killed"
		st.Git.Branch, st.Git.HeadSHA = "main", "1111111"
		st.CurrentAgent = "claude"
		st.Git.Worktrees = []Worktree{
			{Path: "/r", Branch: "main", Head: "1111111", Dirty: &clean},
			{
				Path:            "/r/.worktrees/claude-new",
				Branch:          "claude/288-test-spec-citations",
				Head:            "aaaaaaa",
				HeadCommittedAt: now.Add(-5 * time.Minute),
				Dirty:           &clean,
				AheadCount:      1,
			},
			{
				Path:            "/r/.worktrees/claude-old",
				Branch:          "claude/older-than-cache",
				Head:            "bbbbbbb",
				HeadCommittedAt: now.Add(-30 * time.Minute),
				Dirty:           &clean,
				AheadCount:      1,
			},
		}

		removeNew, reasonNew := st.PruneVerdict(st.Git.Worktrees[1], "claude", "/r", false)
		if removeNew || reasonNew != "PR unknown (cache predates branch) (cache 15m)" {
			t.Fatalf("new branch PruneVerdict = (%v, %q), want (false, %q)", removeNew, reasonNew, "PR unknown (cache predates branch) (cache 15m)")
		}
		removeOld, reasonOld := st.PruneVerdict(st.Git.Worktrees[2], "claude", "/r", false)
		if removeOld || reasonOld != "no PR (cache 15m)" {
			t.Fatalf("old branch PruneVerdict = (%v, %q), want (false, %q)", removeOld, reasonOld, "no PR (cache 15m)")
		}

		var out bytes.Buffer
		if Prune(context.Background(), "/r", st, "claude", true, false, &out) {
			t.Fatalf("Prune dry run failed:\n%s", out.String())
		}
		wantSyncWarn := "WARNING: Forge board sync failed (serving cached board, synced=" + st.BoardSyncedAt.Format(time.RFC3339) + "): gh issue list failed: signal: killed\n"
		for _, want := range []string{
			wantSyncWarn,
			"kept claude/288-test-spec-citations: PR unknown (cache predates branch) (cache 15m)\n",
			"kept claude/older-than-cache: no PR (cache 15m)\n",
		} {
			if !strings.Contains(out.String(), want) {
				t.Fatalf("prune output missing %q:\n%s", want, out.String())
			}
		}

		summary := BuildPickupSummary(st)
		if !strings.Contains(summary, "- claude/288-test-spec-citations (head=aaaaaaa, dirty=false, ahead=1, PR unknown (cache predates branch) (cache 15m))") {
			t.Fatalf("pickup summary missing PR unknown label:\n%s", summary)
		}
	})

	t.Run("linked worktree with newer forge cache supplies PR cards to main checkout", func(t *testing.T) {
		wtMergedDir := t.TempDir()
		wtOpenDir := t.TempDir()
		wtOlderDir := t.TempDir()
		wtStaleMtimeDir := t.TempDir()

		writeWTCache := func(dir string, syncedAt time.Time, card BoardCard) {
			wtSt := defaultState()
			wtSt.Forge = ForgeConfig{Provider: ForgeGitHub, Owner: "c360", Repo: "ateam"}
			wtSt.BoardSyncedAt = syncedAt
			wtSt.ForgeSyncedAt = map[ForgeProvider]time.Time{ForgeGitHub: syncedAt}
			wtSt.Cards = []BoardCard{card}
			if err := Save(dir, wtSt); err != nil {
				t.Fatal(err)
			}
		}
		writeWTCache(wtMergedDir, now.Add(-2*time.Minute), BoardCard{
			ID:      "GH-PR-289",
			Number:  289,
			Type:    "pr",
			Column:  "done",
			Merged:  true,
			Branch:  "claude/288-test-spec-citations",
			HeadSHA: "aaaaaaa",
			Forge:   "github",
		})
		writeWTCache(wtOpenDir, now.Add(-3*time.Minute), BoardCard{
			ID:      "GH-PR-290",
			Number:  290,
			Type:    "pr",
			Column:  "in-review",
			Merged:  false,
			Branch:  "claude/233-pin-alpha5",
			HeadSHA: "ccccccc",
			Forge:   "github",
		})
		// Linked cache older than main cache (newer file mtime because it was just saved, but older SyncedAtFor):
		// its stale open card must be ignored in favor of main's merged card.
		writeWTCache(wtOlderDir, now.Add(-2*time.Hour), BoardCard{
			ID:      "GH-PR-291",
			Number:  291,
			Type:    "pr",
			Column:  "in-review",
			Merged:  false,
			Branch:  "claude/older-linked-cache",
			HeadSHA: "ddddddd",
			Forge:   "github",
		})
		// Linked cache whose file mtime is older than main's sync timestamp: must be skipped by os.Stat without reading.
		writeWTCache(wtStaleMtimeDir, now.Add(-2*time.Hour), BoardCard{
			ID:      "GH-PR-292",
			Number:  292,
			Type:    "pr",
			Column:  "in-review",
			Merged:  false,
			Branch:  "claude/stale-mtime-cache",
			HeadSHA: "eeeeeee",
			Forge:   "github",
		})
		staleMtime := now.Add(-2 * time.Hour)
		if err := os.Chtimes(CacheFilePath(wtStaleMtimeDir), staleMtime, staleMtime); err != nil {
			t.Fatal(err)
		}

		readCounts := map[string]int{}
		origRead := readWorktreeCacheFile
		readWorktreeCacheFile = func(path string) ([]byte, error) {
			readCounts[filepath.Clean(path)]++
			return origRead(path)
		}
		t.Cleanup(func() { readWorktreeCacheFile = origRead })

		st := defaultState()
		st.Forge = ForgeConfig{Provider: ForgeGitHub, Owner: "c360", Repo: "ateam"}
		st.BoardSyncedAt = now.Add(-1 * time.Hour)
		st.ForgeSyncedAt = map[ForgeProvider]time.Time{ForgeGitHub: now.Add(-1 * time.Hour)}
		st.Git.Branch, st.Git.HeadSHA = "main", "1111111"
		st.CurrentAgent = "claude"
		// Main cache has a stale open card for claude/288-test-spec-citations (which merged in wtMergedDir's newer cache)
		// and newer merged cards for claude/older-linked-cache and claude/stale-mtime-cache.
		st.Cards = []BoardCard{
			{
				ID:      "GH-PR-289",
				Number:  289,
				Type:    "pr",
				Column:  "in-review",
				Merged:  false,
				Branch:  "claude/288-test-spec-citations",
				HeadSHA: "aaaaaaa",
				Forge:   "github",
			},
			{
				ID:      "GH-PR-291",
				Number:  291,
				Type:    "pr",
				Column:  "done",
				Merged:  true,
				Branch:  "claude/older-linked-cache",
				HeadSHA: "ddddddd",
				Forge:   "github",
			},
			{
				ID:      "GH-PR-292",
				Number:  292,
				Type:    "pr",
				Column:  "done",
				Merged:  true,
				Branch:  "claude/stale-mtime-cache",
				HeadSHA: "eeeeeee",
				Forge:   "github",
			},
		}
		st.Git.Worktrees = []Worktree{
			{Path: "/r", Branch: "main", Head: "1111111", Dirty: &clean},
			{
				Path:            wtMergedDir,
				Branch:          "claude/288-test-spec-citations",
				Head:            "aaaaaaa",
				HeadCommittedAt: now.Add(-10 * time.Minute),
				Dirty:           &clean,
				AheadCount:      1,
			},
			{
				Path:            wtOpenDir,
				Branch:          "claude/233-pin-alpha5",
				Head:            "ccccccc",
				HeadCommittedAt: now.Add(-10 * time.Minute),
				Dirty:           &clean,
				AheadCount:      1,
			},
			{
				Path:            wtOlderDir,
				Branch:          "claude/older-linked-cache",
				Head:            "ddddddd",
				HeadCommittedAt: now.Add(-10 * time.Minute),
				Dirty:           &clean,
				AheadCount:      1,
			},
			{
				Path:            wtStaleMtimeDir,
				Branch:          "claude/stale-mtime-cache",
				Head:            "eeeeeee",
				HeadCommittedAt: now.Add(-10 * time.Minute),
				Dirty:           &clean,
				AheadCount:      1,
			},
		}
		st.ReconcileWorktrees()
		if !st.Git.Worktrees[1].MergedIntoMain {
			t.Fatal("expected ReconcileWorktrees to mark claude/288-test-spec-citations MergedIntoMain=true from newer worktree cache")
		}
		removeMerged, reasonMerged := st.PruneVerdict(st.Git.Worktrees[1], "claude", "/r", false)
		if !removeMerged || reasonMerged != "merged via #289" {
			t.Fatalf("merged worktree PruneVerdict = (%v, %q), want (true, %q)", removeMerged, reasonMerged, "merged via #289")
		}
		removeOpen, reasonOpen := st.PruneVerdict(st.Git.Worktrees[2], "claude", "/r", false)
		if removeOpen || reasonOpen != "open PR #290" {
			t.Fatalf("open worktree PruneVerdict = (%v, %q), want (false, %q)", removeOpen, reasonOpen, "open PR #290")
		}
		removeOlderLinked, reasonOlderLinked := st.PruneVerdict(st.Git.Worktrees[3], "claude", "/r", false)
		if !removeOlderLinked || reasonOlderLinked != "merged via #291 (cache 1h)" {
			t.Fatalf("older linked cache worktree PruneVerdict = (%v, %q), want (true, %q)", removeOlderLinked, reasonOlderLinked, "merged via #291 (cache 1h)")
		}
		removeStaleMtime, reasonStaleMtime := st.PruneVerdict(st.Git.Worktrees[4], "claude", "/r", false)
		if !removeStaleMtime || reasonStaleMtime != "merged via #292 (cache 1h)" {
			t.Fatalf("stale mtime linked cache worktree PruneVerdict = (%v, %q), want (true, %q)", removeStaleMtime, reasonStaleMtime, "merged via #292 (cache 1h)")
		}

		summary := BuildPickupSummary(st)
		for _, want := range []string{
			"- claude/288-test-spec-citations (head=aaaaaaa, dirty=false, ahead=1, merged via #289, prune candidate)",
			"- claude/233-pin-alpha5 (head=ccccccc, dirty=false, ahead=1, open PR #290)",
			"- claude/older-linked-cache (head=ddddddd, dirty=false, ahead=1, merged via #291 (cache 1h), prune candidate)",
			"- claude/stale-mtime-cache (head=eeeeeee, dirty=false, ahead=1, merged via #292 (cache 1h), prune candidate)",
		} {
			if !strings.Contains(summary, want) {
				t.Fatalf("pickup summary missing %q:\n%s", want, summary)
			}
		}
		for _, dir := range []string{wtMergedDir, wtOpenDir, wtOlderDir} {
			p := filepath.Clean(CacheFilePath(dir))
			if got := readCounts[p]; got != 1 {
				t.Fatalf("expected linked cache %s to be read exactly once across ReconcileWorktrees + PruneVerdict + BuildPickupSummary, got %d", p, got)
			}
		}
		staleMtimePath := filepath.Clean(CacheFilePath(wtStaleMtimeDir))
		if got := readCounts[staleMtimePath]; got != 0 {
			t.Fatalf("expected stale-mtime linked cache %s to be skipped by os.Stat without reading, got %d reads", staleMtimePath, got)
		}
	})
}
