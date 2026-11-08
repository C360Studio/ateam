package bootstrap

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/c360studio/ateam/internal/state"
)

// PinPath is the deprecated tracked file an adopting repository declares its ateam release in, one line such as
// v0.3.9 (#334); ateam_version in .agents/ateam.json replaces it (#341).
const PinPath = ".agents/ateam-version"

// PinStatus returns root's declared ateam pin, a doctor violation for each binary whose release differs from it
// by exact string, so a v0.3.9-8-g<sha> dev build fails a v0.3.9 pin, and warnings. running is the running binary's
// release. The pin is ateam_version in .agents/ateam.json, else the deprecated PinPath file; when both declare one
// and they differ, the key wins with a warning (#341). An invalid settings file is ignored here, because
// CheckParity already fails doctor on it. An absent hook binary (a CI runner) is not a violation; an unreadable one
// is, because it still runs. Without a pin, and in the ateam checkout, whose build provenance is checked against
// HEAD instead, pin is "".
func PinStatus(root, running string, hook HookBinary) (pin string, violations, warnings []string) {
	if IsAteamCheckout(root) {
		return "", nil, nil
	}
	settings, _, settingsErr := state.LoadSettings(root)
	key := ""
	if settingsErr == nil {
		key = settings.AteamVersion
	}
	filePin, fileExists, fileErr := readPinFile(root)
	source := PinPath
	switch {
	case key != "":
		pin, source = key, state.SettingsPath+" ateam_version"
		if fileExists && filePin != key {
			warnings = append(warnings, fmt.Sprintf("%s holds %q but %s ateam_version is %s; the key wins, so delete %s", PinPath, filePin, state.SettingsPath, key, PinPath))
		}
	case fileErr != nil:
		return "", []string{fileErr.Error()}, nil
	case !fileExists:
		return "", nil, nil
	default:
		pin = filePin
		if !state.IsReleaseTag(pin) {
			return pin, []string{fmt.Sprintf("%s holds %q, not one release such as v0.3.9", PinPath, pin)}, nil
		}
	}
	if running != pin {
		violations = append(violations, fmt.Sprintf("running ateam is %s, not the %s pin %s", running, source, pin))
	}
	switch {
	case hook.Version == "absent":
	case hook.Build == "unreadable":
		violations = append(violations, fmt.Sprintf("hook binary %s reports no version, not the %s pin %s", hook.Path, source, pin))
	case hook.Version != pin:
		violations = append(violations, fmt.Sprintf("hook binary %s is %s, not the %s pin %s", hook.Path, hook.Version, source, pin))
	}
	return pin, violations, warnings
}

// readPinFile returns the trimmed PinPath contents and whether the file exists.
func readPinFile(root string) (string, bool, error) {
	data, err := os.ReadFile(filepath.Join(root, PinPath))
	if errors.Is(err, fs.ErrNotExist) {
		return "", false, nil
	}
	if err != nil {
		return "", true, fmt.Errorf("%s is unreadable: %v", PinPath, err)
	}
	return strings.TrimSpace(string(data)), true, nil
}
