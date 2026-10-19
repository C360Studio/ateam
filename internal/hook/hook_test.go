package hook

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/c360studio/ateam/internal/state"
)

func TestBeforePromptInjectsLadderOnlyOnModeSwitch(t *testing.T) {
	dir := t.TempDir()

	// F-01: Ordinary prompt without a mode switch must NOT re-inject the ladder.
	normalOut, err := Handle(dir, "before-prompt", strings.NewReader(`{"hook_event_name":"UserPromptSubmit","prompt":"implement handler"}`))
	if err != nil {
		t.Fatalf("Handle normal prompt: %v", err)
	}
	if normalOut.AdditionalContext != "" {
		t.Fatalf("expected empty AdditionalContext on normal prompt, got: %q", normalOut.AdditionalContext)
	}

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

	// Subsequent ordinary prompt in ultra mode must also return empty output.
	nextOut, err := Handle(dir, "before-prompt", strings.NewReader(`{"hook_event_name":"BeforeAgent","prompt":"now add tests"}`))
	if err != nil {
		t.Fatalf("Handle subsequent prompt: %v", err)
	}
	if nextOut.AdditionalContext != "" {
		t.Errorf("expected empty AdditionalContext on subsequent prompt, got: %q", nextOut.AdditionalContext)
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
	_, _ = Handle(dir, "session-start", strings.NewReader(`{}`))

	start := time.Now()
	const iterations = 20
	for i := 0; i < iterations; i++ {
		if _, err := Handle(dir, "before-prompt", strings.NewReader(`{"prompt":"implement handler"}`)); err != nil {
			t.Fatalf("Handle before-prompt iteration %d: %v", i, err)
		}
		if _, err := Handle(dir, "turn-end", strings.NewReader(`{"session_id":"s1"}`)); err != nil {
			t.Fatalf("Handle turn-end iteration %d: %v", i, err)
		}
	}
	avg := time.Since(start) / (iterations * 2)
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

func TestSessionStartSurfacesHoldScanErrorsAndWriterLeaseAndUnknownEvent(t *testing.T) {
	dir := t.TempDir()

	// F-08: Unknown event must be a fast no-op returning empty Output{}.
	unknownOut, err := Handle(dir, "Notification", strings.NewReader(`{"hook_event_name":"Notification"}`))
	if err != nil {
		t.Fatalf("Handle unknown event: %v", err)
	}
	if unknownOut.AdditionalContext != "" || unknownOut.HookSpecific != nil {
		t.Fatalf("expected empty Output for unknown event, got: %+v", unknownOut)
	}

	// F-06: Active writer lease from another session must be surfaced in session-start pickup summary.
	leaseData := []byte(`{"agent":"claude","session_id":"sess-other","pid":999999,"last_seen":"` + time.Now().UTC().Format(time.RFC3339) + `"}`)
	_ = os.MkdirAll(filepath.Join(dir, ".ateam"), 0o755)
	if err := os.WriteFile(state.WriterLeasePath(dir), leaseData, 0o644); err != nil {
		t.Fatal(err)
	}

	// Astra #3: Failed Hold/OpenSpec scan (e.g. stray Hold or invalid UTF-8 tasks.md) must NOT silently report holds=0.
	changeDir := filepath.Join(dir, "openspec", "changes", "broken-hold")
	_ = os.MkdirAll(changeDir, 0o755)
	_ = os.WriteFile(filepath.Join(dir, "openspec", "changes", "broken-hold", "tasks.md"), []byte("## 1. Work\nHold: stray hold outside task\n- [ ] 1.1 Task\n"), 0o644)

	out, err := Handle(dir, "session-start", strings.NewReader(`{"session_id":"sess-current"}`))
	if err != nil {
		t.Fatalf("Handle session-start: %v", err)
	}
	if strings.Contains(out.AdditionalContext, "holds=0") {
		t.Fatalf("session-start reported holds=0 on failed Hold scan:\n%s", out.AdditionalContext)
	}
	if !strings.Contains(out.AdditionalContext, "holds=unavailable") || !strings.Contains(out.AdditionalContext, "WARNING: Hold/OpenSpec scan failed") {
		t.Fatalf("session-start missing Hold scan failure warning:\n%s", out.AdditionalContext)
	}
	if !strings.Contains(out.AdditionalContext, "WARNING: another writer may be active") {
		t.Fatalf("session-start missing active writer lease warning:\n%s", out.AdditionalContext)
	}

	// Astra #5 & G-01: Same session ID must NOT warn even when PID differs, and ClearWriterLease clears the lease.
	if _, err := HandleWithAgent(dir, "turn-end", "codex", strings.NewReader(`{"session_id":"same-session"}`)); err != nil {
		t.Fatalf("HandleWithAgent turn-end: %v", err)
	}
	sameOut, err := HandleWithAgent(dir, "session-start", "codex", strings.NewReader(`{"session_id":"same-session"}`))
	if err != nil {
		t.Fatalf("HandleWithAgent session-start same-session: %v", err)
	}
	if !strings.Contains(sameOut.AdditionalContext, "agent=codex") {
		t.Fatalf("expected agent=codex in writer lease line, got:\n%s", sameOut.AdditionalContext)
	}
	if strings.Contains(sameOut.AdditionalContext, "WARNING: another writer may be active") {
		t.Fatalf("same-session lease triggered false competing-writer warning:\n%s", sameOut.AdditionalContext)
	}

	if err := state.ClearWriterLease(dir); err != nil {
		t.Fatalf("ClearWriterLease: %v", err)
	}
	afterHandoffOut, err := HandleWithAgent(dir, "session-start", "claude", strings.NewReader(`{"session_id":"next-session"}`))
	if err != nil {
		t.Fatalf("HandleWithAgent after ClearWriterLease: %v", err)
	}
	if strings.Contains(afterHandoffOut.AdditionalContext, "WARNING: another writer may be active") {
		t.Fatalf("cleared writer lease still triggered warning:\n%s", afterHandoffOut.AdditionalContext)
	}
}

func TestAntigravityHookContract(t *testing.T) {
	dir := t.TempDir()

	// Strict protojson-style oracle for Antigravity SessionStart/PreInvocation stdout.
	type agyStepOracle struct {
		InjectSteps []struct {
			SystemMessage struct {
				SystemMessage string `json:"systemMessage"`
			} `json:"systemMessage"`
		} `json:"injectSteps"`
	}
	// Strict protojson-style oracle for Antigravity Stop stdout.
	type agyStopOracle struct {
		Decision string `json:"decision"`
		Reason   string `json:"reason,omitempty"`
	}

	// 1. SessionStart with --agent agy emits strictly injectSteps (no unknown additionalContext field).
	startOut, err := HandleWithAgent(dir, "session-start", "agy", strings.NewReader(`{"conversationId":"conv-agy-123","modelName":"auto"}`))
	if err != nil {
		t.Fatalf("HandleWithAgent agy session-start: %v", err)
	}
	if !startOut.HasPayload() {
		t.Fatal("expected agy session-start HasPayload() == true")
	}
	rawStart, err := json.Marshal(startOut)
	if err != nil {
		t.Fatalf("marshal startOut: %v", err)
	}
	decStart := json.NewDecoder(bytes.NewReader(rawStart))
	decStart.DisallowUnknownFields()
	var parsedStart agyStepOracle
	if err := decStart.Decode(&parsedStart); err != nil {
		t.Fatalf("agy session-start output violated strict protojson schema (%s): %v", string(rawStart), err)
	}
	if len(parsedStart.InjectSteps) != 1 || !strings.Contains(parsedStart.InjectSteps[0].SystemMessage.SystemMessage, "PONYTAIL MODE [full]") {
		t.Fatalf("expected agy injectSteps with Ponytail ladder, got: %s", string(rawStart))
	}

	// 2. PreInvocation normalizes to before-prompt, records conversationId in writer lease, and emits {} when no mode switch occurs.
	preOut, err := HandleWithAgent(dir, "", "agy", strings.NewReader(`{"event":"PreInvocation","conversationId":"conv-agy-123","invocationNum":1}`))
	if err != nil {
		t.Fatalf("HandleWithAgent agy PreInvocation: %v", err)
	}
	if !preOut.HasPayload() {
		t.Fatal("expected agy PreInvocation HasPayload() == true for empty JSON object")
	}
	rawPre, err := json.Marshal(preOut)
	if err != nil || string(rawPre) != "{}" {
		t.Fatalf("expected agy PreInvocation stdout {}, got %q (err=%v)", string(rawPre), err)
	}
	wl, err := state.ReadWriterLease(dir)
	if err != nil || wl == nil {
		t.Fatalf("expected writer lease after agy PreInvocation, got wl=%+v err=%v", wl, err)
	}
	if wl.Agent != "agy" || wl.SessionID != "conv-agy-123" {
		t.Fatalf("expected writer lease agent=agy session=conv-agy-123, got %+v", wl)
	}

	// 3. Stop (turn-end) with --agent agy emits {"decision":"allow"} and does not trigger competing-writer warning on same conversationId.
	stopOut, err := HandleWithAgent(dir, "turn-end", "agy", strings.NewReader(`{"conversationId":"conv-agy-123","terminationReason":"NO_TOOL_CALL","fullyIdle":true}`))
	if err != nil {
		t.Fatalf("HandleWithAgent agy turn-end: %v", err)
	}
	rawStop, err := json.Marshal(stopOut)
	if err != nil {
		t.Fatalf("marshal stopOut: %v", err)
	}
	decStop := json.NewDecoder(bytes.NewReader(rawStop))
	decStop.DisallowUnknownFields()
	var parsedStop agyStopOracle
	if err := decStop.Decode(&parsedStop); err != nil || parsedStop.Decision != "allow" {
		t.Fatalf("expected agy Stop output {\"decision\":\"allow\"}, got %s (err=%v)", string(rawStop), err)
	}
}
