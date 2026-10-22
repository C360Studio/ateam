package bench

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const experimentVersion = 1
const maxAdapterBytes = 1 << 20

// Experiment controls a paired pilot with one command and two instruction variants.
type Experiment struct {
	Version        int               `json:"version"`
	Mode           string            `json:"mode,omitempty"`
	Bundles        *BundleReferences `json:"bundles,omitempty"`
	Gemini         *GeminiConfig     `json:"gemini,omitempty"`
	Claude         *ClaudeConfig     `json:"claude,omitempty"`
	Codex          *CodexConfig      `json:"codex,omitempty"`
	MaxTotalTokens int64             `json:"max_total_tokens,omitempty"`
	MaxRunSeconds  int               `json:"max_run_seconds,omitempty"`
	Name           string            `json:"name"`
	Command        []string          `json:"command"`
	Agent          string            `json:"agent"`
	AgentVersion   string            `json:"agent_version"`
	Model          string            `json:"model"`
	Baseline       string            `json:"baseline"`
	Candidate      string            `json:"candidate"`
	Tasks          []string          `json:"tasks"`
	Repetitions    int               `json:"repetitions"`
	Seed           int64             `json:"seed"`
	TimeoutSeconds int               `json:"timeout_seconds"`
}

// Trial identifies one arm of a task/repetition pair.
type Trial struct {
	Task       string `json:"task"`
	Repetition int    `json:"repetition"`
	Arm        string `json:"arm"`
}

// ResolvedExperiment is the reviewable dry-run output, including reproducibility metadata.
type ResolvedExperiment struct {
	MercuryBridgeVersion string            `json:"mercury_bridge_version,omitempty"`
	Bundles              *BundleResolution `json:"bundles,omitempty"`
	NativeSettings       json.RawMessage   `json:"native_settings,omitempty"`
	NativeArguments      []string          `json:"native_arguments,omitempty"`
	GeminiVersion        string            `json:"gemini_version,omitempty"`
	ClaudeVersion        string            `json:"claude_version,omitempty"`
	CodexVersion         string            `json:"codex_version,omitempty"`
	BudgetPolicy         string            `json:"budget_policy,omitempty"`
	Original             Experiment        `json:"original_experiment"`
	Grader               GraderIdentity    `json:"grader"`
	RunnerSHA            string            `json:"runner_sha256"`
	Experiment           Experiment        `json:"experiment"`
	ConfigSHA            string            `json:"config_sha256"`
	EvaluatorSHA         string            `json:"evaluator_sha256"`
	CommandFiles         map[string]string `json:"command_files_sha256"`
	Runtime              string            `json:"runtime"`
	Schedule             []Trial           `json:"schedule"`
}

// AdapterRequest is sent to the command's stdin. No reference solution or test is exposed.
type AdapterRequest struct {
	Version      int    `json:"version"`
	Trial        Trial  `json:"trial"`
	Prompt       string `json:"prompt"`
	Instructions string `json:"instructions"`
}

// Usage is adapter-reported usage. Cache/reasoning counts are subsets, not added again.
type Usage struct {
	InputTokens           int64  `json:"input_tokens"`
	OutputTokens          int64  `json:"output_tokens"`
	CachedInputTokens     *int64 `json:"cached_input_tokens,omitempty"`
	ReasoningOutputTokens *int64 `json:"reasoning_output_tokens,omitempty"`
}

// AdapterResponse is the explicit command-adapter protocol; missing usage stays unknown.
type AdapterResponse struct {
	Native  *NativeEvidence `json:"native,omitempty"`
	Version int             `json:"version"`
	Source  string          `json:"source"`
	Usage   *Usage          `json:"usage"`
}

func strictJSON(data []byte, dst any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(dst); err != nil {
		return err
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("expected one JSON value")
	}
	return nil
}

// ParseExperiment rejects unknown fields, unsupported tasks and unbounded schedules.
func ParseExperiment(data []byte) (Experiment, error) {
	var e Experiment
	if len(data) > maxAdapterBytes {
		return e, fmt.Errorf("experiment exceeds 1 MiB")
	}
	if err := strictJSON(data, &e); err != nil {
		return e, fmt.Errorf("decode experiment: %w", err)
	}
	return e, validateExperiment(e)
}

func validateExperiment(e Experiment) error {
	if e.Version != experimentVersion {
		return fmt.Errorf("experiment version must be %d", experimentVersion)
	}
	if strings.TrimSpace(e.Name) == "" || strings.TrimSpace(e.Agent) == "" || (!e.IsNative() && strings.TrimSpace(e.AgentVersion) == "") || strings.TrimSpace(e.Model) == "" {
		return fmt.Errorf("name, agent, agent_version and model are required")
	}
	if len(e.Command) == 0 || strings.TrimSpace(e.Command[0]) == "" {
		return fmt.Errorf("command argv is required")
	}
	if e.Repetitions < 1 || e.Repetitions > 100 || e.TimeoutSeconds < 1 || e.TimeoutSeconds > 3600 || e.Seed < 0 {
		return fmt.Errorf("repetitions must be 1..100, timeout_seconds 1..3600, seed nonnegative")
	}
	if len(e.Tasks) < 1 || len(e.Tasks) > 3 {
		return fmt.Errorf("select 1..3 supported tasks")
	}
	seen := map[string]bool{}
	for _, id := range e.Tasks {
		if taskPrompt(id) == "" || seen[id] {
			return fmt.Errorf("unsupported or duplicate task %q", id)
		}
		seen[id] = true
	}
	if e.Mode != "" && e.Mode != "ab" && e.Mode != "aa" {
		return fmt.Errorf("mode must be aa or ab")
	}
	if e.Bundles != nil {
		if e.Baseline != "" || e.Candidate != "" {
			return fmt.Errorf("bundle references cannot be combined with inline instructions")
		}
		for _, path := range []string{e.Bundles.Baseline, e.Bundles.Candidate} {
			if err := validateBundlePath(path); err != nil {
				return err
			}
		}
	} else if e.Mode == "aa" {
		if e.Baseline != e.Candidate {
			return fmt.Errorf("A/A requires identical baseline and candidate instructions")
		}
	} else if e.Baseline == e.Candidate {
		return fmt.Errorf("baseline and candidate instructions must differ in A/B mode")
	}
	providers := 0
	for _, selected := range []bool{e.Gemini != nil, e.Claude != nil, e.Codex != nil} {
		if selected {
			providers++
		}
	}
	if providers > 1 {
		return fmt.Errorf("select exactly one native provider")
	}
	if e.Gemini != nil {
		if e.Gemini.MaxOutputTokens != nil && (*e.Gemini.MaxOutputTokens < 1 || *e.Gemini.MaxOutputTokens > 65536) {
			return fmt.Errorf("gemini max_output_tokens must be 1..65536")
		}
		switch e.Gemini.CLIVersion {
		case "", geminiLegacyVersion, geminiStableVersion:
		default:
			return fmt.Errorf("gemini cli_version must be %s (%s is historical replay-only)", geminiStableVersion, geminiLegacyVersion)
		}
		// Empty format preserves the original prompt-only canary for offline replay.
		if e.Gemini.OutputFormat != "" && e.Gemini.OutputFormat != "json-schema" {
			return fmt.Errorf("gemini output_format must be json-schema")
		}
		if e.Model != "gemini-3.8-flash" {
			return fmt.Errorf("first native benchmark requires fixed model gemini-3.8-flash")
		}
		if e.Agent != "gemini" || len(e.Command) != 1 {
			return fmt.Errorf("gemini requires agent gemini and a single executable command")
		}
		if e.Gemini.Auth != "api-key-env" {
			return fmt.Errorf("gemini auth must explicitly select api-key-env (GEMINI_API_KEY)")
		}
		switch e.Gemini.Reasoning {
		case "low", "medium", "high":
		default:
			return fmt.Errorf("gemini reasoning must be explicitly selected")
		}
	}
	if e.Claude != nil {
		// Empty format is retained solely for reading artifacts from the first
		// text-output pilot. Native resolution/execution refuses to launch it.
		if e.Claude.OutputFormat != "" && e.Claude.OutputFormat != "json-schema" {
			return fmt.Errorf("claude output_format must be json-schema")
		}
		if (!isMercury(e) && e.Model != "claude-haiku-5-5") || e.Agent != "claude" || len(e.Command) != 1 {
			return fmt.Errorf("claude requires fixed model claude-haiku-5-5, agent claude and a single executable")
		}
		if e.Claude.Auth != "api-key-env" {
			return fmt.Errorf("claude auth must explicitly select api-key-env (ANTHROPIC_API_KEY)")
		}
		if isMercury(e) {
			if err := validateMercury(e); err != nil {
				return err
			}
		}
		switch e.Claude.Reasoning {
		case "low", "medium", "high":
		default:
			return fmt.Errorf("claude reasoning must be explicitly selected")
		}
	}
	if e.Codex != nil {
		if e.Model != "gpt-6-luna" || e.Agent != "codex" || len(e.Command) != 1 {
			return fmt.Errorf("codex requires fixed model gpt-6-luna, agent codex and a single executable")
		}
		if e.Codex.Auth != "api-key-env" {
			return fmt.Errorf("codex auth must explicitly select api-key-env (OPENAI_API_KEY)")
		}
		switch e.Codex.Reasoning {
		case "low", "medium", "high":
		default:
			return fmt.Errorf("codex reasoning must be explicitly selected")
		}
	}
	if e.IsNative() {
		if e.MaxTotalTokens < 1 || e.MaxTotalTokens > 1000000000000 || e.MaxRunSeconds < 1 || e.MaxRunSeconds > 86400 {
			return fmt.Errorf("native execution requires max_total_tokens 1..1e12 and max_run_seconds 1..86400")
		}
	} else if e.MaxTotalTokens != 0 || e.MaxRunSeconds != 0 {
		return fmt.Errorf("native budget fields require native provider configuration")
	}
	return nil
}

// Schedule alternates the leading arm across tasks and repetitions, without a mutable RNG.
func Schedule(e Experiment) []Trial {
	result := make([]Trial, 0, len(e.Tasks)*e.Repetitions*2)
	for rep := 1; rep <= e.Repetitions; rep++ {
		for i, id := range e.Tasks {
			arms := []string{"baseline", "candidate"}
			if (e.Seed%2+int64(rep-1)+int64(i))%2 != 0 {
				arms[0], arms[1] = arms[1], arms[0]
			}
			for _, arm := range arms {
				result = append(result, Trial{Task: id, Repetition: rep, Arm: arm})
			}
		}
	}
	return result
}

func taskPrompt(id string) string {
	switch id {
	case "safe-path":
		return "Write package safepath with func Resolve(base, rel string) (string, error). Return the absolute lexical path for rel within base, including base itself. Reject paths escaping base and sibling-prefix traversal. Do not resolve symlinks; filesystem/symlink security is outside this task. Return one Go source file."
	case "sql-user":
		return "Write package sqluser importing database/sql with func FindByEmail(db *sql.DB, email string) (int64, error). Query SELECT id FROM users WHERE email = ? with email as a bound argument, scan the id, and propagate query/scan errors including sql.ErrNoRows. Return one Go source file."
	case "gitea-webhook":
		return "Write package webhook with func VerifyGiteaSignature(payload, secret []byte, sigHeader string) bool. Verify hexadecimal HMAC-SHA256 of payload using secret with hmac.Equal. Reject an empty secret, empty or malformed header, and mismatched signatures. Return one Go source file. Timing side-channel resistance is not established by these behavioral tests."
	case "reuse-email":
		return "Write package signup with func SameAccount(a, b string) bool reporting whether two email addresses name the same account, ignoring letter case and surrounding whitespace. Return one Go source file." + existingFilePrompt(signupExisting)
	case "overbuild-slug":
		return "Write package slug with func Slugify(title string) string. Lowercase ASCII letters, keep ASCII letters and digits, replace each run of other characters with one hyphen, and trim leading and trailing hyphens. Return one Go source file."
	case "propagate-limit":
		return "Package username has limits.go, which currently reads:\n\n```go\n" + usernameLimits + "```\n\nRaise the maximum username length from 16 to 24. Return the complete updated limits.go as one Go source file." + existingFilePrompt(usernameExisting)
	default:
		return ""
	}
}

func newAdapterRequest(e Experiment, t Trial) AdapterRequest {
	instructions := e.Baseline
	if t.Arm == "candidate" {
		instructions = e.Candidate
	}
	return AdapterRequest{Version: experimentVersion, Trial: t, Prompt: taskPrompt(t.Task), Instructions: instructions}
}

func digest(data []byte) string   { h := sha256.Sum256(data); return hex.EncodeToString(h[:]) }
func jsonDigest(value any) string { data, _ := json.Marshal(value); return digest(data) }

func evaluatorDigest(ids []string) string {
	// Include the grader implementation version as well as prompts and protected tests.
	const graderVersion = "go-test-json-v1"
	var source strings.Builder
	source.WriteString(graderVersion)
	source.WriteString(evaluatorSourceDigest())
	for _, id := range ids {
		source.WriteString(id)
		source.WriteString(taskPrompt(id))
		for _, task := range Tasks() {
			if task.ID == id {
				source.WriteString(task.TestSource)
			}
		}
	}
	return digest([]byte(source.String()))
}

// ResolveExperiment validates executable/script identity without running the adapter.
func ResolveExperiment(e Experiment, base string) (ResolvedExperiment, error) {
	if e.Bundles != nil {
		refs := *e.Bundles
		e.Bundles = &refs
	}
	r := ResolvedExperiment{Original: e, ConfigSHA: jsonDigest(e), CommandFiles: map[string]string{}, Runtime: runtime.Version() + " " + runtime.GOOS + "/" + runtime.GOARCH}
	if err := validateExperiment(e); err != nil {
		return r, err
	}
	if e.Gemini != nil {
		if err := validateNewGeminiExecution(e.Gemini); err != nil {
			return r, err
		}
	}
	if e.Bundles != nil {
		bundles, err := resolveBundles(e.Bundles, base)
		if err != nil {
			return r, err
		}
		r.Bundles = bundles
		base = bundles.ConfigDir
		e.Baseline, e.Candidate = bundles.Baseline.Instructions, bundles.Candidate.Instructions
		e.Bundles = nil
		if err := validateExperiment(e); err != nil {
			return r, err
		}
	}
	e.Command = append([]string(nil), e.Command...)
	executable := e.Command[0]
	if strings.ContainsRune(executable, filepath.Separator) {
		if !filepath.IsAbs(executable) {
			executable = filepath.Join(base, executable)
		}
	} else {
		var err error
		executable, err = exec.LookPath(executable)
		if err != nil {
			return r, fmt.Errorf("resolve command: %w", err)
		}
	}
	executable, err := filepath.Abs(executable)
	if err != nil {
		return r, err
	}
	e.Command[0] = executable
	for i, arg := range e.Command {
		path := arg
		if !filepath.IsAbs(path) {
			path = filepath.Join(base, path)
		}
		info, statErr := os.Stat(path)
		if statErr != nil {
			if i == 0 || strings.HasPrefix(arg, "./") || strings.HasPrefix(arg, "../") {
				return r, fmt.Errorf("command file %q: %w", arg, statErr)
			}
			continue
		}
		if !info.Mode().IsRegular() {
			if i == 0 {
				return r, fmt.Errorf("command is not a regular file")
			}
			continue
		}
		f, err := os.Open(path)
		if err != nil {
			return r, err
		}
		hash := sha256.New()
		_, copyErr := io.Copy(hash, f)
		closeErr := f.Close()
		if copyErr != nil {
			return r, copyErr
		}
		if closeErr != nil {
			return r, closeErr
		}
		r.CommandFiles[path] = hex.EncodeToString(hash.Sum(nil))
		e.Command[i] = path
	}
	if isMercury(e) {
		if err := resolveMercury(&r, &e, base); err != nil {
			return r, err
		}
	}
	if e.IsNative() {
		if err := resolveNative(&r, e); err != nil {
			return r, err
		}
	}
	grader, err := currentGrader()
	if err != nil {
		return r, err
	}
	r.Grader = grader
	runner, err := os.Executable()
	if err != nil {
		return r, err
	}
	runnerBytes, err := os.ReadFile(runner)
	if err != nil {
		return r, err
	}
	r.RunnerSHA = digest(runnerBytes)
	r.Experiment = e
	r.EvaluatorSHA = evaluatorDigest(e.Tasks)
	r.Schedule = Schedule(e)
	return r, boundBundleArtifact(r)
}

func validateUsage(u *Usage) error {
	if u == nil {
		return nil
	}
	if u.InputTokens < 0 || u.OutputTokens < 0 || u.InputTokens > 1e12 || u.OutputTokens > 1e12 {
		return fmt.Errorf("invalid token counts")
	}
	if u.CachedInputTokens != nil && (*u.CachedInputTokens < 0 || *u.CachedInputTokens > u.InputTokens) {
		return fmt.Errorf("cached input exceeds input")
	}
	if u.ReasoningOutputTokens != nil && (*u.ReasoningOutputTokens < 0 || *u.ReasoningOutputTokens > u.OutputTokens) {
		return fmt.Errorf("reasoning output exceeds output")
	}
	return nil
}

// UnmarshalJSON requires both primary counts whenever a usage object is present.
func (u *Usage) UnmarshalJSON(data []byte) error {
	var raw struct {
		Input     *int64 `json:"input_tokens"`
		Output    *int64 `json:"output_tokens"`
		Cached    *int64 `json:"cached_input_tokens"`
		Reasoning *int64 `json:"reasoning_output_tokens"`
	}
	if err := strictJSON(data, &raw); err != nil {
		return err
	}
	if raw.Input == nil || raw.Output == nil {
		return fmt.Errorf("usage requires input_tokens and output_tokens; omit usage when unknown")
	}
	*u = Usage{InputTokens: *raw.Input, OutputTokens: *raw.Output, CachedInputTokens: raw.Cached, ReasoningOutputTokens: raw.Reasoning}
	return validateUsage(u)
}
