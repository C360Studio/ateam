// Package mutate implements non-destructive Go -overlay mutation checking (ateam mutate / task mutate:check).
package mutate

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Outcome classifies the result of a mutation check.
type Outcome string

// Mutation check outcomes matching the SemEngine testing contract.
const (
	OutcomeDetection    Outcome = "detection"
	OutcomeSurvivor     Outcome = "survivor"
	OutcomeInvalid      Outcome = "invalid"
	OutcomeInconclusive Outcome = "inconclusive"
)

// Options configures a single mutation check run.
type Options struct {
	Pkg        string
	Test       string
	File       string
	Mutant     string
	Expect     []string
	ExpectText []string
	Runs       int
	Timeout    time.Duration
	Seed       uint64
}

// Report summarizes a completed mutation check.
type Report struct {
	Verdict     Outcome  `json:"verdict"`
	Reason      string   `json:"reason"`
	Pkg         string   `json:"pkg"`
	Test        string   `json:"test"`
	File        string   `json:"file"`
	TargetSHA   string   `json:"target_sha256"`
	MatchedLine []string `json:"matched_lines,omitempty"`
}

type stringSliceFlag []string

func (s *stringSliceFlag) String() string { return strings.Join(*s, ",") }
func (s *stringSliceFlag) Set(v string) error {
	*s = append(*s, v)
	return nil
}

// ParseArgs parses CLI flags for `ateam mutate`.
func ParseArgs(args []string) (Options, error) {
	opts := Options{
		Runs:    1,
		Timeout: 2 * time.Minute,
		Seed:    1,
	}
	fs := flag.NewFlagSet("mutate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&opts.Pkg, "pkg", "", "target package (e.g. ./internal/state)")
	fs.StringVar(&opts.Test, "test", "", "exact test or Test/Subtest name")
	fs.StringVar(&opts.File, "file", "", "target Go source file (non-test)")
	fs.StringVar(&opts.Mutant, "mutant", "", "mutant Go file outside repository")
	var expect, expectText stringSliceFlag
	fs.Var(&expect, "expect", "expected failure location (file_test.go:N)")
	fs.Var(&expectText, "expect-text", "expected substring in failure output")
	fs.IntVar(&opts.Runs, "runs", 1, "number of baseline and mutant runs")
	fs.DurationVar(&opts.Timeout, "timeout", 2*time.Minute, "per-run go test timeout")
	fs.Uint64Var(&opts.Seed, "seed", 1, "Rapid/property seed (must be > 0)")
	if err := fs.Parse(args); err != nil {
		return opts, err
	}
	opts.Expect = expect
	opts.ExpectText = expectText
	return opts, nil
}

// Validate checks pre-run refusal rules before any test process starts.
func Validate(repoRoot string, opts Options, goflags string) (string, string, []byte, []byte, error) {
	if strings.TrimSpace(opts.Pkg) == "" || strings.Contains(opts.Pkg, "...") {
		return "", "", nil, nil, errors.New("-pkg is required and must name a single package without '...'")
	}
	if strings.TrimSpace(opts.Test) == "" {
		return "", "", nil, nil, errors.New("-test is required")
	}
	if strings.TrimSpace(opts.File) == "" {
		return "", "", nil, nil, errors.New("-file is required")
	}
	if !strings.HasSuffix(opts.File, ".go") || strings.HasSuffix(opts.File, "_test.go") {
		return "", "", nil, nil, errors.New("-file must be a non-test .go source file")
	}
	if strings.TrimSpace(opts.Mutant) == "" {
		return "", "", nil, nil, errors.New("-mutant is required")
	}
	if len(opts.Expect) == 0 && len(opts.ExpectText) == 0 {
		return "", "", nil, nil, errors.New("at least one -expect (file.go:N) or -expect-text is required")
	}
	if opts.Seed == 0 {
		return "", "", nil, nil, errors.New("-seed 0 is refused because generated checks require a deterministic seed")
	}
	if opts.Runs < 1 {
		return "", "", nil, nil, errors.New("-runs must be >= 1")
	}
	if strings.Contains(goflags, "-overlay") || strings.Contains(goflags, "-cover") {
		return "", "", nil, nil, errors.New("GOFLAGS already contains -overlay or -cover")
	}

	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return "", "", nil, nil, err
	}
	if evalRoot, err := filepath.EvalSymlinks(absRoot); err == nil {
		absRoot = evalRoot
	}
	absTarget := opts.File
	if !filepath.IsAbs(absTarget) {
		absTarget = filepath.Join(absRoot, filepath.FromSlash(opts.File))
	}
	if evalTarget, err := filepath.EvalSymlinks(absTarget); err == nil {
		absTarget = evalTarget
	}
	absMutant, err := filepath.Abs(opts.Mutant)
	if err != nil {
		return "", "", nil, nil, err
	}
	if evalMutant, err := filepath.EvalSymlinks(absMutant); err == nil {
		absMutant = evalMutant
	}
	relMut, err := filepath.Rel(absRoot, absMutant)
	if err == nil && relMut != ".." && !strings.HasPrefix(relMut, ".."+string(filepath.Separator)) {
		return "", "", nil, nil, errors.New("-mutant must be outside the repository tree")
	}

	targetBytes, err := os.ReadFile(absTarget)
	if err != nil {
		return "", "", nil, nil, fmt.Errorf("read -file: %w", err)
	}
	mutantBytes, err := os.ReadFile(absMutant)
	if err != nil {
		return "", "", nil, nil, fmt.Errorf("read -mutant: %w", err)
	}
	if bytes.Equal(targetBytes, mutantBytes) {
		return "", "", nil, nil, errors.New("-mutant is identical to -file")
	}
	return absRoot, absTarget, targetBytes, mutantBytes, nil
}

// Run executes the baseline, mutant (-overlay), after, and optional reach runs.
// Returns Report, exitCode (0 = detection, 1 = non-detection verdict, 2 = refused/error).
func Run(ctx context.Context, repoRoot string, opts Options, stdout, stderr io.Writer) (Report, int) {
	absRoot, absTarget, targetBytes, mutantBytes, err := Validate(repoRoot, opts, os.Getenv("GOFLAGS"))
	if err != nil {
		fmt.Fprintf(stderr, "ateam mutate: refused, and no run was started: %v\n", err)
		return Report{}, 2
	}

	beforeSHA := sha256Hex(targetBytes)
	rep := Report{Pkg: opts.Pkg, Test: opts.Test, File: opts.File, TargetSHA: beforeSHA}

	tmpDir, err := os.MkdirTemp("", "ateam-mutate-*")
	if err != nil {
		fmt.Fprintf(stderr, "ateam mutate: temp dir: %v\n", err)
		return rep, 2
	}
	defer os.RemoveAll(tmpDir)

	overlayPath, err := prepareOverlay(tmpDir, absTarget, mutantBytes)
	if err != nil {
		fmt.Fprintf(stderr, "ateam mutate: %v\n", err)
		return rep, 2
	}

	for i := 1; i <= opts.Runs; i++ {
		if res := runGoTest(ctx, absRoot, opts, "", ""); res.outcome != "pass" {
			rep.Verdict = OutcomeInconclusive
			rep.Reason = fmt.Sprintf("baseline run %d of %d did not pass: %s", i, opts.Runs, res.reason)
			writeSummary(stdout, rep)
			return rep, 1
		}
	}

	var mutantOutcomes []runAnalysis
	allPassed := true
	for i := 1; i <= opts.Runs; i++ {
		res := runGoTest(ctx, absRoot, opts, overlayPath, "")
		mutantOutcomes = append(mutantOutcomes, res)
		allPassed = allPassed && (res.outcome == "pass")
		if len(res.matchedLines) > 0 && len(rep.MatchedLine) == 0 {
			rep.MatchedLine = res.matchedLines
		}
	}

	afterRes := runGoTest(ctx, absRoot, opts, "", "")
	afterBytes, readErr := os.ReadFile(absTarget)
	if readErr != nil || sha256Hex(afterBytes) != beforeSHA {
		rep.Verdict, rep.Reason = OutcomeInconclusive, "the target file changed during the check"
		writeSummary(stdout, rep)
		return rep, 1
	}
	if afterRes.outcome != "pass" {
		rep.Verdict, rep.Reason = OutcomeInconclusive, "the after-run on the unchanged code did not pass: "+afterRes.reason
		writeSummary(stdout, rep)
		return rep, 1
	}

	rep.Verdict, rep.Reason = finalizeVerdict(ctx, absRoot, tmpDir, absTarget, targetBytes, mutantBytes, opts, mutantOutcomes, allPassed)
	writeSummary(stdout, rep)
	if rep.Verdict == OutcomeDetection {
		return rep, 0
	}
	return rep, 1
}

func prepareOverlay(tmpDir, absTarget string, mutantBytes []byte) (string, error) {
	mutantCopy := filepath.Join(tmpDir, "mutant-"+filepath.Base(absTarget))
	if err := os.WriteFile(mutantCopy, mutantBytes, 0o600); err != nil {
		return "", fmt.Errorf("write mutant copy: %w", err)
	}
	overlayPath := filepath.Join(tmpDir, "overlay.json")
	overlayData, err := json.Marshal(map[string]map[string]string{"Replace": {absTarget: mutantCopy}})
	if err != nil {
		return "", fmt.Errorf("marshal overlay: %w", err)
	}
	if err := os.WriteFile(overlayPath, overlayData, 0o600); err != nil {
		return "", fmt.Errorf("write overlay: %w", err)
	}
	return overlayPath, nil
}

func finalizeVerdict(ctx context.Context, absRoot, tmpDir, absTarget string, targetBytes, mutantBytes []byte, opts Options, mutantOutcomes []runAnalysis, allPassed bool) (Outcome, string) {
	first := mutantOutcomes[0]
	for idx, m := range mutantOutcomes {
		if m.outcome == OutcomeInconclusive {
			return OutcomeInconclusive, fmt.Sprintf("mutant run %d: %s", idx+1, m.reason)
		}
		if m.outcome != first.outcome {
			return OutcomeInconclusive, "the mutant runs disagreed across iterations"
		}
	}
	if first.outcome == OutcomeInvalid {
		return OutcomeInvalid, first.reason
	}
	if first.outcome == OutcomeDetection {
		return OutcomeDetection, fmt.Sprintf("each of the %d mutant run(s) failed %s with an expected assertion, and the after-run passed", len(mutantOutcomes), opts.Test)
	}
	if allPassed {
		coverPath := filepath.Join(tmpDir, "reach.cover")
		reachRes := runGoTest(ctx, absRoot, opts, "", coverPath)
		if reachRes.outcome != "pass" {
			return OutcomeInconclusive, "the reach run on the unchanged code did not pass: " + reachRes.reason
		}
		reachedChange, err := checkReach(targetBytes, mutantBytes, filepath.Base(absTarget), coverPath)
		if err != nil || reachedChange {
			return OutcomeSurvivor, fmt.Sprintf("the named test passed in every mutant run and reached the wrong change (seed %d)", opts.Seed)
		}
		return OutcomeInvalid, "the test did not reach the wrong change: no executed coverage block touched the mutated lines"
	}
	return OutcomeInconclusive, first.reason
}

type runAnalysis struct {
	outcome      Outcome // or "pass"
	reason       string
	matchedLines []string
}

type testEvent struct {
	Action      string `json:"Action"`
	Test        string `json:"Test"`
	Output      string `json:"Output"`
	FailedBuild string `json:"FailedBuild"`
}

func runGoTest(ctx context.Context, repoRoot string, opts Options, overlayPath, coverProfile string) runAnalysis {
	runPattern := buildExactRunPattern(opts.Test)
	args := []string{"test", "-json", "-race", "-count=1", "-timeout=" + opts.Timeout.String(), "-run=" + runPattern}
	if overlayPath != "" {
		args = append(args, "-overlay="+overlayPath)
	}
	if coverProfile != "" {
		args = append(args, "-coverprofile="+coverProfile)
	}
	args = append(args, opts.Pkg)

	boundCtx, cancel := context.WithTimeout(ctx, 2*opts.Timeout)
	defer cancel()

	cmd := exec.CommandContext(boundCtx, "go", args...)
	cmd.Dir = repoRoot
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("RAPID_SEED=%d", opts.Seed),
		"RAPID_NOFAILFILE=true",
	)
	out, err := cmd.CombinedOutput()
	if errors.Is(boundCtx.Err(), context.DeadlineExceeded) {
		return runAnalysis{outcome: OutcomeInconclusive, reason: "run exceeded twice the timeout bound"}
	}

	var (
		buildFailed  bool
		selected     bool
		timedOut     bool
		result       string
		expectedHits []string
	)
	isOurs := func(t string) bool {
		return t == opts.Test || strings.HasPrefix(t, opts.Test+"/")
	}

	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Bytes()
		var ev testEvent
		if jsonErr := json.Unmarshal(line, &ev); jsonErr != nil {
			ev = testEvent{Action: "output", Output: string(line)}
		}
		switch ev.Action {
		case "build-fail":
			buildFailed = true
		case "run":
			if ev.Test == opts.Test {
				selected = true
			}
		case "pass", "fail", "skip":
			if ev.Test == opts.Test {
				result = ev.Action
			}
			if ev.Test == "" && ev.FailedBuild != "" {
				buildFailed = true
			}
		case "output":
			text := strings.TrimRight(ev.Output, "\n")
			trimmed := strings.TrimLeft(text, " \t")
			if strings.HasPrefix(trimmed, "panic: test timed out after") {
				timedOut = true
			}
			if ev.Test == "" && (strings.Contains(text, "[build failed]") || strings.Contains(text, "[setup failed]")) {
				buildFailed = true
			}
			if isOurs(ev.Test) && matchesExpectation(text, opts.Expect, opts.ExpectText) {
				expectedHits = append(expectedHits, strings.TrimSpace(text))
			}
		}
	}

	isMutant := overlayPath != ""
	switch {
	case buildFailed && isMutant:
		return runAnalysis{outcome: OutcomeInvalid, reason: "the wrong change does not build"}
	case buildFailed:
		return runAnalysis{outcome: OutcomeInconclusive, reason: "the unchanged code does not build"}
	case timedOut:
		return runAnalysis{outcome: OutcomeInconclusive, reason: "go test timed out"}
	case !selected:
		return runAnalysis{outcome: OutcomeInconclusive, reason: fmt.Sprintf("test %q was not selected in %s", opts.Test, opts.Pkg)}
	case result == "pass" && err == nil:
		return runAnalysis{outcome: "pass", reason: "passed"}
	case result == "fail" && isMutant && len(expectedHits) > 0:
		return runAnalysis{outcome: OutcomeDetection, reason: "failed with expected assertion", matchedLines: expectedHits}
	case result == "fail" && isMutant:
		return runAnalysis{outcome: OutcomeInconclusive, reason: "failed without matching any -expect or -expect-text assertion"}
	case result == "skip":
		return runAnalysis{outcome: OutcomeInconclusive, reason: "the named test was skipped"}
	default:
		return runAnalysis{outcome: OutcomeInconclusive, reason: "test did not report expected pass/fail outcome"}
	}
}

func buildExactRunPattern(testName string) string {
	parts := strings.Split(testName, "/")
	for i, p := range parts {
		parts[i] = "^" + regexp.QuoteMeta(p) + "$"
	}
	return strings.Join(parts, "/")
}

func matchesExpectation(line string, locations, texts []string) bool {
	trimmed := strings.TrimLeft(line, " \t")
	for _, loc := range locations {
		if strings.HasPrefix(trimmed, loc+":") {
			return true
		}
	}
	for _, txt := range texts {
		if strings.Contains(line, txt) {
			return true
		}
	}
	return false
}

var coverLineRe = regexp.MustCompile(`^(.+):([0-9]+)\.[0-9]+,([0-9]+)\.[0-9]+ [0-9]+ ([0-9]+)$`)

func checkReach(orig, mutant []byte, baseName, coverPath string) (bool, error) {
	coverData, err := os.ReadFile(coverPath)
	if err != nil {
		return false, err
	}
	changedLines := diffChangedLines(orig, mutant)
	if len(changedLines) == 0 {
		return true, nil
	}
	var sawFileBlock bool
	for _, line := range strings.Split(string(coverData), "\n") {
		m := coverLineRe.FindStringSubmatch(strings.TrimSpace(line))
		if len(m) != 5 {
			continue
		}
		if !strings.HasSuffix(m[1], "/"+baseName) && m[1] != baseName {
			continue
		}
		sawFileBlock = true
		startLine, _ := strconv.Atoi(m[2])
		endLine, _ := strconv.Atoi(m[3])
		count, _ := strconv.Atoi(m[4])
		if count <= 0 {
			continue
		}
		for ln := range changedLines {
			if ln >= startLine && ln <= endLine {
				return true, nil
			}
		}
	}
	if !sawFileBlock {
		return true, nil
	}
	return false, nil
}

func diffChangedLines(orig, mutant []byte) map[int]bool {
	origLines := strings.Split(string(orig), "\n")
	mutLines := strings.Split(string(mutant), "\n")
	changed := make(map[int]bool)
	maxLen := len(origLines)
	if len(mutLines) > maxLen {
		maxLen = len(mutLines)
	}
	for i := 0; i < maxLen; i++ {
		var a, b string
		if i < len(origLines) {
			a = origLines[i]
		}
		if i < len(mutLines) {
			b = mutLines[i]
		}
		if a != b {
			lineNo := i + 1
			if lineNo > len(origLines) && len(origLines) > 0 {
				lineNo = len(origLines)
			}
			changed[lineNo] = true
		}
	}
	return changed
}

func writeSummary(w io.Writer, rep Report) {
	fmt.Fprintf(w, "pkg: %s\ntest: %s\nfile: %s (sha256: %s)\n", rep.Pkg, rep.Test, rep.File, rep.TargetSHA)
	for _, line := range rep.MatchedLine {
		fmt.Fprintf(w, "matched: %s\n", line)
	}
	fmt.Fprintf(w, "verdict: %s (%s)\n", rep.Verdict, rep.Reason)
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
