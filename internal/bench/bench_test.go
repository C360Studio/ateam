package bench

import (
	"context"
	"strings"
	"testing"
)

func TestSelfTestAndMutation(t *testing.T) {
	rep, err := RunSelfTest(context.Background())
	if err != nil {
		t.Fatalf("RunSelfTest failed: %v", err)
	}
	if !rep.SelfTestPassed {
		t.Fatalf("expected SelfTestPassed=true, got %+v", rep)
	}
	if rep.SafetyPassRatePct != 100 {
		t.Errorf("SafetyPassRatePct = %d, want 100", rep.SafetyPassRatePct)
	}
	if rep.MutationCatchPct != 100 {
		t.Errorf("MutationCatchPct = %d, want 100", rep.MutationCatchPct)
	}
	md := FormatMarkdown(rep)
	if !strings.Contains(md, "relay-openspec-hold") || !strings.Contains(md, "gitea-webhook") {
		t.Errorf("FormatMarkdown missing expected task rows:\n%s", md)
	}
}

func TestCommentSpoofingAndEarlyReturnBypassRejected(t *testing.T) {
	// Probe 8: An always-accepting path resolver with expected keywords in a comment must fail evaluation.
	spoofed := `package safepath
import "path/filepath"
// filepath.Separator HasPrefix !strings.HasPrefix
func Resolve(base, rel string) (string, error) {
	return filepath.Join(base, rel), nil
}`
	if CheckCodingCandidate("safe-path", spoofed) {
		t.Fatal("CheckCodingCandidate accepted comment-spoofed always-accepting path resolver")
	}

	// Astra #4: Unconditional/early return (`if true`) before safety check must be rejected.
	earlyReturnBypass := `package safepath
import ("errors"; "path/filepath"; "strings")
func Resolve(base, rel string) (string, error) {
	if true { return rel, nil }
	absBase, err := filepath.Abs(base)
	if err != nil { return "", err }
	target, err := filepath.Abs(filepath.Join(absBase, rel))
	if err != nil { return "", err }
	if target != absBase && !strings.HasPrefix(target, absBase+string(filepath.Separator)) {
		return "", errors.New("blocked")
	}
	return target, nil
}`
	if CheckCodingCandidate("safe-path", earlyReturnBypass) {
		t.Fatal("CheckCodingCandidate accepted if true early-return bypass before safety guard")
	}

	// G-03: `if 1 == 1` early-return bypass must also be rejected behaviorally.
	oneEqOneBypass := `package safepath
import ("errors"; "path/filepath"; "strings")
func Resolve(base, rel string) (string, error) {
	if 1 == 1 { return rel, nil }
	absBase, err := filepath.Abs(base)
	if err != nil { return "", err }
	target, err := filepath.Abs(filepath.Join(absBase, rel))
	if err != nil { return "", err }
	if target != absBase && !strings.HasPrefix(target, absBase+string(filepath.Separator)) {
		return "", errors.New("blocked")
	}
	return target, nil
}`
	if CheckCodingCandidate("safe-path", oneEqOneBypass) {
		t.Fatal("CheckCodingCandidate accepted if 1 == 1 early-return bypass before safety guard")
	}

	nonConstEarlyReturn := `package safepath
import ("errors"; "path/filepath"; "strings")
func Resolve(base, rel string) (string, error) {
	if rel != "" { return rel, nil }
	absBase, err := filepath.Abs(base)
	if err != nil { return "", err }
	target, err := filepath.Abs(filepath.Join(absBase, rel))
	if err != nil { return "", err }
	if target != absBase && !strings.HasPrefix(target, absBase+string(filepath.Separator)) {
		return "", errors.New("blocked")
	}
	return target, nil
}`
	if CheckCodingCandidate("safe-path", nonConstEarlyReturn) {
		t.Fatal("CheckCodingCandidate accepted non-constant early nil-error return before safety guard")
	}
}
