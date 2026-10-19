package bench

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"time"
)

// Attempt retains the evidence for one trial, including unsuccessful execution.
type Attempt struct {
	Request    AdapterRequest      `json:"request"`
	Status     string              `json:"status"`
	Response   *AdapterResponse    `json:"response"`
	Stdout     string              `json:"stdout"`
	Stderr     string              `json:"stderr"`
	Diagnostic string              `json:"diagnostic"`
	SourceSHA  string              `json:"source_sha256"`
	AgentNanos int64               `json:"agent_nanos"`
	GradeNanos int64               `json:"grade_nanos"`
	Grade      CandidateEvaluation `json:"grade"`
}

// ExperimentRun is a complete immutable pilot artifact. Digest detects corruption, not forgery.
type ExperimentRun struct {
	StopReason string             `json:"stop_reason,omitempty"`
	Version    int                `json:"version"`
	Created    time.Time          `json:"created"`
	Resolved   ResolvedExperiment `json:"resolved"`
	Attempts   []Attempt          `json:"attempts"`
	Digest     string             `json:"sha256"`
}

type limitedBuffer struct {
	bytes.Buffer
	overflow bool
}

// ReadFrom prevents io.Copy from bypassing Write through the embedded bytes.Buffer.
func (b *limitedBuffer) ReadFrom(r io.Reader) (int64, error) {
	return io.Copy(struct{ io.Writer }{b}, r)
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	n := len(p)
	remaining := maxAdapterBytes - b.Len()
	if len(p) > remaining {
		p = p[:remaining]
		b.overflow = true
	}
	_, _ = b.Buffer.Write(p)
	return n, nil
}

func executeAdapter(ctx context.Context, e Experiment, request AdapterRequest, dir string) (AdapterResponse, string, string, string, error) {
	if e.Codex != nil {
		return executeCodex(ctx, e, request, dir)
	}
	if e.Claude != nil {
		return executeClaude(ctx, e, request, dir)
	}
	if e.Gemini != nil {
		return executeGemini(ctx, e, request, dir)
	}
	var response AdapterResponse
	payload, err := json.Marshal(request)
	if err != nil {
		return response, "", "", "adapter-error", err
	}
	bounded, cancel := context.WithTimeout(ctx, time.Duration(e.TimeoutSeconds)*time.Second)
	defer cancel()
	cmd := exec.CommandContext(bounded, e.Command[0], e.Command[1:]...)
	configureBenchmarkProcess(cmd)
	defer cleanupBenchmarkProcess(cmd)
	cmd.Dir = dir
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + dir, "TMPDIR=" + dir, "XDG_CONFIG_HOME=" + dir, "XDG_CACHE_HOME=" + dir, "GOWORK=off", "GOENV=off"}
	cmd.Stdin = bytes.NewReader(payload)
	var stdout, stderr limitedBuffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if bounded.Err() != nil {
		status := "timeout"
		if ctx.Err() != nil {
			status = "cancelled"
		}
		return response, stdout.String(), stderr.String(), status, bounded.Err()
	}
	if stdout.overflow || stderr.overflow {
		return response, stdout.String(), stderr.String(), "adapter-error", fmt.Errorf("adapter output exceeded 1 MiB per stream")
	}
	if err != nil {
		return response, stdout.String(), stderr.String(), "adapter-error", err
	}
	if err := strictJSON(stdout.Bytes(), &response); err != nil {
		return response, stdout.String(), stderr.String(), "adapter-error", fmt.Errorf("decode response: %w", err)
	}
	if response.Version != experimentVersion || response.Source == "" {
		return response, stdout.String(), stderr.String(), "adapter-error", fmt.Errorf("response requires version 1 and nonempty source")
	}
	if err := validateUsage(response.Usage); err != nil {
		return response, stdout.String(), stderr.String(), "adapter-error", err
	}
	return response, stdout.String(), stderr.String(), "completed", nil
}

// RunExperiment executes a resolved paired pilot and returns its unique artifact path.
func RunExperiment(ctx context.Context, resolved ResolvedExperiment, outputRoot string) (string, error) {
	// Re-resolve before any execution so dry-run metadata cannot silently become stale.
	current, err := ResolveExperiment(resolved.Experiment, ".")
	if err != nil {
		return "", err
	}
	if !reflect.DeepEqual(current.CommandFiles, resolved.CommandFiles) || current.EvaluatorSHA != resolved.EvaluatorSHA || current.Grader != resolved.Grader || current.GeminiVersion != resolved.GeminiVersion || current.ClaudeVersion != resolved.ClaudeVersion || current.CodexVersion != resolved.CodexVersion || (current.Experiment.IsNative() && !jsonEquivalent(current.NativeSettings, resolved.NativeSettings)) || !reflect.DeepEqual(current.NativeArguments, resolved.NativeArguments) || !reflect.DeepEqual(current.Schedule, resolved.Schedule) {
		return "", fmt.Errorf("resolved experiment changed before execution")
	}
	if err := os.MkdirAll(outputRoot, 0700); err != nil {
		return "", err
	}
	dir, err := os.MkdirTemp(outputRoot, "experiment-")
	if err != nil {
		return "", err
	}
	if resolved.Experiment.IsNative() {
		keyName := nativeKeyName(resolved.Experiment)
		if os.Getenv(keyName) == "" {
			return dir, fmt.Errorf("%s is required for explicitly selected API authentication", keyName)
		}
		bounded, cancel := context.WithTimeout(ctx, time.Duration(resolved.Experiment.MaxRunSeconds)*time.Second)
		defer cancel()
		ctx = bounded
	}
	run := ExperimentRun{Version: experimentVersion, Created: time.Now().UTC(), Resolved: resolved}
	if err := writeExclusiveJSON(filepath.Join(dir, "resolved.json"), resolved); err != nil {
		return dir, err
	}
	var usedTokens int64
	var stopErr error
	sessions := map[string]bool{}
	for i, trial := range resolved.Schedule {
		if err := ctx.Err(); err != nil {
			run.StopReason = "cancelled"
			if err == context.DeadlineExceeded {
				run.StopReason = "run-timeout"
			}
			stopErr = err
			break
		}
		a := Attempt{Request: newAdapterRequest(resolved.Experiment, trial)}
		workspace, err := os.MkdirTemp("", "ateam-agent-*")
		if err != nil {
			return dir, err
		}
		start := time.Now()
		response, stdout, stderr, status, runErr := executeAdapter(ctx, resolved.Experiment, a.Request, workspace)
		a.AgentNanos = time.Since(start).Nanoseconds()
		if resolved.Experiment.IsNative() && runErr == nil {
			if reason := nativeIdentityStatus(resolved.Experiment, response.Native); reason != "" {
				status = "adapter-error"
				runErr = fmt.Errorf("%s: native evidence does not confirm the configured model/settings", reason)
			}
		}
		a.Stdout = stdout
		a.Stderr = stderr
		a.Status = status
		removeErr := os.RemoveAll(workspace)
		if response.Native != nil {
			a.Response = &response
			if response.Source != "" {
				a.SourceSHA = digest([]byte(response.Source))
			}
		}
		if runErr != nil {
			a.Diagnostic = runErr.Error()
		} else {
			a.Response = &response
			a.SourceSHA = digest([]byte(response.Source))
			gradeAttempt(ctx, &a)
		}
		if a.Response != nil && a.Response.Native != nil {
			session := a.Response.Native.SessionID
			if session != "" && sessions[session] {
				a.Status = "adapter-error"
				a.Grade = CandidateEvaluation{}
				a.GradeNanos = 0
				a.Diagnostic = "native session ID reused across attempts"
			}
			sessions[session] = true
		}
		if err := writeExclusiveJSON(filepath.Join(dir, fmt.Sprintf("attempt-%04d.json", i)), a); err != nil {
			return dir, err
		}
		run.Attempts = append(run.Attempts, a)
		if removeErr != nil {
			return dir, fmt.Errorf("remove workspace: %w", removeErr)
		}
		reason, err := nativeStop(ctx, resolved.Experiment, a, &usedTokens, i+1 == len(resolved.Schedule))
		if reason != "" {
			run.StopReason, stopErr = reason, err
			break
		}
	}
	run.Digest = runDigest(run)
	path := filepath.Join(dir, "run.json")
	if err := writeExclusiveJSON(path, run); err != nil {
		return dir, err
	}
	return path, stopErr
}

func nativeStop(ctx context.Context, e Experiment, a Attempt, used *int64, last bool) (string, error) {
	if ctx.Err() != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "run-timeout", ctx.Err()
		}
		return "cancelled", ctx.Err()
	}
	if !e.IsNative() {
		return "", nil
	}
	if a.Diagnostic == "native session ID reused across attempts" {
		return "session-reuse", fmt.Errorf("native session reused; stopped before further calls")
	}
	if a.Response != nil {
		if reason := nativeIdentityStatus(e, a.Response.Native); reason != "" {
			return reason, fmt.Errorf("native model accounting invalid; stopped before further calls")
		}
	}
	if a.Response == nil || a.Response.Usage == nil {
		return "unknown-usage", fmt.Errorf("native usage unknown; stopped before further calls")
	}
	*used += a.Response.Usage.InputTokens + a.Response.Usage.OutputTokens
	if *used >= e.MaxTotalTokens && !last {
		return "token-limit", fmt.Errorf("native token ceiling reached; stopped before further calls")
	}
	return "", nil
}

func gradeAttempt(ctx context.Context, a *Attempt) {
	start := time.Now()
	grade, err := EvaluateCodingCandidate(ctx, a.Request.Trial.Task, a.Response.Source)
	a.GradeNanos = time.Since(start).Nanoseconds()
	a.Grade = grade
	switch {
	case err != nil:
		a.Status = "grading-error"
		a.Diagnostic = err.Error()
	case !grade.Supported:
		a.Status = "grading-error"
		a.Diagnostic = "unsupported evaluator"
	case grade.Passed:
		a.Status = "passed"
		a.Diagnostic = ""
	default:
		a.Status = "failed"
		a.Diagnostic = ""
	}
}

func writeExclusiveJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if len(data) > 64<<20 {
		return fmt.Errorf("artifact exceeds 64 MiB; reduce repetitions or adapter output")
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, writeErr := f.Write(append(data, '\n'))
	closeErr := f.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}
func runDigest(r ExperimentRun) string { r.Digest = ""; return jsonDigest(r) }

// LoadExperimentRun verifies complete or explicitly stopped history and evidence consistency.
func LoadExperimentRun(path string) (ExperimentRun, error) {
	var r ExperimentRun
	f, err := os.Open(path)
	if err != nil {
		return r, err
	}
	defer f.Close()
	// shortcut: pilot artifacts are capped at 64 MiB; use streaming artifacts before larger task suites.
	data, err := readBounded(f, 64<<20)
	if err != nil {
		return r, err
	}
	if err := strictJSON(data, &r); err != nil {
		return r, err
	}
	if err := validateRun(r); err != nil {
		return r, err
	}
	return r, nil
}

func validateRun(r ExperimentRun) error {
	if r.Version != experimentVersion || r.Created.IsZero() || r.Digest != runDigest(r) {
		return fmt.Errorf("invalid run version, timestamp or digest")
	}
	cfg := r.Resolved.Experiment
	if err := validateExperiment(r.Resolved.Original); err != nil {
		return err
	}
	originalShape := cfg
	originalShape.Command = r.Resolved.Original.Command
	if r.Resolved.ConfigSHA != jsonDigest(r.Resolved.Original) || !reflect.DeepEqual(originalShape, r.Resolved.Original) {
		return fmt.Errorf("original configuration provenance mismatch")
	}
	if err := validateExperiment(cfg); err != nil {
		return err
	}
	if r.Resolved.EvaluatorSHA != evaluatorDigest(cfg.Tasks) {
		return fmt.Errorf("evaluator mismatch: rescore with the matching task/grader version")
	}
	schedule := Schedule(cfg)
	if !reflect.DeepEqual(schedule, r.Resolved.Schedule) || len(r.Attempts) > len(schedule) || (r.StopReason == "" && len(r.Attempts) != len(schedule)) {
		return fmt.Errorf("incomplete or inconsistent paired schedule")
	}
	switch r.StopReason {
	case "", "cancelled", "run-timeout":
	case "unknown-usage", "token-limit", "session-reuse", "model-mismatch", "model-unverified":
		if !cfg.IsNative() {
			return fmt.Errorf("native stop reason on command adapter")
		}
	default:
		return fmt.Errorf("unknown run stop reason")
	}
	if cfg.IsNative() && (((cfg.Gemini != nil && r.Resolved.GeminiVersion == "") || (cfg.Claude != nil && r.Resolved.ClaudeVersion == "") || (cfg.Codex != nil && r.Resolved.CodexVersion == "")) || r.Resolved.BudgetPolicy == "" || len(r.Resolved.NativeSettings) == 0 || len(r.Resolved.NativeArguments) == 0) {
		return fmt.Errorf("missing native provenance")
	}
	if cfg.IsNative() {
		expected, err := nativeSettings(cfg)
		if err != nil || !jsonEquivalent(expected, r.Resolved.NativeSettings) || !reflect.DeepEqual(nativeArguments(cfg), r.Resolved.NativeArguments) {
			return fmt.Errorf("native execution settings provenance mismatch")
		}
	}
	if len(r.Resolved.CommandFiles) == 0 || r.Resolved.ConfigSHA == "" || r.Resolved.Runtime == "" || r.Resolved.RunnerSHA == "" || r.Resolved.Grader.Path == "" || r.Resolved.Grader.SHA == "" || r.Resolved.Grader.Version == "" {
		return fmt.Errorf("missing provenance")
	}
	sessions := map[string]bool{}
	for i, a := range r.Attempts {
		if !reflect.DeepEqual(a.Request, newAdapterRequest(cfg, schedule[i])) || a.AgentNanos < 0 || a.GradeNanos < 0 || a.AgentNanos > int64(24*time.Hour) || a.GradeNanos > int64(24*time.Hour) {
			return fmt.Errorf("attempt %d: request/timing mismatch", i)
		}
		if cfg.IsNative() {
			parsed, parseErr := parseNativeOutput(cfg, a.Stdout)
			if a.Response != nil && a.Response.Native != nil && a.Response.Native.OutputTruncated {
				if a.Status != "adapter-error" || a.Diagnostic != cfg.Agent+" output exceeded 1 MiB per stream" {
					return fmt.Errorf("attempt %d: invalid truncation outcome", i)
				}
				markGeminiTruncated(&parsed)
			}
			if a.Response == nil || !reflect.DeepEqual(&parsed, a.Response) {
				return fmt.Errorf("attempt %d: native response differs from raw output", i)
			}
			if err := validateUsage(a.Response.Usage); err != nil {
				return err
			}
			if a.Response.Source != "" && a.SourceSHA != digest([]byte(a.Response.Source)) {
				return fmt.Errorf("attempt %d: native source digest mismatch", i)
			}
			session := a.Response.Native.SessionID
			if a.Status == "passed" || a.Status == "failed" || a.Status == "grading-error" {
				if parseErr != nil || nativeIdentityStatus(cfg, a.Response.Native) != "" || (session != "" && sessions[session]) {
					return fmt.Errorf("attempt %d: incomplete or reused native session", i)
				}
			}
			sessions[session] = true
		}
		switch a.Status {
		case "passed", "failed", "grading-error":
			if a.Response == nil || a.Response.Version != experimentVersion || a.Response.Source == "" || a.SourceSHA != digest([]byte(a.Response.Source)) {
				return fmt.Errorf("attempt %d: missing or corrupt candidate", i)
			}
			var parsed AdapterResponse
			if err := strictJSON([]byte(a.Stdout), &parsed); !cfg.IsNative() && (err != nil || !reflect.DeepEqual(&parsed, a.Response)) {
				return fmt.Errorf("attempt %d: response differs from raw output", i)
			}
			if err := validateUsage(a.Response.Usage); err != nil {
				return err
			}
			if a.Status == "passed" && (!a.Grade.Supported || !a.Grade.Passed || a.Grade.Verdict != "passed") {
				return fmt.Errorf("attempt %d: contradictory pass", i)
			}
			if a.Status == "failed" && (!a.Grade.Supported || a.Grade.Passed || a.Grade.Verdict != "failed") {
				return fmt.Errorf("attempt %d: contradictory failure", i)
			}
			if a.Status == "grading-error" && a.Diagnostic == "" {
				return fmt.Errorf("attempt %d: missing grading diagnostic", i)
			}
		case "adapter-error", "timeout", "cancelled":
			if (!cfg.IsNative() && (a.Response != nil || a.SourceSHA != "")) || a.Grade.Passed || a.Diagnostic == "" {
				return fmt.Errorf("attempt %d: invalid execution failure", i)
			}
		default:
			return fmt.Errorf("attempt %d: unknown status %q", i, a.Status)
		}
	}
	return validateNativeStop(r)
}

func validateNativeStop(r ExperimentRun) error {
	if !r.Resolved.Experiment.IsNative() {
		return nil
	}
	var used int64
	sessions := map[string]bool{}
	for i, a := range r.Attempts {
		last := i+1 == len(r.Attempts)
		if a.Response == nil || a.Response.Native == nil {
			return fmt.Errorf("missing native evidence")
		}
		session := a.Response.Native.SessionID
		reused := session != "" && sessions[session]
		sessions[session] = true
		if reused {
			if !last || (r.StopReason != "session-reuse" && r.StopReason != "cancelled" && r.StopReason != "run-timeout") || a.Diagnostic != "native session ID reused across attempts" {
				return fmt.Errorf("invalid session reuse stop")
			}
			return nil
		}
		if reason := nativeIdentityStatus(r.Resolved.Experiment, a.Response.Native); reason != "" {
			if !last || (r.StopReason != reason && r.StopReason != "cancelled" && r.StopReason != "run-timeout") {
				return fmt.Errorf("native execution continued with unverified or changed model")
			}
			return nil
		}
		if a.Response.Usage == nil {
			if !last || (r.StopReason != "unknown-usage" && r.StopReason != "cancelled" && r.StopReason != "run-timeout") {
				return fmt.Errorf("native execution continued after unknown usage")
			}
			return nil
		}
		used += a.Response.Usage.InputTokens + a.Response.Usage.OutputTokens
		if used >= r.Resolved.Experiment.MaxTotalTokens && !last {
			return fmt.Errorf("native execution continued after token ceiling")
		}
	}
	switch r.StopReason {
	case "token-limit":
		if len(r.Attempts) == len(r.Resolved.Schedule) || used < r.Resolved.Experiment.MaxTotalTokens {
			return fmt.Errorf("unsubstantiated token stop")
		}
	case "unknown-usage", "session-reuse", "model-mismatch", "model-unverified":
		return fmt.Errorf("unsubstantiated native stop")
	}
	return nil
}

func nativeModelStatus(evidence *NativeEvidence, model string) string {
	if evidence == nil || len(evidence.ReportedModels) == 0 {
		return "model-unverified"
	}
	if len(evidence.ReportedModels) != 1 || evidence.ReportedModels[0] != model {
		return "model-mismatch"
	}
	return ""
}

func jsonEquivalent(a, b []byte) bool {
	var left, right any
	if uniqueJSON(a) != nil || uniqueJSON(b) != nil || json.Unmarshal(a, &left) != nil || json.Unmarshal(b, &right) != nil {
		return false
	}
	return reflect.DeepEqual(left, right)
}

// RescoreExperiment re-evaluates retained source without executing any adapter or overwriting evidence.
func RescoreExperiment(ctx context.Context, path string) (string, error) {
	return rescoreExperiment(ctx, path, gradeAttempt)
}

func rescoreExperiment(ctx context.Context, path string, grade func(context.Context, *Attempt)) (string, error) {
	run, err := LoadExperimentRun(path)
	if err != nil {
		return "", err
	}
	grader, err := currentGrader()
	if err != nil {
		return "", err
	}
	if grader != run.Resolved.Grader {
		return "", fmt.Errorf("grader toolchain changed; use the recorded Go toolchain for rescore")
	}
	for i := range run.Attempts {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		if run.Attempts[i].Status == "passed" || run.Attempts[i].Status == "failed" || run.Attempts[i].Status == "grading-error" {
			grade(ctx, &run.Attempts[i])
		}
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	run.Created = time.Now().UTC()
	run.Digest = runDigest(run)
	dir, err := os.MkdirTemp(filepath.Dir(path), "rescore-")
	if err != nil {
		return "", err
	}
	output := filepath.Join(dir, "run.json")
	if err := writeExclusiveJSON(output, run); err != nil {
		return "", err
	}
	return output, nil
}
