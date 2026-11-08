package bootstrap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTeamSettings(t *testing.T, root, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, ".agents"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".agents", "ateam.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// spec: openspec/changes/team-settings-333/specs/harness/spec.md#Requirement: Committed team settings file
func TestDoctorTeamSettings(t *testing.T) {
	t.Run("absent file reports nothing", func(t *testing.T) {
		root, _ := initWithPolicy(t, "")
		if v := CheckParity(root); len(v) != 0 {
			t.Fatalf("violations without .agents/ateam.json: %v", v)
		}
		if w := CheckMergeMethodDocs(root); len(w) != 0 {
			t.Fatalf("warnings without .agents/ateam.json: %v", w)
		}
	})
	t.Run("invalid file fails the check", func(t *testing.T) {
		root, _ := initWithPolicy(t, "")
		writeTeamSettings(t, root, `{"version":2}`)
		requireViolation(t, CheckParity(root), ".agents/ateam.json", "version must be 1")
	})
	t.Run("valid file passes the check", func(t *testing.T) {
		root, _ := initWithPolicy(t, "")
		writeTeamSettings(t, root, `{"version":1,"default_branch":"trunk"}`)
		if v := CheckParity(root); len(v) != 0 {
			t.Fatalf("violations for a valid file: %v", v)
		}
		if w := CheckMergeMethodDocs(root); len(w) != 0 {
			t.Fatalf("warned without merge_method: %v", w)
		}
	})
	t.Run("protocol naming another method warns", func(t *testing.T) {
		root, _ := initWithPolicy(t, "") // protocol written with the squash command
		writeTeamSettings(t, root, `{"version":1,"merge_method":"merge"}`)
		w := CheckMergeMethodDocs(root)
		if len(w) != 1 || !strings.Contains(w[0], "`gh pr merge --merge --auto`") || !strings.Contains(w[0], "ateam init --force") {
			t.Fatalf("got warnings %q, want one naming gh pr merge --merge --auto and ateam init --force", w)
		}
		if v := CheckParity(root); len(v) != 0 {
			t.Fatalf("a stale protocol is a warning, not a violation: %v", v)
		}
	})
	t.Run("protocol naming the method is quiet", func(t *testing.T) {
		root, _ := initWithPolicy(t, "")
		writeTeamSettings(t, root, `{"version":1,"merge_method":"squash"}`)
		if w := CheckMergeMethodDocs(root); len(w) != 0 {
			t.Fatalf("warned although protocol names gh pr merge --squash --auto: %v", w)
		}
	})
}
