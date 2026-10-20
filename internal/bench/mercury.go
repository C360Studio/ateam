package bench

import (
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// MercuryConfig selects Inception through the existing Claude runtime.
type MercuryConfig struct {
	Bridge string `json:"bridge"`
}

const mercuryBridgeVersion = "LiteLLM: Current Version = 1.99.0"
const mercuryAPIBase = "https://api.inceptionlabs.ai/v1"
const mercuryBudgetPolicy = "INCEPTION_API_KEY only in the per-attempt LiteLLM 1.99.0 bridge; Claude receives a local bridge token; model mercury-3, low effort, 8192 output tokens/request, max4 provider requests, no retries/fallback; raw provider usage required for every request; stop on missing/incomplete usage; process groups killed on exit/cancel; max_run_seconds includes grading; max_total_tokens checked between attempts; no hard dollar cap; CLI cost ignored"

//go:embed mercury_hook.py
var mercuryHook string

func isMercury(e Experiment) bool { return e.Claude != nil && e.Claude.Mercury != nil }

func validateMercury(e Experiment) error {
	if e.Model != "mercury-3" || e.Claude.Reasoning != "low" || e.Claude.OutputFormat != "json-schema" || strings.TrimSpace(e.Claude.Mercury.Bridge) == "" {
		return fmt.Errorf("Mercury requires model mercury-3, low reasoning, json-schema and an explicit LiteLLM bridge executable")
	}
	return nil
}

func mercuryEnvironment(home string) []string {
	return []string{"PATH=" + os.Getenv("PATH"), "HOME=" + home, "TMPDIR=" + home, "XDG_CACHE_HOME=" + home,
		"PYTHON_DOTENV_DISABLED=1", "LITELLM_LOCAL_MODEL_COST_MAP=True", "DO_NOT_TRACK=1", "LITELLM_LOG=ERROR"}
}

func resolveMercury(r *ResolvedExperiment, e *Experiment, base string) error {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		return fmt.Errorf("Mercury bridge execution requires macOS or Linux process groups")
	}
	path := e.Claude.Mercury.Bridge
	var err error
	if !strings.ContainsRune(path, filepath.Separator) {
		path, err = exec.LookPath(path)
	} else if !filepath.IsAbs(path) {
		path = filepath.Join(base, path)
	}
	if err != nil {
		return fmt.Errorf("resolve Mercury bridge: %w", err)
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read Mercury bridge: %w", err)
	}
	home, err := os.MkdirTemp("", "ateam-bridge-version-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(home)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "--version")
	configureBenchmarkProcess(cmd)
	defer cleanupBenchmarkProcess(cmd)
	cmd.Env = mercuryEnvironment(home)
	cmd.Dir = home
	var out limitedBuffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("probe Mercury bridge version: %w", err)
	}
	if out.overflow || strings.TrimSpace(out.String()) != mercuryBridgeVersion {
		return fmt.Errorf("Mercury requires %s", mercuryBridgeVersion)
	}
	claude := *e.Claude
	mercury := *claude.Mercury
	mercury.Bridge = path
	claude.Mercury = &mercury
	e.Claude = &claude
	r.CommandFiles[path] = digest(data)
	r.MercuryBridgeVersion = mercuryBridgeVersion
	r.MercuryHookSHA = digest([]byte(mercuryHook))
	return nil
}

func validateMercuryProvenance(r ResolvedExperiment) error {
	if !isMercury(r.Experiment) || !filepath.IsAbs(r.Experiment.Claude.Mercury.Bridge) || r.CommandFiles[r.Experiment.Claude.Mercury.Bridge] == "" || r.MercuryBridgeVersion != mercuryBridgeVersion || r.MercuryHookSHA != digest([]byte(mercuryHook)) {
		return fmt.Errorf("missing or changed Mercury bridge/hook provenance")
	}
	return nil
}

func executeMercury(ctx context.Context, e Experiment, request AdapterRequest, dir string) (AdapterResponse, string, string, string, error) {
	return executeMercuryAt(ctx, e, request, dir, mercuryAPIBase)
}

func finishMercuryOutput(raw, diagnostic, capturePath, key, status string, runErr error) (AdapterResponse, string, string, string, error) {
	journal := ""
	f, err := os.Open(capturePath)
	if err == nil {
		b, readErr := readBounded(f, maxAdapterBytes)
		_ = f.Close()
		if readErr == nil {
			journal = string(b)
		} else if runErr == nil {
			runErr = readErr
		}
	} else if runErr == nil {
		runErr = fmt.Errorf("Mercury provider capture missing")
	}
	envelope := mercuryOutput{Claude: redactGeminiOutput(raw, key), Capture: redactGeminiOutput(journal, key)}
	if runErr != nil {
		if status == "completed" {
			status = "adapter-error"
		}
		envelope.Error = "Mercury execution failed"
	}
	data, _ := json.Marshal(envelope)
	response, parseErr := parseMercuryOutput(string(data))
	if runErr == nil && parseErr != nil {
		runErr = parseErr
		status = "adapter-error"
	}
	return response, string(data), redactGeminiOutput(diagnostic, key), status, runErr

}

// Only tests supply a local provider URL; no environment or experiment endpoint override.
func executeMercuryAt(ctx context.Context, e Experiment, request AdapterRequest, dir, apiBase string) (AdapterResponse, string, string, string, error) {
	var raw, diagnostic string
	capturePath := filepath.Join(dir, "mercury-capture.jsonl")
	key := os.Getenv("INCEPTION_API_KEY")
	finish := func(status string, runErr error) (AdapterResponse, string, string, string, error) {
		return finishMercuryOutput(raw, diagnostic, capturePath, key, status, runErr)
	}

	if err := validateMercury(e); err != nil {
		return finish("adapter-error", err)
	}
	if key == "" {
		return finish("adapter-error", fmt.Errorf("INCEPTION_API_KEY is required for Mercury"))
	}
	if err := checkClaudeManagedSettings(); err != nil {
		return finish("adapter-error", err)
	}
	bounded, cancel := context.WithTimeout(ctx, time.Duration(e.TimeoutSeconds)*time.Second)
	defer cancel()
	tokenBytes := make([]byte, 24)
	if _, err := rand.Read(tokenBytes); err != nil {
		return finish("adapter-error", err)
	}
	token := "sk-" + hex.EncodeToString(tokenBytes)
	bridgeDir := filepath.Join(dir, "mercury-bridge")
	if err := os.Mkdir(bridgeDir, 0700); err != nil {
		return finish("adapter-error", err)
	}
	if err := os.WriteFile(filepath.Join(bridgeDir, "mercury_hook.py"), []byte(mercuryHook), 0600); err != nil {
		return finish("adapter-error", err)
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return finish("adapter-error", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	_ = listener.Close()
	config := map[string]any{
		"model_list":       []any{map[string]any{"model_name": "mercury-3", "litellm_params": map[string]any{"model": "inception/mercury-3", "api_base": apiBase, "api_key": "os.environ/INCEPTION_API_KEY", "max_retries": 0}}},
		"litellm_settings": map[string]any{"drop_params": false, "num_retries": 0, "telemetry": false, "callbacks": []string{"mercury_hook.capture"}},
		"router_settings":  map[string]any{"num_retries": 0}, "general_settings": map[string]any{"master_key": token},
	}
	data, _ := json.Marshal(config)
	configPath := filepath.Join(bridgeDir, "proxy.yaml")
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return finish("adapter-error", err)
	}
	bridge := exec.CommandContext(bounded, e.Claude.Mercury.Bridge, "--config", configPath, "--host", "127.0.0.1", "--port", fmt.Sprint(port))
	configureBenchmarkProcess(bridge)
	bridge.Dir = bridgeDir
	bridge.Env = append(mercuryEnvironment(bridgeDir), "INCEPTION_API_KEY="+key, "MERCURY_API_BASE="+apiBase, "MERCURY_CAPTURE_PATH="+capturePath)
	var bridgeOutput limitedBuffer
	bridge.Stdout = &bridgeOutput
	bridge.Stderr = &bridgeOutput
	if err := bridge.Start(); err != nil {
		return finish("adapter-error", fmt.Errorf("start Mercury bridge: %w", err))
	}
	done := make(chan struct{})
	var bridgeErr error
	go func() { bridgeErr = bridge.Wait(); close(done) }()
	stopped := false
	stop := func() {
		if !stopped {
			cleanupBenchmarkProcess(bridge)
			<-done
			stopped = true
			diagnostic += "\n" + redactGeminiOutput(bridgeOutput.String(), token)
		}
	}
	defer stop()
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	if err := waitMercuryBridge(bounded, baseURL, token, done); err != nil {
		stop()
		status := "adapter-error"
		if bounded.Err() != nil {
			status = "timeout"
			if ctx.Err() != nil {
				status = "cancelled"
			}
		}
		return finish(status, err)
	}
	// shortcut: only the existing single-file schema task path is supported; qualify general multi-turn tools before expanding benchmark tasks.
	_, raw, diagnostic, status, runErr := executeClaudeProcess(bounded, e, request, dir, token, baseURL)
	select {
	case <-done:
		if runErr == nil {
			runErr = fmt.Errorf("Mercury bridge exited during attempt: %v", bridgeErr)
			status = "adapter-error"
		}
	default:
	}
	stop() // Provider sockets are closed before capture acceptance or workspace removal.
	if bounded.Err() != nil {
		status = "timeout"
		if ctx.Err() != nil {
			status = "cancelled"
		}
		runErr = bounded.Err()
	}
	raw = redactGeminiOutput(raw, token)
	diagnostic = redactGeminiOutput(diagnostic, token)
	if bridgeOutput.overflow && runErr == nil {
		runErr = fmt.Errorf("Mercury bridge diagnostics exceeded 1 MiB")
		status = "adapter-error"
	}
	return finish(status, runErr)
}

func waitMercuryBridge(ctx context.Context, baseURL, token string, done <-chan struct{}) error {
	client := &http.Client{Timeout: 250 * time.Millisecond, Transport: &http.Transport{DisableKeepAlives: true}}
	defer client.CloseIdleConnections()
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-done:
			return fmt.Errorf("Mercury bridge exited before readiness")
		case <-ticker.C:
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/v1/models", nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		response, err := client.Do(req)
		if err != nil {
			continue
		}
		body, readErr := readBounded(response.Body, 65536)
		_ = response.Body.Close()
		var models struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if response.StatusCode == http.StatusOK && readErr == nil && json.Unmarshal(body, &models) == nil {
			for _, model := range models.Data {
				if model.ID == "mercury-3" {
					return nil
				}
			}
		}
	}
}
