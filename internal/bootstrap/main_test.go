package bootstrap

import (
	"os"
	"testing"
)

// TestMain clears ATEAM_HOOK_SCOPE so a developer's exported knob neither fails the parity tests nor lets
// doctor read their real home directory; with the knob unset doctor reads no user-scope file, and tests of
// the knob set it, and HOME, themselves.
func TestMain(m *testing.M) {
	if err := os.Unsetenv(hookScopeEnv); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}
