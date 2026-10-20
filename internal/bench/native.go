package bench

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// IsNative distinguishes built-in provider execution from the command protocol.
func (e Experiment) IsNative() bool { return e.Gemini != nil || e.Claude != nil || e.Codex != nil }

func nativeKeyName(e Experiment) string {
	if e.Codex != nil {
		return "OPENAI_API_KEY"
	}
	if e.Claude != nil {
		return "ANTHROPIC_API_KEY"
	}
	return "GEMINI_API_KEY"
}

func nativeSettings(e Experiment) ([]byte, error) {
	if e.Codex != nil {
		return codexSettings(e)
	}
	if e.Claude != nil {
		return claudeSettings(e)
	}
	return geminiSettings(e)
}

func nativeArguments(e Experiment) []string {
	if e.Codex != nil {
		return codexArguments(e)
	}
	if e.Claude != nil {
		return claudeArguments(e)
	}
	return geminiArguments(e)
}

func parseNativeOutput(e Experiment, output string) (AdapterResponse, error) {
	if e.Codex != nil {
		return parseCodexOutput(output)
	}
	if e.Claude != nil {
		if e.Claude.OutputFormat == "json-schema" {
			return parseClaudeStructuredOutput(output)
		}
		return parseClaudeOutput(output)
	}
	return parseGeminiVersionedOutput(e.Gemini, output)
}

func resolveNative(r *ResolvedExperiment, e Experiment) error {
	if e.Gemini != nil {
		if err := validateNewGeminiExecution(e.Gemini); err != nil {
			return err
		}
	}
	if e.Claude != nil && e.Claude.OutputFormat != "json-schema" {
		return fmt.Errorf("claude new execution requires output_format json-schema; legacy text format is replay-only")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, e.Command[0], "--version")
	configureBenchmarkProcess(cmd)
	defer cleanupBenchmarkProcess(cmd)
	var out limitedBuffer
	cmd.Stdout = &out
	cmd.Env = []string{"PATH=" + os.Getenv("PATH")}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("probe %s version: %w", e.Agent, err)
	}
	version := strings.TrimSpace(out.String())
	want := geminiStableVersion
	if e.Codex != nil {
		want = "codex-cli 0.160.1"
	}
	if e.Claude != nil {
		want = "2.1.292 (Claude Code)"
	}
	if out.overflow || version != want {
		return fmt.Errorf("%s CLI %s is required for the verified native telemetry/config contract", e.Agent, want)
	}
	if e.Codex != nil {
		r.CodexVersion = version
		r.BudgetPolicy = "OPENAI_API_KEY only via fixed direct OpenAI provider; isolated app-server stdio; CLI-resolved model/effort/provider checked before inference, backend identity unknown; reroutes refused; stop between attempts at max_total_tokens or unknown usage; max_run_seconds includes grading; per-attempt timeout_seconds; zero HTTP/SSE retries; no supported per-request output or model-turn cap; no in-flight monetary guarantee; cumulative native thread usage once; cost unknown"
	} else if e.Claude != nil {
		r.ClaudeVersion = version
		r.BudgetPolicy = "ANTHROPIC_API_KEY only via bare mode; local managed configuration refused; remote policy remains enforced and requires live preflight; stop between attempts at max_total_tokens or unknown usage; max_run_seconds includes grading; per-attempt timeout_seconds; no in-flight monetary guarantee; CLI-reported whole-call modelUsage; 4 agentic turns/attempt, requested 8192 output tokens/request, MAX_STRUCTURED_OUTPUT_RETRIES=1 (one structured-output attempt) and 0 transport retries; no fallback accepted; native cost is a client estimate, not billing"
	} else {
		r.GeminiVersion = version
		r.BudgetPolicy = fmt.Sprintf("GEMINI_API_KEY only; GEMINI_CLI_TRUST_WORKSPACE=true only in the generated workspace child process; stop between attempts at max_total_tokens or unknown usage; max_run_seconds includes grading; per-attempt timeout_seconds; no in-flight monetary guarantee; best-effort CLI-reported usage, missing usage stays unknown and cost is not reconciled; 4 model turns/attempt and %d output tokens/turn; no fallback accepted; cost unknown", geminiMaxOutputTokens(e.Gemini))
	}
	settings, err := nativeSettings(e)
	if err != nil {
		return err
	}
	r.NativeSettings = settings
	r.NativeArguments = nativeArguments(e)
	return nil
}

// Codex reports resolved CLI configuration, not provider per-model accounting.
// Keep that assurance distinct and preserve the existing Gemini/Claude checks.
func nativeIdentityStatus(e Experiment, evidence *NativeEvidence) string {
	if e.Codex == nil {
		return nativeModelStatus(evidence, e.Model)
	}
	if evidence == nil || evidence.Codex == nil || evidence.Codex.IdentitySource != "cli-thread-start" || evidence.Codex.ModelProvider == nil || evidence.ResolvedModel == nil || evidence.ResolvedReasoning == nil {
		return "model-unverified"
	}
	if *evidence.Codex.ModelProvider != "benchmark_openai" || *evidence.ResolvedModel != e.Model || *evidence.ResolvedReasoning != e.Codex.Reasoning {
		return "model-mismatch"
	}
	return ""
}
