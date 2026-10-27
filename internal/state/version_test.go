package state

import (
	"runtime/debug"
	"testing"
)

// spec: openspec/specs/harness/spec.md#Requirement: Two-tool MCP context budget
func TestReleaseVersionResolution(t *testing.T) {
	withMain := func(v string) *debug.BuildInfo {
		return &debug.BuildInfo{Main: debug.Module{Path: "github.com/c360studio/ateam", Version: v}}
	}
	cases := []struct {
		name    string
		stamped string
		info    *debug.BuildInfo
		ok      bool
		want    string
	}{
		{"ldflags stamp wins over module tag", "v9.9.9", withMain("v0.3.3"), true, "v9.9.9"},
		{"module tag from go install or a tagged build", "", withMain("v0.3.3"), true, "v0.3.3"},
		{"pseudo-version between tags", " ", withMain("v0.3.3-0.20261027190805-fd9d205be754"), true, "v0.3.3-0.20261027190805-fd9d205be754"},
		{"devel module version", "", withMain("(devel)"), true, "dev"},
		{"empty module version", "", withMain(""), true, "dev"},
		{"no build info", "", nil, false, "dev"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := releaseVersion(tc.stamped, tc.info, tc.ok); got != tc.want {
				t.Fatalf("releaseVersion(%q) = %q, want %q", tc.stamped, got, tc.want)
			}
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Two-tool MCP context budget
func TestVersionStringUsesStampedVersion(t *testing.T) {
	orig := Version
	t.Cleanup(func() { Version = orig })
	Version = "v9.9.9"
	if got, want := VersionString(), "ateam v9.9.9 ("+BuildLabel()+")"; got != want {
		t.Fatalf("VersionString() = %q, want %q", got, want)
	}
}
