package bootstrap

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// PinPath is the tracked file an adopting repository declares its ateam release in, one line such as v0.3.9 (#334).
const PinPath = ".agents/ateam-version"

// PinStatus returns root's declared ateam pin and a doctor violation for each binary whose release differs from it
// by exact string, so a v0.3.9-8-g<sha> dev build fails a v0.3.9 pin. running is the running binary's release.
// An absent hook binary (a CI runner) is not a violation; an unreadable one is, because it still runs. Without the
// file, and in the ateam checkout, whose build provenance is checked against HEAD instead, pin is "".
func PinStatus(root, running string, hook HookBinary) (pin string, violations []string) {
	if IsAteamCheckout(root) {
		return "", nil
	}
	data, err := os.ReadFile(filepath.Join(root, PinPath))
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", []string{fmt.Sprintf("%s is unreadable: %v", PinPath, err)}
	}
	pin = strings.TrimSpace(string(data))
	if !isReleaseTag(pin) {
		return pin, []string{fmt.Sprintf("%s holds %q, not one release such as v0.3.9", PinPath, pin)}
	}
	if running != pin {
		violations = append(violations, fmt.Sprintf("running ateam is %s, not the %s pin %s", running, PinPath, pin))
	}
	switch {
	case hook.Version == "absent":
	case hook.Build == "unreadable":
		violations = append(violations, fmt.Sprintf("hook binary %s reports no version, not the %s pin %s", hook.Path, PinPath, pin))
	case hook.Version != pin:
		violations = append(violations, fmt.Sprintf("hook binary %s is %s, not the %s pin %s", hook.Path, hook.Version, PinPath, pin))
	}
	return pin, violations
}

// isReleaseTag reports whether s is v<major>.<minor>.<patch> with an optional -<pre-release> suffix.
func isReleaseTag(s string) bool {
	core, ok := strings.CutPrefix(s, "v")
	if !ok {
		return false
	}
	core, pre, hasPre := strings.Cut(core, "-")
	if hasPre && (pre == "" || strings.ContainsAny(pre, " \t\r\n")) {
		return false
	}
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return false
	}
	for _, p := range parts {
		if p == "" || strings.Trim(p, "0123456789") != "" {
			return false
		}
	}
	return true
}
