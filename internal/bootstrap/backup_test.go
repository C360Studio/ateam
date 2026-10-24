package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

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
	for _, rel := range []string{"AGENTS.md", "CLAUDE.md", ".agents/protocol.md", ".agents/rules-index.md"} {
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
	root := t.TempDir()
	hand := "# My project\n\nNo tabs.\n"
	earlier := "an earlier backup\n"
	writeRel(t, root, "AGENTS.md", hand)
	writeRel(t, root, "AGENTS.md.ateam-backup-20261024-130509", earlier)

	_, err := InitRepo(context.Background(), root, Options{Force: true, now: fixedClock()})
	if err == nil {
		t.Fatal("InitRepo succeeded although the backup path already existed")
	}
	if got := readRel(t, root, "AGENTS.md"); got != hand {
		t.Fatalf("AGENTS.md replaced after failed backup: %q", got)
	}
	if got := readRel(t, root, "AGENTS.md.ateam-backup-20261024-130509"); got != earlier {
		t.Fatalf("existing backup overwritten: %q", got)
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
