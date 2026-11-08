package state

import "strings"

// IsReleaseTag reports whether s is a semver release tag: v<major>.<minor>.<patch> without leading zeros, with an
// optional -<pre-release> of dot-separated [0-9A-Za-z-] identifiers whose numeric ones carry no leading zero either.
func IsReleaseTag(s string) bool {
	core, ok := strings.CutPrefix(s, "v")
	if !ok {
		return false
	}
	core, pre, hasPre := strings.Cut(core, "-")
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return false
	}
	for _, p := range parts {
		if !isNumericIdent(p) {
			return false
		}
	}
	if !hasPre {
		return true
	}
	for _, id := range strings.Split(pre, ".") {
		if strings.Trim(id, "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz-") != "" {
			return false
		}
		// All digits, or empty: an empty identifier fails here too.
		if strings.Trim(id, "0123456789") == "" && !isNumericIdent(id) {
			return false
		}
	}
	return true
}

// isNumericIdent reports whether s is a semver numeric identifier: digits, and no leading zero unless it is "0".
func isNumericIdent(s string) bool {
	return s != "" && strings.Trim(s, "0123456789") == "" && (s == "0" || s[0] != '0')
}
