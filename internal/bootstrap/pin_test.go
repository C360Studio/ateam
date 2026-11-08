package bootstrap

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// spec: openspec/specs/harness/spec.md#Requirement: Doctor checks a declared ateam pin
func TestPinStatus(t *testing.T) {
	const hookPath = "/repo/.tools/bin/ateam"
	at := func(v string) HookBinary { return HookBinary{Path: hookPath, Version: v, Build: "module"} }
	absent := HookBinary{Version: "absent"}
	unreadable := HookBinary{Path: hookPath, Build: "unreadable"}
	for _, tc := range []struct {
		name     string
		file     *string // nil: no pin file
		checkout bool
		running  string
		hook     HookBinary
		wantPin  string
		want     []string
	}{
		{name: "no pin file", running: "v0.3.9", hook: at("v0.3.8")},
		{name: "both at the pin", file: ptr("v0.3.9\n"), running: "v0.3.9", hook: at("v0.3.9"), wantPin: "v0.3.9"},
		{name: "running differs", file: ptr("v0.3.9"), running: "v0.3.8", hook: at("v0.3.9"), wantPin: "v0.3.9",
			want: []string{"running ateam is v0.3.8, not the .agents/ateam-version pin v0.3.9"}},
		{name: "dev hook build against a release pin", file: ptr("v0.3.9"), running: "v0.3.9", hook: at("v0.3.9-8-gbd6941f"), wantPin: "v0.3.9",
			want: []string{"hook binary /repo/.tools/bin/ateam is v0.3.9-8-gbd6941f, not the .agents/ateam-version pin v0.3.9"}},
		{name: "both differ", file: ptr("v0.3.9"), running: "dev", hook: at("v0.3.7"), wantPin: "v0.3.9",
			want: []string{
				"running ateam is dev, not the .agents/ateam-version pin v0.3.9",
				"hook binary /repo/.tools/bin/ateam is v0.3.7, not the .agents/ateam-version pin v0.3.9",
			}},
		{name: "absent hook binary on a CI runner", file: ptr("v0.3.9"), running: "v0.3.9", hook: absent, wantPin: "v0.3.9"},
		{name: "unreadable hook binary", file: ptr("v0.3.9"), running: "v0.3.9", hook: unreadable, wantPin: "v0.3.9",
			want: []string{"hook binary /repo/.tools/bin/ateam reports no version, not the .agents/ateam-version pin v0.3.9"}},
		{name: "pre-release pin", file: ptr("v0.4.0-rc.1"), running: "v0.4.0-rc.1", hook: at("v0.4.0-rc.1"), wantPin: "v0.4.0-rc.1"},
		{name: "missing v", file: ptr("0.3.9"), running: "v0.3.9", hook: at("v0.3.9"), wantPin: "0.3.9",
			want: []string{`.agents/ateam-version holds "0.3.9", not one release such as v0.3.9`}},
		{name: "two parts", file: ptr("v0.3"), running: "v0.3", hook: at("v0.3"), wantPin: "v0.3",
			want: []string{`.agents/ateam-version holds "v0.3", not one release such as v0.3.9`}},
		{name: "non-numeric part", file: ptr("v0.3.x"), running: "v0.3.x", hook: at("v0.3.x"), wantPin: "v0.3.x",
			want: []string{`.agents/ateam-version holds "v0.3.x", not one release such as v0.3.9`}},
		{name: "empty part", file: ptr("v0..9"), running: "v0..9", hook: at("v0..9"), wantPin: "v0..9",
			want: []string{`.agents/ateam-version holds "v0..9", not one release such as v0.3.9`}},
		{name: "two lines", file: ptr("v0.3.9\nv0.3.8\n"), running: "v0.3.9", hook: at("v0.3.9"), wantPin: "v0.3.9\nv0.3.8",
			want: []string{`.agents/ateam-version holds "v0.3.9\nv0.3.8", not one release such as v0.3.9`}},
		{name: "empty pre-release", file: ptr("v0.3.9-"), running: "v0.3.9-", hook: at("v0.3.9-"), wantPin: "v0.3.9-",
			want: []string{`.agents/ateam-version holds "v0.3.9-", not one release such as v0.3.9`}},
		{name: "pre-release with hyphens and numbers", file: ptr("v1.0.0-alpha-2.0.x7"), running: "v1.0.0-alpha-2.0.x7", hook: at("v1.0.0-alpha-2.0.x7"), wantPin: "v1.0.0-alpha-2.0.x7"},
		{name: "pre-release with a slash", file: ptr("v0.4.0-rc/1"), running: "v0.4.0-rc/1", hook: at("v0.4.0-rc/1"), wantPin: "v0.4.0-rc/1",
			want: []string{`.agents/ateam-version holds "v0.4.0-rc/1", not one release such as v0.3.9`}},
		{name: "empty pre-release identifier", file: ptr("v0.4.0-rc..1"), running: "v0.4.0-rc..1", hook: at("v0.4.0-rc..1"), wantPin: "v0.4.0-rc..1",
			want: []string{`.agents/ateam-version holds "v0.4.0-rc..1", not one release such as v0.3.9`}},
		{name: "leading-zero pre-release number", file: ptr("v0.4.0-rc.01"), running: "v0.4.0-rc.01", hook: at("v0.4.0-rc.01"), wantPin: "v0.4.0-rc.01",
			want: []string{`.agents/ateam-version holds "v0.4.0-rc.01", not one release such as v0.3.9`}},
		{name: "leading-zero major", file: ptr("v01.2.3"), running: "v01.2.3", hook: at("v01.2.3"), wantPin: "v01.2.3",
			want: []string{`.agents/ateam-version holds "v01.2.3", not one release such as v0.3.9`}},
		{name: "build metadata", file: ptr("v0.3.9+meta"), running: "v0.3.9+meta", hook: at("v0.3.9+meta"), wantPin: "v0.3.9+meta",
			want: []string{`.agents/ateam-version holds "v0.3.9+meta", not one release such as v0.3.9`}},
		{name: "zero parts", file: ptr("v0.0.0"), running: "v0.0.0", hook: at("v0.0.0"), wantPin: "v0.0.0"},
		{name: "empty file", file: ptr("\n"), running: "v0.3.9", hook: at("v0.3.9"),
			want: []string{`.agents/ateam-version holds "", not one release such as v0.3.9`}},
		{name: "ignored in the ateam checkout", file: ptr("v0.3.9"), checkout: true, running: "dev", hook: at("v0.3.7")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			if tc.checkout {
				writeGoMod(t, root, ateamGoMod)
			}
			if tc.file != nil {
				if err := os.MkdirAll(filepath.Join(root, ".agents"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(root, ".agents", "ateam-version"), []byte(*tc.file), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			pin, got := PinStatus(root, tc.running, tc.hook)
			if pin != tc.wantPin || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("PinStatus = %q, %q; want %q, %q", pin, got, tc.wantPin, tc.want)
			}
		})
	}
}
