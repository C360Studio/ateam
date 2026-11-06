package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"

	"github.com/c360studio/ateam/internal/review"
)

// Host launch for a cross-agent review (#295): the host-specific part is the
// data file .agents/hosts/<host>.toml, never a host switch in the binary.

// hostManifest is .agents/hosts/<host>.toml. A command is an argument array
// run without a shell. A tool is the text printed instead where the host has no
// one-command form: an MCP tool, or several CLI calls that pass an id along.
type hostManifest struct {
	Launch hostAction `toml:"launch"`
	Notify hostAction `toml:"notify"`
}

type hostAction struct {
	Command []string `toml:"command"`
	Tool    string   `toml:"tool"`
}

// launchRunner runs an argument array without a shell; tests substitute a recorder.
type launchRunner func(ctx context.Context, argv []string) error

func execLaunch(ctx context.Context, argv []string) error {
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Stdout, cmd.Stderr = os.Stderr, os.Stderr
	return cmd.Run()
}

var hostName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,31}$`)

// detectHost names the host ateam runs in: ATEAM_HOST first, so a wrong guess
// is one variable away, then the T3 Code, herdr and Enjoy markers, else cli.
// return_to is the T3 Code thread or the herdr pane; Enjoy puts no
// conversation id in the environment, so its agent passes --return-to.
func detectHost(getenv func(string) string) (host, returnTo string) {
	switch {
	case getenv("ATEAM_HOST") != "":
		host = getenv("ATEAM_HOST")
	case getenv("T3CODE_THREAD_ID") != "":
		host = "t3code"
	case getenv("HERDR_ENV") == "1":
		host = "herdr"
	case getenv("ENJOY_MCP_ENDPOINT") != "":
		host = "enjoy"
	default:
		host = "cli"
	}
	switch host {
	case "t3code":
		returnTo = getenv("T3CODE_THREAD_ID")
	case "herdr":
		returnTo = getenv("HERDR_PANE_ID")
	}
	return host, returnTo
}

// errManifest is fixed text: a malformed manifest's content never reaches the
// agent's context, as the task lease's errors do not.
var errManifest = errors.New("host manifest is malformed: want [launch] and [notify] tables holding command = [\"arg\", ...] string arrays or tool strings")

// loadHostManifest reads .agents/hosts/<host>.toml; a missing file is nil, nil.
func loadHostManifest(root, host string) (*hostManifest, error) {
	if !hostName.MatchString(host) {
		return nil, fmt.Errorf("host name %q must be lower-case letters, digits and hyphens", host)
	}
	data, err := readReviewFile(filepath.Join(root, ".agents", "hosts", host+".toml"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("host manifest .agents/hosts/%s.toml: %w", host, err)
	}
	var m hostManifest
	if err := toml.NewDecoder(bytes.NewReader(data)).DisallowUnknownFields().Decode(&m); err != nil {
		return nil, fmt.Errorf(".agents/hosts/%s.toml: %w", host, errManifest)
	}
	for _, argv := range [][]string{m.Launch.Command, m.Notify.Command} {
		if len(argv) > 0 && strings.TrimSpace(argv[0]) == "" {
			return nil, fmt.Errorf(".agents/hosts/%s.toml: %w", host, errManifest)
		}
	}
	return &m, nil
}

// expandArgs substitutes the placeholders inside each argument in one pass, so
// a value holding "{sha}" is never expanded again and no argument is split.
func expandArgs(argv []string, values map[string]string) []string {
	pairs := make([]string, 0, 2*len(values))
	for _, k := range []string{"pr", "prompt", "sha", "agent", "model", "effort", "return_to"} {
		pairs = append(pairs, "{"+k+"}", values[k])
	}
	r := strings.NewReplacer(pairs...)
	out := make([]string, len(argv))
	for i, a := range argv {
		out[i] = r.Replace(a)
	}
	return out
}

// launchSpec is the object ateam review request prints (design D2).
type launchSpec struct {
	PR           int                `json:"pr"`
	HeadSHA      string             `json:"head_sha"`
	Prompt       string             `json:"prompt"`
	Implementers []review.Run       `json:"implementers"`
	Eligible     []review.Candidate `json:"eligible"`
	Host         string             `json:"host"`
	ReturnTo     string             `json:"return_to"`
	Launch       string             `json:"launch"`
	Command      []string           `json:"command,omitempty"`
	Tool         string             `json:"tool,omitempty"`
}

// readPullBody reads a pull request's description and head; tests substitute it.
var readPullBody = func(ctx context.Context, root string, pr int) (string, string, error) {
	cfg, _, err := gateForge(ctx, root, false)
	if err != nil {
		return "", "", err
	}
	return review.ReadPullBody(ctx, cfg, pr)
}

// reviewRequest prints the launch spec for a cross-agent review of a pull
// request and, only under review_launch: agent with a manifest launch command,
// runs it. Exit 0 printed or executed, 1 the launch failed, 2 unusable input.
func reviewRequest(ctx context.Context, root string, args []string, getenv func(string) string, out io.Writer, run launchRunner) int {
	fs := flag.NewFlagSet("review request", flag.ContinueOnError)
	var implementers, reviewers runSpecs
	fs.Var(&implementers, "implementer", "implementing run as agent:model:effort; repeat for several (default: the PR's implemented-by: line)")
	fs.Var(&reviewers, "reviewer", "reviewing run as agent:model:effort for {agent} {model} {effort} (default: the first eligible run)")
	returnTo := fs.String("return-to", "", "session the reviewer notifies (default: the host's thread or pane id)")
	dryRun := fs.Bool("dry-run", false, "print the expanded launch command without running it")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	pr, err := strconv.Atoi(fs.Arg(0))
	if fs.NArg() != 1 || err != nil || pr <= 0 || len(reviewers) > 1 {
		fmt.Fprintln(os.Stderr, "usage: ateam review request [--implementer AGENT:MODEL:EFFORT]... [--reviewer AGENT:MODEL:EFFORT] [--return-to ID] [--dry-run] PR")
		return 2
	}
	var policy review.Policy
	data, err := readReviewFile(filepath.Join(root, ".agents", "review-policy.json"))
	switch {
	case errors.Is(err, os.ErrNotExist):
	case err != nil:
		fmt.Fprintf(os.Stderr, "ateam review: %v\n", err)
		return 2
	default:
		if policy, err = review.ParsePolicy(data); err != nil {
			fmt.Fprintf(os.Stderr, "ateam review: .agents/review-policy.json: %v\n", err)
			return 2
		}
	}
	head := ""
	if len(implementers) == 0 {
		body, h, err := readPullBody(ctx, root, pr)
		if err == nil {
			implementers, err = review.Implementers(body)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "ateam review: request could not read PR #%d's implemented-by: line (%v); pass --implementer agent:model:effort\n", pr, err)
			return 2
		}
		head = h
	} else if out, err := exec.CommandContext(ctx, "git", "-C", root, "rev-parse", "HEAD").Output(); err == nil {
		head = strings.TrimSpace(string(out))
	}
	host, envReturn := detectHost(getenv)
	spec := launchSpec{PR: pr, HeadSHA: head, Prompt: fmt.Sprintf("/review-pr %d", pr), Implementers: implementers,
		Eligible: policy.EligibleReviewers(implementers), Host: host, ReturnTo: envReturn, Launch: review.LaunchManual}
	if *returnTo != "" {
		spec.ReturnTo = *returnTo
	}
	if spec.Eligible == nil {
		spec.Eligible = []review.Candidate{}
	}
	manifest, err := loadHostManifest(root, host)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ateam review: %v\n", err)
		return 2
	}
	if manifest != nil {
		values := map[string]string{"pr": strconv.Itoa(pr), "prompt": spec.Prompt, "sha": head, "return_to": spec.ReturnTo}
		if len(reviewers) == 1 {
			values["agent"], values["model"], values["effort"] = reviewers[0].Provider, reviewers[0].Model, reviewers[0].Effort
		} else if len(spec.Eligible) > 0 {
			values["agent"], values["model"], values["effort"] = spec.Eligible[0].Agent, spec.Eligible[0].Model, spec.Eligible[0].Effort
		}
		if len(manifest.Launch.Command) > 0 {
			spec.Command = expandArgs(manifest.Launch.Command, values)
		}
		if manifest.Launch.Tool != "" {
			spec.Tool = expandArgs([]string{manifest.Launch.Tool}, values)[0]
		}
	}
	if policy.ReviewLaunch == review.LaunchAgent {
		spec.Launch = "printed"
		if spec.Command != nil && !*dryRun {
			if err := run(ctx, spec.Command); err != nil {
				fmt.Fprintf(os.Stderr, "ateam review: launch command %q failed: %v\n", spec.Command[0], err)
				return 1
			}
			spec.Launch = "executed"
		}
	}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(spec); err != nil {
		fmt.Fprintf(os.Stderr, "ateam review: %v\n", err)
		return 1
	}
	return 0
}

// notifyText is the whole wake message (design D6): it enters the implementing
// session as a prompt, so nothing from the record body, which is model output,
// is ever part of it.
func notifyText(pr int, sha string) string {
	return fmt.Sprintf("review record for PR #%d landed at %s; run ateam state pickup", pr, sha)
}

// reviewNotify tells the implementing session that the review record landed,
// through the host manifest's notify command, or prints the host tool to call.
// It reads only the record's sha; on the cli host it sends nothing.
func reviewNotify(ctx context.Context, root string, args []string, getenv func(string) string, out io.Writer, run launchRunner) int {
	fs := flag.NewFlagSet("review notify", flag.ContinueOnError)
	recordPath := fs.String("record", "", "the review record comment file that was posted")
	returnTo := fs.String("return-to", "", "the implementing session from the launch spec's return_to")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	pr, err := strconv.Atoi(fs.Arg(0))
	if fs.NArg() != 1 || err != nil || pr <= 0 || *recordPath == "" {
		fmt.Fprintln(os.Stderr, "usage: ateam review notify --record FILE [--return-to ID] PR")
		return 2
	}
	body, err := readReviewFile(*recordPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ateam review: record: %v\n", err)
		return 2
	}
	rec, ok := review.ParseRecordComment(string(body))
	if !ok {
		fmt.Fprintln(os.Stderr, "ateam review: the record names no single reviewing agent and reviewed commit")
		return 2
	}
	host, envReturn := detectHost(getenv)
	target := envReturn
	if *returnTo != "" {
		target = *returnTo
	}
	if host == "cli" {
		return 0
	}
	if strings.ContainsAny(target, "\x00\r\n") {
		fmt.Fprintln(os.Stderr, "ateam review: --return-to must be one line")
		return 2
	}
	manifest, err := loadHostManifest(root, host)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ateam review: %v\n", err)
		return 2
	}
	msg := notifyText(pr, rec.SHA)
	switch {
	case manifest != nil && len(manifest.Notify.Command) > 0:
		if target == "" {
			fmt.Fprintln(os.Stderr, "ateam review: notify needs --return-to on this host")
			return 2
		}
		argv := expandArgs(manifest.Notify.Command, map[string]string{"pr": strconv.Itoa(pr), "sha": rec.SHA, "return_to": target})
		// The message is appended as its own argument, never substituted into one.
		if err := run(ctx, append(argv, msg)); err != nil {
			fmt.Fprintf(os.Stderr, "ateam review: notify command %q failed: %v\n", argv[0], err)
			return 1
		}
	case manifest != nil && manifest.Notify.Tool != "":
		tool := expandArgs([]string{manifest.Notify.Tool}, map[string]string{"pr": strconv.Itoa(pr), "sha": rec.SHA, "return_to": target})[0]
		fmt.Fprintf(out, "notify: %s; message: %s\n", tool, msg)
	default:
		fmt.Fprintf(out, "notify: host %s has no notify in .agents/hosts/%s.toml; send %q to %q\n", host, host, msg, target)
	}
	return 0
}

// reviewLaunchContext bounds a launch or notify command: a host's launch
// returns once the reviewer is started, and a synchronous cli reviewer is not
// started through a manifest.
func reviewLaunchContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 2*time.Minute)
}
