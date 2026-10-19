package bench

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
)

const codexTestKey = "unit-test-key\"with\\escape"

func fakeCodexExperiment(t *testing.T) Experiment {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake executable uses a POSIX launcher; native parser tests remain portable")
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	launcher := filepath.Join(t.TempDir(), "fake-codex")
	quoted := "'" + strings.ReplaceAll(exe, "'", "'\"'\"'") + "'"
	if err := os.WriteFile(launcher, []byte("#!/bin/sh\nexec "+quoted+" -test.run=^TestCodexCLIHelper$ -- \"$@\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	e := testExperiment()
	e.Command = []string{launcher}
	e.Model = "explicit-test-model"
	e.Codex = &CodexConfig{Reasoning: "low", Auth: "api-key-env"}
	t.Setenv("CODEX_API_KEY", codexTestKey)
	t.Setenv("ATEAM_SECRET_PROBE", "must-not-inherit")
	t.Setenv("OPENAI_API_KEY", "must-not-inherit")
	return e
}

// Explicit API auth reaches the CLI, but ambient config/secrets and prior arm files must not.
func TestCodexExecutionIsolationAndRedaction(t *testing.T) {
	e := fakeCodexExperiment(t)
	previousSession := ""
	for _, arm := range []string{"baseline", "candidate"} {
		req := newAdapterRequest(e, Trial{Task: "safe-path", Repetition: 1, Arm: arm})
		req.Instructions = "same-visible-instructions"
		r, stdout, stderr, status, err := executeCodex(context.Background(), e, req, t.TempDir())
		if err != nil || status != "completed" {
			t.Fatalf("status=%s error=%v stderr=%s stdout=%s", status, err, stderr, stdout)
		}
		if r.Source != "package safepath // key=[REDACTED]" || r.Usage == nil || r.Usage.InputTokens != 100 || r.Usage.OutputTokens != 25 {
			t.Fatal("decoded source must redact a doubly JSON-escaped API credential and retain native usage")
		}
		encoded, _ := json.Marshal(codexTestKey)
		if strings.Contains(stdout+stderr, codexTestKey) || strings.Contains(stdout+stderr, string(encoded[1:len(encoded)-1])) || !strings.Contains(stdout, "[REDACTED]") || !strings.Contains(stderr, "[REDACTED]") {
			t.Fatal("persisted output did not redact exact API credential")
		}
		if r.Native.SessionID == "" || r.Native.SessionID == previousSession {
			t.Fatal("independent attempts reused a session")
		}
		previousSession = r.Native.SessionID
	}
}

func TestCodexExecutionFailureEvidence(t *testing.T) {
	e := fakeCodexExperiment(t)
	for _, tc := range []struct{ instruction, status string }{{"failure", "adapter-error"}, {"timeout", "timeout"}, {"overflow", "adapter-error"}, {"stdout-cap", "adapter-error"}, {"stderr-cap", "adapter-error"}, {"malformed", "adapter-error"}} {
		t.Run(tc.instruction, func(t *testing.T) {
			e.TimeoutSeconds = 5
			if tc.instruction == "timeout" {
				e.TimeoutSeconds = 1
			}
			req := newAdapterRequest(e, Schedule(e)[0])
			req.Instructions = tc.instruction
			r, stdout, _, status, err := executeCodex(context.Background(), e, req, t.TempDir())
			if err == nil || status != tc.status || r.Native == nil || len(stdout) > maxAdapterBytes {
				t.Fatalf("status=%s response=%+v error=%v", status, r, err)
			}
			if tc.instruction == "failure" && (r.Usage == nil || r.Usage.InputTokens != 41 || r.Native.Terminal != "failed") {
				t.Fatal("failed process lost reported native usage")
			}
			if strings.HasSuffix(tc.instruction, "-cap") && (r.Usage != nil || !r.Native.OutputTruncated || r.Native.ReportedUsage == nil || r.Native.UsageStatus != "truncated") {
				t.Fatal("truncated trace must not supply accountable usage from a completed prefix")
			}
		})
	}
	t.Setenv("CODEX_API_KEY", "")
	_, stdout, _, status, err := executeCodex(context.Background(), e, AdapterRequest{}, t.TempDir())
	if err == nil || status != "adapter-error" || stdout != "" || !strings.Contains(err.Error(), "CODEX_API_KEY") {
		t.Fatalf("missing explicit auth did not fail before process launch: status=%s error=%v", status, err)
	}
}

func TestCodexExecutionCancellation(t *testing.T) {
	e := fakeCodexExperiment(t)
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := newAdapterRequest(e, Schedule(e)[0])
	req.Instructions = "timeout"
	result := make(chan string, 1)
	go func() {
		_, _, _, status, _ := executeCodex(ctx, e, req, dir)
		result <- status
	}()
	deadline := time.NewTimer(4 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(5 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case status := <-result:
			t.Fatalf("process ended before cancellation synchronization: %s", status)
		case <-deadline.C:
			t.Fatal("fake CLI did not signal readiness")
		case <-tick.C:
			if _, err := os.Stat(filepath.Join(dir, "ready")); err == nil {
				cancel()
				select {
				case status := <-result:
					if status != "cancelled" {
						t.Fatalf("cancelled process status=%s", status)
					}
				case <-deadline.C:
					t.Fatal("cancelled process did not exit")
				}
				return
			}
		}
	}
}

func TestCodexCLIHelper(_ *testing.T) {
	separator := slices.Index(os.Args, "--")
	if separator < 0 {
		return
	}
	args := os.Args[separator+1:]
	if err := validateFakeCodexInvocation(args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(9)
	}
	prompt, _ := io.ReadAll(os.Stdin)
	if strings.Contains(string(prompt), `"arm"`) || strings.Contains(string(prompt), "baseline") || strings.Contains(string(prompt), "candidate") || !strings.Contains(string(prompt), "func Resolve") {
		os.Exit(10)
	}
	cwd, _ := os.Getwd()
	encoder := json.NewEncoder(os.Stdout)
	_ = encoder.Encode(map[string]string{"type": "thread.started", "thread_id": filepath.Base(cwd)})
	_ = encoder.Encode(map[string]string{"type": "turn.started"})
	if strings.Contains(string(prompt), "timeout") {
		_ = os.WriteFile(filepath.Join(cwd, "ready"), nil, 0600)
		time.Sleep(8 * time.Second)
		os.Exit(11)
	}
	if strings.Contains(string(prompt), "overflow") {
		_, _ = os.Stdout.WriteString(strings.Repeat("x", maxAdapterBytes+1))
		os.Exit(0)
	}
	if strings.Contains(string(prompt), "malformed") {
		_, _ = os.Stdout.WriteString("{")
		os.Exit(0)
	}
	if strings.Contains(string(prompt), "failure") {
		_, _ = os.Stdout.WriteString("{\"type\":\"turn.failed\",\"usage\":{\"input_tokens\":41,\"output_tokens\":7}}\n")
		os.Exit(7)
	}
	_ = encoder.Encode(map[string]any{"type": "item.completed", "item": map[string]string{"id": "reason", "type": "reasoning", "text": os.Getenv("CODEX_API_KEY")}})
	_, _ = os.Stderr.WriteString(os.Getenv("CODEX_API_KEY"))
	answer, _ := json.Marshal(map[string]string{"source": "package safepath // key=" + os.Getenv("CODEX_API_KEY")})
	_ = encoder.Encode(map[string]any{"type": "item.completed", "item": map[string]string{"id": "answer", "type": "agent_message", "text": string(answer)}})
	_, _ = os.Stdout.WriteString("{\"type\":\"turn.completed\",\"usage\":{\"input_tokens\":100,\"output_tokens\":25}}\n")
	if strings.Contains(string(prompt), "stdout-cap") {
		_, _ = os.Stdout.WriteString(strings.Repeat(" ", maxAdapterBytes))
	}
	if strings.Contains(string(prompt), "stderr-cap") {
		_, _ = os.Stderr.WriteString(strings.Repeat(" ", maxAdapterBytes))
	}
	os.Exit(0)
}

func validateFakeCodexInvocation(args []string) error {
	for _, name := range []string{"ATEAM_SECRET_PROBE", "OPENAI_API_KEY"} {
		if os.Getenv(name) != "" {
			return fmt.Errorf("ambient secret inherited")
		}
	}
	if os.Getenv("CODEX_API_KEY") != codexTestKey {
		return fmt.Errorf("explicit credential missing")
	}
	cwd, _ := os.Getwd()
	cwd, _ = filepath.EvalSymlinks(cwd)
	home, _ := filepath.EvalSymlinks(os.Getenv("HOME"))
	codexHome, _ := filepath.EvalSymlinks(os.Getenv("CODEX_HOME"))
	if cwd != home || codexHome != filepath.Join(home, ".codex") {
		return fmt.Errorf("home or CLI config was not isolated")
	}
	marker := filepath.Join(codexHome, "attempt-marker")
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		return fmt.Errorf("previous attempt state leaked")
	}
	if err := os.WriteFile(marker, nil, 0600); err != nil {
		return err
	}
	for _, required := range []string{"exec", "--json", "--ephemeral", "--ignore-user-config", "--ignore-rules", "--skip-git-repo-check", "read-only", "explicit-test-model", `model_reasoning_effort="low"`, `model_provider="openai"`, `approval_policy="never"`, `shell_environment_policy.inherit="none"`, "project_doc_max_bytes=0"} {
		if !slices.Contains(args, required) {
			return fmt.Errorf("missing required argument %s", required)
		}
	}
	i := slices.Index(args, "--output-schema")
	if i < 0 || i+1 >= len(args) {
		return fmt.Errorf("missing response schema")
	}
	schema, err := os.ReadFile(args[i+1])
	if err != nil || string(schema) != codexSourceSchema {
		return fmt.Errorf("invalid schema")
	}
	return nil
}
