package bootstrap

import (
	"context"
	"errors"
	"io/fs"

	"github.com/c360studio/ateam/internal/taskrunner"
)

// CheckVerifyTarget fails doctor --check when `task verify` cannot run in root: the shipped preflight and review-pr
// skills and the protocol call it by name, so a repository without it breaks every review (#415). It lists targets
// the way `ateam task` does, so a verify task that exists only as an alias does not count.
func CheckVerifyTarget(ctx context.Context, root string) []string {
	targets, err := taskrunner.ListTargets(ctx, root)
	if errors.Is(err, fs.ErrNotExist) {
		return []string{"Taskfile.yml: missing; the shipped skills and protocol run task verify"}
	}
	if err != nil {
		return []string{"Taskfile.yml: cannot list tasks (" + err.Error() + "); the shipped skills and protocol run task verify"}
	}
	for _, t := range targets {
		if t.Name == "verify" {
			return nil
		}
	}
	return []string{"Taskfile.yml: no verify target; the shipped skills and protocol run task verify"}
}
