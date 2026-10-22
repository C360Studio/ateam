package bench

import (
	"context"
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

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Owned Mercury bridge startup
func TestMercuryBridgeKernelPort(t *testing.T) {
	dir := t.TempDir()
	e := testExperiment()
	e.Claude = &ClaudeConfig{Mercury: &MercuryConfig{Bridge: filepath.Join(dir, "unused-bridge")}}
	cmd, err := mercuryBridgeCommand(context.Background(), e, dir, "http://127.0.0.1:12345/v1", "local-test-token")
	if err != nil {
		t.Fatal(err)
	}
	port := ""
	for i, arg := range cmd.Args {
		if arg == "--port" && i+1 < len(cmd.Args) {
			port = cmd.Args[i+1]
		}
	}
	if port != "0" {
		t.Fatalf("bridge must bind its own kernel port, got --port %s", port)
	}
	if _, err := os.Stat(filepath.Join(dir, "proxy.yaml")); err != nil {
		t.Fatal(err)
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Owned Mercury bridge startup
func TestMercuryReadinessDoesNotProbeForeignListener(t *testing.T) {
	var requests atomic.Int64
	foreign := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		_, _ = io.WriteString(w, `{"data":[{"id":"mercury-3"}]}`)
	}))
	defer foreign.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	done := make(chan struct{})
	_, err := waitMercuryBridge(ctx, make(chan string), "local-test-token", done)
	if err == nil || requests.Load() != 0 {
		t.Fatalf("foreign listener accepted/probed without child startup: err=%v requests=%d", err, requests.Load())
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Owned Mercury bridge startup
func TestMercuryStartupOutput(t *testing.T) {
	const line = "INFO:     Uvicorn running on http://127.0.0.1:54321 (Press CTRL+C to quit)\n"
	for split := 0; split <= len(line); split++ {
		output := mercuryBridgeOutput{address: make(chan string, 1)}
		_, _ = output.Write([]byte(line[:split]))
		_, _ = output.Write([]byte(line[split:]))
		select {
		case got := <-output.address:
			if got != "http://127.0.0.1:54321" {
				t.Fatalf("split%d address=%q", split, got)
			}
		default:
			t.Fatalf("split%d lost startup address", split)
		}
		_, _ = output.Write([]byte(strings.ReplaceAll(line, "54321", "54322")))
		select {
		case <-output.address:
			t.Fatalf("split%d accepted second announcement", split)
		default:
		}
		if output.captured.String() != line+strings.ReplaceAll(line, "54321", "54322") {
			t.Fatal("startup parser changed retained diagnostics")
		}
	}
	t.Logf("streaming reference model: %d two-chunk splits completed", len(line)+1)
	output := mercuryBridgeOutput{address: make(chan string, 1)}
	// Hide Reader.WriteTo so io.Copy exercises any ReaderFrom on the destination.
	if _, err := io.Copy(&output, struct{ io.Reader }{strings.NewReader(line)}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-output.address:
	default:
		t.Fatal("io.Copy bypassed startup parsing")
	}
	for _, bad := range []string{
		strings.ReplaceAll(line, "127.0.0.1", "0.0.0.0"), strings.ReplaceAll(line, "127.0.0.1", "localhost"), strings.ReplaceAll(line, "http:", "https:"),
		strings.ReplaceAll(line, "54321", "0"), strings.ReplaceAll(line, "54321", "65536"), strings.ReplaceAll(line, "54321", "-1"), strings.ReplaceAll(line, "54321", "+1"), strings.ReplaceAll(line, "54321", "01"), strings.ReplaceAll(line, "54321", "1/path"),
		strings.ReplaceAll(line, "Uvicorn running", "Other server running"), strings.TrimSuffix(line, "\n"),
	} {
		output := mercuryBridgeOutput{address: make(chan string, 1)}
		_, _ = output.Write([]byte(bad))
		select {
		case got := <-output.address:
			t.Fatalf("invalid startup accepted: %q -> %q", bad, got)
		default:
		}
	}
	if mercuryStartupAddress(strings.ReplaceAll(line, "54321", "65535")) != "http://127.0.0.1:65535" {
		t.Fatal("maximum valid port refused")
	}
	boundary := mercuryBridgeOutput{address: make(chan string, 1)}
	_, _ = boundary.Write([]byte(strings.Repeat("x", maxAdapterBytes-len(line)-1) + "\n" + line))
	if boundary.captured.Len() != maxAdapterBytes || boundary.captured.overflow {
		t.Fatal("exact diagnostic boundary rejected")
	}
	select {
	case <-boundary.address:
	default:
		t.Fatal("complete startup line at diagnostic boundary lost")
	}
	output = mercuryBridgeOutput{address: make(chan string, 1)}
	_, _ = output.Write([]byte(strings.Repeat("x", maxAdapterBytes+1) + "\n" + line))
	if output.captured.Len() != maxAdapterBytes || !output.captured.overflow {
		t.Fatal("startup diagnostics unbounded")
	}
	select {
	case <-output.address:
		t.Fatal("truncated startup output became address")
	default:
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Owned Mercury bridge startup
func TestMercuryStartupExitAndCancellation(t *testing.T) {
	for _, cause := range []string{"exit", "cancel", "exit-after-announce"} {
		t.Run(cause, func(t *testing.T) {
			var requests atomic.Int64
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				requests.Add(1)
				_, _ = io.WriteString(w, `{"data":[{"id":"mercury-3"}]}`)
			}))
			defer server.Close()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan struct{})
			address := make(chan string, 1)
			if cause == "cancel" {
				cancel()
			} else {
				close(done)
			}
			if cause == "exit-after-announce" {
				address <- server.URL
			}
			if _, err := waitMercuryBridge(ctx, address, "local-test-token", done); err == nil {
				t.Fatal("stopped child accepted")
			}
			if requests.Load() != 0 {
				t.Fatal("stopped child was probed")
			}
		})
	}
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Owned Mercury bridge startup
func TestMercuryBridgePortZeroLifecycle(t *testing.T) {
	e := mercuryFixture(t)
	checkMercuryBridgeStartup(t, e.Claude.Mercury.Bridge)
}

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Owned Mercury bridge startup
func TestMercuryPinnedBridgeStartup(t *testing.T) {
	bridge := os.Getenv("ATEAM_MERCURY_BRIDGE")
	if bridge == "" {
		t.Skip("opt-in pinned LiteLLM startup only; no model requests")
	}
	checkMercuryBridgeStartup(t, bridge)
}

func checkMercuryBridgeStartup(t *testing.T, bridge string) {
	t.Helper()
	var requests atomic.Int64
	// Hold an unrelated listener throughout startup. Readiness must never visit it.
	foreign := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { requests.Add(1); w.WriteHeader(http.StatusTeapot) }))
	defer foreign.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	e := testExperiment()
	e.Claude = &ClaudeConfig{Mercury: &MercuryConfig{Bridge: bridge}}
	cmd, err := mercuryBridgeCommand(ctx, e, t.TempDir(), foreign.URL+"/v1", "sk-local-startup-only")
	if err != nil {
		t.Fatal(err)
	}
	output := mercuryBridgeOutput{address: make(chan string, 1)}
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { _ = cmd.Wait(); close(done) }()
	defer func() {
		cleanupBenchmarkProcess(cmd)
		<-done
		if t.Failed() {
			t.Logf("bridge startup output: %s", output.captured.String())
		}
	}()
	base, err := waitMercuryBridge(ctx, output.address, "sk-local-startup-only", done)
	if err != nil {
		t.Fatalf("bridge startup failed: %v", err)
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.Hostname() != "127.0.0.1" || parsed.Port() == "0" || base == foreign.URL {
		t.Fatalf("invalid actual address %q (%v)", base, err)
	}
	if requests.Load() != 0 {
		t.Fatalf("foreign provider listener got %d startup requests", requests.Load())
	}
	t.Log("kernel-assigned loopback address received; authenticated readiness passed; unrelated listener requests=0")
}
