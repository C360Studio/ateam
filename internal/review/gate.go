package review

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/c360studio/ateam/internal/state"
)

// The merge check (openspec spec harness, "Merge gate and CI workflow
// enforcement"), ported from scripts/merge-check.sh, which now only runs
// `ateam review gate`. It reads the forge when it runs, so a re-run sees the
// state of that moment.
//
//  1. The rules in force on main must require the Required check with the
//     up-to-date setting on (GitHub: an active ruleset with
//     strict_required_status_checks_policy; Gitea: branch protection with the
//     status check CI / Required (pull_request) and block_on_outdated_branch,
//     the latter printed strict=unverified when the token cannot read the rule).
//  2. On a pull-request run, a non-draft code pull request needs one
//     implemented-by: and one reviewed-by: line, a reviewer not weaker than every
//     implementer once .agents/review-policy.json exists, and a review record
//     comment naming one PR commit with no later content commit.
//  3. On a pull-request run, while an issue labelled class:flake is open, fail
//     unless this pull request closes every open one. There is no waiver.
//
// A read that fails, or an answer that is not the shape asked for, exits 2
// naming the read; it is never taken as "no known flake" or "no finding".

// GateOptions carries what the merge check reads outside the forge.
type GateOptions struct {
	// PR is the pull request number exactly as given; Gate validates it.
	PR string
	// Actions is GITHUB_ACTIONS=true (Gitea Actions sets it too); only then is
	// Event read.
	Actions bool
	// Event is GITHUB_EVENT_NAME: pull_request or push.
	Event string
	// PolicyPath is .agents/review-policy.json of the checkout being judged from.
	PolicyPath string
}

const (
	gateLabel     = "class:flake"
	gateLimit     = 100
	gatePolicy    = ".agents/review-policy.json"
	gateHeading   = "### Independent Cross-Agent Review"
	giteaRequired = "CI / Required (pull_request)"
)

// gateExit ends the run with its own status and line (a FAIL or a usage stop).
type gateExit struct {
	code int
	msg  string
}

func (e *gateExit) Error() string { return e.msg }

func gateDie(format string, a ...any) error {
	return &gateExit{code: 1, msg: fmt.Sprintf(format, a...)}
}

// errCommitFilesUnproven is the Gitea answer for a commit after the review
// record: no file list is trusted there until a live run shows it is complete
// (owner ruling 5 on #143), so every such commit needs a new review.
var errCommitFilesUnproven = errors.New("commit file lists are not proven complete on this forge")

type gatePull struct {
	Draft        bool
	Bot          bool
	Body         string
	Head         string
	ChangedFiles int
	// Commits is GitHub's count; Gitea reports its count with the commit list.
	Commits int
}

type gateFile struct{ Name, Previous string }

type gateCommit struct {
	SHA     string
	Parents int
	Message string
	Files   []gateFile
}

type gateIssue struct {
	Number int
	URL    string
	Title  string
}

// gateSource is one forge's reads. Every list it returns is complete or an
// error; an error is "unavailable" (exit 2) unless it is a *gateExit.
type gateSource interface {
	// upToDate prints the rule lines and reports whether the up-to-date rule holds.
	upToDate(ctx context.Context, out io.Writer) (bool, error)
	pullRequest(ctx context.Context, pr int) (gatePull, error)
	files(ctx context.Context, pr int) ([]gateFile, error)
	// commits returns the PR's commits oldest first and the count the forge reports.
	commits(ctx context.Context, pr int, pull gatePull) ([]gateCommit, int, error)
	comments(ctx context.Context, pr int) ([]string, error)
	commit(ctx context.Context, pr int, sha string) (gateCommit, error)
	labelExists(ctx context.Context, label string) (bool, error)
	openIssues(ctx context.Context, label string) ([]gateIssue, error)
	// closingRefs returns the URLs of the issues the pull request closes; only
	// those among open are needed.
	closingRefs(ctx context.Context, pr int, open []gateIssue) ([]string, error)
	// fileAt returns path's content at ref, or nil when the forge answers 404.
	fileAt(ctx context.Context, path, ref string) ([]byte, error)
}

// Gate runs the merge check against cfg's forge and returns the exit status:
// 0 pass, 1 fail, 2 unavailable.
func Gate(ctx context.Context, out io.Writer, cfg ForgeConfig, opts GateOptions) int {
	var src gateSource
	switch cfg.Provider {
	case "github":
		src = ghSource{}
	case "gitea":
		root, err := forgeRoot(cfg)
		if err != nil {
			fmt.Fprintf(out, "merge-check: unavailable: %v\n", err)
			return 2
		}
		src = &giteaSource{cfg: cfg, root: root}
	default:
		fmt.Fprintf(out, "merge-check: unavailable: unsupported forge %q\n", cfg.Provider)
		return 2
	}
	if opts.PolicyPath == "" {
		opts.PolicyPath = gatePolicy
	}
	actions := "GitHub Actions"
	if cfg.Provider == "gitea" {
		actions = "Gitea Actions"
	}
	g := &gateRun{out: out, src: src, opts: opts, actions: actions}
	err := g.run(ctx)
	var exit *gateExit
	switch {
	case err == nil:
		return 0
	case errors.As(err, &exit):
		fmt.Fprintln(out, "merge-check: "+exit.msg)
		return exit.code
	default:
		fmt.Fprintln(out, "merge-check: unavailable: "+err.Error())
		return 2
	}
}

type gateRun struct {
	out      io.Writer
	src      gateSource
	opts     GateOptions
	actions  string // the Actions host named in the run line
	pr       int
	findings int
}

func (g *gateRun) say(format string, a ...any) {
	fmt.Fprintf(g.out, "merge-check: "+format+"\n", a...)
}

func (g *gateRun) finding(what, line string) {
	g.say("review finding: %s; the line to write: %s", what, line)
	g.findings++
}

func (g *gateRun) run(ctx context.Context) error {
	push, err := g.kind()
	if err != nil {
		return err
	}
	failed := false
	ok, err := g.src.upToDate(ctx, g.out)
	if err != nil {
		return err
	}
	if !ok {
		failed = true
	}
	if push {
		return gateFinish(g, failed)
	}
	reviewFailed, err := g.review(ctx)
	if err != nil {
		return err
	}
	return g.flakes(ctx, failed || reviewFailed)
}

func gateFinish(g *gateRun, failed bool) error {
	if failed {
		return gateDie("FAILED")
	}
	g.say("ok")
	return nil
}

// kind reads the run kind and the pull request number, reporting a push run.
func (g *gateRun) kind() (bool, error) {
	how := "outside " + g.actions + "; GITHUB_EVENT_NAME not read"
	if g.opts.Actions {
		switch g.opts.Event {
		case "pull_request":
			how = g.actions + ", pull_request event"
		case "push":
			g.say("push run (%s, push event); the known-flake check does not apply, the up-to-date rule is checked", g.actions)
			g.say("the review check does not apply to a push run; no pull request is read")
			return true, nil
		case "":
			g.say("inside %s with GITHUB_EVENT_NAME unset", g.actions)
			return false, gateDie("the event is missing")
		default:
			g.say("inside %s on event %s", g.actions, g.opts.Event)
			return false, gateDie("event %s has no rule here; add one before the workflow triggers on it", g.opts.Event)
		}
	}
	pr := g.opts.PR
	shown := pr
	if shown == "" {
		shown = "?"
	}
	g.say("pull-request run for #%s (%s)", shown, how)
	if pr == "" {
		return false, gateDie("a pull request number is required: merge-check.sh <pr-number>")
	}
	n, err := strconv.Atoi(pr)
	if err != nil || pr[0] == '0' || strings.Trim(pr, "0123456789") != "" || n <= 0 {
		return false, gateDie("pull request number '%s' is not a positive integer", pr)
	}
	g.pr = n
	return false, nil
}

// review is the review check; it reports whether it fails the run.
func (g *gateRun) review(ctx context.Context) (bool, error) {
	pull, err := g.src.pullRequest(ctx, g.pr)
	if err != nil {
		return false, err
	}
	files, err := g.src.files(ctx, g.pr)
	if err != nil {
		return false, err
	}
	if len(files) != pull.ChangedFiles {
		return false, fmt.Errorf("the file list of pull request #%d is incomplete: it has %d of %d files", g.pr, len(files), pull.ChangedFiles)
	}
	code := gateCodeNames(files)
	if len(code) == 0 {
		g.say("pull request #%d is documents only (%d changed files); the review check passes", g.pr, pull.ChangedFiles)
		return false, nil
	}
	g.say("pull request #%d is a code pull request; its code files (%d, up to ten shown):", g.pr, len(code))
	for _, name := range code[:min(10, len(code))] {
		g.say("code file: %s", name)
	}
	g.approvalNotes(ctx, pull.Head, files)
	lines, err := g.reviewLines(pull)
	if err != nil {
		return false, err
	}
	recSHA := ""
	if g.findings == 0 {
		recSHA, err = g.record(ctx, pull, files, lines.reviewer)
		if err != nil {
			return false, err
		}
	}
	switch {
	case g.findings == 0:
		g.say("the review check passes; reviewing agent: %s (record at %s)", lines.reviewer, recSHA)
	case pull.Draft:
		g.say("pull request #%d is a draft: a review finding does not fail a draft (%d above)", g.pr, g.findings)
	default:
		g.say("FAIL: the review check has %d finding(s); editing the description starts no run, so run the merge-check job again after the edit", g.findings)
		return true, nil
	}
	return false, nil
}

// approvalNotes prints a note for each OpenSpec change the pull request touches
// whose proposal.md at head has no filled Approved-by: (#182). The proposal is
// documentation read from the pull request's tree, and a note never changes
// the verdict, so a failed read is a note too, never unavailable.
func (g *gateRun) approvalNotes(ctx context.Context, head string, files []gateFile) {
	for _, dir := range gateChangeDirs(files) {
		path := dir + "/proposal.md"
		data, err := g.src.fileAt(ctx, path, head)
		if err != nil {
			g.say("note: %s was not read, so its Approved-by is unchecked: %v", path, err)
			continue
		}
		if approved, _ := state.ParseProposal(data); approved == "" {
			g.say("note: OpenSpec change %s has no Approved-by; the spec was not human-reviewed", dir[strings.LastIndex(dir, "/")+1:])
		}
	}
}

// gateChangeDirs lists, sorted and once each, the OpenSpec change directories
// (openspec/changes/<id> or openspec/changes/archive/<dated-id>) holding a
// changed file's current name.
func gateChangeDirs(files []gateFile) []string {
	var dirs []string
	for _, f := range files {
		rest, ok := strings.CutPrefix(f.Name, "openspec/changes/")
		if !ok {
			continue
		}
		prefix := "openspec/changes/"
		if r, ok := strings.CutPrefix(rest, "archive/"); ok {
			rest, prefix = r, prefix+"archive/"
		}
		id, _, ok := strings.Cut(rest, "/")
		if !ok {
			continue
		}
		if dir := prefix + id; !slices.Contains(dirs, dir) {
			dirs = append(dirs, dir)
		}
	}
	slices.Sort(dirs)
	return dirs
}

// gateContent decodes a contents-API answer, which GitHub and Gitea both give
// as base64 (GitHub wrapping it in newlines).
func gateContent(content, encoding *string) ([]byte, bool) {
	if content == nil || encoding == nil || *encoding != "base64" {
		return nil, false
	}
	b, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(*content, "\n", ""))
	return b, err == nil
}

type gateLines struct {
	reviewer string
}

// reviewLines checks the implemented-by: and reviewed-by: lines and, once the
// policy exists, the reviewer's strength. Each line starts in the first column;
// a carriage return before the line feed is dropped.
func (g *gateRun) reviewLines(pull gatePull) (gateLines, error) {
	var impl, rev []string
	for _, l := range strings.Split(pull.Body, "\n") {
		l = strings.TrimSuffix(l, "\r")
		if strings.HasPrefix(l, "implemented-by:") {
			impl = append(impl, l)
		}
		if strings.HasPrefix(l, "reviewed-by:") {
			rev = append(rev, l)
		}
	}
	first := func(ls []string, prefix string) string {
		if len(ls) == 0 {
			return ""
		}
		return strings.TrimPrefix(ls[0], prefix)
	}
	iagents := strings.Join(gateAgents(first(impl, "implemented-by:")), " ")
	ragents := strings.Join(gateAgents(first(rev, "reviewed-by:")), " ")
	imodel := gateModel.MatchString(first(impl, "implemented-by:"))
	rmodel := gateModel.MatchString(first(rev, "reviewed-by:"))

	// ADR 0002: once the policy file exists, both lines carry a native effort
	// after each model and compare decides strength; the different-agent rule
	// applies only when the policy sets cross_provider to true.
	policy, err := os.ReadFile(g.opts.PolicyPath)
	hasPolicy := err == nil
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return gateLines{}, fmt.Errorf("could not read cross_provider from %s: %v", gatePolicy, err)
	}
	cross := false
	effortHint := ""
	if hasPolicy {
		effortHint = ", <effort>"
		var p struct {
			CrossProvider any `json:"cross_provider"`
		}
		if err := json.Unmarshal(policy, &p); err != nil {
			return gateLines{}, fmt.Errorf("could not read cross_provider from %s: %v", gatePolicy, err)
		}
		cross = p.CrossProvider == true
	}
	implSingle, others := "", ""
	wantline := "reviewed-by: <claude, codex, or gemini> (<model>" + effortHint + ")"
	if cross && !pull.Bot && len(impl) == 1 {
		switch iagents {
		case "claude":
			implSingle, others, wantline = "claude", "codex or gemini", "reviewed-by: codex (or reviewed-by: gemini)"
		case "codex":
			implSingle, others, wantline = "codex", "claude or gemini", "reviewed-by: claude (or reviewed-by: gemini)"
		case "gemini":
			implSingle, others, wantline = "gemini", "claude or codex", "reviewed-by: claude (or reviewed-by: codex)"
		}
	}
	implline := "implemented-by: claude (<model or persona>), implemented-by: codex (<model or persona>), or implemented-by: gemini (<model or persona>)"
	if !pull.Bot {
		switch {
		case len(impl) == 0:
			g.finding("no line starts implemented-by:, and the implemented-by: line must name claude, codex, or gemini, the agent that wrote the commits", implline)
		case len(impl) > 1:
			g.finding(fmt.Sprintf("%d lines start implemented-by:, and exactly one may", len(impl)), implline)
		case iagents == "":
			g.finding("the implemented-by: line must name claude, codex, or gemini, the agent that wrote the commits", implline)
		case !imodel:
			g.finding("the implemented-by: line must name the implementing model in parentheses after the agent (model, effort pairs once a policy exists)", "implemented-by: "+iagents+" (<model>"+effortHint+")")
		}
	}
	switch {
	case len(rev) == 0:
		g.finding("no line starts reviewed-by:", wantline)
	case len(rev) > 1:
		g.finding(fmt.Sprintf("%d lines start reviewed-by:, and exactly one may", len(rev)), wantline)
	case ragents != "claude" && ragents != "codex" && ragents != "gemini":
		g.finding("the reviewed-by: line must name exactly one of claude, codex, and gemini", wantline)
	case implSingle != "" && ragents == implSingle:
		g.finding(fmt.Sprintf("the reviewed-by: line names %s, which the implemented-by: line names; another agent (%s) reviews", ragents, others), wantline)
	case !rmodel:
		g.finding("the reviewed-by: line must name the reviewing model in parentheses after the agent", "reviewed-by: "+ragents+" (<model>"+effortHint+")")
	}
	if g.findings == 0 && !pull.Bot && hasPolicy {
		if err := g.strength(policy, impl, rev, wantline); err != nil {
			return gateLines{}, err
		}
	}
	return gateLines{reviewer: ragents}, nil
}

// strength reads each line's parenthesised group as model, effort pairs and
// compares them under the policy (ADR 0002 rule 2). A malformed group is a
// finding, not an undecided comparison.
func (g *gateRun) strength(policy []byte, impl, rev []string, wantline string) error {
	var implRuns, revRuns []string
	for _, l := range impl {
		implRuns = append(implRuns, gateRuns(strings.TrimPrefix(l, "implemented-by:"))...)
	}
	for _, l := range rev {
		revRuns = append(revRuns, gateRuns(strings.TrimPrefix(l, "reviewed-by:"))...)
	}
	colon := false
	for _, v := range append(slices.Clone(implRuns), revRuns...) {
		if strings.Count(v, ":") >= 3 {
			colon = true
		}
	}
	if len(implRuns) < 1 || len(revRuns) != 1 || colon {
		g.finding("the implemented-by: group must hold one or more model, effort pairs and the reviewed-by: group exactly one, with no colon in a model or effort", "<agent> (<model>, <effort>) on both lines")
		return nil
	}
	undecided := func(err error) error {
		return fmt.Errorf("ateam review compare exited 2 (the policy or a run could not be read): %v", err)
	}
	p, err := ParsePolicy(policy)
	if err != nil {
		return undecided(fmt.Errorf("%s: %w", gatePolicy, err))
	}
	var implementers []Run
	for _, v := range implRuns {
		r, err := ParseRunSpec(v)
		if err != nil {
			return undecided(err)
		}
		implementers = append(implementers, r)
	}
	reviewer, err := ParseRunSpec(revRuns[0])
	if err != nil {
		return undecided(err)
	}
	findings, notes := p.Compare(implementers, reviewer)
	for _, n := range notes {
		g.say("strength: note: %s", n)
	}
	for _, f := range findings {
		if strings.Contains(f, "names no effort") {
			g.finding(f, "<agent> (<model>, <effort>) for every run on both lines")
			continue
		}
		g.finding(f, wantline+", not weaker by tier, then effort band, than every implementer run in "+gatePolicy)
	}
	if len(findings) == 0 {
		g.say("strength: pass: the reviewer is not weaker than any implementer")
	}
	return nil
}

// record finds the latest review record comment and fails any later content
// commit; it returns the recorded commit.
func (g *gateRun) record(ctx context.Context, pull gatePull, files []gateFile, reviewer string) (string, error) {
	commits, total, err := g.src.commits(ctx, g.pr, pull)
	if err != nil {
		return "", err
	}
	if len(commits) == 0 {
		return "", fmt.Errorf("pull request #%d has no commits", g.pr)
	}
	if len(commits) != total || !strings.EqualFold(commits[len(commits)-1].SHA, pull.Head) {
		return "", fmt.Errorf("the commit list of pull request #%d is incomplete or does not end at head: it has %d of %d commits", g.pr, len(commits), total)
	}
	bodies, err := g.src.comments(ctx, g.pr)
	if err != nil {
		return "", err
	}
	idx := -1
	for _, b := range bodies {
		if !gateIsRecord(b, reviewer) {
			continue
		}
		if idxs := gateDesignated(b, commits); len(idxs) == 1 {
			idx = idxs[0]
		}
	}
	if idx < 0 {
		g.finding(fmt.Sprintf("no review record comment names reviewed-by: %s (or ### Independent Cross-Agent Review) and a single designated reviewed commit in pull request #%d", reviewer, g.pr),
			"post a review record comment with reviewed-by: "+reviewer+" and Reviewed at <commit-sha>")
		return "", nil
	}
	recSHA := commits[idx].SHA
	for _, later := range commits[idx+1:] {
		detail, err := g.src.commit(ctx, g.pr, later.SHA)
		if errors.Is(err, errCommitFilesUnproven) {
			g.finding(fmt.Sprintf("review record at %s is older than commit %s, and on Gitea every commit after the review record needs a new review until a live run shows its commit file lists are complete; request re-review", recSHA, later.SHA),
				"post a new review record comment from "+reviewer+" naming the current head commit")
			break
		}
		if err != nil {
			return "", err
		}
		if !gateCommitAllowed(detail, files) {
			g.finding(fmt.Sprintf("review record at %s is older than content commit %s; request re-review", recSHA, later.SHA),
				"post a new review record comment from "+reviewer+" naming the current head commit")
			break
		}
	}
	return recSHA, nil
}

// flakes is the known-flake check, the last step of a pull-request run.
func (g *gateRun) flakes(ctx context.Context, failed bool) error {
	exists, err := g.src.labelExists(ctx, gateLabel)
	if err != nil {
		return err
	}
	if !exists {
		return gateDie("FAIL: the label %s is missing; without it an empty issue list would read as no known flake", gateLabel)
	}
	issues, err := g.src.openIssues(ctx, gateLabel)
	if err != nil {
		return err
	}
	if len(issues) == 0 {
		g.say("no open %s issue", gateLabel)
		return gateFinish(g, failed)
	}
	closing, err := g.src.closingRefs(ctx, g.pr, issues)
	if err != nil {
		return err
	}
	// Compared by URL, so the same number in another repository does not count.
	var notClosed []string
	for _, is := range issues {
		if !slices.Contains(closing, is.URL) {
			notClosed = append(notClosed, fmt.Sprintf("  #%d %s %s", is.Number, is.URL, is.Title))
		}
	}
	if len(notClosed) > 0 {
		g.say("FAIL: known flakes are open and not closed by this pull request:")
		fmt.Fprintln(g.out, strings.Join(notClosed, "\n"))
		return gateDie("fix them in this pull request (Closes #n for each), or wait for the pull request that does")
	}
	var exempting []string
	for _, is := range issues {
		exempting = append(exempting, fmt.Sprintf("#%d", is.Number))
		g.say("closing #%d exempts pull request #%d (%s)", is.Number, g.pr, is.URL)
	}
	fmt.Fprintf(g.out, "::warning::pull request #%d is exempt from the known-flake check by closing %s; the reviewer confirms the reproduction before and after the fix\n", g.pr, strings.Join(exempting, ", "))
	return gateFinish(g, failed)
}

// gateIsDoc reports a document name: it ends in .md, or starts with openspec/
// and ends in .yml, .yaml, or .json, and is not under .claude/agents/ or
// .agents/contracts/. Every other name is code.
func gateIsDoc(name string) bool {
	doc := strings.HasSuffix(name, ".md") ||
		(strings.HasPrefix(name, "openspec/") && (strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".json")))
	return doc && !strings.HasPrefix(name, ".claude/agents/") && !strings.HasPrefix(name, ".agents/contracts/")
}

// gateNames lists every name and previous name once, sorted.
func gateNames(files []gateFile) []string {
	var names []string
	for _, f := range files {
		names = append(names, f.Name)
		if f.Previous != "" {
			names = append(names, f.Previous)
		}
	}
	slices.Sort(names)
	return slices.Compact(names)
}

// gateCodeNames lists the code names among files, a rename's previous name included.
func gateCodeNames(files []gateFile) []string {
	var code []string
	for _, n := range gateNames(files) {
		if !gateIsDoc(n) {
			code = append(code, n)
		}
	}
	return code
}

var gateMergeMain = regexp.MustCompile(`(?i)^merge\b.*\b(origin/)?main\b`)

// gateCommitAllowed allows a commit after the review record only when it is a
// merge of main touching none of the PR's changed files, or a non-merge commit
// that changes documents only.
func gateCommitAllowed(c gateCommit, prFiles []gateFile) bool {
	cf := gateNames(c.Files)
	if c.Parents >= 2 {
		subject, _, _ := strings.Cut(c.Message, "\n")
		if !gateMergeMain.MatchString(strings.TrimSuffix(subject, "\r")) {
			return false
		}
		prf := gateNames(prFiles)
		for _, f := range cf {
			if slices.Contains(prf, f) {
				return false
			}
		}
		return true
	}
	if len(cf) == 0 {
		return false
	}
	for _, f := range cf {
		if !gateIsDoc(f) {
			return false
		}
	}
	return true
}

var (
	gateAgent = regexp.MustCompile(`(?i)claude|codex|gemini`)
	// The model is the parenthesised group directly after the first agent name
	// (spaces and backticks allowed between), with a character other than a
	// space in it.
	gateModel  = regexp.MustCompile("^[ \\t]*`?(?i:claude|codex|gemini)`?[ \\t]*\\([^)]*[^\\s\\p{Z})][^)]*\\)")
	gateRunsRe = regexp.MustCompile("^[ \\t]*`?((?i:claude|codex|gemini))`?[ \\t]*\\(([^)]*)\\)")
	gateByRe   = regexp.MustCompile("(?i)reviewed-by:[ \\t]*`?(claude|codex|gemini)")
	// The designations of a reviewed commit; the first three start at a word
	// boundary, the last at a line start (scanned with a line feed prepended).
	gateDesignators = []*regexp.Regexp{
		regexp.MustCompile("(?i)(?:re-?)?reviewed[ \\t]+at[ \\t]*:?[ \\t]*`?([0-9a-f]{7,40})"),
		regexp.MustCompile("(?i)(?:(?:re-?)?reviewed[ \\t]+(?:(?:head[ \\t]+)?commit|head)|(?:re-?)?review[ \\t]+of)(?:[ \\t]*\\([^)\\n]*\\))?[ \\t*_]*:?[ \\t*_]*`?([0-9a-f]{7,40})"),
		regexp.MustCompile("(?i)(?:re-?)?reviewed[ \\t]+base[ \\t]+`?[0-9a-f]{7,40}`?[ \\t]+(?:->|→|to)[ \\t]+head[ \\t*_]*:?[ \\t*_]*`?([0-9a-f]{7,40})"),
		regexp.MustCompile("(?i)\\n[ \\t>*_-]*(?:reviewed-commit|head(?:[ \\t]+commit)?)(?:[ \\t]*\\([^)\\n]*\\))?[ \\t*_]*:[ \\t*_]*`?([0-9a-f]{7,40})"),
	}
)

func gateLetterOrDigit(r rune) bool { return unicode.IsLetter(r) || unicode.IsNumber(r) }

func gateWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsMark(r) || unicode.IsNumber(r) || unicode.Is(unicode.Pc, r)
}

func gateRuneBefore(s string, i int, class func(rune) bool) bool {
	r, n := utf8.DecodeLastRuneInString(s[:i])
	return n > 0 && class(r)
}

func gateRuneAt(s string, i int, class func(rune) bool) bool {
	r, n := utf8.DecodeRuneInString(s[i:])
	return n > 0 && class(r)
}

func gateASCIIAlnum(r rune) bool {
	return r < utf8.RuneSelf && (r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z')
}

// gateAgents lists the agent names in s, lower case, sorted and once each; a
// name counts only with no letter or digit directly before or after it.
func gateAgents(s string) []string {
	var found []string
	for _, m := range gateAgent.FindAllStringIndex(s, -1) {
		if gateRuneBefore(s, m[0], gateLetterOrDigit) || gateRuneAt(s, m[1], gateLetterOrDigit) {
			continue
		}
		found = append(found, strings.ToLower(s[m[0]:m[1]]))
	}
	slices.Sort(found)
	return slices.Compact(found)
}

// gateRuns reads a line's group as agent:model:effort runs (an odd last model
// has an empty effort).
func gateRuns(rest string) []string {
	m := gateRunsRe.FindStringSubmatch(rest)
	if m == nil {
		return nil
	}
	agent := strings.ToLower(m[1])
	var v []string
	for _, part := range strings.Split(m[2], ",") {
		if part = strings.TrimSpace(part); part != "" {
			v = append(v, part)
		}
	}
	var runs []string
	for k := 0; k < len(v); k += 2 {
		effort := ""
		if k+1 < len(v) {
			effort = v[k+1]
		}
		runs = append(runs, agent+":"+v[k]+":"+effort)
	}
	return runs
}

// gateIsRecord reports a review record by reviewer: its reviewed-by: names only
// reviewer, or it names no reviewed-by: agent and has an Independent
// Cross-Agent Review heading naming no agent or only reviewer.
func gateIsRecord(body, reviewer string) bool {
	var by []string
	for _, m := range gateByRe.FindAllStringSubmatchIndex(body, -1) {
		if !gateRuneAt(body, m[3], gateLetterOrDigit) {
			by = append(by, strings.ToLower(body[m[2]:m[3]]))
		}
	}
	slices.Sort(by)
	by = slices.Compact(by)
	if len(by) > 0 {
		return slices.Equal(by, []string{reviewer})
	}
	headings := 0
	var named []string
	for _, l := range strings.Split(body, "\n") {
		l = strings.TrimSuffix(l, "\r")
		t := strings.TrimLeft(l, " \t")
		if !strings.HasPrefix(t, gateHeading) || gateRuneAt(t, len(gateHeading), gateWordRune) {
			continue
		}
		headings++
		named = append(named, gateAgents(l)...)
	}
	slices.Sort(named)
	named = slices.Compact(named)
	return headings > 0 && (len(named) == 0 || slices.Equal(named, []string{reviewer}))
}

// gateDesignated returns the indices of the commits body designates as
// reviewed, or none when any designated SHA does not name exactly one commit.
func gateDesignated(body string, commits []gateCommit) []int {
	var toks []string
	for i, re := range gateDesignators {
		s := body
		if i == len(gateDesignators)-1 {
			s = "\n" + body
		}
		for pos := 0; pos < len(s); {
			loc := re.FindStringSubmatchIndex(s[pos:])
			if loc == nil {
				break
			}
			start, end := pos+loc[0], pos+loc[3]
			// A SHA ends at a character other than a letter or digit; the
			// first three forms start at a word boundary.
			if (i < 3 && gateRuneBefore(s, start, gateWordRune)) || gateRuneAt(s, end, gateASCIIAlnum) {
				_, size := utf8.DecodeRuneInString(s[start:])
				pos = start + size
				continue
			}
			toks = append(toks, strings.ToLower(s[pos+loc[2]:end]))
			pos += loc[1]
		}
	}
	slices.Sort(toks)
	toks = slices.Compact(toks)
	if len(toks) == 0 {
		return nil
	}
	var idxs []int
	for _, t := range toks {
		var hits []int
		for i, c := range commits {
			if strings.HasPrefix(strings.ToLower(c.SHA), t) {
				hits = append(hits, i)
			}
		}
		if len(hits) != 1 {
			return nil
		}
		idxs = append(idxs, hits[0])
	}
	slices.Sort(idxs)
	return slices.Compact(idxs)
}
