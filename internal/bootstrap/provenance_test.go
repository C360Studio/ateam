package bootstrap

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const ateamGoMod = "module github.com/c360studio/ateam\n\ngo 1.26\n"

// writeVersionFake writes an executable /bin/sh fake printing line and runs it once outside any deadline
// with ATEAM_FAKE_PREWARM set, so macOS's first-exec check is paid before the code under test runs (#232).
func writeVersionFake(t *testing.T, path, line string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\n[ -n \"$ATEAM_FAKE_PREWARM\" ] && exit 0\necho \"$FAKE_LINE\"\n"
	script = strings.Replace(script, "$FAKE_LINE", line, 1)
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(path)
	cmd.Env = append(os.Environ(), "ATEAM_FAKE_PREWARM=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("prewarm %s: %v\n%s", path, err, out)
	}
}

func writeClaudeHook(t *testing.T, root, argv0 string) {
	t.Helper()
	cmd := argv0 + " hook --event session-start --agent claude"
	body := fmt.Sprintf(`{"hooks":{"SessionStart":[{"hooks":[{"type":"command","command":%q}]}]}}`, cmd)
	if err := os.MkdirAll(filepath.Join(root, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".claude", "settings.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Build provenance names the binary the hooks run
func TestHookBinaryStatusComparesPathBinaryWithDiskBuild(t *testing.T) {
	t.Setenv(hookScopeEnv, "")
	advice := func(p string) string {
		return fmt.Sprintf(`replace %s: task build && ln -sf "$PWD/bin/ateam" %s, or go install ./cmd/ateam`, p, p)
	}
	for _, tc := range []struct {
		name        string
		checkout    bool
		pathLine    string // "" leaves PATH without ateam; "link" symlinks PATH ateam to ./bin/ateam
		absHook     bool   // the Claude hook names an absolute path outside PATH
		wantVersion string
		wantBuild   string
		wantWarning func(path string) string
	}{
		{name: "differs in the checkout", checkout: true, pathLine: "ateam v0.3.3 (def5678)", wantVersion: "v0.3.3", wantBuild: "def5678",
			wantWarning: func(p string) string {
				return "hook binary " + p + " is build def5678, not ./bin/ateam build abc1234; " + advice(p)
			}},
		{name: "module install in the checkout", checkout: true, pathLine: "ateam v0.3.3 (module)", wantVersion: "v0.3.3", wantBuild: "module",
			wantWarning: func(p string) string {
				return "hook binary " + p + " is v0.3.3 (module install), not ./bin/ateam build abc1234; " + advice(p)
			}},
		{name: "symlink to ./bin/ateam matches", checkout: true, pathLine: "link", wantVersion: "v0.3.4", wantBuild: "abc1234"},
		{name: "differs outside the checkout", pathLine: "ateam v0.3.3 (def5678)", wantVersion: "v0.3.3", wantBuild: "def5678"},
		{name: "absent", checkout: true, wantVersion: "absent"},
		{name: "absolute hook path is read from the Claude config", checkout: true, pathLine: "ateam v0.3.3 (def5678)", absHook: true, wantVersion: "v0.3.3", wantBuild: "fff0000",
			wantWarning: func(p string) string {
				return "hook binary " + p + " is build fff0000, not ./bin/ateam build abc1234; " + advice(p)
			}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			pathDir := t.TempDir()
			t.Setenv("PATH", pathDir)
			if tc.checkout {
				writeGoMod(t, root, ateamGoMod)
			}
			writeVersionFake(t, filepath.Join(root, "bin", "ateam"), "ateam v0.3.4 (abc1234)")
			wantPath := ""
			switch tc.pathLine {
			case "":
			case "link":
				wantPath = filepath.Join(pathDir, "ateam")
				if err := os.Symlink(filepath.Join(root, "bin", "ateam"), wantPath); err != nil {
					t.Fatal(err)
				}
			default:
				wantPath = filepath.Join(pathDir, "ateam")
				writeVersionFake(t, wantPath, tc.pathLine)
			}
			argv0 := "ateam"
			if tc.absHook {
				wantPath = filepath.Join(t.TempDir(), "ateam")
				writeVersionFake(t, wantPath, "ateam v0.3.3 (fff0000)")
				argv0 = wantPath
			}
			writeClaudeHook(t, root, argv0)

			diskBuild, _ := DiskBuildStatus(context.Background(), root)
			hb, warnings := HookBinaryStatus(context.Background(), root, diskBuild)
			want := HookBinary{Path: wantPath, Version: tc.wantVersion, Build: tc.wantBuild}
			if hb != want {
				t.Fatalf("hook_binary = %+v, want %+v", hb, want)
			}
			var wantWarnings []string
			if tc.wantWarning != nil {
				wantWarnings = []string{tc.wantWarning(wantPath)}
			}
			if fmt.Sprint(warnings) != fmt.Sprint(wantWarnings) {
				t.Fatalf("warnings = %q, want %q", warnings, wantWarnings)
			}
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Build provenance names the binary the hooks run
func TestDoctorResolvesModuleTagToCommit(t *testing.T) {
	root := t.TempDir()
	writeGoMod(t, root, ateamGoMod)
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-c", "user.name=T", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false", "-c", "tag.gpgsign=false"}, args...)...)
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-q", "-b", "main")
	git("commit", "-q", "--allow-empty", "-m", "c1")
	git("tag", "-a", "-m", "release", "v0.3.3") // annotated: rev-parse must peel it to the commit
	git("commit", "-q", "--allow-empty", "-m", "c2")
	git("tag", "v0.3.4")
	tagShort := git("rev-parse", "--short=7", "v0.3.3^{commit}")
	head := git("rev-parse", "--short=7", "HEAD")
	const bin = "/opt/bin/ateam"
	advice := `replace /opt/bin/ateam: task build && ln -sf "$PWD/bin/ateam" /opt/bin/ateam, or go install ./cmd/ateam`

	for _, tc := range []struct {
		name, tag string
		want      []string
	}{
		{"tag behind HEAD", "v0.3.3", []string{"binary v0.3.3 (module install, " + tagShort + ") is behind HEAD " + head + "; " + advice}},
		{"tag at HEAD", "v0.3.4", nil},
		{"tag not in this checkout", "v9.9.9", []string{"binary v9.9.9 (module install) is not built from HEAD " + head + "; " + advice}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, got := BuildProvenanceStatus(context.Background(), root, tc.tag, head, bin)
			if fmt.Sprint(got) != fmt.Sprint(tc.want) {
				t.Fatalf("BuildProvenanceStatus(%s) = %q, want %q", tc.tag, got, tc.want)
			}
		})
	}
}
