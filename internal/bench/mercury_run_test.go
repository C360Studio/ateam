package bench

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func mercuryFixture(t *testing.T) Experiment {
	t.Helper()
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("Mercury process groups require macOS/Linux")
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for _, role := range []string{"claude", "bridge"} {
		script := "#!/bin/sh\nexec '" + strings.ReplaceAll(exe, "'", "'\\''") + "' -test.run=^TestMercuryProcessHelper$ -- " + role + " \"$@\"\n"
		if err := os.WriteFile(filepath.Join(dir, role), []byte(script), 0700); err != nil {
			t.Fatal(err)
		}
	}
	e := testExperiment()
	e.Agent = "claude"
	e.AgentVersion = ""
	e.Model = "mercury-3"
	e.Mode = "aa"
	e.Candidate = e.Baseline
	e.Command = []string{filepath.Join(dir, "claude")}
	e.Claude = &ClaudeConfig{Auth: "api-key-env", Reasoning: "low", OutputFormat: "json-schema", Mercury: &MercuryConfig{Bridge: filepath.Join(dir, "bridge")}}
	e.Tasks = []string{"safe-path"}
	e.Repetitions = 1
	e.MaxTotalTokens = 1000
	e.MaxRunSeconds = 60
	e.TimeoutSeconds = 5
	t.Setenv("INCEPTION_API_KEY", "fixture-inception-secret")
	t.Setenv("ANTHROPIC_API_KEY", "must-not-inherit")
	t.Setenv("HTTP_PROXY", "must-not-inherit")
	return e
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Controlled bounded experiments
func TestMercuryProcessHelper(_ *testing.T) {
	index := -1
	for i, arg := range os.Args {
		if arg == "--" {
			index = i
			break
		}
	}
	if index < 0 {
		return
	}
	role := os.Args[index+1]
	args := os.Args[index+2:]
	if len(args) == 1 && args[0] == "--version" {
		if role == "claude" {
			fmt.Println("2.1.292 (Claude Code)")
		} else {
			fmt.Println(mercuryBridgeVersion)
		}
		os.Exit(0)
	}
	fail := func(message string) { fmt.Fprintln(os.Stderr, message); os.Exit(2) }
	if os.Getenv("HTTP_PROXY") != "" {
		fail("inherited proxy")
	}
	if role == "claude" {
		if os.Getenv("INCEPTION_API_KEY") != "" || os.Getenv("ANTHROPIC_API_KEY") == "must-not-inherit" {
			fail("provider credential reached native CLI")
		}
		if !strings.Contains(strings.Join(args, " "), "--json-schema") {
			fail("schema missing")
		}
		prompt, _ := io.ReadAll(os.Stdin)
		if strings.Contains(string(prompt), "fixture-stderr-cap") {
			fmt.Fprint(os.Stderr, strings.Repeat("x", maxAdapterBytes+1))
			fmt.Fprintln(os.Stdout, `{"type":"result","subtype":"success","is_error":false,"session_id":"stderr-cap","structured_output":{"source":"package safepath"}}`)
			os.Exit(0)
		}
		req, _ := http.NewRequest(http.MethodPost, os.Getenv("ANTHROPIC_BASE_URL")+"/v1/messages", strings.NewReader(string(prompt)))
		req.Header.Set("Authorization", "Bearer "+os.Getenv("ANTHROPIC_API_KEY"))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fail("local bridge request failed")
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			fail("local bridge returned error")
		}
		_, _ = io.Copy(os.Stdout, resp.Body)
		os.Exit(0)
	}
	if os.Getenv("INCEPTION_API_KEY") != "" || os.Getenv("ANTHROPIC_API_KEY") != "" || os.Getenv("MERCURY_CAPTURE_PATH") != "" || os.Getenv("MERCURY_API_BASE") != "" {
		fail("bridge credential isolation")
	}
	var configPath, port string
	for i := 0; i+1 < len(args); i++ {
		if args[i] == "--config" {
			configPath = args[i+1]
		}
		if args[i] == "--port" {
			port = args[i+1]
		}
	}
	var config struct {
		Models []struct {
			Params struct {
				Base    string   `json:"api_base"`
				Key     string   `json:"api_key"`
				Retries *int     `json:"max_retries"`
				Allowed []string `json:"allowed_openai_params"`
			} `json:"litellm_params"`
		} `json:"model_list"`
		General struct {
			Key string `json:"master_key"`
		} `json:"general_settings"`
	}
	b, err := os.ReadFile(configPath)
	if err != nil || json.Unmarshal(b, &config) != nil {
		fail("bad bridge config")
	}
	if len(config.Models) != 1 || !strings.HasPrefix(config.Models[0].Params.Base, "http://127.0.0.1:") || config.Models[0].Params.Key == "" || config.Models[0].Params.Key == "fixture-inception-secret" || config.Models[0].Params.Retries == nil || *config.Models[0].Params.Retries != 0 || len(config.Models[0].Params.Allowed) != 1 || config.Models[0].Params.Allowed[0] != "max_retries" {
		fail("bad bridge provider isolation or retry controls")
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/models", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+config.General.Key {
			w.WriteHeader(401)
			return
		}
		_, _ = io.WriteString(w, `{"data":[{"id":"mercury-3"}]}`)
	})
	mux.HandleFunc("/v1/messages", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+config.General.Key {
			w.WriteHeader(401)
			return
		}
		prompt, _ := io.ReadAll(r.Body)
		body, err := json.Marshal(map[string]any{
			"model": "mercury-3", "reasoning_effort": "low", "max_tokens": 8192,
			"stream": true, "stream_options": map[string]bool{"include_usage": true},
			"messages": []map[string]string{{"role": "user", "content": string(prompt)}},
		})
		if err != nil {
			fail("provider request encoding")
		}
		upstream, err := http.NewRequestWithContext(r.Context(), http.MethodPost, config.Models[0].Params.Base+"/chat/completions", strings.NewReader(string(body)))
		if err != nil {
			fail("provider request construction")
		}
		upstream.Header.Set("Authorization", "Bearer "+config.Models[0].Params.Key)
		upstream.Header.Set("Content-Type", "application/json")
		upstream.Header.Set("X-Stainless-Retry-Count", "0")
		response, err := http.DefaultClient.Do(upstream)
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		_, readErr := io.Copy(io.Discard, response.Body)
		_ = response.Body.Close()
		if response.StatusCode != http.StatusOK || readErr != nil {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		source := ""
		for _, task := range Tasks() {
			if task.ID == "safe-path" {
				source = task.GoodSource
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "result", "subtype": "success", "is_error": false, "session_id": fmt.Sprint(os.Getpid()), "structured_output": map[string]string{"source": source}, "modelUsage": map[string]any{"invented": map[string]int{"inputTokens": 99999}}})
	})
	listener, err := net.Listen("tcp4", "127.0.0.1:"+port)
	if err != nil {
		fail("bridge listener failed")
	}
	fmt.Fprintf(os.Stderr, "INFO:     Uvicorn running on http://%s (Press CTRL+C to quit)\n", listener.Addr())
	if err := http.Serve(listener, mux); err != nil {
		fail("bridge listener failed")
	}
	os.Exit(0)
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Bounded native pilots
// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Retained independently graded artifacts
func TestMercuryRunAndReplay(t *testing.T) {
	for _, scenario := range []string{"complete", "missing", "oversize"} {
		t.Run(scenario, func(t *testing.T) {
			missing := scenario != "complete"
			e := mercuryFixture(t)
			if missing {
				e.Baseline = "fixture-" + scenario
				e.Candidate = e.Baseline
			}
			resolved, err := ResolveExperiment(e, ".")
			if err != nil {
				t.Fatal(err)
			}
			if resolved.MercuryBridgeVersion != mercuryBridgeVersion || nativeKeyName(e) != "INCEPTION_API_KEY" {
				t.Fatal("Mercury provenance missing")
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer fixture-inception-secret" {
					t.Error("provider endpoint or credential missing")
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				requestBody, _ := io.ReadAll(r.Body)
				if missing && !strings.Contains(string(requestBody), "fixture-"+scenario) {
					t.Error("native prompt not forwarded")
				}
				w.Header().Set("Content-Type", "text/event-stream")
				body := mercuryTestBody(mercuryTestUsage)
				if scenario == "missing" {
					body = "data: {\"model\":\"mercury-3\",\"choices\":[]}\n\ndata: [DONE]\n\n"
				}
				if scenario == "oversize" {
					body = strings.Repeat("x", maxAdapterBytes+1)
				}
				_, _ = io.WriteString(w, body)
			}))
			defer server.Close()
			execute := func(ctx context.Context, experiment Experiment, request AdapterRequest, dir string) (AdapterResponse, string, string, string, error) {
				return executeMercuryAt(ctx, experiment, request, dir, server.URL+"/v1")
			}
			path, runErr := runExperiment(context.Background(), resolved, t.TempDir(), execute)
			if (runErr != nil) != missing {
				t.Fatalf("run=%v", runErr)
			}
			run, err := LoadExperimentRun(path)
			if err != nil {
				t.Fatal(err)
			}
			if missing {
				if run.StopReason != "adapter-error" || len(run.Attempts) != 1 || run.Attempts[0].Response.Usage != nil {
					t.Fatal("missing provider usage did not stop before second call")
				}
			} else {
				if len(run.Attempts) != 2 || run.Attempts[0].Status != "passed" || run.Attempts[0].Response.Usage.InputTokens != 20 || run.Attempts[0].Response.Usage.OutputTokens != 5 {
					t.Fatal("Mercury source/accounting mismatch")
				}
				run.Attempts[0].Response.Usage.InputTokens++
				run.Digest = runDigest(run)
				if validateRun(run) == nil {
					t.Fatal("tampered Mercury totals accepted")
				}
				original, _ := os.ReadFile(path)
				if _, err := RescoreExperiment(context.Background(), path); err != nil {
					t.Fatal(err)
				}
				after, _ := os.ReadFile(path)
				if string(original) != string(after) {
					t.Fatal("replay mutated raw evidence")
				}
			}
		})
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Controlled bounded experiments
func TestMercuryCancellationClosesProviderSocket(t *testing.T) {
	for _, timeout := range []bool{false, true} {
		t.Run(fmt.Sprint(timeout), func(t *testing.T) {
			e := mercuryFixture(t)
			if timeout {
				e.TimeoutSeconds = 5
			}
			connected, closed, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, r.Body)
				_ = r.Body.Close()
				w.WriteHeader(200)
				w.(http.Flusher).Flush()
				close(connected)
				select {
				case <-r.Context().Done():
					close(closed)
				case <-release:
				}

			}))
			defer server.Close()
			defer close(release)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			type outcome struct {
				status string
				err    error
			}
			done := make(chan outcome, 1)
			go func() {
				_, _, _, status, err := executeMercuryAt(ctx, e, AdapterRequest{Instructions: "fixture-wait"}, t.TempDir(), server.URL+"/v1")
				done <- outcome{status, err}
			}()
			select {
			case <-connected:
			case <-time.After(4 * time.Second):
				t.Fatal("provider handshake absent")
			}
			if !timeout {
				cancel()
			}
			select {
			case got := <-done:
				want := "cancelled"
				if timeout {
					want = "timeout"
				}
				if got.err == nil || got.status != want {
					t.Fatalf("got %s/%v want %s", got.status, got.err, want)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("attempt did not stop")
			}
			select {
			case <-closed:
			case <-time.After(3 * time.Second):
				t.Fatal("bridge provider socket survived cleanup")
			}
		})
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Controlled bounded experiments
func TestMercuryDryRunAndValidation(t *testing.T) {
	e := mercuryFixture(t)
	t.Setenv("INCEPTION_API_KEY", "")
	r, err := ResolveExperiment(e, ".")
	if err != nil {
		t.Fatal(err)
	}
	if r.ClaudeVersion != "2.1.292 (Claude Code)" || r.MercuryBridgeVersion != mercuryBridgeVersion {
		t.Fatal("dry-run provenance")
	}
	for _, change := range []func(*Experiment){func(e *Experiment) { e.Model = "claude-haiku-5-5" }, func(e *Experiment) { e.Claude.Reasoning = "high" }, func(e *Experiment) { e.Claude.Mercury.Bridge = "" }} {
		bad := e
		c := *e.Claude
		m := *c.Mercury
		c.Mercury = &m
		bad.Claude = &c
		change(&bad)
		if validateExperiment(bad) == nil {
			t.Fatal("unsupported Mercury config accepted")
		}
	}
	if err := os.WriteFile(e.Claude.Mercury.Bridge, []byte("#!/bin/sh\nexit 1\n"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := RunExperiment(context.Background(), r, t.TempDir()); err == nil {
		t.Fatal("changed bridge accepted")
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Retained independently graded artifacts
// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Bounded native pilots
func TestMercuryStderrTruncationSurvivesReplay(t *testing.T) {
	e := mercuryFixture(t)
	e.Baseline, e.Candidate = "fixture-stderr-cap", "fixture-stderr-cap"
	resolved, err := ResolveExperiment(e, ".")
	if err != nil {
		t.Fatal(err)
	}
	path, runErr := RunExperiment(context.Background(), resolved, t.TempDir())
	if runErr == nil {
		t.Fatal("truncated attempt succeeded")
	}
	run, err := LoadExperimentRun(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(run.Attempts) != 1 || run.StopReason != "adapter-error" {
		t.Fatalf("truncation history: %+v", run)
	}
	a := &run.Attempts[0]
	if a.Response == nil || a.Response.Native == nil || !a.Response.Native.OutputTruncated || a.Response.Native.UsageStatus != "truncated" || a.Response.Usage != nil {
		t.Fatalf("truncation flag/accounting lost: %+v", a.Response)
	}
	var envelope map[string]any
	if err := json.Unmarshal([]byte(a.Stdout), &envelope); err != nil {
		t.Fatal(err)
	}
	delete(envelope, "output_truncated")
	data, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	a.Stdout = string(data)
	run.Digest = runDigest(run)
	if validateRun(run) == nil {
		t.Fatal("removed truncation envelope flag accepted")
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Retained independently graded artifacts
func TestMercuryCaptureReadFailures(t *testing.T) {
	for _, oversized := range []bool{false, true} {
		t.Run(fmt.Sprint(oversized), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "capture.jsonl")
			if oversized {
				if err := os.WriteFile(path, []byte(strings.Repeat("x", maxAdapterBytes+1)), 0600); err != nil {
					t.Fatal(err)
				}
			}
			native := `{"type":"result","subtype":"success","is_error":false,"session_id":"capture-read","structured_output":{"source":"package safepath"}}`
			response, raw, _, status, err := finishMercuryOutput(native, "", path, "fixture-inception-secret", "completed", nil, false)
			if err == nil || status != "adapter-error" || response.Usage != nil {
				t.Fatalf("capture read failure accepted: %s %v %+v", status, err, response)
			}
			var envelope mercuryOutput
			if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
				t.Fatal(err)
			}
			if envelope.Capture != "" || envelope.Error == "" {
				t.Fatal("capture read failure not retained")
			}
			if replay, replayErr := parseMercuryOutput(raw); replayErr == nil || replay.Usage != nil {
				t.Fatal("capture failure became valid on replay")
			}
		})
	}
}
