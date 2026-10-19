package bench

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// CodexConfig opts into the built-in, API-key-only Codex CLI adapter.
type CodexConfig struct {
	Reasoning string `json:"reasoning"`
	Auth      string `json:"auth"`
}

// CodexReportedUsage preserves missing native fields as unknown, including on failures.
type CodexReportedUsage struct {
	InputTokens           *int64 `json:"input_tokens,omitempty"`
	OutputTokens          *int64 `json:"output_tokens,omitempty"`
	CachedInputTokens     *int64 `json:"cached_input_tokens,omitempty"`
	CacheWriteInputTokens *int64 `json:"cache_write_input_tokens,omitempty"`
	ReasoningOutputTokens *int64 `json:"reasoning_output_tokens,omitempty"`
}

// CodexEvidence describes observed events, not inferred skill or tool activation.
type CodexEvidence struct {
	SessionID          string              `json:"session_id"`
	Terminal           string              `json:"terminal"`
	UsageStatus        string              `json:"usage_status"`
	ReportedUsage      *CodexReportedUsage `json:"reported_usage"`
	ObservedItemTypes  []string            `json:"observed_item_types"`
	CompletedItemTypes []string            `json:"completed_item_types"`
	EventCount         int                 `json:"event_count"`
	ResolvedModel      *string             `json:"resolved_model"`
	ResolvedReasoning  *string             `json:"resolved_reasoning"`
	OutputTruncated    bool                `json:"output_truncated,omitempty"`
}

const codexSourceSchema = `{"type":"object","properties":{"source":{"type":"string"}},"required":["source"],"additionalProperties":false}`

func executeCodex(ctx context.Context, e Experiment, request AdapterRequest, dir string) (AdapterResponse, string, string, string, error) {
	empty, _ := parseCodexEvents("")
	if e.Codex == nil || e.Codex.Auth != "api-key-env" || len(e.Command) != 1 {
		return empty, "", "", "adapter-error", fmt.Errorf("codex requires api-key-env authentication and one executable")
	}
	key := os.Getenv("CODEX_API_KEY")
	if key == "" {
		return empty, "", "", "adapter-error", fmt.Errorf("CODEX_API_KEY is required for the explicit api-key-env policy")
	}
	if err := os.MkdirAll(filepath.Join(dir, ".codex"), 0700); err != nil {
		return empty, "", "", "adapter-error", err
	}
	schema := filepath.Join(dir, "response-schema.json")
	if err := os.WriteFile(schema, []byte(codexSourceSchema), 0600); err != nil {
		return empty, "", "", "adapter-error", err
	}
	bounded, cancel := context.WithTimeout(ctx, time.Duration(e.TimeoutSeconds)*time.Second)
	defer cancel()
	cmd := exec.CommandContext(bounded, e.Command[0], codexArguments(e, schema)...)
	configureBenchmarkProcess(cmd)
	defer cleanupBenchmarkProcess(cmd)
	cmd.Dir = dir
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + dir, "CODEX_HOME=" + filepath.Join(dir, ".codex"), "CODEX_API_KEY=" + key,
		"TMPDIR=" + dir, "XDG_CONFIG_HOME=" + dir, "XDG_CACHE_HOME=" + dir, "GOWORK=off", "GOENV=off"}
	// Arm names and repetition numbers do not enter the model prompt.
	cmd.Stdin = strings.NewReader("Instructions:\n" + request.Instructions + "\n\nTask:\n" + request.Prompt + "\n\nReturn a JSON object with source containing the complete Go source file.")
	var stdout, stderr limitedBuffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	runErr := cmd.Run()
	out, diagnostic := redactCodexOutput(stdout.String(), key), redactCodexOutput(stderr.String(), key)
	response, parseErr := parseCodexEvents(out)
	if stdout.overflow || stderr.overflow {
		markCodexTruncated(&response)
		return response, out, diagnostic, "adapter-error", fmt.Errorf("codex output exceeded 1 MiB per stream")
	}
	if bounded.Err() != nil {
		status := "timeout"
		if ctx.Err() != nil {
			status = "cancelled"
		}
		return response, out, diagnostic, status, bounded.Err()
	}
	if runErr != nil {
		return response, out, diagnostic, "adapter-error", fmt.Errorf("codex execution: %w", runErr)
	}
	if parseErr != nil {
		return response, out, diagnostic, "adapter-error", parseErr
	}
	return response, out, diagnostic, "completed", nil
}

func markCodexTruncated(response *AdapterResponse) {
	response.Native.OutputTruncated = true
	response.Native.UsageStatus = "truncated"
	response.Usage = nil
}

func codexArguments(e Experiment, schema string) []string {
	return []string{"exec", "--json", "--ephemeral", "--ignore-user-config", "--ignore-rules", "--skip-git-repo-check",
		"--sandbox", "read-only", "--output-schema", schema, "--model", e.Model,
		"-c", "model_reasoning_effort=" + strconv.Quote(e.Codex.Reasoning),
		"-c", `model_provider="openai"`, "-c", `approval_policy="never"`,
		"-c", `shell_environment_policy.inherit="none"`, "-c", "project_doc_max_bytes=0", "-"}
}

func redactCodexOutput(output, key string) string {
	if key == "" {
		return output
	}
	// A final-message JSON string embeds another JSON document. Match escaped
	// forms from deepest to shallowest so replacing one form cannot expose another.
	forms := []string{key}
	for depth := 0; depth < 64; depth++ {
		encoded, _ := json.Marshal(forms[len(forms)-1])
		next := string(encoded[1 : len(encoded)-1])
		if next == forms[len(forms)-1] || len(next) > maxAdapterBytes {
			break
		}
		forms = append(forms, next)
	}
	for i := len(forms) - 1; i >= 0; i-- {
		output = strings.ReplaceAll(output, forms[i], "[REDACTED]")
	}
	return output
}
