//go:build unix

package bench

import (
	"os"
	"syscall"
)

// Do not block if a regular input is replaced with a FIFO before descriptor checks.
const bundleReadFlags = os.O_RDONLY | syscall.O_NONBLOCK
