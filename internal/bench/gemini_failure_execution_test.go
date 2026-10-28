package bench

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// spec: openspec/specs/benchmark-experiments/spec.md#Requirement: Gemini fatal stderr evidence
func TestGeminiTurnLimitExecutionGuards(t *testing.T) {
	for _, tc := range []struct {
		name, stdout              string
		code                      int
		overflow, timeout, cancel bool
	}{
		{name: "other-exit", code: 1}, {name: "successful-exit", code: 0},
		{name: "stdout-present", stdout: "{}", code: 53}, {name: "stdout-whitespace", stdout: " ", code: 53},
		{name: "truncated", code: 53, overflow: true}, {name: "timeout", code: 53, timeout: true}, {name: "cancelled", code: 53, cancel: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := nativeTestExperiment(t, `null`)
			stderr := turnLimitFixture("Reached max session turns.")
			if tc.overflow {
				stderr += "\n" + strings.Repeat("x", maxAdapterBytes)
			}
			// Write fixture text as a here-document, not executable shell interpolation.
			// The prewarm guard pays macOS's first-exec check on this new file outside the adapter deadline (#234).
			script := "#!/bin/sh\n[ -n \"$ATEAM_FAKE_PREWARM\" ] && exit 0\ncat >/dev/null\ncat >&2 <<'ATEAM_FIXTURE'\n" + stderr + "\nATEAM_FIXTURE\n"
			if tc.stdout != "" {
				script += "printf '%s' '" + tc.stdout + "'\n"
			}
			if tc.timeout {
				e.TimeoutSeconds = 1
				script += "sleep 3\n"
			}
			script += fmt.Sprintf("exit %d\n", tc.code)
			if err := os.WriteFile(e.Command[0], []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			prewarm := exec.Command(e.Command[0])
			prewarm.Env = append(os.Environ(), "ATEAM_FAKE_PREWARM=1")
			if out, err := prewarm.CombinedOutput(); err != nil {
				t.Fatalf("prewarm fixture: %v\n%s", err, out)
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if tc.cancel {
				cancel()
			}
			response, _, _, status, err := executeGemini(ctx, e, newAdapterRequest(e, Schedule(e)[0]), t.TempDir())
			if err == nil || response.Native == nil || response.Native.GeminiFailure != nil || response.Usage != nil {
				t.Fatalf("execution guard allowed fatal classification: status=%s evidence=%+v error=%v", status, response.Native, err)
			}
			want := "adapter-error"
			if tc.timeout {
				want = "timeout"
			}
			if tc.cancel {
				want = "cancelled"
			}
			if status != want {
				t.Fatalf("got status=%s want=%s", status, want)
			}
		})
	}
}
