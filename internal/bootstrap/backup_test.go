package bootstrap

import (
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

var harnessDocs = []string{"AGENTS.md", "CLAUDE.md", ".agents/protocol.md", ".agents/rules-index.md"}

func fixedClock() func() time.Time {
	return func() time.Time { return time.Date(2026, 10, 24, 13, 5, 9, 0, time.UTC) }
}

func writeRel(t *testing.T, root, rel, data string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readRel(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// spec: openspec/specs/harness/spec.md#Requirement: Agent initialization and configuration parity
func TestForceInitBacksUpModifiedHarnessDocuments(t *testing.T) {
	templates := defaultTemplateFiles()
	for _, rel := range harnessDocs {
		t.Run(rel, func(t *testing.T) {
			root := t.TempDir()
			hand := "# My project\n\nNo tabs. Run make test.\n"
			writeRel(t, root, rel, hand)

			res, err := InitRepo(context.Background(), root, Options{Force: true, now: fixedClock()})
			if err != nil {
				t.Fatal(err)
			}
			backup := rel + ".ateam-backup-20261024-130509"
			if !slices.Equal(res.BackedUp, []string{backup}) {
				t.Fatalf("BackedUp = %v, want [%s]", res.BackedUp, backup)
			}
			if got := readRel(t, root, backup); got != hand {
				t.Fatalf("backup bytes = %q, want %q", got, hand)
			}
			if got := readRel(t, root, rel); got != templates[rel] {
				t.Fatalf("%s was not replaced by the template", rel)
			}
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Agent initialization and configuration parity
func TestForceInitSkipsBackupOfUnmodifiedTemplate(t *testing.T) {
	root := t.TempDir()
	if _, err := InitRepo(context.Background(), root, Options{}); err != nil {
		t.Fatal(err)
	}
	res, err := InitRepo(context.Background(), root, Options{Force: true, now: fixedClock()})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.BackedUp) != 0 {
		t.Fatalf("BackedUp = %v, want none for unmodified templates", res.BackedUp)
	}
	if got, err := json.Marshal(res.BackedUp); err != nil || string(got) != "[]" {
		t.Fatalf("BackedUp JSON = %s (%v), want []", got, err)
	}
	matches, err := filepath.Glob(filepath.Join(root, "*.ateam-backup-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("unexpected backup files: %v", matches)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Agent initialization and configuration parity
func TestForceInitFailedBackupKeepsOriginal(t *testing.T) {
	for _, failing := range harnessDocs {
		t.Run(failing, func(t *testing.T) {
			root := t.TempDir()
			for _, rel := range harnessDocs {
				writeRel(t, root, rel, "# hand-written "+rel+"\n")
			}
			earlier := "an earlier backup\n"
			collision := failing + ".ateam-backup-20261024-130509"
			writeRel(t, root, collision, earlier)

			res, err := InitRepo(context.Background(), root, Options{Force: true, now: fixedClock()})
			if err == nil {
				t.Fatal("InitRepo succeeded although the backup path already existed")
			}
			for _, rel := range harnessDocs {
				if got := readRel(t, root, rel); got != "# hand-written "+rel+"\n" {
					t.Fatalf("%s replaced although a backup failed (%s): %q", rel, failing, got)
				}
			}
			if got := readRel(t, root, collision); got != earlier {
				t.Fatalf("existing backup overwritten: %q", got)
			}
			if len(res.Written) != 0 {
				t.Fatalf("files written although a backup failed: %v", res.Written)
			}
			if _, err := os.Stat(filepath.Join(root, ".agents", "README.md")); !os.IsNotExist(err) {
				t.Fatalf(".agents/README.md created although a backup failed: %v", err)
			}
			for _, b := range res.BackedUp {
				if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(b))); err != nil {
					t.Fatalf("reported backup %s missing: %v", b, err)
				}
			}
		})
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Agent initialization and configuration parity
func TestForceInitFailureAfterBackupReportsBackups(t *testing.T) {
	root := t.TempDir()
	hand := "# Mine\n"
	writeRel(t, root, "AGENTS.md", hand)
	// A regular file where a skill directory belongs makes a later mkdir fail.
	writeRel(t, root, ".agents/skills", "not a directory\n")

	res, err := InitRepo(context.Background(), root, Options{Force: true, now: fixedClock()})
	if err == nil {
		t.Fatal("InitRepo succeeded although .agents/skills is a file")
	}
	backup := "AGENTS.md.ateam-backup-20261024-130509"
	if !slices.Equal(res.BackedUp, []string{backup}) {
		t.Fatalf("BackedUp on failure = %v, want [%s]", res.BackedUp, backup)
	}
	if got := readRel(t, root, backup); got != hand {
		t.Fatalf("backup bytes = %q, want %q", got, hand)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Agent initialization and configuration parity
func TestForceInitSkipsBackupOfNonHarnessFiles(t *testing.T) {
	root := t.TempDir()
	if _, err := InitRepo(context.Background(), root, Options{}); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{".agents/skills/ponytail/SKILL.md", ".agents/README.md", "openspec/config.yaml"} {
		writeRel(t, root, rel, "# locally edited\n")
	}
	res, err := InitRepo(context.Background(), root, Options{Force: true, now: fixedClock()})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.BackedUp) != 0 {
		t.Fatalf("BackedUp = %v, want none for non-harness files", res.BackedUp)
	}
	err = filepath.WalkDir(root, func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if strings.Contains(path, ".ateam-backup-") {
			t.Errorf("unexpected backup %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: Agent initialization and configuration parity
func TestForceInitBackupLeavesClientConfigPreserved(t *testing.T) {
	root := t.TempDir()
	settings := `{"permissions":{"allow":["Bash(make test)"]}}`
	writeRel(t, root, ".claude/settings.json", settings)
	writeRel(t, root, "AGENTS.md", "# Mine\n")

	res, err := InitRepo(context.Background(), root, Options{Force: true, now: fixedClock()})
	if err != nil {
		t.Fatal(err)
	}
	if got := readRel(t, root, ".claude/settings.json"); got != settings {
		t.Fatalf("client config overwritten under --force: %q", got)
	}
	if !slices.Equal(res.BackedUp, []string{"AGENTS.md.ateam-backup-20261024-130509"}) {
		t.Fatalf("BackedUp = %v, want only the AGENTS.md backup", res.BackedUp)
	}
	if slices.Contains(res.Written, ".claude/settings.json") {
		t.Fatal("client config reported as written")
	}
}
