package state

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/c360studio/ateam/internal/review"
)

// reviewPolicyFile is read from the checkout being checked (design_paths, hold_my_beer); editing it is itself a
// gate-file change.
const reviewPolicyFile = ".agents/review-policy.json"

// CheckDesignPaths applies the policy's design_paths globs (#401): every changed path matching one needs an
// added or modified openspec/changes/**/design.md in the same diff. The diff runs from the merge base of base
// and HEAD to the working tree, untracked files included. An empty base tries the default branch on origin;
// when that cannot resolve, the check reports that it did not run instead of passing silently. It returns
// the report, the paths that failed (one message each) and an error for a check that could not run as asked.
func CheckDesignPaths(ctx context.Context, root, base string) (string, []string, error) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(reviewPolicyFile)))
	if errors.Is(err, os.ErrNotExist) {
		return "", nil, nil
	}
	if err != nil {
		return "", nil, fmt.Errorf("design paths: read %s: %w", reviewPolicyFile, err)
	}
	policy, err := review.ParsePolicy(data)
	if err != nil {
		return "", nil, fmt.Errorf("design paths: %s: %w", reviewPolicyFile, err)
	}
	if len(policy.DesignPaths) == 0 {
		return "", nil, nil
	}
	explicit := base != ""
	if !explicit {
		base = defaultDesignBase(ctx, root)
		if base == "" {
			return "design paths: not checked (no origin default branch to diff against; pass --base <ref>)\n", nil, nil
		}
	}
	changes, err := changedPaths(ctx, root, base)
	if err != nil {
		if !explicit {
			return fmt.Sprintf("design paths: not checked (%v; pass --base <ref>)\n", err), nil, nil
		}
		return "", nil, fmt.Errorf("design paths: %w", err)
	}
	hasDesign := false
	for p, deleted := range changes {
		if !deleted && isChangeDesign(p) {
			hasDesign = true
			break
		}
	}
	var failures []string
	for _, p := range sortedKeys(changes) {
		for _, g := range policy.DesignPaths {
			if matchDesignGlob(g, p) {
				if !hasDesign {
					failures = append(failures, fmt.Sprintf("design paths: %s matches design_paths %q, but the diff against %s adds or modifies no openspec/changes/**/design.md (put this change on the design track)", p, g, base))
				}
				break
			}
		}
	}
	if len(failures) > 0 {
		return "", failures, nil
	}
	return fmt.Sprintf("design paths: ok (%d changed paths against %s, %d globs)\n", len(changes), base, len(policy.DesignPaths)), nil, nil
}

// matchDesignGlob matches a slash path against a path.Match glob; a trailing /** matches anything below the
// directories the rest of the glob names.
func matchDesignGlob(glob, p string) bool {
	if prefix, ok := strings.CutSuffix(glob, "/**"); ok {
		n := len(strings.Split(prefix, "/"))
		segs := strings.Split(p, "/")
		if len(segs) <= n {
			return false
		}
		ok, _ := path.Match(prefix, strings.Join(segs[:n], "/"))
		return ok
	}
	ok, _ := path.Match(glob, p)
	return ok
}

// isChangeDesign reports an OpenSpec change's design.md, in flight or archived.
func isChangeDesign(p string) bool {
	in, _ := path.Match("openspec/changes/*/design.md", p)
	archived, _ := path.Match("openspec/changes/archive/*/design.md", p)
	return in || archived
}

// defaultDesignBase names origin's default branch from settings, else origin/main or origin/master, or "".
func defaultDesignBase(ctx context.Context, root string) string {
	candidates := []string{"origin/main", "origin/master"}
	if b := settingsOrDefault(root).DefaultBranch; b != "" {
		candidates = []string{"origin/" + b}
	}
	for _, c := range candidates {
		if _, err := gitRun(ctx, root, "rev-parse", "--verify", "--quiet", c+"^{commit}"); err == nil {
			return c
		}
	}
	return ""
}

// changedPaths maps each root-relative path changed since the merge base of base and HEAD (working tree and
// untracked files included) to whether it was deleted. Renames count as a delete plus an add.
func changedPaths(ctx context.Context, root, base string) (map[string]bool, error) {
	if strings.HasPrefix(base, "-") {
		return nil, fmt.Errorf("--base %q: a ref cannot start with '-'", base)
	}
	if _, err := gitRun(ctx, root, "rev-parse", "--verify", "--quiet", base+"^{commit}"); err != nil {
		return nil, fmt.Errorf("--base %q does not resolve to a commit here (fetch it, e.g. git fetch origin <branch>)", base)
	}
	mergeBase, err := gitRun(ctx, root, "merge-base", base, "HEAD")
	if err != nil {
		return nil, fmt.Errorf("%s has no merge base with HEAD (shallow clone? use actions/checkout fetch-depth: 0)", base)
	}
	out, err := gitRun(ctx, root, "diff", "--name-status", "--no-renames", "-z", "--relative", strings.TrimSpace(mergeBase), "--")
	if err != nil {
		return nil, err
	}
	changes := map[string]bool{}
	fields := strings.Split(out, "\x00")
	for i := 0; i+1 < len(fields); i += 2 {
		changes[fields[i+1]] = fields[i] == "D"
	}
	untracked, err := gitRun(ctx, root, "ls-files", "--others", "--exclude-standard", "-z")
	if err != nil {
		return nil, err
	}
	for _, p := range strings.Split(untracked, "\x00") {
		if p != "" {
			changes[p] = false
		}
	}
	return changes, nil
}

func gitRun(ctx context.Context, root string, args ...string) (string, error) {
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

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// RunSpecCheck is spec-check as both the CLI and the MCP spec_check action run it: OpenSpec validation, then
// design_paths (#401). A design_paths failure is appended to out, so a caller that prints out on failure shows it.
func RunSpecCheck(ctx context.Context, root, base string) (string, int, error) {
	out, code, err := CheckOpenSpec(ctx, root)
	if code != 0 || err != nil {
		return out, code, err
	}
	report, failures, err := CheckDesignPaths(ctx, root, base)
	if err != nil {
		return out + err.Error() + "\n", 1, err
	}
	if len(failures) > 0 {
		return out + strings.Join(failures, "\n") + "\n", 1, fmt.Errorf("design paths: %d changed path(s) need a design.md in the same diff", len(failures))
	}
	return out + report, 0, nil
}
