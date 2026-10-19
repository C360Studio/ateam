package mutate

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func setupTinyGoMod(t *testing.T) (string, string) {
	t.Helper()
	repo := t.TempDir()
	mustWrite(t, filepath.Join(repo, "go.mod"), "module example.com/tiny\n\ngo 1.24\n")
	src := `package tiny

func ClampMax(n, max int) int {
	if n > max {
		return max
	}
	return n
}

func DeadHelper(x int) int {
	return x + 1
}
`
	targetFile := filepath.Join(repo, "tiny.go")
	mustWrite(t, targetFile, src)
	testSrc := `package tiny

import "testing"

func TestClampMax(t *testing.T) {
	if got := ClampMax(12, 10); got != 10 {
		t.Fatalf("want clamped max 10, got %d", got)
	}
	if got := ClampMax(5, 10); got != 5 {
		t.Fatalf("want 5, got %d", got)
	}
}
`
	mustWrite(t, filepath.Join(repo, "tiny_test.go"), testSrc)
	return repo, targetFile
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRefusals(t *testing.T) {
	repo, target := setupTinyGoMod(t)
	outsideDir := t.TempDir()
	validMutant := filepath.Join(outsideDir, "mutant.go")
	mustWrite(t, validMutant, "package tiny\nfunc ClampMax(n, max int) int { return n }\n")

	base := Options{
		Pkg:        ".",
		Test:       "TestClampMax",
		File:       "tiny.go",
		Mutant:     validMutant,
		ExpectText: []string{"want clamped max 10"},
		Runs:       1,
		Timeout:    30 * time.Second,
		Seed:       1,
	}

	if _, _, _, _, err := Validate(repo, base, ""); err != nil {
		t.Fatalf("expected valid options, got %v", err)
	}

	// 1. Refuse _test.go target
	badTestTarget := base
	badTestTarget.File = "tiny_test.go"
	if _, _, _, _, err := Validate(repo, badTestTarget, ""); err == nil {
		t.Fatal("expected refusal for _test.go file")
	}

	// 2. Refuse mutant inside repo
	inRepoMutant := filepath.Join(repo, "inrepo.go")
	mustWrite(t, inRepoMutant, "package tiny\n")
	badInRepo := base
	badInRepo.Mutant = inRepoMutant
	if _, _, _, _, err := Validate(repo, badInRepo, ""); err == nil {
		t.Fatal("expected refusal for mutant inside repo")
	}

	// 3. Refuse identical mutant
	identicalMut := filepath.Join(outsideDir, "identical.go")
	origBytes, _ := os.ReadFile(target)
	mustWrite(t, identicalMut, string(origBytes))
	badIdentical := base
	badIdentical.Mutant = identicalMut
	if _, _, _, _, err := Validate(repo, badIdentical, ""); err == nil {
		t.Fatal("expected refusal for identical mutant")
	}

	// 4. Refuse missing -expect / -expect-text
	badNoExpect := base
	badNoExpect.ExpectText = nil
	if _, _, _, _, err := Validate(repo, badNoExpect, ""); err == nil {
		t.Fatal("expected refusal when neither -expect nor -expect-text is provided")
	}

	// 5. Refuse seed 0
	badSeed := base
	badSeed.Seed = 0
	if _, _, _, _, err := Validate(repo, badSeed, ""); err == nil {
		t.Fatal("expected refusal for -seed 0")
	}

	// 6. Refuse GOFLAGS with -overlay or -cover
	if _, _, _, _, err := Validate(repo, base, "-cover"); err == nil {
		t.Fatal("expected refusal when GOFLAGS contains -cover")
	}
}

func TestRunOutcomes(t *testing.T) {
	repo, target := setupTinyGoMod(t)
	outsideDir := t.TempDir()
	origBytes, _ := os.ReadFile(target)

	// 1. Detection: break ClampMax boundary and match expected failure text
	detMutant := filepath.Join(outsideDir, "det.go")
	mustWrite(t, detMutant, strings.Replace(string(origBytes), "return max", "return n", 1))
	var out, errBuf bytes.Buffer
	rep, code := Run(context.Background(), repo, Options{
		Pkg:        ".",
		Test:       "TestClampMax",
		File:       "tiny.go",
		Mutant:     detMutant,
		ExpectText: []string{"want clamped max 10"},
		Runs:       1,
		Timeout:    30 * time.Second,
		Seed:       1,
	}, &out, &errBuf)
	if code != 0 || rep.Verdict != OutcomeDetection {
		t.Fatalf("expected detection (code 0), got code=%d verdict=%s reason=%s out=%s", code, rep.Verdict, rep.Reason, out.String())
	}

	// 2. Survivor: off-by-one `n >= max` still returns `max` when n == max, so reached mutant passes
	survMutant := filepath.Join(outsideDir, "surv.go")
	mustWrite(t, survMutant, strings.Replace(string(origBytes), "if n > max", "if n >= max", 1))
	out.Reset()
	rep, code = Run(context.Background(), repo, Options{
		Pkg:        ".",
		Test:       "TestClampMax",
		File:       "tiny.go",
		Mutant:     survMutant,
		ExpectText: []string{"want clamped max 10"},
		Runs:       1,
		Timeout:    30 * time.Second,
		Seed:       1,
	}, &out, &errBuf)
	if code != 1 || rep.Verdict != OutcomeSurvivor {
		t.Fatalf("expected survivor (code 1), got code=%d verdict=%s reason=%s", code, rep.Verdict, rep.Reason)
	}

	// 3. Invalid (unreached): mutate DeadHelper which TestClampMax never calls
	unreachedMut := filepath.Join(outsideDir, "unreached.go")
	mustWrite(t, unreachedMut, strings.Replace(string(origBytes), "return x + 1", "return x - 99", 1))
	out.Reset()
	rep, code = Run(context.Background(), repo, Options{
		Pkg:        ".",
		Test:       "TestClampMax",
		File:       "tiny.go",
		Mutant:     unreachedMut,
		ExpectText: []string{"want clamped max 10"},
		Runs:       1,
		Timeout:    30 * time.Second,
		Seed:       1,
	}, &out, &errBuf)
	if code != 1 || rep.Verdict != OutcomeInvalid {
		t.Fatalf("expected invalid (unreached, code 1), got code=%d verdict=%s reason=%s", code, rep.Verdict, rep.Reason)
	}

	// 4. Invalid (build failure): syntax error in mutant
	brokenMut := filepath.Join(outsideDir, "broken.go")
	mustWrite(t, brokenMut, "package tiny\nfunc ClampMax(n, max int) int { return }\n")
	out.Reset()
	rep, code = Run(context.Background(), repo, Options{
		Pkg:        ".",
		Test:       "TestClampMax",
		File:       "tiny.go",
		Mutant:     brokenMut,
		ExpectText: []string{"want clamped max 10"},
		Runs:       1,
		Timeout:    30 * time.Second,
		Seed:       1,
	}, &out, &errBuf)
	if code != 1 || rep.Verdict != OutcomeInvalid {
		t.Fatalf("expected invalid (build fail, code 1), got code=%d verdict=%s reason=%s", code, rep.Verdict, rep.Reason)
	}

	// 5. Inconclusive: mutant fails, but with a different assertion than -expect-text
	out.Reset()
	rep, code = Run(context.Background(), repo, Options{
		Pkg:        ".",
		Test:       "TestClampMax",
		File:       "tiny.go",
		Mutant:     detMutant,
		ExpectText: []string{"some completely unrelated assertion message"},
		Runs:       1,
		Timeout:    30 * time.Second,
		Seed:       1,
	}, &out, &errBuf)
	if code != 1 || rep.Verdict != OutcomeInconclusive {
		t.Fatalf("expected inconclusive (wrong assertion, code 1), got code=%d verdict=%s reason=%s", code, rep.Verdict, rep.Reason)
	}

	// 6. Astra #4: Build-excluded target file (//go:build ignore) has no coverage blocks -> must report invalid, not survivor
	mustWrite(t, filepath.Join(repo, "excluded.go"), "//go:build ignore\n\npackage tiny\nfunc Excluded() int { return 1 }\n")
	excludedMut := filepath.Join(outsideDir, "excluded_mut.go")
	mustWrite(t, excludedMut, "//go:build ignore\n\npackage tiny\nfunc Excluded() int { return 2 }\n")
	out.Reset()
	rep, code = Run(context.Background(), repo, Options{
		Pkg:        ".",
		Test:       "TestClampMax",
		File:       "excluded.go",
		Mutant:     excludedMut,
		ExpectText: []string{"want clamped max 10"},
		Runs:       1,
		Timeout:    30 * time.Second,
		Seed:       1,
	}, &out, &errBuf)
	if code != 1 || rep.Verdict != OutcomeInvalid {
		t.Fatalf("expected invalid for build-excluded target file, got code=%d verdict=%s reason=%s", code, rep.Verdict, rep.Reason)
	}

	// 7. Astra #3: Baseline passing t.Log matching -expect-text followed by an unrelated t.Fatal must report inconclusive, not detection
	mustWrite(t, filepath.Join(repo, "tiny_test.go"), `package tiny
import "testing"
func TestClampMax(t *testing.T) {
	t.Log("expected clamp assertion")
	if ClampMax(12, 10) != 10 {
		t.Fatal("unrelated failure assertion")
	}
}
`)
	out.Reset()
	rep, code = Run(context.Background(), repo, Options{
		Pkg:        ".",
		Test:       "TestClampMax",
		File:       "tiny.go",
		Mutant:     detMutant,
		ExpectText: []string{"expected clamp assertion"},
		Runs:       1,
		Timeout:    30 * time.Second,
		Seed:       1,
	}, &out, &errBuf)
	if code != 1 || rep.Verdict != OutcomeInconclusive {
		t.Fatalf("expected inconclusive when only baseline log matched -expect-text, got code=%d verdict=%s reason=%s", code, rep.Verdict, rep.Reason)
	}

	// Verify target file on disk was never modified
	afterBytes, _ := os.ReadFile(target)
	if !bytes.Equal(origBytes, afterBytes) {
		t.Fatal("target file in repository was modified during mutation check")
	}
}
