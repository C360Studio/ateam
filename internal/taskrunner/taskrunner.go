// Package taskrunner parses Taskfile.yml and executes validated task targets.
package taskrunner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Target describes a single task in Taskfile.yml.
type Target struct {
	Name string `json:"name"`
	Desc string `json:"desc"`
}

// ListTargets returns the sorted task targets in root. It asks the task CLI (`task --list-all --json`) so Task v3
// includes: are resolved, and falls back to scanning the root Taskfile.yml when the CLI is absent or fails. A canceled
// ctx is returned as an error rather than a reason to fall back.
func ListTargets(ctx context.Context, root string) ([]Target, error) {
	// Task searches ancestor directories when root has no Taskfile; require root's own so another project's tasks are
	// never listed or run.
	if _, err := os.Stat(filepath.Join(root, "Taskfile.yml")); err != nil {
		return nil, fmt.Errorf("read Taskfile.yml: %w", err)
	}
	out, err := listWithTask(ctx, root)
	if err == nil {
		return out, nil
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return scanTaskfile(root)
}

func listWithTask(ctx context.Context, root string) ([]Target, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "task", "--taskfile", "Taskfile.yml", "--list-all", "--json")
	cmd.Dir = root
	// A descendant that inherits stdout would otherwise hold Output open past the deadline.
	cmd.WaitDelay = time.Second
	raw, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	// name is Task's display label (a task's label: when set); task is the key that `task <target>` accepts.
	var listed struct {
		Tasks []struct {
			Task string `json:"task"`
			Desc string `json:"desc"`
		} `json:"tasks"`
	}
	if err := json.Unmarshal(raw, &listed); err != nil {
		return nil, fmt.Errorf("parse task --list-all --json: %w", err)
	}
	out := make([]Target, 0, len(listed.Tasks))
	for _, t := range listed.Tasks {
		out = append(out, Target{Name: t.Task, Desc: t.Desc})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// scanTaskfile parses Taskfile.yml in root using stdlib.
// shortcut: indentation-based fallback sees only top-level tasks: in the root Taskfile.yml, upgrade to a YAML AST if offline listing must resolve includes:
func scanTaskfile(root string) ([]Target, error) {
	p := filepath.Join(root, "Taskfile.yml")
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("read Taskfile.yml: %w", err)
	}
	tasks := parseTaskfileTargets(string(data))
	out := make([]Target, 0, len(tasks))
	for name, desc := range tasks {
		out = append(out, Target{Name: name, Desc: desc})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func parseTaskfileTargets(raw string) map[string]string {
	tasks := make(map[string]string)
	inTasks := false
	currentTask := ""
	lines := strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		if indent == 0 {
			inTasks = trimmed == "tasks:"
			currentTask = ""
			continue
		}
		if !inTasks {
			continue
		}
		if indent == 2 && strings.HasSuffix(trimmed, ":") {
			name := strings.TrimSpace(strings.TrimSuffix(trimmed, ":"))
			name = strings.Trim(name, `"'`)
			if name != "" {
				currentTask = name
				tasks[currentTask] = ""
			}
			continue
		}
		if indent == 4 && currentTask != "" && strings.HasPrefix(trimmed, "desc:") {
			desc := strings.TrimSpace(strings.TrimPrefix(trimmed, "desc:"))
			desc = strings.Trim(desc, `"'`)
			tasks[currentTask] = desc
		}
	}
	return tasks
}

// Run executes a validated target defined in Taskfile.yml using the task CLI.
func Run(ctx context.Context, root, target string, extraArgs []string) (string, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", fmt.Errorf("target name is required")
	}
	targets, err := ListTargets(ctx, root)
	if err != nil {
		return "", err
	}
	allowed := false
	for _, t := range targets {
		if t.Name == target {
			allowed = true
			break
		}
	}
	if !allowed {
		return "", fmt.Errorf("unknown task target %q (not in Taskfile.yml)", target)
	}

	cctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	args := []string{"--taskfile", "Taskfile.yml", target}
	if len(extraArgs) > 0 {
		args = append(args, "--")
		args = append(args, extraArgs...)
	}
	cmd := exec.CommandContext(cctx, "task", args...)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("task %s failed: %w\n%s", target, err, string(out))
	}
	return string(out), nil
}
