//go:build !unix

package bench

import (
	"os/exec"
	"time"
)

func configureBenchmarkProcess(cmd *exec.Cmd) {
	// shortcut: non-Unix platforms kill only the direct child; use a container/job-object adapter before untrusted workloads.
	cmd.WaitDelay = time.Second
}

func cleanupBenchmarkProcess(_ *exec.Cmd) {}
