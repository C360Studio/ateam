package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"

	"github.com/c360studio/ateam/internal/state"
)

// herdrRunner invokes the herdr binary; tests substitute a recorder.
type herdrRunner func(ctx context.Context, bin string, args ...string) ([]byte, error)

func execHerdr(ctx context.Context, bin string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, bin, args...).Output()
	if err != nil {
		return nil, fmt.Errorf("herdr %s: %w", args[0], err)
	}
	return out, nil
}

// herdrBin returns HERDR_BIN_PATH when running as a herdr plugin command, else "".
func herdrBin() string {
	if os.Getenv("HERDR_ENV") != "1" {
		return ""
	}
	return os.Getenv("HERDR_BIN_PATH")
}

func runStateHerdr(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: ateam state herdr open-pane <pickup|spec-queue> | on-event")
		return 1
	}
	ctx := context.Background()
	var err error
	switch args[0] {
	case "open-pane":
		if len(args) != 2 {
			fmt.Fprintln(os.Stderr, "usage: ateam state herdr open-pane <pickup|spec-queue>")
			return 1
		}
		err = herdrOpenPane(ctx, execHerdr, args[1])
	case "on-event":
		err = herdrOnEvent(ctx, execHerdr, os.Stdout)
	default:
		err = fmt.Errorf("unknown subcommand %q", args[0])
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "ateam state herdr: %v\n", err)
		return 1
	}
	return 0
}

// herdrOpenPane opens a pane declared in .agents/herdr/herdr-plugin.toml.
func herdrOpenPane(ctx context.Context, run herdrRunner, id string) error {
	if id != "pickup" && id != "spec-queue" {
		return fmt.Errorf("unknown pane %q (want pickup or spec-queue)", id)
	}
	bin := herdrBin()
	if bin == "" {
		return fmt.Errorf("not inside herdr (HERDR_ENV=1 and HERDR_BIN_PATH required); run `ateam state %s` instead", id)
	}
	// shortcut: placement comes from the manifest (the CLI flag has no popup) and the pane cwd
	// is the action's pane when HERDR_PANE_ID is set; confirm both in the 3.2 manual session.
	args := []string{"plugin", "pane", "open", "--plugin", "ateam", "--entrypoint", id, "--focus"}
	if ws := os.Getenv("HERDR_WORKSPACE_ID"); ws != "" {
		args = append(args, "--workspace", ws)
	}
	if paneID := os.Getenv("HERDR_PANE_ID"); paneID != "" {
		if cwd := herdrPaneCWD(ctx, run, bin, paneID); cwd != "" {
			args = append(args, "--cwd", cwd)
		}
	}
	_, err := run(ctx, bin, args...)
	return err
}

// herdrOnEvent handles worktree.created and pane.agent_status_changed plugin events: it toasts
// spec-queue caveats not yet notified for the repo and, on worktree.created, prints pickup.
// It runs as a herdr plugin command, never on the synchronous hook path.
func herdrOnEvent(ctx context.Context, run herdrRunner, out io.Writer) error {
	bin := herdrBin()
	if bin == "" {
		return nil
	}
	event, data := parseHerdrEvent()
	var dir string
	switch event {
	case "worktree.created":
		dir = data.Worktree.Path
	case "pane.agent_status_changed":
		dir = herdrPaneCWD(ctx, run, bin, data.PaneID)
	default:
		return nil
	}
	if dir == "" {
		return nil
	}
	if abs, err := filepath.Abs(dir); err == nil {
		dir = abs
	}
	if resolved, err := filepath.EvalSymlinks(dir); err == nil {
		dir = resolved
	}
	repo := findRepoRoot(dir)
	if info, err := os.Stat(filepath.Join(repo, "openspec")); err != nil || !info.IsDir() {
		return nil
	}
	if event == "worktree.created" {
		st, err := state.Refresh(ctx, repo, false)
		if err != nil {
			return fmt.Errorf("pickup %s: %w", repo, err)
		}
		fmt.Fprintln(out, state.BuildPickupSummary(st))
	}
	return toastNewCaveats(ctx, run, bin, repo)
}

type herdrEventData struct {
	PaneID   string `json:"pane_id"`
	Worktree struct {
		Path string `json:"path"`
	} `json:"worktree"`
}

// parseHerdrEvent accepts both the subscription envelope ({"event","data"}) and a bare payload,
// since which one HERDR_PLUGIN_EVENT_JSON carries is unverified. A malformed payload yields an
// empty event, which the caller ignores.
func parseHerdrEvent() (string, herdrEventData) {
	var env struct {
		Event string          `json:"event"`
		Data  json.RawMessage `json:"data"`
	}
	raw := []byte(os.Getenv("HERDR_PLUGIN_EVENT_JSON"))
	if err := json.Unmarshal(raw, &env); err != nil {
		return "", herdrEventData{}
	}
	if len(env.Data) > 0 {
		raw = env.Data
	}
	var data herdrEventData
	if err := json.Unmarshal(raw, &data); err != nil {
		return "", herdrEventData{}
	}
	event := os.Getenv("HERDR_PLUGIN_EVENT")
	if event == "" {
		event = env.Event
	}
	return event, data
}

// herdrPaneCWD returns the pane's cwd from `herdr pane get`, or "" when unavailable.
// shortcut: the CLI output shape is unverified; this takes the first "cwd" (then
// "foreground_cwd") found at any depth. Pin the exact path after the 3.2 manual session.
func herdrPaneCWD(ctx context.Context, run herdrRunner, bin, paneID string) string {
	if paneID == "" {
		return ""
	}
	raw, err := run(ctx, bin, "pane", "get", paneID)
	if err != nil {
		return ""
	}
	var v any
	if json.Unmarshal(raw, &v) != nil {
		return ""
	}
	if cwd := findJSONString(v, "cwd"); cwd != "" {
		return cwd
	}
	return findJSONString(v, "foreground_cwd")
}

func findJSONString(v any, key string) string {
	switch t := v.(type) {
	case map[string]any:
		if s, ok := t[key].(string); ok && s != "" {
			return s
		}
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if s := findJSONString(t[k], key); s != "" {
				return s
			}
		}
	case []any:
		for _, e := range t {
			if s := findJSONString(e, key); s != "" {
				return s
			}
		}
	}
	return ""
}

// notifyKey identifies a repository across its spellings and linked worktrees: the git common
// dir when repo is a git checkout, else the resolved repo path.
func notifyKey(repo string) string {
	key := repo
	if common, ok := state.GitCommonDir(repo); ok {
		key = common
	}
	if resolved, err := filepath.EvalSymlinks(key); err == nil {
		key = resolved
	}
	return filepath.Clean(key)
}

// toastNewCaveats toasts HALT/RED/BLOCKED caveats (BLOCKED includes Hold: lines) never notified
// for this repository. The record is additive per repository, so main and its worktrees share it
// and do not flip-flop; a resolved caveat that later returns does not toast again. The record is
// saved before toasting: delivery is at most once, and a failed toast is not retried.
func toastNewCaveats(ctx context.Context, run herdrRunner, bin, repo string) error {
	stateDir := os.Getenv("HERDR_PLUGIN_STATE_DIR")
	if stateDir == "" {
		return nil // without a record every turn would re-toast the same caveats
	}
	changes, err := state.ScanOpenSpecQueue(repo)
	if err != nil {
		return fmt.Errorf("scan queue %s: %w", repo, err)
	}
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		return fmt.Errorf("state dir: %w", err)
	}
	path := filepath.Join(stateDir, "notified.json")
	notified := map[string][]string{}
	if data, err := os.ReadFile(path); err == nil {
		if json.Unmarshal(data, &notified) != nil {
			// A corrupt record is replaced; the cost is one repeat toast per caveat.
			notified = map[string][]string{}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read %s: %w", path, err)
	}
	key := notifyKey(repo)
	seen := map[string]bool{}
	for _, fp := range notified[key] {
		seen[fp] = true
	}
	type toast struct{ title, body string }
	var pending []toast
	for _, ch := range changes {
		for _, c := range ch.Caveats {
			if c.Marker != "HALT" && c.Marker != "RED" && c.Marker != "BLOCKED" {
				continue
			}
			// Line numbers are left out so edits elsewhere in tasks.md do not re-toast.
			fp := ch.Name + "|" + c.Marker + "|" + c.Text
			if seen[fp] {
				continue
			}
			seen[fp] = true
			notified[key] = append(notified[key], fp)
			pending = append(pending, toast{fmt.Sprintf("ateam: %s %s", ch.Name, c.Marker), c.Text})
		}
	}
	if len(pending) == 0 {
		return nil
	}
	// shortcut: concurrent events can race this read-modify-write and double-toast once;
	// add a lock file if herdr turns out to run event commands in parallel.
	if err := writeNotified(stateDir, path, notified); err != nil {
		return err
	}
	for _, t := range pending {
		if _, err := run(ctx, bin, "notification", "show", t.title, "--body", t.body, "--sound", "request"); err != nil {
			return err
		}
	}
	return nil
}

func writeNotified(stateDir, path string, notified map[string][]string) error {
	data, err := json.MarshalIndent(notified, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(stateDir, "notified-*.json")
	if err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return fmt.Errorf("write %s: %w", path, err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		os.Remove(tmp.Name())
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
