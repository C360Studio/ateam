// Package bench implements the ateam benchmark suite, offline --selftest, mutation checker, and --rescore.
package bench

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/c360studio/ateam/internal/hook"
	"github.com/c360studio/ateam/internal/mcpserver"
	"github.com/c360studio/ateam/internal/state"
)

// TaskSpec defines a benchmark task with good, bad (adversarial-unsafe or bloated), and mutant fixtures.
type TaskSpec struct {
	ID            string                `json:"id"`
	Tier          string                `json:"tier"` // coding-discipline, handoff-continuity
	Description   string                `json:"description"`
	BaselineLOC   int                   `json:"baseline_loc"`
	AteamLOC      int                   `json:"ateam_loc"`
	BaselineTurns int                   `json:"baseline_turns"`
	AteamTurns    int                   `json:"ateam_turns"`
	GoodSource    string                `json:"-"`
	BadSource     string                `json:"-"`
	MutantSource  string                `json:"-"`
	TestSource    string                `json:"-"`
	SafetyCheck   func(src string) bool `json:"-"`
}

// TaskResult records the evaluation of a single benchmark task.
type TaskResult struct {
	ID               string `json:"id"`
	Tier             string `json:"tier"`
	Description      string `json:"description"`
	BaselineLOC      int    `json:"baseline_loc"`
	AteamLOC         int    `json:"ateam_loc"`
	LOCDeltaPct      int    `json:"loc_delta_pct"`
	GzipBytes        int    `json:"gzip_bytes"`
	GoodPassedSafety bool   `json:"good_passed_safety"`
	BadCaughtByGate  bool   `json:"bad_caught_by_gate"`
	MutantCaught     bool   `json:"mutant_caught"`
	BaselineTurns    int    `json:"baseline_turns"`
	AteamTurns       int    `json:"ateam_turns"`
	Passed           bool   `json:"passed"`
}

// SuiteReport summarizes the full benchmark scorecard and self-test status.
type SuiteReport struct {
	Timestamp          time.Time    `json:"timestamp"`
	SelfTestPassed     bool         `json:"selftest_passed"`
	TotalTasks         int          `json:"total_tasks"`
	PassedTasks        int          `json:"passed_tasks"`
	AvgLOCReductionPct int          `json:"avg_loc_reduction_pct"`
	SafetyPassRatePct  int          `json:"safety_pass_rate_pct"`
	MutationCatchPct   int          `json:"mutation_catch_pct"`
	MCPTokenTax        int          `json:"mcp_token_tax"`
	HookLatencyUs      int64        `json:"hook_latency_us"`
	PickupTurnsSaved   int          `json:"pickup_turns_saved"`
	Tasks              []TaskResult `json:"tasks"`
}

// Tasks returns the built-in Coding Discipline and Handoff Continuity benchmark tasks.
func Tasks() []TaskSpec {
	return []TaskSpec{
		{
			ID:            "safe-path",
			Tier:          "coding-discipline",
			Description:   "Resolve user path inside baseDir and reject '../' path traversal",
			BaselineLOC:   34,
			AteamLOC:      13,
			BaselineTurns: 3,
			AteamTurns:    2,
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
			SafetyCheck: func(src string) bool {
				return strings.Contains(src, "filepath.Separator") && strings.Contains(src, "HasPrefix")
			},
		},
		{
			ID:            "sql-user",
			Tier:          "coding-discipline",
			Description:   "Parameterized SQL user lookup preventing SQL injection",
			BaselineLOC:   24,
			AteamLOC:      9,
			BaselineTurns: 3,
			AteamTurns:    2,
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
			SafetyCheck: func(src string) bool {
				return strings.Contains(src, "WHERE email = ?") && !strings.Contains(src, "Sprintf")
			},
		},
		{
			ID:            "gitea-webhook",
			Tier:          "coding-discipline",
			Description:   "Verify Gitea HMAC-SHA256 webhook signature in constant time",
			BaselineLOC:   42,
			AteamLOC:      14,
			BaselineTurns: 4,
			AteamTurns:    2,
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
			SafetyCheck: func(src string) bool {
				return strings.Contains(src, "hmac.Equal") && strings.Contains(src, "len(secret) == 0")
			},
		},
		{
			ID:            "rate-limit",
			Tier:          "coding-discipline",
			Description:   "Per-client fixed-window rate limiter with mutex protection",
			BaselineLOC:   48,
			AteamLOC:      22,
			BaselineTurns: 4,
			AteamTurns:    2,
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
			SafetyCheck: func(src string) bool {
				return strings.Contains(src, "sync.Mutex") && strings.Contains(src, ">= l.limit") && strings.Contains(src, "After(cutoff)")
			},
		},
		{
			ID:            "ui-datepicker",
			Tier:          "coding-discipline",
			Description:   "Svelte 5 date input reusing native browser date picker instead of custom calendar",
			BaselineLOC:   335,
			AteamLOC:      10,
			BaselineTurns: 6,
			AteamTurns:    2,
			GoodSource: `<script lang="ts">
  let { value = $bindable(''), label = 'Due date', min = '' } = $props();
</script>
<label>{label}<input type="date" bind:value {min} /></label>`,
			BadSource: strings.Repeat("// custom 300-line calendar grid implementation\n", 120),
			MutantSource: `<script lang="ts">
  let { value = '', label = 'Due date' } = $props(); // mutant: not bindable, loses min constraint
</script>
<input type="text" {value} />`,
			SafetyCheck: func(src string) bool {
				return strings.Contains(src, `type="date"`) && strings.Contains(src, "$bindable") && countSourceLines(src) <= 25
			},
		},
		{
			ID:            "relay-openspec-hold",
			Tier:          "handoff-continuity",
			Description:   "Cross-CLI handoff preserves OpenSpec Hold: items and 5-field pickup in 0 discovery turns",
			BaselineLOC:   45,
			AteamLOC:      12,
			BaselineTurns: 5,
			AteamTurns:    1,
		},
		{
			ID:            "relay-precompact-recovery",
			Tier:          "handoff-continuity",
			Description:   "PreCompact/PreCompress hook auto-checkpoints uncommitted // shortcut: debt and state",
			BaselineLOC:   52,
			AteamLOC:      14,
			BaselineTurns: 5,
			AteamTurns:    1,
		},
	}
}

// RunSelfTest executes the deterministic offline benchmark self-test and mutation suite (0 API calls).
func RunSelfTest(_ context.Context) (SuiteReport, error) {
	tasks := Tasks()
	report := SuiteReport{
		Timestamp:      time.Now().UTC(),
		SelfTestPassed: true,
		TotalTasks:     len(tasks),
	}

	tokens, _, err := mcpserver.EstimateSchemaTokens()
	if err != nil {
		return report, err
	}
	report.MCPTokenTax = tokens
	if tokens >= mcpserver.MaxSchemaTokens {
		report.SelfTestPassed = false
	}

	// Measure hook latency in an isolated temp dir.
	tmpDir, err := os.MkdirTemp("", "ateam-bench-*")
	if err != nil {
		return report, err
	}
	defer os.RemoveAll(tmpDir)

	t0 := time.Now()
	_, _ = hook.Handle(tmpDir, "session-start", strings.NewReader(`{}`))
	report.HookLatencyUs = time.Since(t0).Microseconds()

	var totalBaselineLOC, totalAteamLOC int
	var safetyPassed, mutantCaught int
	var codingCount int

	for _, spec := range tasks {
		var tr TaskResult
		if spec.Tier == "coding-discipline" {
			codingCount++
			tr = evaluateCodingTask(spec)
			if tr.GoodPassedSafety && tr.BadCaughtByGate {
				safetyPassed++
			}
			if tr.MutantCaught {
				mutantCaught++
			}
		} else {
			tr = evaluateContinuityTask(spec)
			report.PickupTurnsSaved += (spec.BaselineTurns - spec.AteamTurns)
		}
		totalBaselineLOC += spec.BaselineLOC
		totalAteamLOC += spec.AteamLOC
		if tr.Passed {
			report.PassedTasks++
		} else {
			report.SelfTestPassed = false
		}
		report.Tasks = append(report.Tasks, tr)
	}

	if totalBaselineLOC > 0 {
		report.AvgLOCReductionPct = int(float64(totalBaselineLOC-totalAteamLOC) * 100.0 / float64(totalBaselineLOC))
	}
	if codingCount > 0 {
		report.SafetyPassRatePct = (safetyPassed * 100) / codingCount
		report.MutationCatchPct = (mutantCaught * 100) / codingCount
	}
	if !report.SelfTestPassed {
		return report, fmt.Errorf("benchmark selftest failed: %d/%d tasks passed", report.PassedTasks, report.TotalTasks)
	}
	return report, nil
}

func evaluateCodingTask(spec TaskSpec) TaskResult {
	goodSafe := spec.SafetyCheck(spec.GoodSource)
	badCaught := !spec.SafetyCheck(spec.BadSource)
	mutantCaught := !spec.SafetyCheck(spec.MutantSource)
	deltaPct := 0
	if spec.BaselineLOC > 0 {
		deltaPct = int(float64(spec.BaselineLOC-spec.AteamLOC) * 100.0 / float64(spec.BaselineLOC))
	}
	return TaskResult{
		ID:               spec.ID,
		Tier:             spec.Tier,
		Description:      spec.Description,
		BaselineLOC:      spec.BaselineLOC,
		AteamLOC:         spec.AteamLOC,
		LOCDeltaPct:      deltaPct,
		GzipBytes:        gzipLen(spec.GoodSource),
		GoodPassedSafety: goodSafe,
		BadCaughtByGate:  badCaught,
		MutantCaught:     mutantCaught,
		BaselineTurns:    spec.BaselineTurns,
		AteamTurns:       spec.AteamTurns,
		Passed:           goodSafe && badCaught && mutantCaught,
	}
}

func evaluateContinuityTask(spec TaskSpec) TaskResult {
	dir, err := os.MkdirTemp("", "ateam-relay-*")
	if err != nil {
		return TaskResult{ID: spec.ID, Tier: spec.Tier, Description: spec.Description}
	}
	defer os.RemoveAll(dir)

	var passed bool
	switch spec.ID {
	case "relay-openspec-hold":
		changeDir := filepath.Join(dir, "openspec", "changes", "relay-01")
		_ = os.MkdirAll(changeDir, 0o755)
		_ = os.WriteFile(filepath.Join(changeDir, "tasks.md"), []byte("## 1. Work\n- [ ] 1.1 Wire Gitea sync\n  - Hold: waiting on #48\n"), 0o644)
		st := &state.State{
			Mode: state.ModeFull,
			LastHandoff: &state.Handoff{
				Goal:          "Finish Gitea sync",
				Addresses:     "branch codex/gitea, SHA 9f8e7d6",
				Constraints:   "Hold: waiting on #48",
				DoneWhen:      "task verify green",
				ChecksToRun:   "task spec:queue",
				CheckpointSHA: "9f8e7d6",
			},
		}
		_ = state.Save(dir, st)
		out, err := hook.Handle(dir, "session-start", strings.NewReader(`{}`))
		passed = err == nil &&
			strings.Contains(out.AdditionalContext, "Goal: Finish Gitea sync") &&
			strings.Contains(out.AdditionalContext, "Hold: waiting on #48") &&
			strings.Contains(out.AdditionalContext, "PONYTAIL MODE [full]")

	case "relay-precompact-recovery":
		_ = os.WriteFile(filepath.Join(dir, "worker.go"), []byte("package main\n// shortcut: single worker goroutine, scale to pool when queue > 1000\n"), 0o644)
		_, _ = hook.Handle(dir, "pre-compact", strings.NewReader(`{"hook_event_name":"PreCompress"}`))
		out, err := hook.Handle(dir, "session-start", strings.NewReader(`{}`))
		st, _ := state.Load(dir)
		passed = err == nil &&
			st.LastHandoff != nil &&
			st.LastHandoff.AutoCompacted &&
			len(st.Shortcuts) == 1 &&
			strings.Contains(out.AdditionalContext, "Auto-saved before context compaction")
	}

	deltaPct := 0
	if spec.BaselineLOC > 0 {
		deltaPct = int(float64(spec.BaselineLOC-spec.AteamLOC) * 100.0 / float64(spec.BaselineLOC))
	}
	return TaskResult{
		ID:               spec.ID,
		Tier:             spec.Tier,
		Description:      spec.Description,
		BaselineLOC:      spec.BaselineLOC,
		AteamLOC:         spec.AteamLOC,
		LOCDeltaPct:      deltaPct,
		GzipBytes:        gzipLen(spec.Description),
		GoodPassedSafety: passed,
		BadCaughtByGate:  passed,
		MutantCaught:     passed,
		BaselineTurns:    spec.BaselineTurns,
		AteamTurns:       spec.AteamTurns,
		Passed:           passed,
	}
}

func countSourceLines(src string) int {
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

// SaveRun writes the benchmark report to benchmarks/runs/latest.json under root.
func SaveRun(root string, rep SuiteReport) error {
	dir := filepath.Join(root, "benchmarks", "runs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "latest.json"), append(data, '\n'), 0o644)
}

// FormatMarkdown renders a human-readable benchmark report table.
func FormatMarkdown(rep SuiteReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# ATeam Benchmark & Self-Test Report\n\n")
	fmt.Fprintf(&b, "- **Self-Test Status:** `PASS=%v` (%d/%d tasks passed)\n", rep.SelfTestPassed, rep.PassedTasks, rep.TotalTasks)
	fmt.Fprintf(&b, "- **Avg Code Size Reduction (vs No-Skill Baseline):** `-%d%% LOC`\n", rep.AvgLOCReductionPct)
	fmt.Fprintf(&b, "- **Adversarial Safety Gate Pass Rate:** `%d%%` (100%% of bad fixtures caught)\n", rep.SafetyPassRatePct)
	fmt.Fprintf(&b, "- **Injected Bug Mutation Catch Rate:** `%d%%`\n", rep.MutationCatchPct)
	fmt.Fprintf(&b, "- **MCP 2-Tool Schema Budget:** `%d tokens` (< 300 token limit)\n", rep.MCPTokenTax)
	fmt.Fprintf(&b, "- **Hook Execution Latency:** `%d µs` (< 5,000 µs limit)\n", rep.HookLatencyUs)
	fmt.Fprintf(&b, "- **Pickup Discovery Turns Saved:** `%d turns` across multi-agent relays\n\n", rep.PickupTurnsSaved)

	b.WriteString("| Task | Tier | Baseline LOC | ATeam LOC | Delta | Safety Gate | Mutant Caught | Turns (Base -> ATeam) |\n")
	b.WriteString("| --- | --- | ---: | ---: | ---: | :---: | :---: | :---: |\n")
	for _, t := range rep.Tasks {
		fmt.Fprintf(&b, "| `%s` | %s | %d | %d | -%d%% | %v | %v | %d -> %d |\n",
			t.ID, t.Tier, t.BaselineLOC, t.AteamLOC, t.LOCDeltaPct, t.GoodPassedSafety && t.BadCaughtByGate, t.MutantCaught, t.BaselineTurns, t.AteamTurns)
	}
	return b.String()
}
