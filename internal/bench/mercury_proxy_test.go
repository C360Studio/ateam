package bench

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const mercuryProxyRequest = `{"model":"mercury-3","reasoning_effort":"low","max_tokens":8192,"stream":true,"stream_options":{"include_usage":true},"messages":[{"role":"user","content":"fixture"}]}`

func mercuryProxyPost(t *testing.T, base, body string, change func(*http.Request)) (int, string) {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, base+"/chat/completions", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer bridge-token")
	request.Header.Set("X-Stainless-Retry-Count", "0")
	request.Header.Set("Idempotency-Key", "must-not-forward")
	if change != nil {
		change(request)
	}
	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, string(data)
}

func mercuryProxyStart(ctx context.Context, t *testing.T, providerURL string) (string, string, func() error) {
	t.Helper()
	capture := filepath.Join(t.TempDir(), "capture.jsonl")
	base, stop, err := startMercuryProxy(ctx, providerURL+"/v1", "provider-secret", "bridge-token", capture)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = stop() })
	return base, capture, stop
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Controlled bounded experiments
// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Retained independently graded artifacts
func TestMercuryProxyCaptureAndCredentials(t *testing.T) {
	var calls atomic.Int64
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Method != "POST" || r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer provider-secret" || r.Header.Get("Idempotency-Key") != "" || r.Header.Get("X-Stainless-Retry-Count") != "" || r.Header.Get("Accept-Encoding") != "identity" {
			t.Errorf("unsafe provider request: %s %s headers=%v", r.Method, r.URL.Path, r.Header)
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != mercuryProxyRequest {
			t.Errorf("provider body changed: %s", body)
		}
		_, _ = io.WriteString(w, mercuryTestBody(mercuryTestUsage))
	}))
	t.Cleanup(provider.Close)
	// Environment proxies cannot redirect a capture-owned provider request.
	t.Setenv("HTTP_PROXY", "http://127.0.0.1:1")
	base, capture, stop := mercuryProxyStart(context.Background(), t, provider.URL)
	head, err := http.NewRequest(http.MethodHead, strings.TrimSuffix(base, "/v1")+"/", nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.DefaultClient.Do(head)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != 200 || calls.Load() != 0 {
		t.Fatal("local HEAD probe reached provider or failed")
	}
	for range 2 {
		status, body := mercuryProxyPost(t, base, mercuryProxyRequest, nil)
		if status != 200 || body != mercuryTestBody(mercuryTestUsage) {
			t.Fatalf("provider capture changed response: %d %s", status, body)
		}
	}
	stop()
	stop()
	data, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	usage, count, err := parseMercuryCapture(string(data))
	if err != nil || count != 2 || usage.InputTokens != 40 || usage.OutputTokens != 10 {
		t.Fatalf("captured provider accounting: %+v count=%d err=%v", usage, count, err)
	}
	if calls.Load() != 2 || strings.Contains(string(data), "provider-secret") || strings.Contains(string(data), "bridge-token") {
		t.Fatalf("credential/count leak in journal: %s", data)
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Controlled bounded experiments
func TestMercuryProxyRequestGuards(t *testing.T) {
	for name, change := range map[string]func(*http.Request){
		"authorization": func(r *http.Request) { r.Header.Set("Authorization", "Bearer provider-secret") },
		"retry":         func(r *http.Request) { r.Header.Set("X-Stainless-Retry-Count", "1") },
		"missing-retry": func(r *http.Request) { r.Header.Del("X-Stainless-Retry-Count") },
		"method":        func(r *http.Request) { r.Method = http.MethodGet },
		"path":          func(r *http.Request) { r.URL.Path = "/v1/other" },
		"query":         func(r *http.Request) { r.URL.RawQuery = "destination=elsewhere" },
	} {
		t.Run(name, func(t *testing.T) { mercuryProxyGuardCase(t, mercuryProxyRequest, change) })
	}
	for name, body := range map[string]string{
		"conflicting-output-limit": strings.Replace(mercuryProxyRequest, `"max_tokens":8192`, `"max_tokens":8192,"max_completion_tokens":1000000`, 1),
		"duplicate-output-limit":   strings.Replace(mercuryProxyRequest, `"max_tokens":8192`, `"max_tokens":8192,"max_completion_tokens":8192`, 1),
		"sdk-retry-control":        strings.Replace(mercuryProxyRequest, `"max_tokens":8192`, `"max_tokens":8192,"max_retries":0`, 1),
		"sdk-allowlist":            strings.Replace(mercuryProxyRequest, `"max_tokens":8192`, `"max_tokens":8192,"allowed_openai_params":[]`, 1),
		"model":                    strings.Replace(mercuryProxyRequest, "mercury-3", "other", 1),
		"effort":                   strings.Replace(mercuryProxyRequest, "low", "high", 1),
		"tokens":                   strings.Replace(mercuryProxyRequest, "8192", "8193", 1),
		"nonstream":                strings.Replace(mercuryProxyRequest, `"stream":true`, `"stream":false`, 1),
		"usage":                    strings.Replace(mercuryProxyRequest, `"include_usage":true`, `"include_usage":false`, 1),
		"duplicate":                strings.Replace(mercuryProxyRequest, `"model":`, `"model":"other","model":`, 1),
		"alias":                    strings.Replace(mercuryProxyRequest, `"max_tokens":`, `"MAX_TOKENS":`, 1),
		"fraction":                 strings.Replace(mercuryProxyRequest, "8192", "8192.0", 1),
		"malformed":                "{", "oversize": strings.Repeat("x", maxAdapterBytes+1),
	} {
		t.Run(name, func(t *testing.T) { mercuryProxyGuardCase(t, body, nil) })
	}
}

func mercuryProxyGuardCase(t *testing.T, body string, change func(*http.Request)) {
	t.Helper()
	var calls atomic.Int64
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		_, _ = io.WriteString(w, mercuryTestBody(mercuryTestUsage))
	}))
	t.Cleanup(provider.Close)
	base, capture, stop := mercuryProxyStart(context.Background(), t, provider.URL)
	// Prior success must become unusable when a later request breaks the contract.
	if status, _ := mercuryProxyPost(t, base, mercuryProxyRequest, nil); status != 200 {
		t.Fatal("baseline failed")
	}
	if status, _ := mercuryProxyPost(t, base, body, change); status == 200 {
		t.Fatal("invalid provider request accepted")
	}
	if status, _ := mercuryProxyPost(t, base, mercuryProxyRequest, nil); status == 200 {
		t.Fatal("failure latch ignored")
	}
	stop()
	data, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	if usage, _, parseErr := parseMercuryCapture(string(data)); parseErr == nil || usage != nil {
		t.Fatalf("earlier success accepted after capture failure: %s", data)
	}
	if calls.Load() != 1 {
		t.Fatalf("invalid request escaped proxy: calls=%d", calls.Load())
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Controlled bounded experiments
func TestMercuryProxyResponseGuards(t *testing.T) {
	for _, kind := range []string{"redirect", "http-error", "encoding", "missing-usage", "truncated", "body-limit", "journal-limit"} {
		t.Run(kind, func(t *testing.T) {
			var calls atomic.Int64
			provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				calls.Add(1)
				switch kind {
				case "redirect":
					w.Header().Set("Location", "/elsewhere")
					w.WriteHeader(http.StatusTemporaryRedirect)
				case "http-error":
					w.WriteHeader(http.StatusTooManyRequests)
				case "encoding":
					w.Header().Set("Content-Encoding", "gzip")
				case "missing-usage":
					_, _ = io.WriteString(w, mercuryTestBody("null"))
					return
				case "truncated":
					w.Header().Set("Content-Length", "99999")
				case "body-limit":
					_, _ = io.WriteString(w, strings.Repeat("x", maxAdapterBytes+1))
					return
				case "journal-limit":
					_, _ = io.WriteString(w, ": "+strings.Repeat("\t", maxAdapterBytes/2)+"\n\n"+mercuryTestBody(mercuryTestUsage))
					return
				}
				_, _ = io.WriteString(w, mercuryTestBody(mercuryTestUsage))
			}))
			t.Cleanup(provider.Close)
			base, capture, stop := mercuryProxyStart(context.Background(), t, provider.URL)
			if status, _ := mercuryProxyPost(t, base, mercuryProxyRequest, nil); status == 200 {
				t.Fatal("invalid provider response accepted")
			}
			if status, _ := mercuryProxyPost(t, base, mercuryProxyRequest, nil); status == 200 {
				t.Fatal("response failure latch ignored")
			}
			stop()
			data, err := os.ReadFile(capture)
			if err != nil {
				t.Fatal(err)
			}
			if len(data) > maxAdapterBytes {
				t.Fatalf("journal exceeds bound: %d", len(data))
			}
			if usage, _, parseErr := parseMercuryCapture(string(data)); parseErr == nil || usage != nil {
				t.Fatalf("failed response usage accepted: %s", data)
			}
			if calls.Load() != 1 {
				t.Fatalf("failed/redirected request retried: calls=%d", calls.Load())
			}
		})
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Controlled bounded experiments
func TestMercuryProxyRequestLimit(t *testing.T) {
	var calls atomic.Int64
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		_, _ = io.WriteString(w, mercuryTestBody(mercuryTestUsage))
	}))
	t.Cleanup(provider.Close)
	base, capture, stop := mercuryProxyStart(context.Background(), t, provider.URL)
	for i := range 5 {
		status, _ := mercuryProxyPost(t, base, mercuryProxyRequest, nil)
		if (i < 4) != (status == 200) {
			t.Fatalf("four-exchange cap at %d: status=%d", i, status)
		}
	}
	stop()
	data, err := os.ReadFile(capture)
	if err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 4 {
		t.Fatalf("provider dispatch count=%d", calls.Load())
	}
	if usage, _, parseErr := parseMercuryCapture(string(data)); parseErr == nil || usage != nil {
		t.Fatal("fifth request did not invalidate prior accounting")
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Controlled bounded experiments
func TestMercuryProxyCancellation(t *testing.T) {
	for _, mode := range []string{"owner", "downstream", "stop"} {
		t.Run(mode, func(t *testing.T) {
			received, closed := make(chan struct{}), make(chan struct{})
			abort := make(chan struct{})
			defer close(abort)
			provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, r.Body)
				_, _ = io.WriteString(w, "data: {\"model\":\"mercury-3\"}\n\n")
				w.(http.Flusher).Flush()
				close(received)
				select {
				case <-r.Context().Done():
					close(closed)
				case <-abort:
				}
			}))
			t.Cleanup(provider.Close)
			owner, cancelOwner := context.WithCancel(context.Background())
			defer cancelOwner()
			base, capture, stop := mercuryProxyStart(owner, t, provider.URL)
			downstream, cancelDownstream := context.WithCancel(context.Background())
			defer cancelDownstream()
			request, err := http.NewRequestWithContext(downstream, http.MethodPost, base+"/chat/completions", strings.NewReader(mercuryProxyRequest))
			if err != nil {
				t.Fatal(err)
			}
			request.Header.Set("Authorization", "Bearer bridge-token")
			request.Header.Set("X-Stainless-Retry-Count", "0")
			finished := make(chan struct{})
			go func() {
				defer close(finished)
				response, doErr := http.DefaultClient.Do(request)
				if doErr == nil {
					_ = response.Body.Close()
				}
			}()
			select {
			case <-received:
			case <-time.After(3 * time.Second):
				t.Fatal("provider never received request")
			}
			if mode == "owner" {
				cancelOwner()
			} else if mode == "downstream" {
				cancelDownstream()
			} else {
				stop()
			}
			select {
			case <-closed:
			case <-time.After(3 * time.Second):
				t.Fatal("cancellation did not close provider socket")
			}
			stop()
			select {
			case <-finished:
			case <-time.After(3 * time.Second):
				t.Fatal("client remained blocked after stop")
			}
			data, err := os.ReadFile(capture)
			if err != nil {
				t.Fatal(err)
			}
			if usage, _, parseErr := parseMercuryCapture(string(data)); parseErr == nil || usage != nil {
				t.Fatal("cancelled response accounted as complete")
			}
			// stop waits for the journal writer; retained JSONL is stable and valid JSON.
			for _, line := range strings.Split(strings.TrimSuffix(string(data), "\n"), "\n") {
				if !json.Valid([]byte(line)) {
					t.Fatalf("stop raced journal write: %q", line)
				}
			}
		})
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Controlled bounded experiments
func TestMercuryProxyStorageFailure(t *testing.T) {
	endpoint, err := url.Parse("http://127.0.0.1:1/v1/chat/completions")
	if err != nil {
		t.Fatal(err)
	}
	t.Run("preserved-success", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "capture.jsonl")
		original := mercuryTestJournal(t, mercuryTestBody(mercuryTestUsage))
		if err := os.WriteFile(path, []byte(original), 0600); err != nil {
			t.Fatal(err)
		}
		journal, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		base, stop, err := serveMercuryProxy(context.Background(), endpoint, "provider-secret", "bridge-token", journal)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = stop() })
		if status, _ := mercuryProxyPost(t, base, mercuryProxyRequest, nil); status == 200 {
			t.Fatal("read-only capture accepted")
		}
		first, second := stop(), stop()
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != original {
			t.Fatal("read-only fixture must preserve earlier success journal")
		}
		if usage, _, err := parseMercuryCapture(string(data)); err != nil || usage == nil {
			t.Fatal("fixture must retain independently valid-looking accounting")
		}
		if first == nil || second == nil || first.Error() != second.Error() {
			t.Fatalf("stop lost capture failure despite preserved valid journal: first=%v second=%v", first, second)
		}
	})
	t.Run("close-error", func(t *testing.T) {
		journal, err := os.Create(filepath.Join(t.TempDir(), "capture.jsonl"))
		if err != nil {
			t.Fatal(err)
		}
		_, stop, err := serveMercuryProxy(context.Background(), endpoint, "provider-secret", "bridge-token", journal)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = stop() })
		if err := journal.Close(); err != nil {
			t.Fatal(err)
		}
		first, second := stop(), stop()
		if first == nil || second == nil || first.Error() != second.Error() {
			t.Fatalf("stop lost journal close error: first=%v second=%v", first, second)
		}
	})
}
