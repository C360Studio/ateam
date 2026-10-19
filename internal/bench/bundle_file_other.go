//go:build !unix

package bench

import "os"

const bundleReadFlags = os.O_RDONLY
