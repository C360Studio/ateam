package review

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// BaselinePolicy reads policy from the forge-reported base, never the candidate
// branch. It requires a clean checkout of the exact reviewed head and an
// up-to-date branch. Initial policy adoption must land separately before gating.
func BaselinePolicy(ctx context.Context, root, base, head string) ([]byte, error) {
	if !commitID(base) || !commitID(head) {
		return nil, fmt.Errorf("base and head must be full commit SHAs")
	}
	current, err := gitRead(ctx, root, "rev-parse", "HEAD")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(string(current)) != head {
		return nil, fmt.Errorf("checkout does not match reviewed head %s", head)
	}
	dirty, err := gitRead(ctx, root, "status", "--porcelain", "--untracked-files=normal")
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(dirty)) != 0 {
		return nil, fmt.Errorf("review check requires a clean checkout")
	}
	if _, err := gitRead(ctx, root, "merge-base", "--is-ancestor", base, head); err != nil {
		return nil, fmt.Errorf("reviewed head must contain the current base: %w", err)
	}
	data, err := gitRead(ctx, root, "show", base+":.agents/review-policy.json")
	if err != nil {
		return nil, fmt.Errorf("trusted base has no readable review policy; adopt owner-approved policy separately: %w", err)
	}
	return data, nil
}

// GitRemote returns the configured origin URL without executing a network call.
func GitRemote(ctx context.Context, root string) (string, error) {
	data, err := gitRead(ctx, root, "remote", "get-url", "origin")
	return strings.TrimSpace(string(data)), err
}

func gitRead(ctx context.Context, root string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = root
	out := &boundedOutput{}
	cmd.Stdout = out
	// Do not return arbitrary subprocess stderr as evidence or credential output.
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git %s failed: %w", args[0], err)
	}
	return out.Bytes(), nil
}

type boundedOutput struct{ buf bytes.Buffer }

func (b *boundedOutput) Bytes() []byte { return b.buf.Bytes() }
func (b *boundedOutput) Write(p []byte) (int, error) {
	if b.buf.Len()+len(p) > 2<<20 {
		return 0, fmt.Errorf("git output exceeds 2 MiB")
	}
	return b.buf.Write(p)
}
