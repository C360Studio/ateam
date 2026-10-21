package bench

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Controlled bounded experiments
// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Retained independently graded artifacts
func TestMercuryPinnedOffline(t *testing.T) {
	if os.Getenv("ATEAM_MERCURY_NATIVE_OFFLINE") != "1" {
		t.Skip("opt-in pinned native/bridge loopback test")
	}
	cli, bridge := os.Getenv("ATEAM_MERCURY_CLAUDE"), os.Getenv("ATEAM_MERCURY_BRIDGE")
	if cli == "" || bridge == "" {
		t.Fatal("explicit pinned executables required")
	}
	var mutex sync.Mutex
	requests := 0
	source := ""
	for _, task := range Tasks() {
		if task.ID == "safe-path" {
			source = task.GoodSource
		}
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mutex.Lock()
		defer mutex.Unlock()
		if r.Method != http.MethodPost || r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer offline-provider-fixture" {
			t.Errorf("unexpected provider route or credential: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		requests++
		var request map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			w.WriteHeader(400)
			return
		}
		var tools []struct {
			Function struct {
				Name string `json:"name"`
			} `json:"function"`
		}
		_ = json.Unmarshal(request["tools"], &tools)
		names := []string{}
		for _, tool := range tools {
			names = append(names, tool.Function.Name)
		}
		t.Logf("provider request%d path=%s model=%s effort=%s max_tokens=%s tool_names=%v response_format=%s", requests, r.URL.Path, request["model"], request["reasoning_effort"], request["max_tokens"], names, request["response_format"])
		w.Header().Set("Content-Type", "text/event-stream")
		args, _ := json.Marshal(map[string]string{"source": source})
		chunk := func(choices any, usage any) {
			value := map[string]any{"id": "offline-mercury", "object": "chat.completion.chunk", "created": 1, "model": "mercury-3", "choices": choices}
			if usage != nil {
				value["usage"] = usage
			}
			b, _ := json.Marshal(value)
			_, _ = fmt.Fprintf(w, "data: %s\n\n", b)
		}
		chunk([]any{map[string]any{"index": 0, "delta": map[string]any{"role": "assistant", "tool_calls": []any{map[string]any{"index": 0, "id": "structured-output-1", "type": "function", "function": map[string]any{"name": "StructuredOutput", "arguments": string(args)}}}}, "finish_reason": nil}}, nil)
		chunk([]any{map[string]any{"index": 0, "delta": map[string]any{}, "finish_reason": "tool_calls"}}, nil)
		chunk([]any{}, map[string]any{"prompt_tokens": 20, "completion_tokens": 5, "total_tokens": 25, "prompt_tokens_details": map[string]int{"cached_tokens": 3}, "completion_tokens_details": map[string]int{"reasoning_tokens": 2}})
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer server.Close()
	e := testExperiment()
	e.Command = []string{cli}
	e.Agent = "claude"
	e.Model = "mercury-3"
	e.Claude = &ClaudeConfig{Auth: "api-key-env", Reasoning: "low", OutputFormat: "json-schema", Mercury: &MercuryConfig{Bridge: bridge}}
	e.TimeoutSeconds = 30
	t.Setenv("INCEPTION_API_KEY", "offline-provider-fixture")
	dir := t.TempDir()
	response, stdout, stderr, status, err := executeMercuryAt(context.Background(), e, newAdapterRequest(e, Trial{Task: "safe-path", Repetition: 1, Arm: "baseline"}), dir, server.URL+"/v1")
	output := os.Getenv("ATEAM_MERCURY_EVIDENCE")
	if output != "" {
		if err := os.MkdirAll(output, 0700); err != nil {
			t.Fatal(err)
		}
		_ = os.WriteFile(filepath.Join(output, "stdout.json"), []byte(stdout), 0600)
		_ = os.WriteFile(filepath.Join(output, "stderr.txt"), []byte(stderr), 0600)
	}
	if err != nil || status != "completed" {
		t.Fatalf("native Mercury status=%s error=%v stderr=%s", status, err, stderr)
	}
	if response.Source != source || response.Usage == nil || response.Usage.InputTokens != 20 || response.Usage.OutputTokens != 5 {
		t.Fatalf("native schema/accounting mismatch: %+v", response)
	}
	mutex.Lock()
	count := requests
	mutex.Unlock()
	if count != 1 {
		t.Fatalf("expected one provider exchange, got%d", count)
	}
	parsed, err := parseNativeOutput(e, stdout)
	if err != nil || parsed.Source != source {
		t.Fatalf("native envelope replay: %v", err)
	}
}
