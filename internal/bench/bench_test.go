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
	if rep.AvgLOCReductionPct < 50 {
		t.Errorf("AvgLOCReductionPct = %d%%, want >= 50%%", rep.AvgLOCReductionPct)
	}
	md := FormatMarkdown(rep)
	if !strings.Contains(md, "relay-openspec-hold") || !strings.Contains(md, "gitea-webhook") {
		t.Errorf("FormatMarkdown missing expected task rows:\n%s", md)
	}
}

func TestCommentSpoofingRejected(t *testing.T) {
	// Probe 8: An always-accepting path resolver with expected keywords in a comment must fail evaluation.
	spoofed := `package safepath
import "path/filepath"
// filepath.Separator HasPrefix !strings.HasPrefix
func Resolve(base, rel string) (string, error) {
	return filepath.Join(base, rel), nil
}`
	if checkSafePathAST(spoofed) {
		t.Fatal("checkSafePathAST accepted comment-spoofed always-accepting path resolver")
	}
}
