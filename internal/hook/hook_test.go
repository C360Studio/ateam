package hook

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/c360studio/ateam/internal/state"
)

func TestBeforePromptInjectsLadderAndSwitchesMode(t *testing.T) {
	dir := t.TempDir()

	// Switch to ultra via Gemini BeforeAgent payload.
	payload := `{"hook_event_name":"BeforeAgent","prompt":"Please run /ponytail ultra on this module"}`
	out, err := Handle(dir, "", strings.NewReader(payload))
	if err != nil {
		t.Fatalf("Handle BeforeAgent: %v", err)
	}
	if !strings.Contains(out.AdditionalContext, "PONYTAIL MODE [ultra]") {
		t.Errorf("expected ultra ladder in AdditionalContext, got: %q", out.AdditionalContext)
	}

	st, err := state.Load(dir)
	if err != nil {
		t.Fatalf("state.Load: %v", err)
	}
	if st.Mode != state.ModeUltra {
		t.Errorf("persisted mode = %q, want %q", st.Mode, state.ModeUltra)
	}

	// Switch off and confirm 0 tokens injected.
	offPayload := `{"hook_event_name":"UserPromptSubmit","prompt":"/mode off"}`
	outOff, err := Handle(dir, "before-prompt", strings.NewReader(offPayload))
	if err != nil {
		t.Fatalf("Handle mode off: %v", err)
	}
	if outOff.AdditionalContext != "" {
		t.Errorf("expected empty AdditionalContext when off, got: %q", outOff.AdditionalContext)
	}
}

func TestPreCompactSavesAutoHandoff(t *testing.T) {
	dir := t.TempDir()
	out, err := Handle(dir, "pre-compact", strings.NewReader(`{"hook_event_name":"PreCompress"}`))
	if err != nil {
		t.Fatalf("Handle pre-compact: %v", err)
	}
	if !strings.Contains(out.AdditionalContext, "Goal: Auto-saved before context compaction") {
		t.Errorf("unexpected pre-compact output: %q", out.AdditionalContext)
	}
	st, err := state.Load(dir)
	if err != nil {
		t.Fatalf("state.Load: %v", err)
	}
	if st.LastHandoff == nil || !st.LastHandoff.AutoCompacted {
		t.Errorf("expected AutoCompacted handoff in state, got: %+v", st.LastHandoff)
	}
}

func TestHookLatencyBudget(t *testing.T) {
	dir := t.TempDir()
	// Warm up filesystem cache in temp dir.
	_, _ = Handle(dir, "session-start", strings.NewReader(`{}`))

	start := time.Now()
	const iterations = 20
	for i := 0; i < iterations; i++ {
		_, err := Handle(dir, "before-prompt", strings.NewReader(`{"prompt":"implement handler"}`))
		if err != nil {
			t.Fatalf("Handle iteration %d: %v", i, err)
		}
	}
	avg := time.Since(start) / iterations
	if avg > 5*time.Millisecond {
		t.Errorf("average hook latency %v exceeds 5ms budget", avg)
	}
}

func TestSessionStartDetectsHeadDrift(t *testing.T) {
	dir := t.TempDir()
	headsDir := filepath.Join(dir, ".git", "refs", "heads")
	if err := os.MkdirAll(headsDir, 0o755); err != nil {
		t.Fatalf("mkdir .git/refs/heads: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".git", "HEAD"), []byte("ref: refs/heads/main\n"), 0o644); err != nil {
		t.Fatalf("write .git/HEAD: %v", err)
	}
	if err := os.WriteFile(filepath.Join(headsDir, "main"), []byte("cd8ea792914382474208bd59658c26d00bed9860\n"), 0o644); err != nil {
		t.Fatalf("write refs/heads/main: %v", err)
	}

	st := &state.State{
		Mode: state.ModeFull,
		Git:  state.GitStatus{Branch: "main", HeadSHA: "148420d"},
		LastHandoff: &state.Handoff{
			Goal:          "Complete dual-forge webhook receiver",
			Addresses:     "openspec/changes/dual-forge-webhooks",
			Constraints:   "Hold on task 2.1",
			DoneWhen:      "task verify green",
			ChecksToRun:   "task verify",
			Branch:        "main",
			CheckpointSHA: "148420d",
		},
	}
	if err := state.Save(dir, st); err != nil {
		t.Fatalf("state.Save: %v", err)
	}

	out, err := Handle(dir, "session-start", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("Handle session-start: %v", err)
	}
	wantDrift := "Last Handoff (HEAD drifted from 148420d -> cd8ea79):"
	if !strings.Contains(out.AdditionalContext, wantDrift) {
		t.Errorf("expected %q in SessionStart AdditionalContext, got:\n%s", wantDrift, out.AdditionalContext)
	}
}
