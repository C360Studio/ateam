package state

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// wholeReviewTagPrefix names the annotated tags that record a whole-repository review (#399).
const wholeReviewTagPrefix = "review/whole/"

// WholeReview is what landed on the default branch's remote tip since the newest whole-repository
// review tag reachable from it. Tag is "" when no such tag exists ("never").
type WholeReview struct {
	Tag     string `json:"tag,omitempty"`
	Date    string `json:"date,omitempty"`
	Commits int    `json:"commits"`
	Added   int    `json:"added"`
	Deleted int    `json:"deleted"`
}

// String renders the pickup-line value of since-review=.
func (w WholeReview) String() string {
	if w.Tag == "" {
		return "never"
	}
	return fmt.Sprintf("%d commits, +%d/-%d lines (%s)", w.Commits, w.Added, w.Deleted, w.Date)
}

var shortstatCount = regexp.MustCompile(`(\d+) (insertion|deletion)`)

// parseShortstat returns the insertions and deletions in `git diff --shortstat` output.
func parseShortstat(s string) (added, deleted int) {
	for _, m := range shortstatCount.FindAllStringSubmatch(s, -1) {
		n, _ := strconv.Atoi(m[1]) // the pattern only matches digits
		if m[2] == "insertion" {
			added = n
		} else {
			deleted = n
		}
	}
	return added, deleted
}

// firstParentIndex maps each commit on ref's first-parent chain to its distance from ref (ref itself is 0).
// A commit that reached the branch through a merge's second parent is absent: it never was the branch.
func firstParentIndex(ctx context.Context, root, ref string) map[string]int {
	index := map[string]int{}
	for i, sha := range strings.Fields(runGitOutput(ctx, root, "rev-list", "--first-parent", ref)) {
		index[sha] = i
	}
	return index
}

// snapshotWholeReview counts first-parent commits and diff lines from the newest review tag on tip's
// first-parent chain to tip. "Newest" is the tag closest to tip on that chain; tags on one commit go to
// the later name, which sorts by date. Tags on merged side branches or off the branch are ignored.
func snapshotWholeReview(ctx context.Context, root, tip string) WholeReview {
	chain := firstParentIndex(ctx, root, tip)
	var best WholeReview
	bestCommit := ""
	tags := runGitOutput(ctx, root, "tag", "--list", wholeReviewTagPrefix+"*", "--sort=refname", "--format=%(refname:short)")
	for _, tag := range strings.Fields(tags) {
		commit := runGitLine(ctx, root, "rev-list", "-n", "1", tag)
		count, onChain := chain[commit]
		if !onChain {
			continue
		}
		if best.Tag == "" || count <= best.Commits {
			// The date is the one RecordWholeReview put in the name: review/whole/<YYYY-MM-DD>-<short sha>.
			date := strings.TrimPrefix(tag, wholeReviewTagPrefix)
			if len(date) > 10 {
				date = date[:10]
			}
			best = WholeReview{Tag: tag, Date: date, Commits: count}
			bestCommit = commit
		}
	}
	if best.Tag != "" {
		best.Added, best.Deleted = parseShortstat(runGitOutput(ctx, root, "diff", "--shortstat", bestCommit, tip))
	}
	return best
}

// RecordWholeReview creates the annotated tag review/whole/<date>-<short sha> at commit `at`, which must
// be on the default branch's first-parent chain, and pushes it to origin unless push is false. It returns the tag name.
func RecordWholeReview(ctx context.Context, root, at, note string, push bool, now time.Time) (string, error) {
	sha, err := gitCombined(ctx, root, "rev-parse", "--verify", "--quiet", at+"^{commit}")
	if err != nil || sha == "" {
		return "", fmt.Errorf("review whole: %q is not a commit", at)
	}
	branch := ""
	for _, ref := range defaultBranchRefs(root) {
		if _, ok := firstParentIndex(ctx, root, ref)[sha]; ok {
			branch = ref
			break
		}
	}
	if branch == "" {
		return "", fmt.Errorf("review whole: %s is not on the default branch's first-parent history (%s)", sha[:7], strings.Join(defaultBranchRefs(root), ", "))
	}
	tag := wholeReviewTagPrefix + now.Format("2006-01-02") + "-" + sha[:7]
	msg := "Whole-repository review at " + sha
	if note = strings.TrimSpace(note); note != "" {
		msg += "\n\n" + note
	}
	if out, err := gitCombined(ctx, root, "tag", "-a", tag, sha, "-m", msg); err != nil {
		return "", fmt.Errorf("review whole: git tag %s: %v: %s", tag, err, out)
	}
	if push {
		if out, err := gitCombined(ctx, root, "push", "origin", "refs/tags/"+tag); err != nil {
			return tag, fmt.Errorf("review whole: created %s but git push failed (run git push origin refs/tags/%s): %v: %s", tag, tag, err, out)
		}
	}
	return tag, nil
}

// defaultBranchRefs lists the refs a reviewed commit may sit on: origin's and the local copy of the
// configured default branch, or of main and master when none is configured.
func defaultBranchRefs(root string) []string {
	branches := fallbackDefaultBranches[:]
	if b := settingsOrDefault(root).DefaultBranch; b != "" {
		branches = []string{b}
	}
	var refs []string
	for _, b := range branches {
		refs = append(refs, "refs/remotes/origin/"+b, "refs/heads/"+b)
	}
	return refs
}
