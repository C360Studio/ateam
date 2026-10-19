//go:build unix

package bench

import (
	"os/exec"
	"syscall"
	"time"
)

// configureBenchmarkProcess bounds pipe waits and cancels the whole process group.
func configureBenchmarkProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.WaitDelay = time.Second
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
}

func cleanupBenchmarkProcess(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
