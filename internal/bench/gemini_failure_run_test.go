package bench

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

const geminiFatalSession = "stderr-only-session"
const geminiFatalMessage = "Maximum number of turns exceeded. Credential: [REDACTED]"

func geminiFatalRun(t *testing.T) (string, ExperimentRun, string, string) {
	t.Helper()
	e := nativeTestExperiment(t, `null`)
	t.Setenv("GEMINI_API_KEY", geminiTestKey)
	root := t.TempDir()
	calls, graderCalls := filepath.Join(root, "agent-calls"), filepath.Join(root, "grader-calls")
	quote := func(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'" }
	envelope, err := json.MarshalIndent(map[string]any{
		"session_id": geminiFatalSession,
		"error":      map[string]any{"type": "FatalTurnLimitedError", "message": "Maximum number of turns exceeded. Credential: " + geminiTestKey, "code": 53},
	}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	// Version probes do not count as model invocations. The two identical native
	// envelopes reproduce the CLI's bare and [ERROR]-prefixed stderr emissions.
	script := "#!/bin/sh\nif [ \"$1\" = --version ]; then echo '0.43.0'; exit 0; fi\n" +
		"cat >/dev/null\nprintf 'called\\n' >> " + quote(calls) + "\n" +
		"printf '%s\\n' " + quote("Warning: optional terminal integration unavailable") + " " + quote(string(envelope)) + " " + quote("[ERROR] "+string(envelope)) + " >&2\nexit 53\n"
	if err := os.WriteFile(e.Command[0], []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	// A separate executable side effect proves grading never ran, even if a
	// regression were to clear Grade/GradeNanos after invoking the grader.
	goScript := "#!/bin/sh\nif [ \"$1\" = version ]; then echo 'go version go1.test fixture/fixture'; exit 0; fi\nprintf 'graded\\n' >> " + quote(graderCalls) + "\nexit 99\n"
	if err := os.WriteFile(filepath.Join(root, "go"), []byte(goScript), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", root+string(os.PathListSeparator)+os.Getenv("PATH"))
	resolved, err := ResolveExperiment(e, ".")
	if err != nil {
		t.Fatal(err)
	}
	path, err := RunExperiment(context.Background(), resolved, t.TempDir())
	if err == nil {
		t.Fatal("fatal stderr exit did not stop execution")
	}
	run, err := LoadExperimentRun(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(run.Resolved.Schedule) != 2 || len(run.Attempts) != 1 || run.StopReason != "adapter-error" {
		t.Fatalf("fatal stderr history mismatch: scheduled=%d retained=%d stop=%s", len(run.Resolved.Schedule), len(run.Attempts), run.StopReason)
	}
	assertGeminiFatalNoExtraWork(t, calls, graderCalls)
	return path, run, calls, graderCalls
}

func assertGeminiFatalNoExtraWork(t *testing.T, calls, graderCalls string) {
	t.Helper()
	data, err := os.ReadFile(calls)
	if err != nil || string(data) != "called\n" {
		t.Fatalf("fatal stderr launched more than one attempt: %q err=%v", data, err)
	}
	if _, err := os.Stat(graderCalls); !os.IsNotExist(err) {
		t.Fatalf("fatal stderr invoked grader: %v", err)
	}
}

func replayGeminiFatalRun(t *testing.T, path string, run ExperimentRun) ExperimentRun {
	t.Helper()
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	comparison, err := CompareExperiment(run)
	if err != nil {
		t.Fatal(err)
	}
	if comparison.CompletedAttempts != 1 || comparison.UnpairedAttempts != 1 || comparison.Pairs != 0 || comparison.UsagePairs != 0 || comparison.TotalTokenDelta != nil {
		t.Fatalf("failed unpaired attempt gained comparable quality or usage: %+v", comparison)
	}
	gradeCalls := 0
	path2, err := rescoreExperiment(context.Background(), path, func(context.Context, *Attempt) { gradeCalls++ })
	if err != nil {
		t.Fatal(err)
	}
	if gradeCalls != 0 {
		t.Fatal("offline replay graded a failed execution")
	}
	replayed, err := LoadExperimentRun(path2)
	if err != nil {
		t.Fatal(err)
	}
	afterComparison, err := CompareExperiment(replayed)
	if err != nil || !reflect.DeepEqual(comparison, afterComparison) || !reflect.DeepEqual(run.Attempts, replayed.Attempts) || run.StopReason != replayed.StopReason {
		t.Fatalf("offline replay changed failure history or comparison: %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("offline replay altered original evidence bytes")
	}
	return replayed
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Gemini fatal stderr evidence
func TestGeminiFatalStderrRunAndReplay(t *testing.T) {
	path, run, calls, graderCalls := geminiFatalRun(t)
	a := run.Attempts[0]
	if a.Status != "adapter-error" || a.Stdout != "" || a.SourceSHA != "" || a.Response == nil || a.Response.Source != "" || a.Response.Usage != nil || a.GradeNanos != 0 || !reflect.DeepEqual(a.Grade, CandidateEvaluation{}) {
		t.Fatalf("fatal stderr fabricated a candidate, grade or usage: %+v", a)
	}
	n := a.Response.Native
	if n == nil || n.UsageStatus != "missing" || n.Terminal != "incomplete" || n.SessionID != "" || n.ReportedUsage != nil || n.ReportedModels != nil || n.ResolvedModel != nil || n.ResolvedReasoning != nil {
		t.Fatalf("stderr changed stdout-derived accounting or identity: %+v", n)
	}
	want := &GeminiFailureEvidence{SessionID: geminiFatalSession, Type: "FatalTurnLimitedError", Message: geminiFatalMessage, ExitCode: 53}
	if !reflect.DeepEqual(n.GeminiFailure, want) {
		t.Fatalf("fatal stderr evidence missing or wrong: got=%+v want=%+v", n.GeminiFailure, want)
	}
	if !strings.Contains(a.Diagnostic, "four-turn limit") || strings.Contains(strings.ToLower(a.Diagnostic), "quota") {
		t.Fatalf("fatal stderr diagnostic obscured configured limit or invented quota: %q", a.Diagnostic)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	encodedKey, _ := json.Marshal(geminiTestKey)
	if bytes.Contains(raw, []byte(geminiTestKey)) || bytes.Contains(raw, encodedKey[1:len(encodedKey)-1]) || strings.Contains(a.Stderr, geminiTestKey) || !strings.Contains(a.Stderr, "[REDACTED]") {
		t.Fatal("fatal stderr retained an API credential")
	}
	replayGeminiFatalRun(t, path, run)
	assertGeminiFatalNoExtraWork(t, calls, graderCalls)
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Gemini fatal stderr evidence
func TestGeminiFatalStderrLegacyReplay(t *testing.T) {
	_, run, calls, graderCalls := geminiFatalRun(t)
	// Model the historical artifact exactly: identical raw streams but the old
	// stdout-only response and generic process diagnostic, with no new field.
	run.Attempts[0].Response.Native.GeminiFailure = nil
	run.Attempts[0].Diagnostic = "gemini execution: exit status 53"
	run.Digest = runDigest(run)
	path := filepath.Join(t.TempDir(), "legacy-run.json")
	if err := writeExclusiveJSON(path, run); err != nil {
		t.Fatal(err)
	}
	legacy, err := LoadExperimentRun(path)
	if err != nil {
		t.Fatal(err)
	}
	replayed := replayGeminiFatalRun(t, path, legacy)
	a := replayed.Attempts[0]
	if a.Response.Native.GeminiFailure != nil || a.Response.Usage != nil || a.Response.Native.UsageStatus != "missing" || a.Diagnostic != "gemini execution: exit status 53" {
		t.Fatal("legacy replay retroactively enriched unknown failure evidence")
	}
	raw, err := os.ReadFile(path)
	if err != nil || bytes.Contains(raw, []byte(`"gemini_failure"`)) {
		t.Fatal("optional evidence changed legacy serialization")
	}
	assertGeminiFatalNoExtraWork(t, calls, graderCalls)
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Gemini fatal stderr evidence
func TestGeminiFatalStderrRejectsTampering(t *testing.T) {
	_, original, calls, graderCalls := geminiFatalRun(t)
	if original.Attempts[0].Response.Native.GeminiFailure == nil {
		t.Fatal("fatal stderr evidence missing before tampering boundary")
	}
	for _, tc := range []struct {
		name   string
		change func(*Attempt)
	}{
		{"session", func(a *Attempt) { a.Response.Native.GeminiFailure.SessionID = "changed-session" }},
		{"type", func(a *Attempt) { a.Response.Native.GeminiFailure.Type = "QuotaError" }},
		{"message", func(a *Attempt) { a.Response.Native.GeminiFailure.Message = "invented message" }},
		{"exit-code", func(a *Attempt) { a.Response.Native.GeminiFailure.ExitCode = 1 }},
		{"grade", func(a *Attempt) { a.Grade.Supported = true }},
		{"grading-time", func(a *Attempt) { a.GradeNanos = 1 }},
		{"source-digest", func(a *Attempt) { a.SourceSHA = "invented" }},
		{"stdout", func(a *Attempt) { a.Stdout = " " }},
		{"truncated", func(a *Attempt) { a.Response.Native.OutputTruncated = true }},
		{"usage", func(a *Attempt) { a.Response.Usage = &Usage{InputTokens: 1} }},
		{"diagnostic", func(a *Attempt) { a.Diagnostic = "gemini execution: invented quota" }},
		{"raw-stderr", func(a *Attempt) {
			a.Stderr = strings.Replace(a.Stderr, "Maximum number of turns exceeded.", "Conflicting native message.", 1)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(original)
			if err != nil {
				t.Fatal(err)
			}
			var changed ExperimentRun
			if err := json.Unmarshal(data, &changed); err != nil {
				t.Fatal(err)
			}
			tc.change(&changed.Attempts[0])
			changed.Digest = runDigest(changed)
			path := filepath.Join(t.TempDir(), "tampered-run.json")
			if err := writeExclusiveJSON(path, changed); err != nil {
				t.Fatal(err)
			}
			_, err = LoadExperimentRun(path)
			if err == nil {
				t.Fatal("tampered fatal stderr evidence accepted after rehash")
			}
			switch tc.name {
			case "grade", "grading-time", "source-digest", "stdout", "truncated", "diagnostic":
				if !strings.Contains(err.Error(), "invalid Gemini fatal outcome") {
					t.Fatalf("did not reach fatal outcome guard: %v", err)
				}
			}
		})
	}
	assertGeminiFatalNoExtraWork(t, calls, graderCalls)
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Gemini fatal stderr evidence
func TestGeminiFatalStderrRefusesOtherAdapter(t *testing.T) {
	cfg := testExperiment()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Command = []string{executable}
	resolved, err := ResolveExperiment(cfg, ".")
	if err != nil {
		t.Fatal(err)
	}
	run := ExperimentRun{Version: experimentVersion, Created: time.Now().UTC(), Resolved: resolved, StopReason: "cancelled"}
	run.Attempts = []Attempt{{Request: newAdapterRequest(resolved.Experiment, resolved.Schedule[0]), Status: "cancelled", Diagnostic: "cancelled", Response: &AdapterResponse{Version: experimentVersion, Native: &NativeEvidence{GeminiFailure: &GeminiFailureEvidence{ExitCode: 53}}}}}
	run.Digest = runDigest(run)
	if err := validateRun(run); err == nil || !strings.Contains(err.Error(), "Gemini failure evidence on another adapter") {
		t.Fatalf("did not reach adapter-specific failure guard: %v", err)
	}
}
