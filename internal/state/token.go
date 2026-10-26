package state

import (
	"context"
	"os"
	"os/exec"
	"strings"
)

// Forge credentials are named by operation, not by tool (#160): reads (board, sync, pickup, review evidence) use
// ATEAM_TOKEN_READ and writes (comments, labels, statuses, merges) use ATEAM_TOKEN_WRITE, so each token can carry
// only its role's scopes. Either one alone serves both roles.
const (
	envTokenRead  = "ATEAM_TOKEN_READ"
	envTokenWrite = "ATEAM_TOKEN_WRITE"
)

// ReadToken returns the trimmed token for forge reads: ATEAM_TOKEN_READ, else ATEAM_TOKEN_WRITE. It is empty when
// neither is set, and the caller then falls back to its forge's own token resolution.
func ReadToken() string { return firstEnv(envTokenRead, envTokenWrite) }

// WriteToken returns the trimmed token for forge writes: ATEAM_TOKEN_WRITE, else ATEAM_TOKEN_READ. It is empty when
// neither is set, and the caller then falls back to its forge's own token resolution.
func WriteToken() string { return firstEnv(envTokenWrite, envTokenRead) }

// firstEnv returns the first named variable whose trimmed value is not empty; whitespace never suppresses a fallback.
func firstEnv(names ...string) string {
	for _, name := range names {
		if v := strings.TrimSpace(os.Getenv(name)); v != "" {
			return v
		}
	}
	return ""
}

// giteaToken returns the role token, else the Gitea token resolution that predates roles.
func giteaToken(role string) string {
	if role != "" {
		return role
	}
	return firstEnv("GITEA_TOKEN", "ATEAM_GITEA_TOKEN")
}

// ghCommand builds a gh invocation that authenticates with the role token as GH_TOKEN when one is set, and with
// gh's own resolution (GH_TOKEN, GITHUB_TOKEN, gh auth) otherwise.
func ghCommand(ctx context.Context, role string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "gh", args...)
	if role != "" {
		// The last duplicate key wins in exec, so this overrides an inherited GH_TOKEN.
		cmd.Env = append(os.Environ(), "GH_TOKEN="+role)
	}
	return cmd
}
