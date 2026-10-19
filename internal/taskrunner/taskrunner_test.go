package taskrunner

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// spec: openspec/specs/harness/spec.md#Requirement: Task target discovery and validation
func TestListTargetsAndValidation(t *testing.T) {
	dir := t.TempDir()
	tf := `version: '3'
tasks:
  fmt:
    desc: Format Go code
    cmds:
      - gofmt -w .
  verify:
    desc: Run all checks
    cmds:
      - go test ./...
`
	if err := os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte(tf), 0o644); err != nil {
		t.Fatal(err)
	}
	targets, err := ListTargets(dir)
	if err != nil {
		t.Fatalf("ListTargets: %v", err)
	}
	if len(targets) != 2 || targets[0].Name != "fmt" || targets[1].Name != "verify" {
		t.Fatalf("unexpected targets: %+v", targets)
	}

	_, err = Run(context.Background(), dir, "nonexistent", nil)
	if err == nil || !strings.Contains(err.Error(), "unknown task target") {
		t.Fatalf("expected unknown task target error, got: %v", err)
	}
}
