package bench

import (
	"context"
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

//go:embed bench.go process_unix.go process_other.go
var evaluatorSources embed.FS

// GraderIdentity records the local Go driver actually used with toolchain switching disabled.
type GraderIdentity struct {
	Path    string `json:"path"`
	SHA     string `json:"sha256"`
	Version string `json:"version"`
}

func currentGrader() (GraderIdentity, error) {
	var identity GraderIdentity
	path, err := exec.LookPath("go")
	if err != nil {
		return identity, err
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return identity, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return identity, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, path, "version")
	configureBenchmarkProcess(cmd)
	defer cleanupBenchmarkProcess(cmd)
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "GOENV=off", "GOTOOLCHAIN=local"}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return identity, fmt.Errorf("identify grader: %w", err)
	}
	return GraderIdentity{Path: path, SHA: digest(data), Version: strings.TrimSpace(string(output))}, nil
}

func evaluatorSourceDigest() string {
	var source strings.Builder
	for _, name := range []string{"bench.go", "process_unix.go", "process_other.go"} {
		data, _ := evaluatorSources.ReadFile(name)
		source.WriteString(name)
		source.Write(data)
	}
	return digest([]byte(source.String()))
}
