package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const ateamGoMod = "module github.com/c360studio/ateam\n\ngo 1.26\n"

// wantRelinkAdvice is the oracle for main-checkout advice over paths without a single quote.
func wantRelinkAdvice(checkout, p string) string {
	return "replace " + p + ": (cd '" + checkout + "' && task build) && ln -sf '" + checkout + "/bin/ateam' '" + p +
		"', or (cd '" + checkout + "' && GOBIN='" + filepath.Dir(p) + "' go install ./cmd/ateam)"
}

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
	for _, tc := range []struct {
		name        string
		checkout    bool
		pathLine    string // "" leaves PATH without ateam; "link" symlinks PATH ateam to ./bin/ateam
		absHook     bool   // the Claude hook names an absolute path outside PATH
		hookArgv0   string // a hook argv[0] ("{root}" is replaced) that must resolve to root/wantRel
		wantRel     string // default bin/ateam
		decoy       bool   // a different ateam on PATH, so a fallback to PATH shows
		noDisk      bool   // no ./bin/ateam
		wantVersion string
		wantBuild   string
		wantWarning func(root, path string) string
	}{
		{name: "differs in the checkout", checkout: true, pathLine: "ateam v0.3.3 (def5678)", wantVersion: "v0.3.3", wantBuild: "def5678",
			wantWarning: func(root, p string) string {
				return "hook binary " + p + " is build def5678, not ./bin/ateam build abc1234; " + wantRelinkAdvice(root, p)
			}},
		{name: "module install in the checkout", checkout: true, pathLine: "ateam v0.3.3 (module)", wantVersion: "v0.3.3", wantBuild: "module",
			wantWarning: func(root, p string) string {
				return "hook binary " + p + " is v0.3.3 (module install), not ./bin/ateam build abc1234; " + wantRelinkAdvice(root, p)
			}},
		{name: "symlink to ./bin/ateam matches", checkout: true, pathLine: "link", wantVersion: "v0.3.4", wantBuild: "abc1234"},
		{name: "differs outside the checkout", pathLine: "ateam v0.3.3 (def5678)", wantVersion: "v0.3.3", wantBuild: "def5678"},
		{name: "absent", checkout: true, wantVersion: "absent"},
		{name: "absolute hook path is read from the Claude config", checkout: true, pathLine: "ateam v0.3.3 (def5678)", absHook: true, wantVersion: "v0.3.3", wantBuild: "fff0000",
			wantWarning: func(root, p string) string {
				return "hook binary " + p + " is build fff0000, not ./bin/ateam build abc1234; " + wantRelinkAdvice(root, p)
			}},
		{name: "no ./bin/ateam in the checkout", checkout: true, noDisk: true, pathLine: "ateam v0.3.3 (def5678)", wantVersion: "v0.3.3", wantBuild: "def5678",
			wantWarning: func(root, p string) string {
				return "hook binary " + p + " is build def5678, and there is no ./bin/ateam; " + wantRelinkAdvice(root, p)
			}},
		{name: "bare CLAUDE_PROJECT_DIR", checkout: true, hookArgv0: "$CLAUDE_PROJECT_DIR/bin/ateam", wantVersion: "v0.3.4", wantBuild: "abc1234"},
		{name: "quoted CLAUDE_PROJECT_DIR", checkout: true, hookArgv0: `"$CLAUDE_PROJECT_DIR"/bin/ateam`, wantVersion: "v0.3.4", wantBuild: "abc1234"},
		{name: "braced CLAUDE_PROJECT_DIR", checkout: true, hookArgv0: "${CLAUDE_PROJECT_DIR}/bin/ateam", wantVersion: "v0.3.4", wantBuild: "abc1234"},
		{name: "dot-relative hook path", checkout: true, hookArgv0: "./bin/ateam", wantVersion: "v0.3.4", wantBuild: "abc1234"},
		{name: "relative hook path", checkout: true, hookArgv0: "bin/ateam", wantVersion: "v0.3.4", wantBuild: "abc1234"},
		{name: "unquoted control with a PATH decoy", checkout: true, decoy: true, hookArgv0: "$CLAUDE_PROJECT_DIR/bin/ateam", wantVersion: "v0.3.4", wantBuild: "abc1234"},
		{name: "fully quoted bare CLAUDE_PROJECT_DIR", checkout: true, decoy: true, hookArgv0: `"$CLAUDE_PROJECT_DIR/bin/ateam"`, wantVersion: "v0.3.4", wantBuild: "abc1234"},
		{name: "fully quoted braced CLAUDE_PROJECT_DIR", checkout: true, decoy: true, hookArgv0: `"${CLAUDE_PROJECT_DIR}/bin/ateam"`, wantVersion: "v0.3.4", wantBuild: "abc1234"},
		{name: "double-quoted path with spaces", checkout: true, decoy: true, hookArgv0: `"$CLAUDE_PROJECT_DIR/my tools/ateam"`, wantRel: "my tools/ateam", wantVersion: "v0.3.4", wantBuild: "abc1234"},
		{name: "single-quoted absolute path with spaces", checkout: true, decoy: true, hookArgv0: "'{root}/my tools/ateam'", wantRel: "my tools/ateam", wantVersion: "v0.3.4", wantBuild: "abc1234"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			pathDir := t.TempDir()
			t.Setenv("PATH", pathDir)
			if tc.checkout {
				writeGoMod(t, root, ateamGoMod)
			}
			if !tc.noDisk {
				writeVersionFake(t, filepath.Join(root, "bin", "ateam"), "ateam v0.3.4 (abc1234)")
			}
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
			if tc.decoy {
				writeVersionFake(t, filepath.Join(pathDir, "ateam"), "ateam v9.9.9 (bad0000)")
			}
			if tc.hookArgv0 != "" {
				rel := "bin/ateam"
				if tc.wantRel != "" {
					rel = tc.wantRel
					writeVersionFake(t, filepath.Join(root, filepath.FromSlash(rel)), "ateam v0.3.4 (abc1234)")
				}
				wantPath = filepath.Join(root, filepath.FromSlash(rel))
				argv0 = strings.ReplaceAll(tc.hookArgv0, "{root}", root)
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
				wantWarnings = []string{tc.wantWarning(root, wantPath)}
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
	advice := wantRelinkAdvice(root, bin)

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

// spec: openspec/specs/harness/spec.md#Requirement: Build provenance names the binary the hooks run
func TestStaleBinaryAdviceNeverLinksIntoAWorktree(t *testing.T) {
	base := t.TempDir()
	mainDir := filepath.Join(base, "main")
	git := func(dir string, args ...string) string {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-c", "user.name=T", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false"}, args...)...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	if err := os.MkdirAll(mainDir, 0o755); err != nil {
		t.Fatal(err)
	}
	git(mainDir, "init", "-q", "-b", "main")
	writeGoMod(t, mainDir, ateamGoMod)
	git(mainDir, "add", "go.mod")
	git(mainDir, "commit", "-q", "-m", "c1")
	wt := filepath.Join(mainDir, ".worktrees", "wt")
	git(mainDir, "worktree", "add", "-q", "-b", "claude/x", wt)
	// Oracle: git's own answer for the linked worktree's main checkout.
	common := filepath.Dir(git(wt, "rev-parse", "--path-format=absolute", "--git-common-dir"))
	writeVersionFake(t, filepath.Join(mainDir, "bin", "ateam"), "ateam v0.3.4 (abc1234)")
	link := filepath.Join(base, "path", "ateam")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(mainDir, "bin", "ateam"), link); err != nil {
		t.Fatal(err)
	}
	const opt = "/opt/bin/ateam"
	fromWorktree := func(p string) string { return wantRelinkAdvice(common, p) }
	for _, tc := range []struct {
		name, root, binPath, want string
	}{
		{"main checkout, foreign binary", mainDir, opt, wantRelinkAdvice(mainDir, opt)},
		{"binary not named ateam has no go install", mainDir, "/opt/bin/ateam-old",
			"replace /opt/bin/ateam-old: (cd '" + mainDir + "' && task build) && ln -sf '" + mainDir + "/bin/ateam' '/opt/bin/ateam-old'"},
		{"main checkout, PATH link to its ./bin/ateam", mainDir, link, "run task build"},
		{"linked worktree, foreign binary", wt, opt, fromWorktree(opt)},
		{"linked worktree, PATH link to the main checkout", wt, link, fromWorktree(link)},
		{"linked worktree, main checkout's real build", wt, filepath.Join(mainDir, "bin", "ateam"),
			"rebuild " + filepath.Join(mainDir, "bin", "ateam") + " in its checkout: (cd '" + mainDir + "' && task build)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := staleBinaryAdvice(tc.root, tc.binPath); got != tc.want {
				t.Fatalf("staleBinaryAdvice = %q, want %q", got, tc.want)
			}
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Build provenance names the binary the hooks run
func TestHookConfigValidationAcceptsQuotedHookPaths(t *testing.T) {
	// Doctor's parity check and hook_binary read hook commands through one parser (hookWords). A command
	// the shell cannot parse must fail validation; /bin/sh -n is the independent oracle for which is which.
	var claude hookClient
	for _, c := range hookClients {
		if c.agent == "claude" {
			claude = c
		}
	}
	cases := []struct {
		name, format string // format takes the event name
		valid        bool
	}{
		{"quoted bare project dir", `"$CLAUDE_PROJECT_DIR/bin/ateam" hook --event %s --agent claude`, true},
		{"quoted braced project dir", `"${CLAUDE_PROJECT_DIR}/bin/ateam" hook --event %s --agent claude`, true},
		{"single-quoted path with spaces", `'/opt/my tools/ateam' hook --event %s --agent claude`, true},
		{"bare", `ateam hook --event %s --agent claude`, true},
		{"closed quoted agent", `ateam hook --event %s --agent "claude"`, true},
		{"unterminated opening double quote", `ateam hook --event %s --agent "claude`, false},
		{"unterminated trailing double quote", `ateam hook --event %s --agent claude"`, false},
		{"unterminated opening single quote", `'ateam hook --event %s --agent claude`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			byNative := map[string]any{}
			for _, ev := range claude.events {
				cmd := fmt.Sprintf(tc.format, ev.event)
				if err := exec.Command("/bin/sh", "-n", "-c", cmd).Run(); (err == nil) != tc.valid {
					t.Fatalf("oracle: sh -n on %q gave err=%v, want valid=%v", cmd, err, tc.valid)
				}
				byNative[ev.native] = []any{map[string]any{"hooks": []any{map[string]any{"type": "command", "command": cmd}}}}
			}
			data, err := json.Marshal(map[string]any{claude.root: byNative})
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "settings.json")
			if err := os.WriteFile(path, data, 0o644); err != nil {
				t.Fatal(err)
			}
			_, violations := hookFileViolations(hookFile{label: ".claude/settings.json", path: path}, claude)
			if tc.valid && len(violations) != 0 {
				t.Fatalf("violations for valid %s: %v", tc.name, violations)
			}
			if !tc.valid && len(violations) == 0 {
				t.Fatalf("malformed %s accepted with no violations", tc.name)
			}
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Build provenance names the binary the hooks run
func TestStaleBinaryAdviceRunsWithHostilePaths(t *testing.T) {
	// Spaces, $, backticks, and a single quote in every path the advice interpolates must stay literal.
	base := t.TempDir()
	checkout := filepath.Join(base, "my checkout $HOME `false` it's")
	dest := filepath.Join(base, "bin dir $(false) `x`", "ateam")
	for _, dir := range []string{filepath.Join(checkout, "bin"), filepath.Dir(dest)} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(checkout, "bin", "ateam"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dest, []byte("stale"), 0o755); err != nil {
		t.Fatal(err)
	}
	fakeDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(fakeDir, "task"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", fakeDir+string(os.PathListSeparator)+"/bin"+string(os.PathListSeparator)+"/usr/bin")

	advice := staleBinaryAdvice(checkout, dest)
	relink, ok := strings.CutPrefix(advice, "replace "+dest+": ")
	if !ok {
		t.Fatalf("advice = %q, want it to start with replace %s", advice, dest)
	}
	relink, install, ok := strings.Cut(relink, ", or ")
	if !ok || !strings.Contains(install, " go install ./cmd/ateam") {
		t.Fatalf("advice = %q, want a relink and a go install alternative", advice)
	}
	if out, err := exec.Command("/bin/sh", "-c", relink).CombinedOutput(); err != nil {
		t.Fatalf("sh -c %q: %v\n%s", relink, err, out)
	}
	target, err := os.Readlink(dest)
	if err != nil {
		t.Fatalf("%s is not a symlink after the advice ran: %v", dest, err)
	}
	if want := filepath.Join(checkout, "bin", "ateam"); target != want {
		t.Fatalf("link target = %q, want %q", target, want)
	}
	// The install alternative writes to dest's directory: print what GOBIN and cd receive, not run go.
	probe := strings.Replace(install, "go install ./cmd/ateam", `sh -c 'printf "%s\n" "$GOBIN" "$PWD"'`, 1)
	out, err := exec.Command("/bin/sh", "-c", probe).Output()
	if err != nil {
		t.Fatalf("sh -c %q: %v", probe, err)
	}
	gotDir, gotPWD, _ := strings.Cut(strings.TrimSuffix(string(out), "\n"), "\n")
	if gotDir != filepath.Dir(dest) {
		t.Fatalf("GOBIN = %q, want %q", gotDir, filepath.Dir(dest))
	}
	if resolved, _ := filepath.EvalSymlinks(checkout); gotPWD != checkout && gotPWD != resolved {
		t.Fatalf("go install runs in %q, want %q", gotPWD, checkout)
	}
}
