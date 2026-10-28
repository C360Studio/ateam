package hook

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// spec: openspec/specs/harness/spec.md#Requirement: Build provenance names the binary the hooks run
func TestSessionStartLabelsModuleInstallWithoutGit(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/c360studio/ateam\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A PATH git that records each run: the hook must label the tag without resolving it (#251, D1 A).
	fakeDir := t.TempDir()
	marker := filepath.Join(t.TempDir(), "git-ran")
	fake := filepath.Join(fakeDir, "git")
	script := "#!/bin/sh\n[ -n \"$ATEAM_FAKE_PREWARM\" ] && exit 0\necho \"$*\" >> \"$GIT_MARKER\"\nexit 1\n"
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	prewarm := exec.Command(fake)
	prewarm.Env = append(os.Environ(), "ATEAM_FAKE_PREWARM=1")
	if out, err := prewarm.CombinedOutput(); err != nil {
		t.Fatalf("prewarm %s: %v\n%s", fake, err, out)
	}
	t.Setenv("PATH", fakeDir)
	t.Setenv("GIT_MARKER", marker)
	// Activation: the fake is the git a subprocess would find, and it leaves the marker.
	_ = exec.Command("git", "probe").Run()
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("fake git did not record a run: %v", err)
	}
	if err := os.Remove(marker); err != nil {
		t.Fatal(err)
	}

	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	want := "binary v0.3.3 (module install) is not built from HEAD 6e8d841; replace " + exe +
		`: task build && ln -sf "$PWD/bin/ateam" ` + exe + ", or go install ./cmd/ateam"
	if got := sessionBuildWarning(dir, "6e8d841", "v0.3.3"); got != want {
		t.Fatalf("session-start warning = %q, want %q", got, want)
	}
	_ = checkSessionBuildProvenance(dir, "6e8d841")
	if data, err := os.ReadFile(marker); err == nil {
		t.Fatalf("session-start provenance check ran git: %s", data)
	}
}
