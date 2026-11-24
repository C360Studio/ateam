package state

import (
	"context"
	"strings"
	"testing"
	"time"
)

// wholeReviewRepo builds main with commits A (a.txt "1\n2\n"), B (+2 lines in b.txt) and C (a.txt line
// "2" -> "x", +1/-1), a side branch commit D, and points origin/main at C. Totals from A to C are hand
// derived: 2 first-parent commits, +3/-1 lines.
func wholeReviewRepo(t *testing.T) (dir string, a, b, d string) {
	t.Helper()
	dir = t.TempDir()
	gitIn(t, dir, "init", "-q", "-b", "main")
	writeRepoFile(t, dir, "a.txt", "1\n2\n")
	gitIn(t, dir, "add", "-A")
	gitIn(t, dir, "commit", "-q", "-m", "A")
	a = gitIn(t, dir, "rev-parse", "HEAD")
	writeRepoFile(t, dir, "b.txt", "3\n4\n")
	gitIn(t, dir, "add", "-A")
	gitIn(t, dir, "commit", "-q", "-m", "B")
	b = gitIn(t, dir, "rev-parse", "HEAD")
	writeRepoFile(t, dir, "a.txt", "1\nx\n")
	gitIn(t, dir, "commit", "-q", "-am", "C")
	gitIn(t, dir, "update-ref", "refs/remotes/origin/main", "HEAD")
	gitIn(t, dir, "checkout", "-q", "-b", "side", a)
	writeRepoFile(t, dir, "side.txt", "s\n")
	gitIn(t, dir, "add", "-A")
	gitIn(t, dir, "commit", "-q", "-m", "D")
	d = gitIn(t, dir, "rev-parse", "HEAD")
	gitIn(t, dir, "checkout", "-q", "main")
	return dir, a, b, d
}

func record(t *testing.T, dir, at, day string) string {
	t.Helper()
	now, err := time.Parse("2006-01-02", day)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_COMMITTER_NAME", "t")
	t.Setenv("GIT_COMMITTER_EMAIL", "t@t")
	tag, err := RecordWholeReview(context.Background(), dir, at, "findings: #1", false, now)
	if err != nil {
		t.Fatalf("RecordWholeReview(%s): %v", at, err)
	}
	return tag
}

// spec: openspec/specs/harness/spec.md#Requirement: Whole-repository review marker
func TestWholeReviewCounts(t *testing.T) {
	ctx := context.Background()
	dir, a, b, d := wholeReviewRepo(t)

	gs := SnapshotGit(ctx, dir)
	if gs.WholeReview == nil || gs.WholeReview.String() != "never" {
		t.Fatalf("no tag: WholeReview = %+v, want never", gs.WholeReview)
	}
	if got := BuildPickupSummary(&State{Git: gs}); !strings.Contains(got, "| holds=0 | since-review=never |") {
		t.Fatalf("summary lacks since-review=never:\n%s", got)
	}

	if tag := record(t, dir, a, "2026-11-01"); tag != "review/whole/2026-11-01-"+a[:7] {
		t.Fatalf("tag = %q", tag)
	}
	// A tag on a commit origin/main does not contain is never counted.
	gitIn(t, dir, "tag", "-a", "review/whole/2026-12-01-"+d[:7], d, "-m", "side")
	got := *SnapshotGit(ctx, dir).WholeReview
	want := WholeReview{Tag: "review/whole/2026-11-01-" + a[:7], Date: "2026-11-01", Commits: 2, Added: 3, Deleted: 1}
	if got != want {
		t.Fatalf("after A: %+v, want %+v", got, want)
	}
	if s := BuildPickupSummary(&State{Git: SnapshotGit(ctx, dir)}); !strings.Contains(s, "since-review=2 commits, +3/-1 lines (2026-11-01)") {
		t.Fatalf("summary:\n%s", s)
	}

	record(t, dir, b, "2026-11-10")
	got = *SnapshotGit(ctx, dir).WholeReview
	want = WholeReview{Tag: "review/whole/2026-11-10-" + b[:7], Date: "2026-11-10", Commits: 1, Added: 1, Deleted: 1}
	if got != want {
		t.Fatalf("newest of two: %+v, want %+v", got, want)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Whole-repository review marker
func TestWholeReviewWithoutRemoteDefault(t *testing.T) {
	dir := t.TempDir()
	gitIn(t, dir, "init", "-q", "-b", "main")
	writeRepoFile(t, dir, "a.txt", "1\n")
	gitIn(t, dir, "add", "-A")
	gitIn(t, dir, "commit", "-q", "-m", "A")
	gs := SnapshotGit(context.Background(), dir)
	if gs.WholeReview != nil {
		t.Fatalf("no origin/main: WholeReview = %+v, want nil", gs.WholeReview)
	}
	if s := BuildPickupSummary(&State{Git: gs}); strings.Contains(s, "since-review") {
		t.Fatalf("summary prints since-review with nothing to count against:\n%s", s)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Whole-repository review marker
func TestRecordWholeReview(t *testing.T) {
	ctx := context.Background()
	dir, a, _, d := wholeReviewRepo(t)
	origin := t.TempDir()
	gitIn(t, origin, "init", "-q", "--bare")
	gitIn(t, dir, "remote", "add", "origin", origin)
	t.Setenv("GIT_COMMITTER_NAME", "t")
	t.Setenv("GIT_COMMITTER_EMAIL", "t@t")
	now := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)

	if _, err := RecordWholeReview(ctx, dir, d, "", false, now); err == nil || !strings.Contains(err.Error(), "not on the default branch") {
		t.Fatalf("side-branch commit: err = %v, want refusal", err)
	}
	if tags := gitIn(t, dir, "tag", "--list", "review/whole/*"); tags != "" {
		t.Fatalf("refusal left tags: %q", tags)
	}

	tag, err := RecordWholeReview(ctx, dir, a, "findings: #1", false, now)
	if err != nil {
		t.Fatal(err)
	}
	if remote := gitIn(t, dir, "ls-remote", "--tags", "origin"); remote != "" {
		t.Fatalf("--no-push pushed: %q", remote)
	}
	if kind := gitIn(t, dir, "cat-file", "-t", tag); kind != "tag" {
		t.Fatalf("%s is a %s, want an annotated tag", tag, kind)
	}
	if msg := gitIn(t, dir, "tag", "-l", "--format=%(contents)", tag); !strings.Contains(msg, "Whole-repository review at "+a) || !strings.Contains(msg, "findings: #1") {
		t.Fatalf("tag message = %q", msg)
	}

	gitIn(t, dir, "tag", "-d", tag)
	if _, err := RecordWholeReview(ctx, dir, a, "", true, now); err != nil {
		t.Fatal(err)
	}
	if remote := gitIn(t, dir, "ls-remote", "--tags", "origin"); !strings.Contains(remote, "refs/tags/"+tag) {
		t.Fatalf("push: origin tags = %q, want %s", remote, tag)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Whole-repository review marker
func TestWholeReviewCountsMergeCommitOnce(t *testing.T) {
	// A --no-ff merge of a two-commit branch is one first-parent commit on main (one PR), not three.
	dir := t.TempDir()
	gitIn(t, dir, "init", "-q", "-b", "main")
	writeRepoFile(t, dir, "a.txt", "1\n")
	gitIn(t, dir, "add", "-A")
	gitIn(t, dir, "commit", "-q", "-m", "A")
	a := gitIn(t, dir, "rev-parse", "HEAD")
	gitIn(t, dir, "update-ref", "refs/remotes/origin/main", "HEAD")
	record(t, dir, a, "2026-11-01")
	gitIn(t, dir, "checkout", "-q", "-b", "feat")
	for _, f := range []string{"f1.txt", "f2.txt"} {
		writeRepoFile(t, dir, f, "x\n")
		gitIn(t, dir, "add", "-A")
		gitIn(t, dir, "commit", "-q", "-m", f)
	}
	gitIn(t, dir, "checkout", "-q", "main")
	gitIn(t, dir, "merge", "-q", "--no-ff", "-m", "Merge feat", "feat")
	gitIn(t, dir, "update-ref", "refs/remotes/origin/main", "HEAD")
	got := *SnapshotGit(context.Background(), dir).WholeReview
	if got.Commits != 1 || got.Added != 2 || got.Deleted != 0 {
		t.Fatalf("after one merge: %+v, want 1 commit, +2/-0", got)
	}
}
