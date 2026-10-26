//go:build unix

package bench

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// hungProbeFixture makes the first version probe that resolving e runs never
// exit on its own. With signal set, the probe writes to a FIFO before hanging
// and the returned channel closes once it has; otherwise the channel is nil.
func hungProbeFixture(t *testing.T, role string, signal bool) (Experiment, <-chan struct{}, string) {
	t.Helper()
	dir := t.TempDir()
	var e Experiment
	var path, wantErr string
	switch role {
	case "bridge":
		e, wantErr = mercuryFixture(t), "probe Mercury bridge version"
		path = e.Claude.Mercury.Bridge
	case "native":
		e, wantErr = claudeRunFixture(t, "complete"), "probe claude version"
		path = e.Command[0]
	case "grader":
		e, wantErr = testExperiment(), "identify grader"
		path = filepath.Join(dir, "go")
		t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	default:
		t.Fatalf("unknown role %q", role)
	}
	script := "#!/bin/sh\nexec sleep 60\n"
	var started chan struct{}
	if signal {
		fifo := filepath.Join(dir, "started")
		if err := syscall.Mkfifo(fifo, 0600); err != nil {
			t.Fatal(err)
		}
		script = "#!/bin/sh\necho started > '" + fifo + "'\nexec sleep 60\n"
		started = make(chan struct{})
		go func() {
			defer close(started)
			// Opening a FIFO for reading blocks until the probe opens it to write.
			f, err := os.Open(fifo)
			if err != nil {
				return
			}
			_, _ = io.Copy(io.Discard, f)
			_ = f.Close()
		}()
	}
	if err := os.WriteFile(path, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	return e, started, wantErr
}

// A hung --version must end when the caller cancels (Ctrl-C reaches the run
// context, not the probe's own process group) and, absent cancellation, at the
// probe bound. Not parallel: subtests set the package-level bound.
// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Controlled bounded experiments
// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Mercury through the existing Claude runtime
func TestVersionProbeStopsOnCancelAndBound(t *testing.T) {
	saved := versionProbeTimeout
	t.Cleanup(func() { versionProbeTimeout = saved })
	for _, role := range []string{"bridge", "native", "grader"} {
		for _, stop := range []string{"cancel", "bound"} {
			t.Run(role+"/"+stop, func(t *testing.T) {
				e, started, wantErr := hungProbeFixture(t, role, stop == "cancel")
				// An hour leaves cancellation as the only way out; the short
				// bound is the only way out when nothing cancels.
				versionProbeTimeout = time.Hour
				if stop == "bound" {
					versionProbeTimeout = 100 * time.Millisecond
				}
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				done := make(chan error, 1)
				go func() {
					_, err := ResolveExperimentContext(ctx, e, ".")
					done <- err
				}()
				if started != nil {
					select {
					case <-started:
					case err := <-done:
						t.Fatalf("resolve returned before the %s probe started: %v", role, err)
					}
					cancel()
				}
				select {
				case err := <-done:
					if err == nil || !strings.Contains(err.Error(), wantErr) {
						t.Fatalf("hung %s probe: got %v, want %q", role, err, wantErr)
					}
				case <-time.After(20 * time.Second):
					t.Fatalf("hung %s probe outlived the %s", role, stop)
				}
			})
		}
	}
}

// #186 measured a trivial --version at up to 5.6s on a loaded race-enabled
// run, and the former 5s bound flaked. Keep at least double that headroom.
// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Controlled bounded experiments
func TestVersionProbeBoundCoversMeasuredStartup(t *testing.T) {
	const measuredWorstStartup = 5600 * time.Millisecond
	if versionProbeTimeout < 2*measuredWorstStartup {
		t.Fatalf("version probe bound %v is under twice the measured %v startup", versionProbeTimeout, measuredWorstStartup)
	}
}
