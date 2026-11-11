package main

import (
	"bytes"
	"cmp"
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
	"github.com/c360studio/ateam/internal/state"
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
	// Rereview is "same-session" when the round went back to --reviewer-session,
	// else why it fell back to a fresh launch (#372).
	Rereview string `json:"rereview,omitempty"`
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
	reviewerSession := fs.String("reviewer-session", "", "after REQUEST CHANGES, send the re-review to this reviewing session (from its notify message)")
	fresh := fs.Bool("fresh", false, "launch a fresh reviewer even when --reviewer-session is given")
	dryRun := fs.Bool("dry-run", false, "print the expanded launch command without running it")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	pr, err := strconv.Atoi(fs.Arg(0))
	if fs.NArg() != 1 || err != nil || pr <= 0 || len(reviewers) > 1 {
		fmt.Fprintln(os.Stderr, "usage: ateam review request [--implementer AGENT:MODEL:EFFORT]... [--reviewer AGENT:MODEL:EFFORT] [--return-to ID] [--reviewer-session ID] [--fresh] [--dry-run] PR")
		return 2
	}
	if *reviewerSession != "" && !sessionID.MatchString(*reviewerSession) {
		fmt.Fprintln(os.Stderr, "ateam review: --reviewer-session must be 1 to 128 of A-Z a-z 0-9 . _ : -")
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
	// The head always comes from the PR: the local checkout may sit on another commit (#313).
	body, head, err := readPullBody(ctx, root, pr)
	if err != nil && len(implementers) > 0 {
		fmt.Fprintf(os.Stderr, "ateam review: request could not read PR #%d's head (%v)\n", pr, err)
		return 2
	}
	if len(implementers) == 0 {
		if err == nil {
			implementers, err = review.Implementers(body)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "ateam review: request could not read PR #%d's implemented-by: line (%v); pass --implementer agent:model:effort\n", pr, err)
			return 2
		}
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
	if *reviewerSession != "" {
		// return_to enters the reviewer's prompt as command text, so it takes the session grammar.
		if spec.ReturnTo != "" && !sessionID.MatchString(spec.ReturnTo) {
			fmt.Fprintln(os.Stderr, "ateam review: --return-to must be 1 to 128 of A-Z a-z 0-9 . _ : - with --reviewer-session")
			return 2
		}
		spec.Rereview = "fresh"
		if !*fresh {
			spec.Rereview = rereview(ctx, root, &spec, policy, manifest, *reviewerSession, *dryRun, run)
		}
		if spec.Rereview == "same-session" {
			return encodeSpec(out, spec)
		}
	}
	if manifest != nil {
		reviewer, err := chooseReviewer(policy, implementers, reviewers, spec.Eligible)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ateam review: %v\n", err)
			return 2
		}
		// A launch that names {return_to} with none known starts a reviewer that can never notify back (#306).
		if spec.ReturnTo == "" && strings.Contains(manifest.Launch.Tool+strings.Join(manifest.Launch.Command, " "), "{return_to}") {
			fmt.Fprintf(os.Stderr, "ateam review: request: host %s needs --return-to <your session id>; the reviewer cannot notify you otherwise\n", host)
			if !*dryRun {
				return 2
			}
		}
		values := map[string]string{"pr": strconv.Itoa(pr), "prompt": spec.Prompt, "sha": head, "return_to": spec.ReturnTo,
			"agent": reviewer.Provider, "model": reviewer.Model, "effort": reviewer.Effort}
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
	return encodeSpec(out, spec)
}

func encodeSpec(out io.Writer, spec launchSpec) int {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(spec); err != nil {
		fmt.Fprintf(os.Stderr, "ateam review: %v\n", err)
		return 1
	}
	return 0
}

// rereview sends the round back to the reviewing session through the host's
// notify action and returns "same-session", or returns why a fresh launch
// follows instead. It goes back only after REQUEST CHANGES at an older commit,
// to a reviewer run the merge gate still accepts against these implementers.
// The notify command runs, as a launch command does, only under review_launch: agent.
func rereview(ctx context.Context, root string, spec *launchSpec, policy review.Policy, m *hostManifest, session string, dryRun bool, run launchRunner) string {
	rec := cachedRecord(root, spec.PR)
	switch {
	case rec == nil:
		return "no cached record; run ateam state sync"
	case rec.Verdict != "REQUEST CHANGES":
		return "record verdict is " + strings.ToLower(cmp.Or(rec.Verdict, "missing"))
	case spec.HeadSHA == "" || strings.HasPrefix(spec.HeadSHA, rec.SHA):
		return "record is at the head"
	case m == nil || (len(m.Notify.Command) == 0 && m.Notify.Tool == ""):
		return "host " + spec.Host + " has no notify"
	case spec.ReturnTo == "":
		return "no return_to for the reviewer to notify"
	}
	prev := review.Run{Provider: rec.Agent, Model: rec.Model, Effort: rec.Effort}
	if _, err := chooseReviewer(policy, spec.Implementers, []review.Run{prev}, spec.Eligible); err != nil {
		return "previous reviewer: " + err.Error()
	}
	msg := rereviewText(spec.PR, rec.SHA, spec.HeadSHA, session, spec.ReturnTo)
	if len(m.Notify.Command) > 0 {
		spec.Command = append(expandArgs(m.Notify.Command, map[string]string{"pr": strconv.Itoa(spec.PR), "sha": spec.HeadSHA, "return_to": session}), msg)
		if policy.ReviewLaunch == review.LaunchAgent {
			spec.Launch = "printed"
			if !dryRun {
				if err := run(ctx, spec.Command); err != nil {
					spec.Command, spec.Launch = nil, review.LaunchManual
					return "unreachable"
				}
				spec.Launch = "executed"
			}
		}
	} else {
		spec.Tool = notifyTool(m, spec.PR, spec.HeadSHA, session, msg) + fmt.Sprintf("; if that session is gone, run ateam review request --fresh%s %d", returnFlag(spec.ReturnTo), spec.PR)
		if policy.ReviewLaunch == review.LaunchAgent {
			spec.Launch = "printed"
		}
	}
	return "same-session"
}

// cachedRecord is the newest review record the last ateam state sync cached
// for the pull request, or nil.
func cachedRecord(root string, pr int) *review.RecordComment {
	st, err := state.Load(root)
	if err != nil {
		return nil
	}
	for _, c := range st.Cards {
		if c.Type == "pr" && c.Number == pr && c.Review != nil {
			return c.Review
		}
	}
	return nil
}

// chooseReviewer is the --reviewer run, else the first eligible one, refused
// when the merge gate would reject it, so a launch never spends a review run the
// gate discards or leaves the reviewer to the host's defaults.
func chooseReviewer(p review.Policy, implementers, override []review.Run, eligible []review.Candidate) (review.Run, error) {
	var r review.Run
	switch {
	case len(override) == 1:
		r = override[0]
	case len(eligible) > 0:
		r = review.Run{Provider: eligible[0].Agent, Model: eligible[0].Model, Effort: eligible[0].Effort}
	default:
		return r, errors.New("no reviewer in model_tiers is eligible for these implementers; pass --reviewer agent:model:effort")
	}
	for _, dev := range implementers {
		if p.CrossProvider && dev.Provider == r.Provider {
			return r, fmt.Errorf("cross_provider: reviewer %s:%s is the same agent as an implementer", r.Provider, r.Model)
		}
	}
	if findings, _ := p.Compare(implementers, r); len(findings) > 0 {
		return r, fmt.Errorf("reviewer %s:%s:%s would fail the merge gate: %s", r.Provider, r.Model, r.Effort, strings.Join(findings, "; "))
	}
	return r, nil
}

// notifyText is the whole wake message (design D6): it enters the implementing
// session as a prompt, so nothing from the record body, which is model output,
// is ever part of it. It asks for a sync because each checkout caches in its
// own .ateam/state.json and pickup skips the fetch while that cache is fresh.
// A known reviewer session is named so the next round can come back to it (#372).
// The commands it names carry the implementer's own --return-to, which Enjoy
// cannot read from the environment, so they run as printed.
func notifyText(pr int, sha, session, returnTo string) string {
	msg := fmt.Sprintf("review record for PR #%d landed at %s; run ateam state sync, then ateam state pickup", pr, sha)
	if session != "" {
		msg += fmt.Sprintf("; to re-review after fixes run ateam review request --reviewer-session %s%s %d", session, returnFlag(returnTo), pr)
	}
	return msg
}

// declineText is the reviewer's whole answer to a re-review it will not take (#372 D4).
func declineText(pr int, returnTo string) string {
	return fmt.Sprintf("reviewer declined the re-review of PR #%d; run ateam review request --fresh%s %d", pr, returnFlag(returnTo), pr)
}

// returnFlag is " --return-to <id>" for a known session, else "".
func returnFlag(id string) string {
	if id == "" {
		return ""
	}
	return " --return-to " + id
}

// rereviewText is the whole message a re-review sends the reviewing session
// (#372): fixed text over validated ids, never anything from the record body.
func rereviewText(pr int, recordSHA, head, session, returnTo string) string {
	return fmt.Sprintf("PR #%d moved to %s after your REQUEST CHANGES at %s; check out %s detached, review the full base-to-head diff and %s..%s, rerun task verify, post a new record, then run ateam review notify --record <record file> --from %s --return-to %s %d; if your context is past your handoff threshold, run ateam review notify --decline --return-to %s %d instead",
		pr, head, recordSHA, head, recordSHA, head, session, returnTo, pr, returnTo, pr)
}

// sessionID bounds a session id that enters another session's prompt.
var sessionID = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)

// reviewNotify tells the implementing session that the review record landed,
// through the host manifest's notify command, or prints the host tool to call.
// It reads only the record's sha; on the cli host it sends nothing.
func reviewNotify(ctx context.Context, root string, args []string, getenv func(string) string, out io.Writer, run launchRunner) int {
	fs := flag.NewFlagSet("review notify", flag.ContinueOnError)
	recordPath := fs.String("record", "", "the review record comment file that was posted")
	returnTo := fs.String("return-to", "", "the implementing session from the launch spec's return_to")
	from := fs.String("from", "", "your own session, named so the next round can come back to you (default: the host's thread or pane id)")
	decline := fs.Bool("decline", false, "decline a re-review instead of sending a record")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	pr, err := strconv.Atoi(fs.Arg(0))
	if fs.NArg() != 1 || err != nil || pr <= 0 || (*recordPath == "") == !*decline {
		fmt.Fprintln(os.Stderr, "usage: ateam review notify (--record FILE [--from ID] | --decline) [--return-to ID] PR")
		return 2
	}
	host, envReturn := detectHost(getenv)
	target := envReturn
	if *returnTo != "" {
		target = *returnTo
	}
	// The target enters the message as the --return-to of the commands it names.
	if target != "" && !sessionID.MatchString(target) {
		fmt.Fprintln(os.Stderr, "ateam review: --return-to must be 1 to 128 of A-Z a-z 0-9 . _ : -")
		return 2
	}
	var msg, sha string
	if *decline {
		msg = declineText(pr, target)
	} else {
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
		session := envReturn
		if *from != "" {
			session = *from
		}
		if session != "" && !sessionID.MatchString(session) {
			fmt.Fprintln(os.Stderr, "ateam review: --from must be 1 to 128 of A-Z a-z 0-9 . _ : -")
			return 2
		}
		sha = rec.SHA
		msg = notifyText(pr, sha, session, target)
	}
	if host == "cli" {
		if target == "" {
			return 0
		}
		// A silent exit 0 here read as delivery to reviewers whose shell lacked the host marker (#358).
		fmt.Fprintf(out, "notify: host cli has no notify; send %q to %q\n", msg, target)
		fmt.Fprintf(os.Stderr, "ateam review: notify: the host resolved to cli, so nothing was sent to %s; rerun with ATEAM_HOST set to your host%s\n", target, hostManifestList(root))
		return 1
	}
	manifest, err := loadHostManifest(root, host)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ateam review: %v\n", err)
		return 2
	}
	switch {
	case manifest != nil && len(manifest.Notify.Command) > 0:
		if target == "" {
			fmt.Fprintln(os.Stderr, "ateam review: notify needs --return-to on this host")
			return 2
		}
		if err := sendNotify(ctx, manifest, pr, sha, target, msg, run); err != nil {
			fmt.Fprintf(os.Stderr, "ateam review: %v\n", err)
			return 1
		}
	case manifest != nil && manifest.Notify.Tool != "":
		fmt.Fprintln(out, notifyTool(manifest, pr, sha, target, msg))
	default:
		fmt.Fprintf(out, "notify: host %s has no notify in .agents/hosts/%s.toml; send %q to %q\n", host, host, msg, target)
	}
	return 0
}

// hostManifestList is " (one of: a, b)" naming the hosts with a manifest in
// .agents/hosts/, or "" when there are none.
func hostManifestList(root string) string {
	entries, _ := os.ReadDir(filepath.Join(root, ".agents", "hosts"))
	var hosts []string
	for _, e := range entries {
		if name, ok := strings.CutSuffix(e.Name(), ".toml"); ok && hostName.MatchString(name) {
			hosts = append(hosts, name)
		}
	}
	if len(hosts) == 0 {
		return ""
	}
	return " (one of: " + strings.Join(hosts, ", ") + ")"
}

// sendNotify runs the manifest's notify command for target; the message is
// appended as its own argument, never substituted into one.
func sendNotify(ctx context.Context, m *hostManifest, pr int, sha, target, msg string, run launchRunner) error {
	argv := expandArgs(m.Notify.Command, map[string]string{"pr": strconv.Itoa(pr), "sha": sha, "return_to": target})
	if err := run(ctx, append(argv, msg)); err != nil {
		return fmt.Errorf("notify command %q failed: %w", argv[0], err)
	}
	return nil
}

// notifyTool is the line naming the manifest's notify tool for target and the message.
func notifyTool(m *hostManifest, pr int, sha, target, msg string) string {
	tool := expandArgs([]string{m.Notify.Tool}, map[string]string{"pr": strconv.Itoa(pr), "sha": sha, "return_to": target})[0]
	return fmt.Sprintf("notify: %s; message: %s", tool, msg)
}

// reviewLaunchContext bounds a launch or notify command: a host's launch
// returns once the reviewer is started, and a synchronous cli reviewer is not
// started through a manifest.
func reviewLaunchContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 2*time.Minute)
}
