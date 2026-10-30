package testspec

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// lineSet holds the changed lines of one file; all marks an untracked file.
type lineSet struct {
	all   bool
	lines map[int]bool
}

// touches reports whether a changed line falls inside the declaration's span,
// from its first attached comment through its last line.
func (s lineSet) touches(d declaration) bool {
	if s.all {
		return true
	}
	start := d.Line
	if len(d.Comments) > 0 {
		start = d.Comments[0].Line
	}
	for line := start; line <= max(d.End, d.Line); line++ {
		if s.lines[line] {
			return true
		}
	}
	return false
}

// changedLines returns, by root-relative path, the lines added or changed between
// the merge base of base and HEAD and the working tree, plus every untracked file.
func changedLines(ctx context.Context, root, base string) (map[string]lineSet, error) {
	if strings.HasPrefix(base, "-") {
		return nil, fmt.Errorf("--base %q: a ref cannot start with '-'", base)
	}
	git := func(args ...string) (string, error) {
		cmd := exec.CommandContext(ctx, "git", args...)
		cmd.Dir = root
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		out, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("git %s: %w: %s", args[0], err, strings.TrimSpace(stderr.String()))
		}
		return string(out), nil
	}
	if _, err := git("rev-parse", "--verify", "--quiet", base+"^{commit}"); err != nil {
		return nil, fmt.Errorf("--base %q does not resolve to a commit here (fetch it, e.g. git fetch origin <branch>): %w", base, err)
	}
	mergeBase, err := git("merge-base", base, "HEAD")
	if err != nil {
		return nil, fmt.Errorf("--base %q has no merge base with HEAD (shallow clone? fetch full history: git fetch --unshallow, or actions/checkout fetch-depth: 0): %w", base, err)
	}
	patch, err := git("-c", "core.quotePath=false", "diff", "-U0", "--no-ext-diff", "--no-textconv", "--no-color",
		"--relative", "--src-prefix=a/", "--dst-prefix=b/", strings.TrimSpace(mergeBase), "--")
	if err != nil {
		return nil, err
	}
	changed, err := parseHunks(patch)
	if err != nil {
		return nil, err
	}
	untracked, err := git("ls-files", "--others", "--exclude-standard", "-z")
	if err != nil {
		return nil, err
	}
	for _, path := range strings.Split(untracked, "\x00") {
		if path != "" {
			changed[path] = lineSet{all: true}
		}
	}
	return changed, nil
}

// parseHunks reads the added lines of a unified diff from the hunk bodies, so
// context lines (diff.interHunkContext can add them even under -U0) never count.
// A removed line marks the line now at its position, so removing a citation counts.
// Bodies are consumed by their header counts, so payload that looks like a file
// header (an added "++ b/x" line prints as "+++ b/x") cannot switch files.
func parseHunks(patch string) (map[string]lineSet, error) {
	changed := map[string]lineSet{}
	file := ""
	oldLeft, newLeft := 0, 0 // body lines still to consume in the current hunk
	cursor := 0              // new-side line number of the next body line
	mark := func(line int) {
		if file != "" {
			changed[file].lines[line] = true
		}
	}
	for _, line := range strings.Split(patch, "\n") {
		if oldLeft > 0 || newLeft > 0 {
			switch {
			case strings.HasPrefix(line, "-") && oldLeft > 0:
				mark(cursor)
				oldLeft--
			case strings.HasPrefix(line, "+") && newLeft > 0:
				mark(cursor)
				cursor++
				newLeft--
			case strings.HasPrefix(line, " ") && oldLeft > 0 && newLeft > 0:
				cursor++
				oldLeft--
				newLeft--
			case strings.HasPrefix(line, "\\"): // "\ No newline at end of file"
			default:
				return nil, fmt.Errorf("parse git diff: hunk body ended early at %q", line)
			}
			continue
		}
		switch {
		case strings.HasPrefix(line, "diff --git "):
			file = ""
		case strings.HasPrefix(line, "+++ "):
			// git ends a ---/+++ name containing a space with a separator tab; a name
			// with a real tab is quoted instead, so one trailing tab is never part of it.
			name := strings.TrimSuffix(strings.TrimPrefix(line, "+++ "), "\t")
			switch {
			case name == "/dev/null":
				file = ""
			case strings.HasPrefix(name, "b/"):
				file = strings.TrimPrefix(name, "b/")
				if _, ok := changed[file]; !ok {
					changed[file] = lineSet{lines: map[int]bool{}}
				}
			default:
				// shortcut: quoted paths (tabs, quotes, control bytes) fail loudly, unescape them when a test path needs one
				return nil, fmt.Errorf("parse git diff: unsupported path %s", name)
			}
		case strings.HasPrefix(line, "@@ "):
			fields := strings.Fields(line)
			if len(fields) < 3 || !strings.HasPrefix(fields[1], "-") || !strings.HasPrefix(fields[2], "+") {
				return nil, fmt.Errorf("parse git diff: malformed hunk header %q", line)
			}
			_, oldCount, okOld := hunkRange(fields[1])
			start, count, okNew := hunkRange(fields[2])
			if !okOld || !okNew {
				return nil, fmt.Errorf("parse git diff: malformed hunk header %q", line)
			}
			oldLeft, newLeft, cursor = oldCount, count, start
			if count == 0 {
				cursor = start + 1 // git names the line before a pure deletion
			}
		}
	}
	if oldLeft > 0 || newLeft > 0 {
		return nil, fmt.Errorf("parse git diff: truncated hunk")
	}
	return changed, nil
}

// hunkRange parses "-a[,n]" or "+a[,n]"; a missing count means 1.
func hunkRange(field string) (start, count int, ok bool) {
	startText, countText, hasCount := strings.Cut(field[1:], ",")
	start, err := strconv.Atoi(startText)
	count = 1
	if err == nil && hasCount {
		count, err = strconv.Atoi(countText)
	}
	return start, count, err == nil && start >= 0 && count >= 0
}
