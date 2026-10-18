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
