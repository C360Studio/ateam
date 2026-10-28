package state

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// reviewCheckoutRe matches the directory review-pr creates for a detached review checkout (<agent>-review-pr-<n>[-suffix]).
var reviewCheckoutRe = regexp.MustCompile(`^(claude|codex|gemini|agy)-review-pr-([1-9][0-9]*)(?:-[A-Za-z0-9]+)?$`)

// PruneVerdict reports whether `ateam state prune` run as agent would remove wt, and why (#229).
// It reads only cached state, so pickup can call it inside the session-start budget.
// currentPath, when set, names the caller's checkout, which is never removed.
func (st *State) PruneVerdict(wt Worktree, agent, currentPath string) (bool, string) {
	branch := strings.TrimSpace(wt.Branch)
	detached := branch == "" || branch == "detached" || branch == "HEAD"
	label := st.WorktreePRStatus(wt)
	if branch == "main" || branch == "master" {
		return false, label
	}
	if (currentPath != "" && samePath(wt.Path, currentPath)) ||
		(!detached && branch == strings.TrimSpace(st.Git.Branch)) {
		return false, "current checkout"
	}
	owner := inferAgentFromBranch(branch)
	reviewPR := 0
	if detached {
		m := reviewCheckoutRe.FindStringSubmatch(filepath.Base(wt.Path))
		if m == nil {
			return false, label
		}
		owner = m[1]
		reviewPR, _ = strconv.Atoi(m[2])
	}
	if owner == "" {
		return false, "no agent prefix"
	}
	if agent = strings.ToLower(strings.TrimSpace(agent)); owner != agent {
		return false, "owned by " + owner
	}
	if wt.Dirty == nil {
		return false, "dirty state unknown"
	}
	if *wt.Dirty {
		return false, "dirty"
	}
	if detached {
		// The number alone can name PRs on two forges, and a review may have checked out an older head of a
		// PR that is still open, so any open PR with this number keeps the checkout. A head that is no closed
		// PR's head may hold unpushed review work.
		headMatched := false
		for _, c := range st.Cards {
			if !isPRCard(c) || cardPRNumber(c) != reviewPR {
				continue
			}
			if c.Column != "done" {
				return false, fmt.Sprintf("review of open PR #%d", reviewPR)
			}
			if c.HeadSHA != "" && SameCommitSHA(c.HeadSHA, wt.Head) {
				headMatched = true
			}
		}
		if !headMatched {
			return false, fmt.Sprintf("head is not PR #%d's head in the forge cache", reviewPR)
		}
		return true, fmt.Sprintf("review of closed PR #%d", reviewPR)
	}
	if strings.HasPrefix(label, "merged via ") || label == "merged (ancestor of main)" {
		return true, label
	}
	return false, label
}

func samePath(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	resolve := func(p string) string {
		if r, err := filepath.EvalSymlinks(p); err == nil {
			return r
		}
		return filepath.Clean(p)
	}
	return resolve(a) == resolve(b)
}

// Prune removes each linked worktree PruneVerdict clears for agent, with its local branch and, when the
// remote branch still points at the local head, its origin branch. Every other linked worktree is listed
// with the reason it was kept. It reports whether any git step failed.
func Prune(ctx context.Context, root string, st *State, agent string, dryRun bool, w io.Writer) (failed bool) {
	for _, wt := range st.Git.Worktrees {
		branch := strings.TrimSpace(wt.Branch)
		name := branch
		if name == "" || name == "detached" || name == "HEAD" {
			branch, name = "", filepath.Base(wt.Path)
		}
		remove, reason := st.PruneVerdict(wt, agent, root)
		if !remove {
			fmt.Fprintf(w, "kept %s: %s\n", name, reason)
			continue
		}
		if dryRun {
			fmt.Fprintf(w, "would remove %s: %s\n", name, reason)
			continue
		}
		approved, err := gitCombined(ctx, root, "rev-parse", "--verify", "--quiet", wt.Head+"^{commit}")
		if err != nil {
			fmt.Fprintf(w, "failed %s: cannot resolve head %s\n", name, wt.Head)
			failed = true
			continue
		}
		// A detached checkout's commits are reachable only from its own HEAD, which removal discards. Pin it to
		// a branch first: a commit landing after the checks below then advances the pin, and the pin's
		// expected-value delete keeps it.
		wantRef := "refs/heads/" + branch
		if branch == "" {
			wantRef = "refs/heads/ateam-prune/" + name
			if out, err := gitCombined(ctx, wt.Path, "switch", "-q", "-c", strings.TrimPrefix(wantRef, "refs/heads/")); err != nil {
				fmt.Fprintf(w, "failed %s: git switch -c: %s\n", name, out)
				failed = true
				continue
			}
		}
		// A commit made since the refresh leaves the tree clean, which git worktree remove would accept.
		liveHead, _ := gitCombined(ctx, wt.Path, "rev-parse", "HEAD")
		liveRef, _ := gitCombined(ctx, wt.Path, "symbolic-ref", "-q", "HEAD")
		if liveHead != approved || liveRef != wantRef {
			fmt.Fprintf(w, "kept %s: moved since refresh\n", name)
			continue
		}
		// No --force: git refuses a worktree that gained changes or untracked files since the refresh.
		if out, err := gitCombined(ctx, root, "worktree", "remove", wt.Path); err != nil {
			fmt.Fprintf(w, "failed %s: git worktree remove: %s\n", name, out)
			failed = true
			continue
		}
		// Delete the branch only while it still points at the approved head; squash merges rule out git branch -d.
		if out, err := gitCombined(ctx, root, "update-ref", "-d", wantRef, approved); err != nil {
			cur, revErr := gitCombined(ctx, root, "rev-parse", "--verify", "--quiet", wantRef)
			if revErr != nil || cur == approved {
				fmt.Fprintf(w, "failed %s: worktree removed; git update-ref -d %s: %s\n", name, wantRef, out)
				failed = true
				continue
			}
			fmt.Fprintf(w, "removed %s: %s; kept branch %s, which moved since refresh\n", name, reason, strings.TrimPrefix(wantRef, "refs/heads/"))
			continue
		}
		if branch == "" {
			fmt.Fprintf(w, "removed %s: %s\n", name, reason)
			continue
		}
		remoteHead := func() (string, error) {
			out, err := gitCombined(ctx, root, "ls-remote", "origin", wantRef)
			sha, _, _ := strings.Cut(out, "\t")
			if err != nil {
				return out, err
			}
			return sha, nil
		}
		remote, err := remoteHead()
		switch {
		case err != nil:
			fmt.Fprintf(w, "failed %s: worktree and branch removed; git ls-remote: %s\n", name, remote)
			failed = true
		case remote == "":
			fmt.Fprintf(w, "removed %s: %s; no remote branch\n", name, reason)
		case remote != approved:
			fmt.Fprintf(w, "removed %s: %s; kept origin/%s, which moved to %.7s\n", name, reason, branch, remote)
		default:
			// The lease makes the server refuse the delete if the branch moved after ls-remote.
			out, err := gitCombined(ctx, root, "push", "--force-with-lease="+wantRef+":"+approved, "origin", ":"+wantRef)
			if err == nil {
				fmt.Fprintf(w, "removed %s: %s; deleted origin/%s\n", name, reason, branch)
				continue
			}
			if now, lsErr := remoteHead(); lsErr == nil && now != "" && now != approved {
				fmt.Fprintf(w, "removed %s: %s; kept origin/%s, which moved to %.7s\n", name, reason, branch, now)
				continue
			}
			fmt.Fprintf(w, "failed %s: worktree and branch removed; git push --delete: %s\n", name, out)
			failed = true
		}
	}
	return failed
}

func gitCombined(ctx context.Context, dir string, args ...string) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}
