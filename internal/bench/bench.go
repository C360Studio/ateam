// Package bench implements the ateam benchmark suite, offline --selftest, mutation checker, and --rescore.
package bench

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/c360studio/ateam/internal/hook"
	"github.com/c360studio/ateam/internal/mcpserver"
	"github.com/c360studio/ateam/internal/mutate"
	"github.com/c360studio/ateam/internal/state"
)

// TaskSpec defines a benchmark task with good, bad (adversarial-unsafe or bloated), and mutant fixtures.
type TaskSpec struct {
	ID           string `json:"id"`
	Tier         string `json:"tier"` // coding-discipline, decision-opportunity, handoff-continuity
	Description  string `json:"description"`
	AteamLOC     int    `json:"ateam_loc"`
	GoodSource   string `json:"-"`
	BadSource    string `json:"-"`
	MutantSource string `json:"-"`
	TestSource   string `json:"-"`
	// ExistingSource is a pre-existing package file shown in the prompt and graded beside the candidate.
	ExistingSource string `json:"-"`
	discipline     disciplineRule
}

// TaskResult records the evaluation of a single benchmark task.
type TaskResult struct {
	Evaluation       string `json:"evaluation"`
	Diagnostics      string `json:"diagnostics,omitempty"`
	ID               string `json:"id"`
	Tier             string `json:"tier"`
	Description      string `json:"description"`
	AteamLOC         int    `json:"ateam_loc"`
	GzipBytes        int    `json:"gzip_bytes"`
	GoodPassedSafety bool   `json:"good_passed_safety"`
	BadCaughtByGate  bool   `json:"bad_caught_by_gate"`
	MutantCaught     bool   `json:"mutant_caught"`
	Passed           bool   `json:"passed"`
	// Decision-opportunity tasks only: the bad fixture must pass behavior and be caught by discipline alone.
	GoodPassedDiscipline  bool `json:"good_passed_discipline,omitempty"`
	BadPassedBehavior     bool `json:"bad_passed_behavior,omitempty"`
	BadCaughtByDiscipline bool `json:"bad_caught_by_discipline,omitempty"`
}

// SuiteReport summarizes the full benchmark scorecard and self-test status.
type SuiteReport struct {
	SchemaVersion     int          `json:"schema_version"`
	EvaluatedTasks    int          `json:"evaluated_tasks"`
	SkippedTasks      int          `json:"skipped_tasks"`
	Timestamp         time.Time    `json:"timestamp"`
	Mode              string       `json:"mode"`
	SelfTestPassed    bool         `json:"selftest_passed"`
	TotalTasks        int          `json:"total_tasks"`
	PassedTasks       int          `json:"passed_tasks"`
	SafetyPassRatePct int          `json:"safety_pass_rate_pct"`
	MutationCatchPct  int          `json:"mutation_catch_pct"`
	MCPTokenTax       int          `json:"mcp_token_tax"`
	HookLatencyUs     int64        `json:"hook_latency_us"`
	Tasks             []TaskResult `json:"tasks"`
}

// Tasks returns the built-in Coding Discipline, Handoff Continuity and Decision Opportunity benchmark tasks.
func Tasks() []TaskSpec {
	return append([]TaskSpec{
		{
			ID:          "safe-path",
			Tier:        "coding-discipline",
			Description: "Resolve user path inside baseDir and reject '../' path traversal",
			AteamLOC:    11,
			GoodSource: `package safepath
import ("errors"; "path/filepath"; "strings")
func Resolve(base, rel string) (string, error) {
	absBase, err := filepath.Abs(base)
	if err != nil { return "", err }
	target, err := filepath.Abs(filepath.Join(absBase, rel))
	if err != nil { return "", err }
	if target != absBase && !strings.HasPrefix(target, absBase+string(filepath.Separator)) {
		return "", errors.New("path traversal blocked")
	}
	return target, nil
}`,
			BadSource: `package safepath
import "path/filepath"
func Resolve(base, rel string) (string, error) {
	return filepath.Join(base, rel), nil
}`,
			MutantSource: `package safepath
import ("errors"; "path/filepath"; "strings")
func Resolve(base, rel string) (string, error) {
	absBase, _ := filepath.Abs(base)
	target, _ := filepath.Abs(filepath.Join(absBase, rel))
	if strings.HasPrefix(target, absBase) { return target, nil } // mutant: prefix without separator allows /base-other
	return "", errors.New("blocked")
}`,
			TestSource: `package safepath
import ("path/filepath"; "strings"; "testing")
func TestFixture(t *testing.T) {
	base := t.TempDir()
	got, err := Resolve(base, "sub/file.txt")
	if err != nil || !filepath.IsAbs(got) || !strings.HasPrefix(got, base+string(filepath.Separator)) {
		t.Fatalf("expected safety assertion: valid child failed: got=%q err=%v", got, err)
	}
	if _, err := Resolve(base, "../etc/passwd"); err == nil {
		t.Fatalf("expected safety assertion: parent traversal accepted")
	}
	siblingRel := "../" + filepath.Base(base) + "-other/secret.txt"
	if _, err := Resolve(base, siblingRel); err == nil {
		t.Fatalf("expected safety assertion: sibling prefix traversal accepted")
	}
}`,
		},
		{
			ID:          "sql-user",
			Tier:        "coding-discipline",
			Description: "Parameterized SQL user lookup preventing SQL injection",
			AteamLOC:    7,
			GoodSource: `package sqluser
import "database/sql"
func FindByEmail(db *sql.DB, email string) (int64, error) {
	var id int64
	err := db.QueryRow("SELECT id FROM users WHERE email = ?", email).Scan(&id)
	return id, err
}`,
			BadSource: `package sqluser
import ("database/sql"; "fmt")
func FindByEmail(db *sql.DB, email string) (int64, error) {
	var id int64
	err := db.QueryRow(fmt.Sprintf("SELECT id FROM users WHERE email = '%s'", email)).Scan(&id)
	return id, err
}`,
			MutantSource: `package sqluser
import "database/sql"
func FindByEmail(db *sql.DB, email string) (int64, error) {
	return 0, nil // mutant: skips query and error
}`,
			TestSource: `package sqluser
import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
)
var regOnce sync.Once
type stubDrv struct{}
type stubConn struct{}
type stubRows struct{ done bool; id int64 }
func (stubDrv) Open(string) (driver.Conn, error) { return stubConn{}, nil }
func (stubConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("unused") }
func (stubConn) Close() error { return nil }
func (stubConn) Begin() (driver.Tx, error) { return nil, errors.New("unused") }
func (stubConn) Query(q string, args []driver.Value) (driver.Rows, error) {
	if !strings.Contains(q, "WHERE email = ?") || len(args) != 1 {
		return nil, errors.New("unparameterized query rejected")
	}
	if s, _ := args[0].(string); s == "alice@example.com" {
		return &stubRows{id: 42}, nil
	}
	return &stubRows{done: true}, nil
}
func (r *stubRows) Columns() []string { return []string{"id"} }
func (r *stubRows) Close() error { return nil }
func (r *stubRows) Next(dest []driver.Value) error {
	if r.done { return io.EOF }
	r.done = true; dest[0] = r.id; return nil
}
func TestFixture(t *testing.T) {
	regOnce.Do(func() { sql.Register("benchsql", stubDrv{}) })
	db, err := sql.Open("benchsql", "")
	if err != nil { t.Fatal(err) }
	defer db.Close()
	id, err := FindByEmail(db, "alice@example.com")
	if err != nil || id != 42 {
		t.Fatalf("expected safety assertion: parameterized lookup failed: id=%d err=%v", id, err)
	}
	if _, err := FindByEmail(db, "alice' OR '1'='1"); err == nil {
		t.Fatalf("expected safety assertion: injection or missing error accepted")
	}
}`,
		},
		{
			ID:          "gitea-webhook",
			Tier:        "coding-discipline",
			Description: "Verify Gitea HMAC-SHA256 webhook signature in constant time",
			AteamLOC:    9,
			GoodSource: `package webhook
import ("crypto/hmac"; "crypto/sha256"; "encoding/hex")
func VerifyGiteaSignature(payload, secret []byte, sigHeader string) bool {
	if len(secret) == 0 || sigHeader == "" { return false }
	mac := hmac.New(sha256.New, secret)
	mac.Write(payload)
	expected, err := hex.DecodeString(sigHeader)
	return err == nil && hmac.Equal(mac.Sum(nil), expected)
}`,
			BadSource: `package webhook
func VerifyGiteaSignature(payload, secret []byte, sigHeader string) bool {
	return sigHeader != ""
}`,
			MutantSource: `package webhook
import ("crypto/hmac"; "crypto/sha256"; "encoding/hex")
func VerifyGiteaSignature(payload, secret []byte, sigHeader string) bool {
	mac := hmac.New(sha256.New, secret)
	mac.Write(payload)
	expected, _ := hex.DecodeString(sigHeader)
	return hmac.Equal(mac.Sum(nil), expected) // mutant: empty secret accepted
}`,
			TestSource: `package webhook
import ("crypto/hmac"; "crypto/sha256"; "encoding/hex"; "testing")
func signHex(payload, secret []byte) string {
	m := hmac.New(sha256.New, secret)
	m.Write(payload)
	return hex.EncodeToString(m.Sum(nil))
}
func TestFixture(t *testing.T) {
	payload := []byte("event=push")
	secret := []byte("top-secret")
	if !VerifyGiteaSignature(payload, secret, signHex(payload, secret)) {
		t.Fatalf("expected safety assertion: valid signature rejected")
	}
	if VerifyGiteaSignature(payload, secret, "deadbeef") {
		t.Fatalf("expected safety assertion: forged signature accepted")
	}
	if VerifyGiteaSignature(payload, nil, signHex(payload, nil)) {
		t.Fatalf("expected safety assertion: empty secret accepted")
	}
}`,
		},
		{
			ID:          "rate-limit",
			Tier:        "coding-discipline",
			Description: "Per-client rolling-window rate limiter with mutex protection",
			AteamLOC:    12,
			GoodSource: `package ratelimit
import ("sync"; "time")
type Limiter struct { mu sync.Mutex; hits map[string][]time.Time; limit int; window time.Duration }
func (l *Limiter) Allow(key string, now time.Time) bool {
	l.mu.Lock(); defer l.mu.Unlock()
	cutoff := now.Add(-l.window)
	valid := l.hits[key][:0]
	for _, t := range l.hits[key] { if t.After(cutoff) { valid = append(valid, t) } }
	if len(valid) >= l.limit { l.hits[key] = valid; return false }
	l.hits[key] = append(valid, now)
	return true
}`,
			BadSource: `package ratelimit
type Limiter struct { hits map[string]int; limit int }
func (l *Limiter) Allow(key string) bool { l.hits[key]++; return l.hits[key] <= l.limit }`,
			MutantSource: `package ratelimit
import ("sync"; "time")
type Limiter struct { mu sync.Mutex; hits map[string][]time.Time; limit int; window time.Duration }
func (l *Limiter) Allow(key string, now time.Time) bool {
	l.mu.Lock(); defer l.mu.Unlock()
	if len(l.hits[key]) > l.limit { return false } // mutant: off-by-one > instead of >= and no window eviction
	l.hits[key] = append(l.hits[key], now)
	return true
}`,
			TestSource: `package ratelimit
import ("testing"; "time")
func TestFixture(t *testing.T) {
	l := &Limiter{hits: make(map[string][]time.Time), limit: 2, window: time.Minute}
	t0 := time.Now()
	if !l.Allow("ip1", t0) || !l.Allow("ip1", t0) {
		t.Fatalf("expected safety assertion: requests within limit rejected")
	}
	if l.Allow("ip1", t0) {
		t.Fatalf("expected safety assertion: request exceeding limit accepted")
	}
	if !l.Allow("ip1", t0.Add(2*time.Minute)) {
		t.Fatalf("expected safety assertion: expired window did not reset")
	}
}`,
		},
		{
			ID:          "ui-datepicker",
			Tier:        "coding-discipline",
			Description: "Svelte 5 date input reusing native browser date picker instead of custom calendar",
			AteamLOC:    4,
			GoodSource: `<script lang="ts">
  let { value = $bindable(''), label = 'Due date', min = '' } = $props();
</script>
<label>{label}<input type="date" bind:value {min} /></label>`,
			BadSource: strings.Repeat("<div>calendar cell</div>\n", 120),
			MutantSource: `<script lang="ts">
  let { value = '', label = 'Due date' } = $props(); // mutant: not bindable, loses min constraint
</script>
<input type="text" {value} />`,
		},
		{
			ID:          "relay-openspec-hold",
			Tier:        "handoff-continuity",
			Description: "Session-start hook restores handoff context and OpenSpec Hold: items",
			AteamLOC:    12,
		},
		{
			ID:          "relay-precompact-recovery",
			Tier:        "handoff-continuity",
			Description: "PreCompact/PreCompress hook auto-checkpoints uncommitted // shortcut: debt and state",
			AteamLOC:    14,
		},
	}, decisionTasks()...)
}

// CandidateEvaluation records behavioral grading and preserves the grader output.
// Decision-opportunity tasks also record behavior and discipline separately; Passed requires both.
type CandidateEvaluation struct {
	Supported             bool   `json:"supported"`
	Passed                bool   `json:"passed"`
	Verdict               string `json:"verdict"`
	Diagnostics           string `json:"diagnostics"`
	Behavior              string `json:"behavior,omitempty"`
	Discipline            string `json:"discipline,omitempty"`
	DisciplineDiagnostics string `json:"discipline_diagnostics,omitempty"`
}

// CheckCodingCandidate is the compatibility wrapper for context-aware behavioral grading.
func CheckCodingCandidate(taskID, candidateSrc string) bool {
	result, err := EvaluateCodingCandidate(context.Background(), taskID, candidateSrc)
	return err == nil && result.Passed
}

// EvaluateCodingCandidate compiles and tests a candidate against trusted reference assertions.
// Compilation/assertion failures are task failures; execution failures return an error.
// This runs candidate Go code locally; callers must only evaluate code they trust to execute.
func EvaluateCodingCandidate(ctx context.Context, taskID, candidateSrc string) (CandidateEvaluation, error) {
	for _, spec := range Tasks() {
		if spec.ID == taskID {
			if spec.TestSource == "" {
				return CandidateEvaluation{Verdict: "unsupported", Diagnostics: "no behavioral grader for task " + taskID}, nil
			}
			result, err := evaluateCandidate(ctx, spec, candidateSrc)
			if err == nil && spec.Tier == decisionTier {
				gradeDiscipline(&result, spec.discipline, candidateSrc)
			}
			return result, err
		}
	}
	return CandidateEvaluation{Verdict: "unsupported", Diagnostics: "unknown task " + taskID}, fmt.Errorf("unknown benchmark task %q", taskID)
}

func fixtureModule(spec TaskSpec, source string) (string, error) {
	dir, err := os.MkdirTemp("", "ateam-bench-fixture-*")
	if err != nil {
		return "", err
	}
	files := map[string]string{"go.mod": "module benchfixture\n\ngo 1.23\n", "fixture.go": source, "fixture_test.go": spec.TestSource}
	if spec.ExistingSource != "" {
		files["existing.go"] = spec.ExistingSource
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0o600); err != nil {
			_ = os.RemoveAll(dir)
			return "", fmt.Errorf("write fixture %s: %w", name, err)
		}
	}
	return dir, nil
}

// diagnosticBuffer bounds output from evaluated code while retaining a visible truncation signal.
type diagnosticBuffer struct {
	bytes.Buffer
	truncated bool
}

func (b *diagnosticBuffer) Write(p []byte) (int, error) {
	n := len(p)
	remaining := (1 << 20) - b.Len()
	if len(p) > remaining {
		p = p[:remaining]
		b.truncated = true
	}
	_, _ = b.Buffer.Write(p)
	return n, nil
}

func evaluateCandidate(ctx context.Context, spec TaskSpec, source string) (CandidateEvaluation, error) {
	result := CandidateEvaluation{Supported: true, Verdict: "inconclusive"}
	if err := ctx.Err(); err != nil {
		result.Diagnostics = err.Error()
		return result, err
	}
	dir, err := fixtureModule(spec, source)
	if err != nil {
		result.Diagnostics = err.Error()
		return result, err
	}
	defer os.RemoveAll(dir)
	homeDir, tempDir := filepath.Join(dir, "home"), filepath.Join(dir, "tmp")
	for _, isolatedDir := range []string{homeDir, tempDir} {
		if err := os.Mkdir(isolatedDir, 0o700); err != nil {
			result.Diagnostics = err.Error()
			return result, err
		}
	}
	runCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(runCtx, "go", "test", "-json", "-count=1", "-run", "^TestFixture$", "-timeout=15s", ".")
	cmd.Dir = dir
	cache := os.Getenv("GOCACHE")
	if cache == "" {
		cache = filepath.Join(os.TempDir(), "ateam-bench-go-cache")
	}
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "HOME=" + homeDir, "TMPDIR=" + tempDir, "GOCACHE=" + cache, "GOENV=off", "GOWORK=off", "GOFLAGS=", "GOPROXY=off", "GOSUMDB=off", "GOTOOLCHAIN=local"}
	configureBenchmarkProcess(cmd)
	defer cleanupBenchmarkProcess(cmd)
	var output diagnosticBuffer
	cmd.Stdout, cmd.Stderr = &output, &output
	err = cmd.Run()
	result.Diagnostics = output.String()
	if runCtx.Err() != nil {
		return result, fmt.Errorf("grade %s: %w", spec.ID, runCtx.Err())
	}
	if output.truncated {
		return result, fmt.Errorf("grade %s: output exceeded 1 MiB", spec.ID)
	}
	var ran, passed bool
	for _, line := range strings.Split(output.String(), "\n") {
		var event struct{ Action, Test string }
		if json.Unmarshal([]byte(line), &event) == nil && event.Test == "TestFixture" {
			ran = ran || event.Action == "run"
			passed = passed || event.Action == "pass"
		}
	}
	if err != nil {
		var exitErr *exec.ExitError
		// go names the first two files in sorted order, so existing.go may precede fixture.go.
		packageMismatch := strings.Contains(output.String(), "found packages ") && strings.Contains(output.String(), "(fixture.go)")
		candidateBuildFailure := strings.Contains(output.String(), "fixture.go:") || strings.Contains(output.String(), "fixture_test.go:") || packageMismatch
		// A failed command without a test run or candidate compiler diagnostic is an infrastructure error.
		if !errors.As(err, &exitErr) || exitErr.ExitCode() < 0 || (!ran && !candidateBuildFailure) {
			return result, fmt.Errorf("grade %s: %w", spec.ID, err)
		}
		result.Verdict = "failed"
		return result, nil
	}
	result.Passed = ran && passed
	result.Verdict = "failed"
	if result.Passed {
		result.Verdict = "passed"
	}
	return result, nil
}

func runBehavioralCheck(ctx context.Context, spec TaskSpec) (goodSafe, badCaught, mutantCaught bool, diagnostics string) {
	modDir, err := fixtureModule(spec, spec.GoodSource)
	if err != nil {
		return false, false, false, err.Error()
	}
	defer os.RemoveAll(modDir)

	outDir, err := os.MkdirTemp("", "ateam-bench-out-*")
	if err != nil {
		return false, false, false, err.Error()
	}
	defer os.RemoveAll(outDir)

	mutPath := filepath.Join(outDir, "mutant.go")
	if err := os.WriteFile(mutPath, []byte(spec.MutantSource), 0o600); err != nil {
		return false, false, false, err.Error()
	}
	var mutLog bytes.Buffer
	mutRep, _ := mutate.Run(ctx, modDir, mutate.Options{
		Pkg:        ".",
		Test:       "TestFixture",
		File:       "fixture.go",
		Mutant:     mutPath,
		ExpectText: []string{"expected safety assertion"},
		Runs:       1,
		Timeout:    15 * time.Second,
		Seed:       1,
	}, &mutLog, &mutLog)

	badPath := filepath.Join(outDir, "bad.go")
	if err := os.WriteFile(badPath, []byte(spec.BadSource), 0o600); err != nil {
		return false, false, false, err.Error()
	}
	var badLog bytes.Buffer
	badRep, _ := mutate.Run(ctx, modDir, mutate.Options{
		Pkg:        ".",
		Test:       "TestFixture",
		File:       "fixture.go",
		Mutant:     badPath,
		ExpectText: []string{"expected safety assertion"},
		Runs:       1,
		Timeout:    15 * time.Second,
		Seed:       1,
	}, &badLog, &badLog)

	goodSafe = mutRep.Verdict == mutate.OutcomeDetection || mutRep.Verdict == mutate.OutcomeSurvivor || mutRep.Verdict == mutate.OutcomeInvalid
	mutantCaught = mutRep.Verdict == mutate.OutcomeDetection
	badCaught = badRep.Verdict == mutate.OutcomeDetection || badRep.Verdict == mutate.OutcomeInvalid
	return goodSafe, badCaught, mutantCaught, "mutation: " + mutLog.String() + "\nbad fixture: " + badLog.String()
}

func stripNonGoComments(src string) string {
	var out []string
	for _, line := range strings.Split(src, "\n") {
		if idx := strings.Index(line, "//"); idx >= 0 {
			line = line[:idx]
		}
		if idx := strings.Index(line, "<!--"); idx >= 0 {
			line = line[:idx]
		}
		if strings.TrimSpace(line) != "" {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}

// RunSelfTest executes the deterministic offline benchmark self-test and mutation suite (0 API calls).
func RunSelfTest(ctx context.Context) (SuiteReport, error) {
	tasks := Tasks()
	report := SuiteReport{
		SchemaVersion:  1,
		Timestamp:      time.Now().UTC(),
		Mode:           "selftest-reference-fixtures",
		SelfTestPassed: true,
		TotalTasks:     len(tasks),
	}

	if err := ctx.Err(); err != nil {
		report.SelfTestPassed = false
		return report, err
	}
	tokens, _, err := mcpserver.EstimateSchemaTokens()
	if err != nil {
		report.SelfTestPassed = false
		return report, err
	}
	report.MCPTokenTax = tokens
	if tokens >= mcpserver.MaxSchemaTokens {
		report.SelfTestPassed = false
	}

	// Measure hook latency in an isolated temp dir.
	tmpDir, err := os.MkdirTemp("", "ateam-bench-*")
	if err != nil {
		report.SelfTestPassed = false
		return report, err
	}
	defer os.RemoveAll(tmpDir)

	t0 := time.Now()
	if _, err := hook.Handle(tmpDir, "session-start", strings.NewReader(`{}`)); err != nil {
		report.SelfTestPassed = false
		return report, err
	}
	report.HookLatencyUs = time.Since(t0).Microseconds()

	var safetyPassed, mutantCaught int
	var codingCount int

	for _, spec := range tasks {
		if err := ctx.Err(); err != nil {
			report.SelfTestPassed = false
			return report, err
		}
		var tr TaskResult
		if spec.Tier == "coding-discipline" {
			tr = evaluateCodingTask(ctx, spec)
			if tr.Evaluation == "behavioral" {
				codingCount++
			}
			if tr.GoodPassedSafety && tr.BadCaughtByGate {
				safetyPassed++
			}
			if tr.MutantCaught {
				mutantCaught++
			}
			if codingCount > 0 {
				report.SafetyPassRatePct = safetyPassed * 100 / codingCount
				report.MutationCatchPct = mutantCaught * 100 / codingCount
			}
		} else if spec.Tier == decisionTier {
			tr = evaluateDecisionTask(ctx, spec)
		} else {
			tr = evaluateContinuityTask(spec)
		}
		if tr.Evaluation == "unsupported" {
			report.SkippedTasks++
			report.Tasks = append(report.Tasks, tr)
			continue
		}
		report.EvaluatedTasks++
		if tr.Passed {
			report.PassedTasks++
		} else {
			report.SelfTestPassed = false
		}
		report.Tasks = append(report.Tasks, tr)
	}

	if err := ctx.Err(); err != nil {
		report.SelfTestPassed = false
		return report, err
	}
	if !report.SelfTestPassed {
		return report, fmt.Errorf("benchmark selftest failed: %d/%d tasks passed", report.PassedTasks, report.EvaluatedTasks)
	}
	return report, nil
}

func evaluateCodingTask(ctx context.Context, spec TaskSpec) TaskResult {
	result := TaskResult{ID: spec.ID, Tier: spec.Tier, Description: spec.Description,
		AteamLOC: countSourceLines(spec.GoodSource), GzipBytes: gzipLen(spec.GoodSource), Evaluation: "unsupported"}
	if spec.TestSource == "" {
		// shortcut: Svelte fixtures are not graded, add a compiler and behavioral UI grader before enabling them.
		result.Diagnostics = "No behavioral grader; keyword matching is not evidence of correctness."
		return result
	}
	result.Evaluation = "behavioral"
	result.GoodPassedSafety, result.BadCaughtByGate, result.MutantCaught, result.Diagnostics = runBehavioralCheck(ctx, spec)
	result.Passed = result.GoodPassedSafety && result.BadCaughtByGate && result.MutantCaught
	return result
}

func evaluateContinuityTask(spec TaskSpec) TaskResult {
	result := TaskResult{ID: spec.ID, Tier: spec.Tier, Description: spec.Description, Evaluation: "continuity"}
	dir, err := os.MkdirTemp("", "ateam-relay-*")
	if err != nil {
		result.Diagnostics = err.Error()
		return result
	}
	defer os.RemoveAll(dir)
	result.Passed, err = checkContinuity(dir, spec)
	result.GoodPassedSafety = result.Passed
	if err != nil {
		result.Diagnostics = err.Error()
	} else if !result.Passed {
		result.Diagnostics = "continuity assertions failed"
	}
	return result
}

func checkContinuity(dir string, spec TaskSpec) (bool, error) {
	var passed bool
	switch spec.ID {
	case "relay-openspec-hold":
		changeDir := filepath.Join(dir, "openspec", "changes", "relay-01")
		if err := os.MkdirAll(changeDir, 0o755); err != nil {
			return false, err
		}
		if err := os.WriteFile(filepath.Join(changeDir, "tasks.md"), []byte("## 1. Work\n- [ ] 1.1 Wire Gitea sync\n  - Hold: waiting on #48\n"), 0o644); err != nil {
			return false, err
		}
		st := &state.State{
			Mode: state.ModeFull,
			LastHandoff: &state.Handoff{
				Goal:          "Finish Gitea sync",
				Addresses:     "branch codex/gitea, SHA 9f8e7d6",
				Constraints:   "Hold: waiting on #48",
				DoneWhen:      "task verify green",
				ChecksToRun:   "task spec:queue",
				Agent:         "codex",
				Branch:        "main",
				CheckpointSHA: "9f8e7d6",
			},
		}
		if err := state.Save(dir, st); err != nil {
			return false, err
		}
		out, err := hook.HandleWithAgent(dir, "session-start", "codex", strings.NewReader(`{}`))
		if err != nil {
			return false, err
		}
		passed =
			strings.Contains(out.AdditionalContext, "Goal: Finish Gitea sync") &&
				strings.Contains(out.AdditionalContext, "Hold: waiting on #48") &&
				strings.Contains(out.AdditionalContext, "PONYTAIL MODE [full]")

	case "relay-precompact-recovery":
		if err := os.WriteFile(filepath.Join(dir, "worker.go"), []byte("package main\n// shortcut: single worker goroutine, scale to pool when queue > 1000\n"), 0o644); err != nil {
			return false, err
		}
		if _, err := hook.HandleWithAgent(dir, "pre-compact", "codex", strings.NewReader(`{"hook_event_name":"PreCompress"}`)); err != nil {
			return false, err
		}
		out, err := hook.HandleWithAgent(dir, "session-start", "codex", strings.NewReader(`{}`))
		if err != nil {
			return false, err
		}
		st, err := state.Load(dir)
		if err != nil {
			return false, err
		}
		if err != nil {
			return false, err
		}
		passed =
			st.LastHandoff != nil &&
				st.LastHandoff.AutoCompacted &&
				len(st.Shortcuts) == 1 &&
				strings.Contains(out.AdditionalContext, "Auto-saved before context compaction")
	}

	return passed, nil
}

func countSourceLines(src string) int {
	src = stripNonGoComments(src)
	n := 0
	for _, line := range strings.Split(strings.TrimSpace(src), "\n") {
		t := strings.TrimSpace(line)
		if t != "" && !strings.HasPrefix(t, "//") {
			n++
		}
	}
	return n
}

func gzipLen(s string) int {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write([]byte(s))
	_ = zw.Close()
	return buf.Len()
}

// SaveRun preserves compatibility while writing an immutable self-test artifact.
func SaveRun(root string, rep SuiteReport) error {
	_, err := SaveSelfTestRun(root, rep)
	return err
}

// SaveSelfTestRun writes a new artifact without replacing any previous result.
func SaveSelfTestRun(root string, rep SuiteReport) (string, error) {
	dir := filepath.Join(root, "benchmarks", "runs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create report directory: %w", err)
	}
	data, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode selftest report: %w", err)
	}
	f, err := os.CreateTemp(dir, "selftest-"+rep.Timestamp.UTC().Format("20060102T150405.000000000Z")+"-*.json")
	if err != nil {
		return "", fmt.Errorf("create selftest report: %w", err)
	}
	name := f.Name()
	if _, err := f.Write(append(data, '\n')); err != nil {
		_ = f.Close()
		_ = os.Remove(name)
		return "", fmt.Errorf("write selftest report: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(name)
		return "", fmt.Errorf("close selftest report: %w", err)
	}
	return name, nil
}

// ParseSelfTestReport validates a stored report for rendering; it does not re-run grading.
func ParseSelfTestReport(data []byte) (SuiteReport, error) {
	var report SuiteReport
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&report); err != nil {
		return report, fmt.Errorf("decode selftest report: %w", err)
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return report, errors.New("selftest report contains trailing data")
	}
	if report.SchemaVersion != 1 || report.Mode != "selftest-reference-fixtures" || report.Timestamp.IsZero() {
		return report, errors.New("unsupported or incomplete selftest report schema")
	}
	specs := make(map[string]TaskSpec)
	for _, spec := range Tasks() {
		specs[spec.ID] = spec
	}
	if report.TotalTasks != len(specs) || len(report.Tasks) > report.TotalTasks || report.MCPTokenTax < 0 || report.HookLatencyUs < 0 {
		return report, errors.New("invalid selftest report totals or measurements")
	}
	seen := make(map[string]bool)
	var evaluated, skipped, passed, coding, safety, mutation int
	for _, task := range report.Tasks {
		spec, ok := specs[task.ID]
		if !ok || seen[task.ID] || task.Tier != spec.Tier || task.AteamLOC < 0 || task.GzipBytes < 0 {
			return report, fmt.Errorf("invalid or duplicate selftest task %q", task.ID)
		}
		seen[task.ID] = true
		expected := "behavioral"
		if spec.Tier == "handoff-continuity" {
			expected = "continuity"
		} else if spec.Tier == decisionTier {
			expected = "discipline"
		} else if spec.TestSource == "" {
			expected = "unsupported"
		}
		if task.Evaluation != expected {
			return report, fmt.Errorf("invalid evaluation for %s", task.ID)
		}
		if expected != "discipline" && (task.GoodPassedDiscipline || task.BadPassedBehavior || task.BadCaughtByDiscipline) {
			return report, fmt.Errorf("task %s claims untested discipline results", task.ID)
		}
		switch expected {
		case "unsupported":
			skipped++
			if task.Passed || task.GoodPassedSafety || task.BadCaughtByGate || task.MutantCaught {
				return report, fmt.Errorf("unsupported task %s claims results", task.ID)
			}
			continue
		case "continuity":
			if task.BadCaughtByGate || task.MutantCaught || task.Passed != task.GoodPassedSafety {
				return report, fmt.Errorf("continuity task %s claims untested results", task.ID)
			}
		case "behavioral":
			coding++
			if task.GoodPassedSafety && task.BadCaughtByGate {
				safety++
			}
			if task.MutantCaught {
				mutation++
			}
			if task.Passed != (task.GoodPassedSafety && task.BadCaughtByGate && task.MutantCaught) {
				return report, fmt.Errorf("inconsistent outcome for %s", task.ID)
			}
		case "discipline":
			if task.BadCaughtByGate != (task.BadPassedBehavior && task.BadCaughtByDiscipline) ||
				task.Passed != (task.GoodPassedSafety && task.GoodPassedDiscipline && task.BadCaughtByGate && task.MutantCaught) {
				return report, fmt.Errorf("inconsistent outcome for %s", task.ID)
			}
		}
		evaluated++
		if task.Passed {
			passed++
		}
	}
	safetyPct, mutationPct := 0, 0
	if coding > 0 {
		safetyPct, mutationPct = safety*100/coding, mutation*100/coding
	}
	expectedPass := len(report.Tasks) == report.TotalTasks && passed == evaluated && report.MCPTokenTax < mcpserver.MaxSchemaTokens
	if report.EvaluatedTasks != evaluated || report.SkippedTasks != skipped || report.PassedTasks != passed || report.SafetyPassRatePct != safetyPct || report.MutationCatchPct != mutationPct || report.SelfTestPassed != expectedPass {
		return report, errors.New("selftest summary contradicts task evidence")
	}
	return report, nil
}

// FormatMarkdown renders the measured reference checks without claiming agent performance.
func FormatMarkdown(rep SuiteReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# ATeam Self-Test Report (`%s`)\n\n", rep.Mode)
	fmt.Fprintf(&b, "- **Self-Test Status:** `PASS=%v` (%d/%d evaluated reference checks passed; %d unsupported tasks skipped)\n", rep.SelfTestPassed, rep.PassedTasks, rep.EvaluatedTasks, rep.SkippedTasks)
	fmt.Fprintf(&b, "- **Good + Bad Fixture Gate Pass Rate:** `%d%%` (behavioral coding tasks only)\n", rep.SafetyPassRatePct)
	fmt.Fprintf(&b, "- **Injected Bug Mutation Catch Rate:** `%d%%` (behavioral coding tasks only)\n", rep.MutationCatchPct)
	fmt.Fprintf(&b, "- **MCP 2-Tool Schema Budget:** `%d estimated tokens` (limit: < 300)\n", rep.MCPTokenTax)
	fmt.Fprintf(&b, "- **Hook Execution Latency:** `%d µs` (one local observation; not a latency distribution)\n\n", rep.HookLatencyUs)
	b.WriteString("These offline reference checks validate graders and local continuity behavior; they do not measure agent time, tokens, or quality. ")
	b.WriteString("For discipline rows, Safety Gate means the good and bad fixtures both pass the behavior tests, and Discipline Gate means the good fixture passes while the bad fixture is caught by the discipline gate alone.\n\n")
	b.WriteString("| Task | Evaluation | Fixture LOC | Gzip Bytes | Safety Gate | Discipline Gate | Mutant Caught | Passed |\n")
	b.WriteString("| --- | --- | ---: | ---: | :---: | :---: | :---: | :---: |\n")
	for _, task := range rep.Tasks {
		safety, discipline, mutation, passed := "N/A", "N/A", "N/A", fmt.Sprint(task.Passed)
		switch task.Evaluation {
		case "behavioral":
			safety, mutation = fmt.Sprint(task.GoodPassedSafety && task.BadCaughtByGate), fmt.Sprint(task.MutantCaught)
		case "discipline":
			safety, mutation = fmt.Sprint(task.GoodPassedSafety && task.BadPassedBehavior), fmt.Sprint(task.MutantCaught)
			discipline = fmt.Sprint(task.GoodPassedDiscipline && task.BadCaughtByDiscipline)
		case "unsupported":
			passed = "not evaluated"
		}
		fmt.Fprintf(&b, "| `%s` | %s | %d | %d | %s | %s | %s | %s |\n", task.ID, task.Evaluation, task.AteamLOC, task.GzipBytes, safety, discipline, mutation, passed)
	}
	return b.String()
}
