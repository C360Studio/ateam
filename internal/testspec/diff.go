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

// parseHunks reads the new-side line ranges of a -U0 unified diff. A deletion-only
// hunk marks the line after the deletion point, so removing a citation counts.
func parseHunks(patch string) (map[string]lineSet, error) {
	changed := map[string]lineSet{}
	file := ""
	for _, line := range strings.Split(patch, "\n") {
		switch {
		case strings.HasPrefix(line, "+++ "):
			name := strings.TrimPrefix(line, "+++ ")
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
			if file == "" {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) < 3 || !strings.HasPrefix(fields[2], "+") {
				return nil, fmt.Errorf("parse git diff: malformed hunk header %q", line)
			}
			startText, countText, hasCount := strings.Cut(strings.TrimPrefix(fields[2], "+"), ",")
			start, err := strconv.Atoi(startText)
			count := 1
			if err == nil && hasCount {
				count, err = strconv.Atoi(countText)
			}
			if err != nil || start < 0 || count < 0 {
				return nil, fmt.Errorf("parse git diff: malformed hunk header %q", line)
			}
			if count == 0 {
				changed[file].lines[start+1] = true
			}
			for n := start; n < start+count; n++ {
				changed[file].lines[n] = true
			}
		}
	}
	return changed, nil
}
