package bench

import (
	"bufio"
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

const codexTestKey = "unit-test-openai-key\"with\\escape"

func fakeCodexExperiment(t *testing.T) Experiment {
	t.Helper()
	return fakeCodexExperimentScenario(t, "complete")
}

func fakeCodexExperimentScenario(t *testing.T, scenario string) Experiment {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake executable uses POSIX launcher; parser tests remain portable")
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	launcher := filepath.Join(t.TempDir(), "fake-codex")
	quote := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }
	script := "#!/bin/sh\nexec " + quote(exe) + " -test.run=^TestCodexCLIHelper$ -- " + quote(scenario) + " \"$@\"\n"
	if err := os.WriteFile(launcher, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	e := testExperiment()
	e.Command = []string{launcher}
	e.Agent = "codex"
	e.Model = "gpt-6-luna"
	e.Codex = &CodexConfig{Auth: "api-key-env", Reasoning: "low"}
	t.Setenv("OPENAI_API_KEY", codexTestKey)
	for _, name := range []string{"CODEX_API_KEY", "ANTHROPIC_API_KEY", "CODEX_HOME", "OPENAI_BASE_URL", "OPENAI_ORGANIZATION", "OPENAI_PROJECT", "HTTP_PROXY", "HTTPS_PROXY", "NODE_OPTIONS", "ATEAM_SECRET_PROBE"} {
		t.Setenv(name, "must-not-inherit")
	}
	return e
}

func TestCodexExecutionIsolationAndRedaction(t *testing.T) {
	e := fakeCodexExperiment(t)
	previous := ""
	for _, arm := range []string{"baseline", "candidate"} {
		req := newAdapterRequest(e, Trial{Task: "safe-path", Repetition: 1, Arm: arm})
		r, out, diagnostic, status, err := executeCodex(context.Background(), e, req, t.TempDir())
		if err != nil || status != "completed" {
			t.Fatalf("status=%s err=%v stderr=%s stdout=%s", status, err, diagnostic, out)
		}
		if r.Source != Tasks()[0].GoodSource+"\n// [REDACTED]\n" || r.Usage == nil || r.Usage.InputTokens != 10 || r.Usage.OutputTokens != 5 {
			t.Fatalf("source or native accounting lost: %+v", r)
		}
		encoded, _ := json.Marshal(codexTestKey)
		if strings.Contains(out+diagnostic, codexTestKey) || strings.Contains(out+diagnostic, string(encoded[1:len(encoded)-1])) || !strings.Contains(diagnostic, "[REDACTED]") {
			t.Fatal("API credential escaped redaction")
		}
		if r.Native.SessionID == "" || r.Native.SessionID == previous {
			t.Fatal("attempts reused native session")
		}
		previous = r.Native.SessionID
	}
	if os.Getenv("CODEX_HOME") != "must-not-inherit" {
		t.Fatal("child configuration changed parent environment")
	}
}

func TestCodexExecutionRejectsBeforeInference(t *testing.T) {
	for _, scenario := range []string{"mismatchedmodel", "mismatchedeffort", "mismatchedpolicy", "approvalrequest", "unknownpreflight", "mismatchedthread"} {
		t.Run(scenario, func(t *testing.T) {
			e := fakeCodexExperimentScenario(t, scenario)
			dir := t.TempDir()
			_, _, _, status, err := executeCodex(context.Background(), e, newAdapterRequest(e, Schedule(e)[0]), dir)
			if err == nil || status != "adapter-error" {
				t.Fatalf("unsafe native setup accepted: %s %v", status, err)
			}
			if _, err := os.Stat(filepath.Join(dir, "paid-turn")); !os.IsNotExist(err) {
				t.Fatal("inference began before native policy validation")
			}
		})
	}
	e := fakeCodexExperiment(t)
	t.Setenv("OPENAI_API_KEY", "")
	_, out, _, status, err := executeCodex(context.Background(), e, AdapterRequest{}, t.TempDir())
	if err == nil || status != "adapter-error" || out != "" || !strings.Contains(err.Error(), "OPENAI_API_KEY") {
		t.Fatal("missing explicit API credential did not stop launch")
	}
}

func TestCodexExecutionFailureEvidence(t *testing.T) {
	for _, scenario := range []string{"failed", "unknownusage", "timeout", "overflow", "stderr-cap", "toolrequest", "toolitem", "malformed", "secretmethod"} {
		t.Run(scenario, func(t *testing.T) {
			e := fakeCodexExperimentScenario(t, scenario)
			e.TimeoutSeconds = 5
			if scenario == "timeout" {
				e.TimeoutSeconds = 1
			}
			r, out, _, status, err := executeCodex(context.Background(), e, newAdapterRequest(e, Schedule(e)[0]), t.TempDir())
			if scenario == "unknownusage" {
				if err != nil || status != "completed" || r.Usage != nil {
					t.Fatal("missing usage must remain unknown")
				}
				return
			}
			want := "adapter-error"
			if scenario == "timeout" {
				want = "timeout"
			}
			if err != nil && strings.Contains(err.Error(), codexTestKey) {
				t.Fatal("protocol diagnostic exposed API credential")
			}
			if err == nil || status != want || r.Native == nil || len(out) > maxAdapterBytes {
				t.Fatalf("status=%s err=%v response=%+v", status, err, r)
			}
			if scenario == "overflow" || scenario == "stderr-cap" {
				if !r.Native.OutputTruncated || r.Usage != nil {
					t.Fatal("truncation falsely retained complete accounting")
				}
			}
		})
	}
}

func TestCodexExecutionCancellation(t *testing.T) {
	e := fakeCodexExperimentScenario(t, "timeout")
	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	result := make(chan string, 1)
	go func() {
		_, _, _, status, _ := executeCodex(ctx, e, newAdapterRequest(e, Schedule(e)[0]), dir)
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
			t.Fatal("fake Codex did not signal readiness")
		case <-tick.C:
			if _, err := os.Stat(filepath.Join(dir, "ready")); err == nil {
				cancel()
				select {
				case status := <-result:
					if status != "cancelled" {
						t.Fatalf("status=%s", status)
					}
				case <-deadline.C:
					t.Fatal("cancelled child did not exit")
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
	scenario, args := os.Args[separator+1], os.Args[separator+2:]
	if slices.Equal(args, []string{"--version"}) {
		fmt.Println("codex-cli 0.160.1")
		os.Exit(0)
	}
	if err := validateFakeCodexInvocation(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(9)
	}
	cwd, _ := os.Getwd()
	thread := "thread-" + filepath.Base(cwd)
	if scenario == "reusedsession" {
		thread = "reused-thread"
	}
	scanner := bufio.NewScanner(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)
	send := func(v any) {
		if err := encoder.Encode(v); err != nil {
			os.Exit(8)
		}
	}
	read := func(want string) map[string]json.RawMessage {
		if !scanner.Scan() {
			os.Exit(7)
		}
		var message map[string]json.RawMessage
		if json.Unmarshal(scanner.Bytes(), &message) != nil {
			os.Exit(6)
		}
		var method string
		_ = json.Unmarshal(message["method"], &method)
		if method != want {
			os.Exit(5)
		}
		var params map[string]json.RawMessage
		_ = json.Unmarshal(message["params"], &params)
		return params
	}
	init := read("initialize")
	if !strings.Contains(string(init["capabilities"]), `"experimentalApi":true`) {
		os.Exit(4)
	}
	send(map[string]any{"id": 1, "result": map[string]string{"userAgent": "fake-codex/0.160.1"}})
	read("initialized")
	start := read("thread/start")
	for field, want := range map[string]string{"model": `"gpt-6-luna"`, "modelProvider": `"benchmark_openai"`, "ephemeral": "true", "approvalPolicy": `"never"`, "sandbox": `"read-only"`, "allowProviderModelFallback": "false", "dynamicTools": "[]", "environments": "[]"} {
		if string(start[field]) != want {
			fmt.Fprintln(os.Stderr, "invalid thread/start field", field)
			os.Exit(3)
		}
	}
	model, effort, policy := "gpt-6-luna", "low", "never"
	if scenario == "mismatchedmodel" {
		model = "gpt-6-sol"
	}
	if scenario == "mismatchedeffort" {
		effort = "high"
	}
	if scenario == "mismatchedpolicy" {
		policy = "on-request"
	}
	if scenario == "approvalrequest" {
		send(map[string]any{"id": 42, "method": "item/commandExecution/requestApproval", "params": map[string]any{}})
	}
	if scenario == "unknownpreflight" {
		send(map[string]any{"method": "unknown/control", "params": map[string]any{}})
	}
	if scenario == "mismatchedthread" {
		send(map[string]any{"method": "thread/started", "params": map[string]any{"thread": map[string]string{"id": "different-thread"}}})
	}
	send(map[string]any{"id": 2, "result": map[string]any{"thread": map[string]any{"id": thread, "ephemeral": true}, "model": model, "modelProvider": "benchmark_openai", "reasoningEffort": effort, "approvalPolicy": policy, "sandbox": map[string]any{"type": "readOnly", "networkAccess": false}, "instructionSources": []any{}}})
	turn := read("turn/start")
	_ = os.WriteFile(filepath.Join(cwd, "paid-turn"), nil, 0600)
	if _, ok := turn["model"]; ok {
		os.Exit(2)
	}
	if _, ok := turn["effort"]; ok {
		os.Exit(2)
	}
	var inputs []struct {
		Text string `json:"text"`
	}
	_ = json.Unmarshal(turn["input"], &inputs)
	if len(inputs) != 1 || !strings.Contains(inputs[0].Text, "func Resolve") || strings.Contains(inputs[0].Text, `"arm"`) {
		os.Exit(2)
	}
	var schema map[string]any
	_ = json.Unmarshal(turn["outputSchema"], &schema)
	expected := `{"additionalProperties":false,"properties":{"source":{"type":"string"}},"required":["source"],"type":"object"}`
	schemaBytes, _ := json.Marshal(schema)
	if string(schemaBytes) != expected {
		fmt.Fprintln(os.Stderr, "native source schema missing")
		os.Exit(2)
	}
	fakeCodexTurn(send, scenario, cwd, thread)
}

func fakeCodexTurn(send func(any), scenario, cwd, thread string) {
	notify := func(method string, params map[string]any) { send(map[string]any{"method": method, "params": params}) }
	notify("thread/started", map[string]any{"thread": map[string]string{"id": thread}})
	send(map[string]any{"id": 3, "result": map[string]any{"turn": map[string]any{"id": "turn-1", "status": "inProgress", "items": []any{}}}})
	notify("turn/started", map[string]any{"threadId": thread, "turn": map[string]string{"id": "turn-1", "status": "inProgress"}})
	if scenario == "timeout" {
		_ = os.WriteFile(filepath.Join(cwd, "ready"), nil, 0600)
		time.Sleep(8 * time.Second)
		os.Exit(1)
	}
	if scenario == "overflow" {
		fmt.Fprint(os.Stdout, strings.Repeat("x", maxAdapterBytes+10))
		os.Exit(0)
	}
	if scenario == "malformed" {
		fmt.Fprintln(os.Stdout, "{")
		os.Exit(0)
	}
	if scenario == "secretmethod" {
		notify("unknown/"+os.Getenv("OPENAI_API_KEY"), map[string]any{"threadId": thread, "turnId": "turn-1"})
		_, _ = io.Copy(io.Discard, os.Stdin)
		os.Exit(0)
	}
	if scenario == "toolrequest" {
		send(map[string]any{"id": 42, "method": "item/commandExecution/requestApproval", "params": map[string]any{}})
		_, _ = io.Copy(io.Discard, os.Stdin)
		os.Exit(0)
	}
	if scenario == "toolitem" {
		notify("item/started", map[string]any{"threadId": thread, "turnId": "turn-1", "item": map[string]string{"id": "tool-1", "type": "commandExecution"}})
		_, _ = io.Copy(io.Discard, os.Stdin)
		os.Exit(0)
	}
	fmt.Fprintln(os.Stderr, os.Getenv("OPENAI_API_KEY"))
	source := Tasks()[0].GoodSource + "\n// " + os.Getenv("OPENAI_API_KEY") + "\n"
	answer, _ := json.Marshal(map[string]string{"source": source})
	item := map[string]string{"id": "answer-1", "type": "agentMessage", "phase": "final_answer", "text": string(answer)}
	notify("item/started", map[string]any{"threadId": thread, "turnId": "turn-1", "item": item})
	notify("item/completed", map[string]any{"threadId": thread, "turnId": "turn-1", "item": item})
	if scenario != "unknownusage" {
		tokens := map[string]int{"inputTokens": 10, "cachedInputTokens": 3, "cacheWriteInputTokens": 0, "outputTokens": 5, "reasoningOutputTokens": 2, "totalTokens": 15}
		notify("thread/tokenUsage/updated", map[string]any{"threadId": thread, "turnId": "turn-1", "tokenUsage": map[string]any{"total": tokens, "last": tokens}})
	}
	status := "completed"
	if scenario == "failed" {
		status = "failed"
	}
	if scenario == "stderr-cap" {
		fmt.Fprint(os.Stderr, strings.Repeat(" ", maxAdapterBytes))
	}
	notify("turn/completed", map[string]any{"threadId": thread, "turn": map[string]string{"id": "turn-1", "status": status}})
	_, _ = io.Copy(io.Discard, os.Stdin)
	os.Exit(0)
}

func validateFakeCodexInvocation(args []string) error {
	if len(args) < 3 || !slices.Equal(args[:3], []string{"app-server", "--stdio", "--strict-config"}) {
		return fmt.Errorf("unexpected Codex transport")
	}
	settings := map[string]string{}
	for i := 3; i < len(args); i += 2 {
		if args[i] != "-c" || i+1 >= len(args) {
			return fmt.Errorf("unexpected Codex argument")
		}
		key, value, ok := strings.Cut(args[i+1], "=")
		if !ok {
			return fmt.Errorf("malformed setting")
		}
		settings[key] = value
	}
	for key, want := range map[string]string{"model": `"gpt-6-luna"`, "model_reasoning_effort": `"low"`, "model_provider": `"benchmark_openai"`, "forced_login_method": `"api"`, "cli_auth_credentials_store": `"ephemeral"`, "model_providers.benchmark_openai.env_key": `"OPENAI_API_KEY"`, "model_providers.benchmark_openai.base_url": `"https://api.openai.com/v1"`, "model_providers.benchmark_openai.requires_openai_auth": "false", "model_providers.benchmark_openai.request_max_retries": "0", "model_providers.benchmark_openai.stream_max_retries": "0", "approval_policy": `"never"`, "sandbox_mode": `"read-only"`, "shell_environment_policy.inherit": `"none"`, "project_doc_max_bytes": "0", "web_search": `"disabled"`} {
		if settings[key] != want {
			return fmt.Errorf("execution policy missing %s", key)
		}
	}
	for _, name := range []string{"hooks", "plugins", "remote_plugin", "apps", "multi_agent", "shell_tool", "shell_snapshot", "browser_use", "computer_use", "image_generation", "view_image", "skill_search", "skill_mcp_dependency_install", "goals", "unbounded_connection_retries", "code_mode_host"} {
		if settings["features."+name] != "false" {
			return fmt.Errorf("ambient capability enabled: %s", name)
		}
	}
	for _, name := range []string{"imagegen", "openai-docs", "skill-creator", "skill-installer"} {
		if !strings.Contains(settings["skills.config"], name+`/SKILL.md",enabled=false`) {
			return fmt.Errorf("bundled skill enabled")
		}
	}
	allowed := map[string]bool{}
	for _, name := range []string{"PATH", "HOME", "CODEX_HOME", "OPENAI_API_KEY", "TMPDIR", "XDG_CONFIG_HOME", "XDG_CACHE_HOME", "GOWORK", "GOENV", "PWD", "SHLVL", "_"} {
		allowed[name] = true
	}
	for _, assignment := range os.Environ() {
		name, _, _ := strings.Cut(assignment, "=")
		if !allowed[name] {
			return fmt.Errorf("ambient environment inherited: %s", name)
		}
	}
	if os.Getenv("OPENAI_API_KEY") != codexTestKey {
		return fmt.Errorf("API credential not isolated")
	}
	cwd, _ := os.Getwd()
	cwd, _ = filepath.EvalSymlinks(cwd)
	home, _ := filepath.EvalSymlinks(os.Getenv("HOME"))
	config, _ := filepath.EvalSymlinks(os.Getenv("CODEX_HOME"))
	if home != cwd || config != filepath.Join(home, ".codex") {
		return fmt.Errorf("home or Codex config not isolated")
	}
	if _, err := os.Stat(filepath.Join(cwd, ".ateam-benchmark-root")); err != nil {
		return fmt.Errorf("instruction boundary missing")
	}
	marker := filepath.Join(config, "attempt-marker")
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		return fmt.Errorf("prior attempt state leaked")
	}
	return os.WriteFile(marker, nil, 0600)
}

func TestCodexManagedPaths(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing")
	if err := checkCodexManagedPaths([]string{missing}); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"file", "directory", "symlink"} {
		path := filepath.Join(dir, kind)
		var err error
		switch kind {
		case "file":
			err = os.WriteFile(path, nil, 0600)
		case "directory":
			err = os.Mkdir(path, 0700)
		case "symlink":
			err = os.Symlink(missing, path)
		}
		if err != nil {
			t.Fatal(err)
		}
		if err := checkCodexManagedPaths([]string{missing, path}); err == nil || !strings.Contains(err.Error(), "policy review") {
			t.Fatal("managed policy was not refused")
		}
	}
	if err := checkCodexManagedPaths([]string{filepath.Join(dir, "file", "child")}); err == nil {
		t.Fatal("filesystem inspection error was ignored")
	}
}

func TestCodexMissingConfiguration(t *testing.T) {
	e := testExperiment()
	if _, err := codexSettings(e); err == nil {
		t.Fatal("missing native configuration accepted")
	}
	_, _, _, status, err := executeCodex(context.Background(), e, AdapterRequest{}, t.TempDir())
	if err == nil || status != "adapter-error" {
		t.Fatal("invalid native authentication config accepted")
	}
}
