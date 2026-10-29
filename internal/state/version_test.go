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

// spec: openspec/specs/harness/spec.md#Requirement: Build provenance names the binary the hooks run
func TestBuildRevisionLabelsModuleInstalls(t *testing.T) {
	withMain := func(v string, settings ...debug.BuildSetting) *debug.BuildInfo {
		return &debug.BuildInfo{Main: debug.Module{Path: "github.com/c360studio/ateam", Version: v}, Settings: settings}
	}
	vcs := []debug.BuildSetting{{Key: "vcs.revision", Value: "9f11a42c0ffee9f11a42c0ffee9f11a42c0ffee0"}, {Key: "vcs.modified", Value: "true"}}
	cases := []struct {
		name                string
		stamped             string
		info                *debug.BuildInfo
		ok                  bool
		wantShort, wantFull string
		wantVersion         string
	}{
		{"go install module@tag", "", withMain("v0.3.3"), true, "v0.3.3", "v0.3.3", "ateam v0.3.3 (module)"},
		{"go install pre-release tag", "", withMain("v0.3.4-rc1"), true, "v0.3.4-rc1", "v0.3.4-rc1", "ateam v0.3.4-rc1 (module)"},
		{"go install pseudo-version", "", withMain("v0.3.4-0.20261028123456-abcdef123456"), true, "abcdef1", "abcdef123456", "ateam v0.3.4-0.20261028123456-abcdef123456 (abcdef1)"},
		{"pre-release pseudo-version", "", withMain("v0.3.4-rc1.0.20261028123456-0123456789ab"), true, "0123456", "0123456789ab", "ateam v0.3.4-rc1.0.20261028123456-0123456789ab (0123456)"},
		{"go build or go run", "", withMain("(devel)"), true, "unknown", "", "ateam dev (unknown)"},
		{"vcs stamp wins over the module tag", "", withMain("v0.3.3", vcs...), true, "9f11a42", "9f11a42c0ffee9f11a42c0ffee9f11a42c0ffee0", "ateam v0.3.3 (9f11a42-dirty)"},
		{"ldflags commit wins over everything", "ea0c7845a875", withMain("v0.3.3", vcs...), true, "ea0c784", "ea0c7845a875", "ateam v0.3.3 (ea0c784)"},
		{"no build info", "", nil, false, "unknown", "", "ateam dev (unknown)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			short, full, _ := buildRevision(tc.stamped, tc.info, tc.ok)
			if short != tc.wantShort || full != tc.wantFull {
				t.Fatalf("buildRevision = (%q, %q), want (%q, %q)", short, full, tc.wantShort, tc.wantFull)
			}
			if got := versionString("", tc.stamped, tc.info, tc.ok); got != tc.wantVersion {
				t.Fatalf("versionString = %q, want %q", got, tc.wantVersion)
			}
		})
	}
}
