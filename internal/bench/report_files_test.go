package bench

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func reportFileFixture() map[string][]byte {
	return map[string][]byte{"results.json": []byte("{\"count\":2}\n"), "results.md": []byte("# Two attempts\n"), "evidence.json": []byte("{\"private\":false}\n")}
}

// spec: openspec/changes/benchmark-report-exporter/specs/benchmark-reports/spec.md#Requirement: Deterministic non-overwriting output and read-only checks
func TestReportFilesHistory(t *testing.T) {
	ctx := context.Background()
	dir := filepath.Join(t.TempDir(), "report")
	files := reportFileFixture()
	if err := checkReportFiles(ctx, dir, files); err == nil {
		t.Fatal("absent report passed check")
	}
	if err := writeReportFiles(ctx, dir, files); err != nil {
		t.Fatal(err)
	}
	for name, want := range files {
		got, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || string(got) != string(want) {
			t.Fatalf("%s: %q, %v", name, got, err)
		}
	}
	if err := checkReportFiles(ctx, dir, files); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "interpretation.md"), []byte("Human opinion"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := checkReportFiles(ctx, dir, files); err != nil {
		t.Fatal(err)
	}
	if err := writeReportFiles(ctx, dir, files); err == nil {
		t.Fatal("overwrote existing report")
	}
	if err := os.WriteFile(filepath.Join(dir, "results.md"), []byte("tampered"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := checkReportFiles(ctx, dir, files); err == nil {
		t.Fatal("tampered report passed")
	}
	got, err := os.ReadFile(filepath.Join(dir, "results.md"))
	if err != nil || string(got) != "tampered" {
		t.Fatal("check repaired report")
	}
	note, err := os.ReadFile(filepath.Join(dir, "interpretation.md"))
	if err != nil || string(note) != "Human opinion" {
		t.Fatal("human note changed")
	}
}

// spec: openspec/changes/benchmark-report-exporter/specs/benchmark-reports/spec.md#Requirement: Deterministic non-overwriting output and read-only checks
func TestReportFilesCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	dir := filepath.Join(t.TempDir(), "report")
	if err := writeReportFiles(ctx, dir, reportFileFixture()); err == nil {
		t.Fatal("canceled output succeeded")
	}
	if _, err := os.Lstat(dir); !os.IsNotExist(err) {
		t.Fatalf("canceled output created directory: %v", err)
	}
	if err := checkReportFiles(ctx, dir, reportFileFixture()); err == nil {
		t.Fatal("canceled check succeeded")
	}
}

// spec: openspec/changes/benchmark-report-exporter/specs/benchmark-reports/spec.md#Requirement: Deterministic non-overwriting output and read-only checks
func TestReportFilesUnsafePaths(t *testing.T) {
	for _, kind := range []string{"directory", "file", "symlink"} {
		t.Run("destination-"+kind, func(t *testing.T) {
			base := t.TempDir()
			dir := filepath.Join(base, "report")
			target := filepath.Join(base, "target")
			if err := os.Mkdir(target, 0700); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "directory":
				if err := os.Mkdir(dir, 0700); err != nil {
					t.Fatal(err)
				}
			case "file":
				if err := os.WriteFile(dir, []byte("untouched"), 0600); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Symlink(target, dir); err != nil {
					t.Fatal(err)
				}
			}
			if err := writeReportFiles(context.Background(), dir, reportFileFixture()); err == nil {
				t.Fatal("existing destination accepted")
			}
			if err := checkReportFiles(context.Background(), dir, reportFileFixture()); err == nil {
				t.Fatal("unsafe destination accepted")
			}
			entries, err := os.ReadDir(target)
			if err != nil || len(entries) != 0 {
				t.Fatal("symlink target changed")
			}
		})
	}
	for _, kind := range []string{"symlink", "directory", "missing", "oversized", "truncated"} {
		t.Run("file-"+kind, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "report")
			files := reportFileFixture()
			if err := writeReportFiles(context.Background(), dir, files); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(dir, "results.md")
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "symlink":
				if err := os.WriteFile(filepath.Join(dir, "same-bytes.md"), files["results.md"], 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("same-bytes.md", path); err != nil {
					t.Fatal(err)
				}
			case "directory":
				if err := os.Mkdir(path, 0700); err != nil {
					t.Fatal(err)
				}
			case "oversized":
				if err := os.WriteFile(path, make([]byte, 1<<20), 0600); err != nil {
					t.Fatal(err)
				}
			case "truncated":
				if err := os.WriteFile(path, []byte{}, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err := checkReportFiles(context.Background(), dir, files); err == nil {
				t.Fatalf("%s passed", kind)
			}
		})
	}
}

// spec: openspec/changes/benchmark-report-exporter/specs/benchmark-reports/spec.md#Requirement: Deterministic non-overwriting output and read-only checks
func TestReportFilesInvalidNames(t *testing.T) {
	for _, files := range []map[string][]byte{nil, {"results.json": nil, "results.md": nil, "../escape": nil}} {
		dir := filepath.Join(t.TempDir(), "report")
		if err := writeReportFiles(context.Background(), dir, files); err == nil {
			t.Fatal("invalid filenames accepted")
		}
		if _, err := os.Lstat(dir); !os.IsNotExist(err) {
			t.Fatal("invalid filenames created destination")
		}
	}
}

// spec: openspec/changes/benchmark-report-exporter/specs/benchmark-reports/spec.md#Requirement: Deterministic non-overwriting output and read-only checks
func TestReportFilesInterrupted(t *testing.T) {
	for _, cancelAfterFirst := range []bool{false, true} {
		t.Run(fmt.Sprint(cancelAfterFirst), func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "report")
			files := reportFileFixture()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var order []string
			injected := errors.New("write failed")
			err := writeReportFilesWith(ctx, dir, files, func(ctx context.Context, root *os.Root, name string, data []byte) error {
				order = append(order, name)
				if name == "results.md" {
					return injected
				}
				if err := writeReportFile(ctx, root, name, data); err != nil {
					return err
				}
				if cancelAfterFirst {
					cancel()
				}
				return nil
			})
			if err == nil {
				t.Fatal("interrupted export succeeded")
			}
			if len(order) == 0 || order[0] != "results.json" {
				t.Fatalf("write order: %v", order)
			}
			if _, err := os.Lstat(filepath.Join(dir, "evidence.json")); !os.IsNotExist(err) {
				t.Fatal("interrupted output has manifest")
			}
			if err := checkReportFiles(context.Background(), dir, files); err == nil {
				t.Fatal("incomplete output passed")
			}
			if err := writeReportFiles(context.Background(), dir, files); err == nil {
				t.Fatal("retry overwrote incomplete output")
			}
		})
	}
}

type reportFailWriter struct {
	short              bool
	writeErr, closeErr error
	closed             bool
}

func (w *reportFailWriter) Write(data []byte) (int, error) {
	if w.writeErr != nil {
		return 0, w.writeErr
	}
	if w.short {
		return len(data) - 1, nil
	}
	return len(data), nil
}
func (w *reportFailWriter) Close() error { w.closed = true; return w.closeErr }

// spec: openspec/changes/benchmark-report-exporter/specs/benchmark-reports/spec.md#Requirement: Deterministic non-overwriting output and read-only checks
func TestReportFilesWriteAndCloseFailures(t *testing.T) {
	closeErr := errors.New("close failed")
	writeErr := errors.New("write failed")
	for _, tc := range []struct {
		writer reportFailWriter
		want   error
	}{
		{reportFailWriter{short: true}, io.ErrShortWrite}, {reportFailWriter{writeErr: writeErr}, writeErr}, {reportFailWriter{closeErr: closeErr}, closeErr},
	} {
		err := writeCloseReportFile(context.Background(), &tc.writer, []byte("expected"))
		if !errors.Is(err, tc.want) || !tc.writer.closed {
			t.Fatalf("error=%v, closed=%t", err, tc.writer.closed)
		}
	}
}

// spec: openspec/changes/benchmark-report-exporter/specs/benchmark-reports/spec.md#Requirement: Deterministic non-overwriting output and read-only checks
func TestReportFilesDirectoryReplacement(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "report")
	moved := filepath.Join(base, "moved")
	target := filepath.Join(base, "unrelated")
	if err := os.Mkdir(target, 0700); err != nil {
		t.Fatal(err)
	}
	arrived, resume := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- writeReportFilesWith(context.Background(), dir, reportFileFixture(), func(ctx context.Context, root *os.Root, name string, data []byte) error {
			if name == "results.json" {
				close(arrived)
				<-resume
			}
			return writeReportFile(ctx, root, name, data)
		})
	}()
	select {
	case <-arrived:
	case err := <-done:
		t.Fatalf("writer did not reach barrier: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("writer did not reach barrier")
	}
	// Always release the owned goroutine even if a filesystem assertion fails.
	func() {
		defer close(resume)
		if err := os.Rename(dir, moved); err != nil {
			t.Error(err)
			return
		}
		if err := os.Symlink(target, dir); err != nil {
			t.Error(err)
		}
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("replaced directory succeeded")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("writer stuck")
	}
	entries, err := os.ReadDir(target)
	if err != nil || len(entries) != 0 {
		t.Fatalf("unrelated directory changed: %v %v", entries, err)
	}
	if _, err := os.Stat(filepath.Join(moved, "results.json")); err != nil {
		t.Fatalf("write was not anchored to original: %v", err)
	}
	if _, err := os.Stat(filepath.Join(moved, "evidence.json")); !os.IsNotExist(err) {
		t.Fatal("replaced report has manifest")
	}
}

// spec: openspec/changes/benchmark-report-exporter/specs/benchmark-reports/spec.md#Requirement: Deterministic non-overwriting output and read-only checks
func TestReportFilesConcurrentClaim(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "report")
	start := make(chan struct{})
	errs := make(chan error, 2)
	var ready sync.WaitGroup
	ready.Add(2)
	for range 2 {
		go func() {
			ready.Done()
			<-start
			errs <- writeReportFiles(context.Background(), dir, reportFileFixture())
		}()
	}
	ready.Wait()
	close(start)
	success := 0
	for range 2 {
		select {
		case err := <-errs:
			if err == nil {
				success++
			}
		case <-time.After(5 * time.Second):
			t.Fatal("concurrent writer stuck")
		}
	}
	if success != 1 {
		t.Fatalf("successful claims=%d, want exactly one", success)
	}
	if err := checkReportFiles(context.Background(), dir, reportFileFixture()); err != nil {
		t.Fatal(err)
	}
}

// spec: openspec/changes/benchmark-report-exporter/specs/benchmark-reports/spec.md#Requirement: Deterministic non-overwriting output and read-only checks
func TestReportFilesGeneratedHistories(t *testing.T) {
	// Exhaust all 3^4 short histories against an independent absent/complete/drift
	// model. Repeated creation must fail even after tampering; check never repairs.
	reached := map[string]int{}
	for history := range 81 {
		dir := filepath.Join(t.TempDir(), "report")
		exists, complete := false, false
		code := history
		for range 4 {
			action := code % 3
			code /= 3
			switch action {
			case 0:
				err := writeReportFiles(context.Background(), dir, reportFileFixture())
				if (err == nil) == exists {
					t.Fatalf("history %d: create exists=%t err=%v", history, exists, err)
				}
				if !exists {
					exists, complete = true, true
					reached["create"]++
				} else {
					reached["refuse"]++
				}
			case 1:
				err := checkReportFiles(context.Background(), dir, reportFileFixture())
				if (err == nil) != complete {
					t.Fatalf("history %d: check complete=%t err=%v", history, complete, err)
				}
				if complete {
					reached["check-pass"]++
				} else {
					reached["check-fail"]++
				}
			case 2:
				if exists {
					if err := os.WriteFile(filepath.Join(dir, "results.json"), []byte("changed"), 0600); err != nil {
						t.Fatal(err)
					}
					complete = false
					reached["tamper"]++
				}
			}
		}
	}
	for _, transition := range []string{"create", "refuse", "check-pass", "check-fail", "tamper"} {
		if reached[transition] == 0 {
			t.Fatalf("unreached transition %s", transition)
		}
	}
}

// spec: openspec/changes/benchmark-report-exporter/specs/benchmark-reports/spec.md#Requirement: Deterministic non-overwriting output and read-only checks
func TestReportFilesExclusiveFile(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "report")
	err := writeReportFilesWith(context.Background(), dir, reportFileFixture(), func(ctx context.Context, root *os.Root, name string, data []byte) error {
		if name == "results.json" {
			if err := root.WriteFile(name, []byte("other writer"), 0600); err != nil {
				return err
			}
		}
		return writeReportFile(ctx, root, name, data)
	})
	if err == nil {
		t.Fatal("overwrote intervening file")
	}
	got, err := os.ReadFile(filepath.Join(dir, "results.json"))
	if err != nil || string(got) != "other writer" {
		t.Fatalf("intervening content changed: %q %v", got, err)
	}
	if _, err := os.Lstat(filepath.Join(dir, "evidence.json")); !os.IsNotExist(err) {
		t.Fatal("failed report has manifest")
	}
}

// spec: openspec/changes/benchmark-report-exporter/specs/benchmark-reports/spec.md#Requirement: Deterministic non-overwriting output and read-only checks
func TestReportFilesManifestLast(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "report") + string(filepath.Separator)
	var order []string
	err := writeReportFilesWith(context.Background(), dir, reportFileFixture(), func(ctx context.Context, root *os.Root, name string, data []byte) error {
		order = append(order, name)
		return writeReportFile(ctx, root, name, data)
	})
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(order) != "[results.json results.md evidence.json]" {
		t.Fatalf("order=%v", order)
	}
	if err := checkReportFiles(context.Background(), dir, reportFileFixture()); err != nil {
		t.Fatal(err)
	}
}
