package review

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBaselinePolicyRejectsCandidateAuthority(t *testing.T) {
	root := t.TempDir()
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-q")
	git("config", "user.name", "Fixture")
	git("config", "user.email", "fixture@example.invalid")
	if err := os.Mkdir(filepath.Join(root, ".agents"), 0700); err != nil {
		t.Fatal(err)
	}
	policyPath := filepath.Join(root, ".agents", "review-policy.json")
	if err := os.WriteFile(policyPath, []byte("trusted baseline"), 0600); err != nil {
		t.Fatal(err)
	}
	git("add", ".")
	git("commit", "-qm", "baseline")
	base := git("rev-parse", "HEAD")
	if err := os.WriteFile(policyPath, []byte("candidate policy weakening"), 0600); err != nil {
		t.Fatal(err)
	}
	git("add", ".")
	git("commit", "-qm", "candidate")
	head := git("rev-parse", "HEAD")
	got, err := BaselinePolicy(context.Background(), root, base, head)
	if err != nil || string(got) != "trusted baseline" {
		t.Fatalf("candidate controlled governing policy: %q %v", got, err)
	}
	if _, err := BaselinePolicy(context.Background(), root, base, base); err == nil {
		t.Fatal("mismatched checkout accepted")
	}
	if _, err := BaselinePolicy(context.Background(), root, "--help", head); err == nil {
		t.Fatal("non-SHA base accepted")
	}
	if err := os.WriteFile(filepath.Join(root, "untracked"), []byte("dirty"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := BaselinePolicy(context.Background(), root, base, head); err == nil {
		t.Fatal("dirty checkout accepted")
	}
}

func TestBaselinePolicyRequiresAdoptionAndAncestor(t *testing.T) {
	root := t.TempDir()
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-q")
	git("config", "user.name", "Fixture")
	git("config", "user.email", "fixture@example.invalid")
	git("commit", "--allow-empty", "-qm", "no policy")
	base := git("rev-parse", "HEAD")
	if _, err := BaselinePolicy(context.Background(), root, base, base); err == nil {
		t.Fatal("missing baseline policy accepted")
	}
	git("commit", "--allow-empty", "-qm", "later")
	head := git("rev-parse", "HEAD")
	// Current checkout is head but the future object is not an ancestor of it.
	future := git("commit-tree", "HEAD^{tree}", "-p", head, "-m", "future")
	if _, err := BaselinePolicy(context.Background(), root, future, head); err == nil {
		t.Fatal("non-ancestor base accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := BaselinePolicy(ctx, root, base, head); err == nil {
		t.Fatal("cancelled read succeeded")
	}
}

func TestGitOutputBound(t *testing.T) {
	b := &boundedOutput{}
	if _, err := b.Write(make([]byte, 2<<20)); err != nil {
		t.Fatal(err)
	}
	if _, err := b.Write([]byte{1}); err == nil {
		t.Fatal("oversized output accepted")
	}
	if len(b.Bytes()) != 2<<20 {
		t.Fatal("failed write changed buffer")
	}
}
