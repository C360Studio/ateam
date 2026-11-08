package state

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/c360studio/ateam/internal/review"
)

// Commit status contexts ateam sets on a PR head (#302). They are informational: no gate reads them and none is
// ever a required check.
const (
	StatusContextProposal = "ateam/proposal"
	StatusContextHandoff  = "ateam/handoff"
	StatusContextRecord   = "ateam/review-record"
)

// maxStatusDescription is GitHub's commit status description limit in bytes.
const maxStatusDescription = 140

// statusCallTimeout bounds each commit status read or write.
const statusCallTimeout = 1500 * time.Millisecond

// CommitStatus is one commit status to set on a PR head.
type CommitStatus struct {
	Context     string `json:"context"`
	State       string `json:"state"`
	Description string `json:"description"`
	TargetURL   string `json:"target_url,omitempty"`
}

// StatusResult counts one publication's outcome per context.
type StatusResult struct {
	Posted    int `json:"posted"`
	Unchanged int `json:"unchanged"`
	Errors    int `json:"errors"`
}

func (r *StatusResult) add(o StatusResult) {
	r.Posted += o.Posted
	r.Unchanged += o.Unchanged
	r.Errors += o.Errors
}

// CommitStatusesEnabled reports whether commit statuses are on for root. ATEAM_COMMIT_STATUSES "0" or "1"
// overrides on this machine; otherwise commit_statuses in .agents/ateam.json decides, and an absent key or
// an invalid file leaves them on (#333 D4).
func CommitStatusesEnabled(root string) bool {
	switch strings.TrimSpace(os.Getenv("ATEAM_COMMIT_STATUSES")) {
	case "0":
		return false
	case "1":
		return true
	}
	if on := settingsOrDefault(root).CommitStatuses; on != nil {
		return *on
	}
	return true
}

// touchedChange is one OpenSpec change directory a PR touches, read at the PR head.
type touchedChange struct {
	ID         string
	Proposal   string // repository-relative proposal.md path
	ApprovedBy string
	OpenQ      bool
}

// statusInputs is what the D3 chooser decides from; every field is already known locally or from one sync.
type statusInputs struct {
	Provider   ForgeProvider
	HeadSHA    string
	PRURL      string
	Changes    []touchedChange
	Handoff    *Handoff
	HandoffURL string
	Record     *review.RecordComment
	RecordURL  string
}

// chooseStatuses applies the D3 table (design.md), first match wins per context. A context with nothing to say is
// left out rather than posted.
func chooseStatuses(in statusInputs) []CommitStatus {
	var out []CommitStatus
	head7 := sha7(in.HeadSHA)
	link := func(u string) string {
		if u = strings.TrimSpace(u); u != "" {
			return u
		}
		return in.PRURL
	}

	if len(in.Changes) > 0 {
		var waiting, approved []string
		first := in.Changes[0]
		var firstWaiting *touchedChange
		for i, ch := range in.Changes {
			if strings.TrimSpace(ch.ApprovedBy) == "" || ch.OpenQ {
				waiting = append(waiting, ch.ID)
				if firstWaiting == nil {
					firstWaiting = &in.Changes[i]
				}
				continue
			}
			approved = append(approved, "approved by "+strings.TrimSpace(ch.ApprovedBy)+": "+ch.ID)
		}
		if firstWaiting != nil {
			out = append(out, CommitStatus{Context: StatusContextProposal, State: "pending",
				Description: "awaiting owner approval (OPEN-Q): " + strings.Join(waiting, ", "),
				TargetURL:   link(proposalURL(in.Provider, in.PRURL, in.HeadSHA, firstWaiting.Proposal))})
		} else {
			out = append(out, CommitStatus{Context: StatusContextProposal, State: "success",
				Description: strings.Join(approved, "; "),
				TargetURL:   link(proposalURL(in.Provider, in.PRURL, in.HeadSHA, first.Proposal))})
		}
	}

	if h := in.Handoff; h != nil {
		at := sha7(h.CheckpointSHA)
		st := CommitStatus{Context: StatusContextHandoff, State: "pending", TargetURL: link(in.HandoffURL)}
		switch {
		case !SameCommitSHA(h.CheckpointSHA, in.HeadSHA):
			st.Description = "handoff stale: at " + at + ", head " + head7
		case h.Dirty:
			st.Description = "handoff at " + at + " has uncommitted changes"
		default:
			st.State = "success"
			st.Description = "handoff at " + at + ", " + h.UpdatedAt.UTC().Format("2006-01-02T15:04Z")
		}
		out = append(out, st)
	}

	if r := in.Record; r != nil {
		at := sha7(r.SHA)
		st := CommitStatus{Context: StatusContextRecord, State: "pending", TargetURL: link(in.RecordURL)}
		switch {
		case !SameCommitSHA(r.SHA, in.HeadSHA):
			st.Description = "record at " + at + " is behind head " + head7
		case r.Verdict == "APPROVE" || r.Verdict == "REQUEST CHANGES":
			st.State = "success"
			if r.Verdict == "REQUEST CHANGES" {
				st.State = "failure"
			}
			st.Description = r.Verdict + " by " + recordReviewer(*r) + " at " + at
		default:
			st.Description = "record at " + at + " has no single verdict"
		}
		out = append(out, st)
	}

	for i := range out {
		out[i].Description = cleanDescription(out[i].Description)
	}
	return out
}

// recordReviewer spells a record's reviewer as the pickup line does: agent, then model and effort when named.
func recordReviewer(r review.RecordComment) string {
	who := r.Agent
	if who == "" {
		who = "unnamed agent"
	}
	if r.Model != "" {
		who += " (" + r.Model
		if r.Effort != "" {
			who += ", " + r.Effort
		}
		who += ")"
	}
	return who
}

func sha7(sha string) string {
	sha = strings.ToLower(strings.TrimSpace(sha))
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

// cleanDescription drops control characters (record fields come from PR comments) and cuts to the forge's 140-byte
// limit without splitting a rune.
func cleanDescription(s string) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || r == '\u2028' || r == '\u2029' {
			return -1
		}
		return r
	}, strings.ToValidUTF8(s, ""))
	if len(s) <= maxStatusDescription {
		return s
	}
	cut := maxStatusDescription
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut]
}

// repoHTMLURL is the PR URL without its /pull/<n> (GitHub) or /pulls/<n> (Gitea) tail, or "" when it has neither.
func repoHTMLURL(prURL string) string {
	for _, sep := range []string{"/pulls/", "/pull/"} {
		if i := strings.LastIndex(prURL, sep); i > 0 {
			return prURL[:i]
		}
	}
	return ""
}

func proposalURL(p ForgeProvider, prURL, sha, file string) string {
	repo := repoHTMLURL(prURL)
	if repo == "" || sha == "" || file == "" {
		return ""
	}
	if p == ForgeGitea {
		return repo + "/src/commit/" + sha + "/" + file
	}
	return repo + "/blob/" + sha + "/" + file
}

// touchedChanges lists the OpenSpec changes the PR head touches (D2 C), read at the head with git show. The change
// list is the diff from the merge base of baseRef and head. A head object missing locally, or no baseRef, yields
// none: no proposal status rather than a guessed one.
func touchedChanges(ctx context.Context, dir, baseRef, head string) []touchedChange {
	head = strings.TrimSpace(head)
	baseRef = strings.TrimSpace(baseRef)
	if head == "" || baseRef == "" {
		return nil
	}
	if _, ok := gitOutput(ctx, dir, "rev-parse", "--verify", "--quiet", head+"^{commit}"); !ok {
		return nil
	}
	base, ok := gitOutput(ctx, dir, "merge-base", baseRef, head)
	if !ok {
		return nil
	}
	names, ok := gitOutput(ctx, dir, "diff", "--name-only", "-M", strings.TrimSpace(string(base)), head)
	if !ok {
		return nil
	}
	var out []touchedChange
	for _, d := range changeDirs(strings.Split(strings.TrimSpace(string(names)), "\n")) {
		proposal, hasProposal := gitOutput(ctx, dir, "show", head+":"+d+"/proposal.md")
		tasks, hasTasks := gitOutput(ctx, dir, "show", head+":"+d+"/tasks.md")
		if !hasProposal && !hasTasks {
			continue // the PR removed this change
		}
		ch := touchedChange{ID: path.Base(d), Proposal: d + "/proposal.md"}
		if hasProposal {
			ch.ApprovedBy, _ = ParseProposal(proposal)
		}
		if hasTasks {
			if parsed, _, err := ParseTasksMarkdown(d+"/tasks.md", tasks); err == nil {
				ch.OpenQ = slices.ContainsFunc(parsed.Caveats, func(c OpenSpecCaveat) bool { return c.Marker == "OPEN-Q" })
			}
		}
		out = append(out, ch)
	}
	return out
}

// changeDirs lists, sorted and once each, the OpenSpec change directories (openspec/changes/<id> or
// openspec/changes/archive/<dated-id>) holding a changed file. It copies gateChangeDirs (internal/review/gate.go),
// which is unexported and a gate file; drift only mislabels an informational status.
func changeDirs(files []string) []string {
	var dirs []string
	for _, f := range files {
		rest, ok := strings.CutPrefix(strings.TrimSpace(f), "openspec/changes/")
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
		if d := prefix + id; !slices.Contains(dirs, d) {
			dirs = append(dirs, d)
		}
	}
	slices.Sort(dirs)
	return dirs
}

// gitOutput runs git in dir and reports whether it exited 0, unlike runGitOutput, which cannot tell an empty file
// from a missing one.
func gitOutput(ctx context.Context, dir string, args ...string) ([]byte, bool) {
	cctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	return out, err == nil
}

// PublishCommitStatuses sets statuses on sha (D4): one combined-status read, then one write per context whose state
// or description differs from the latest with that context. A status write error names the context, sha, and HTTP
// code or gh exit code, never a response body or token.
func PublishCommitStatuses(ctx context.Context, root string, cfg ForgeConfig, sha string, statuses []CommitStatus) (StatusResult, error) {
	var res StatusResult
	sha = strings.TrimSpace(sha)
	if sha == "" || len(statuses) == 0 {
		return res, nil
	}
	var w statusWriter
	switch cfg.Provider {
	case ForgeGitHub:
		repo := "{owner}/{repo}" // gh resolves these from root's remote
		if cfg.Owner != "" || cfg.Repo != "" {
			validated, err := normalizeForgeConfig(cfg)
			if err != nil {
				return res, err
			}
			repo = validated.Owner + "/" + validated.Repo
		}
		w = ghStatusWriter{root: root, repo: repo}
	case ForgeGitea:
		validated, err := normalizeForgeConfig(cfg)
		if err != nil {
			return res, err
		}
		w = giteaStatusWriter{api: strings.TrimRight(validated.BaseURL, "/") + "/api/v1/repos/" + validated.Owner + "/" + validated.Repo, client: &http.Client{Timeout: statusCallTimeout}}
	default:
		return res, nil
	}
	latest, err := w.latest(ctx, sha)
	if err != nil {
		res.Errors = len(statuses)
		return res, fmt.Errorf("status read on %s: %w", sha7(sha), err)
	}
	var first error
	for _, s := range statuses {
		if prev, ok := latest[s.Context]; ok && prev.State == s.State && prev.Description == s.Description {
			res.Unchanged++
			continue
		}
		if err := w.post(ctx, sha, s); err != nil {
			res.Errors++
			if first == nil {
				first = fmt.Errorf("%s on %s: %w", s.Context, sha7(sha), err)
			}
			continue
		}
		res.Posted++
	}
	return res, first
}

type statusWriter interface {
	latest(ctx context.Context, sha string) (map[string]CommitStatus, error)
	post(ctx context.Context, sha string, s CommitStatus) error
}

// combinedStatus is the combined-status answer; Gitea names each status's state "status".
type combinedStatus struct {
	Statuses []struct {
		Context     string `json:"context"`
		State       string `json:"state"`
		Status      string `json:"status"`
		Description string `json:"description"`
	} `json:"statuses"`
}

func (c combinedStatus) byContext() map[string]CommitStatus {
	out := make(map[string]CommitStatus, len(c.Statuses))
	for _, s := range c.Statuses {
		if _, seen := out[s.Context]; seen {
			continue // newest first
		}
		st := s.State
		if st == "" {
			st = s.Status
		}
		out[s.Context] = CommitStatus{Context: s.Context, State: st, Description: s.Description}
	}
	return out
}

type ghStatusWriter struct{ root, repo string }

// ghRunError reports a gh failure by exit code only: its stderr can echo the response body.
func ghRunError(err error) error {
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return fmt.Errorf("gh exit %d", exit.ExitCode())
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("gh timed out")
	}
	return fmt.Errorf("gh did not run")
}

func (g ghStatusWriter) latest(ctx context.Context, sha string) (map[string]CommitStatus, error) {
	cctx, cancel := context.WithTimeout(ctx, statusCallTimeout)
	defer cancel()
	// shortcut: the combined status lists the first 100 contexts; page it if a sha with more could hide an ateam/* one.
	cmd := ghCommand(cctx, ReadToken(), "api", "repos/"+g.repo+"/commits/"+sha+"/status?per_page=100")
	cmd.Dir = g.root
	out, err := cmd.Output()
	if err != nil {
		return nil, ghRunError(err)
	}
	var c combinedStatus
	if err := json.Unmarshal(out, &c); err != nil {
		return nil, fmt.Errorf("unreadable combined status")
	}
	return c.byContext(), nil
}

func (g ghStatusWriter) post(ctx context.Context, sha string, s CommitStatus) error {
	cctx, cancel := context.WithTimeout(ctx, statusCallTimeout)
	defer cancel()
	args := []string{"api", "-X", "POST", "repos/" + g.repo + "/statuses/" + sha,
		"-f", "state=" + s.State, "-f", "context=" + s.Context, "-f", "description=" + s.Description}
	if s.TargetURL != "" {
		args = append(args, "-f", "target_url="+s.TargetURL)
	}
	cmd := ghCommand(cctx, WriteToken(), args...)
	cmd.Dir = g.root
	if err := cmd.Run(); err != nil {
		return ghRunError(err)
	}
	return nil
}

type giteaStatusWriter struct {
	api    string
	client *http.Client
}

func (g giteaStatusWriter) do(ctx context.Context, method, url, token string, body []byte) ([]byte, error) {
	cctx, cancel := context.WithTimeout(ctx, statusCallTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(cctx, method, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("bad request")
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "token "+token)
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unreachable")
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	if method != http.MethodGet {
		return nil, nil
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("unreadable response")
	}
	return data, nil
}

func (g giteaStatusWriter) latest(ctx context.Context, sha string) (map[string]CommitStatus, error) {
	data, err := g.do(ctx, http.MethodGet, g.api+"/commits/"+sha+"/status", giteaToken(ReadToken()), nil)
	if err != nil {
		return nil, err
	}
	var c combinedStatus
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("unreadable combined status")
	}
	return c.byContext(), nil
}

func (g giteaStatusWriter) post(ctx context.Context, sha string, s CommitStatus) error {
	payload, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("bad status")
	}
	_, err = g.do(ctx, http.MethodPost, g.api+"/statuses/"+sha, giteaToken(WriteToken()), payload)
	return err
}

// statusTarget is one PR head to publish on.
type statusTarget struct {
	dir        string // worktree to read the head's OpenSpec changes from
	baseRef    string
	headSHA    string
	prURL      string
	handoff    *Handoff
	handoffURL string
	record     *review.RecordComment
	recordURL  string
}

func publishTarget(ctx context.Context, root string, cfg ForgeConfig, t statusTarget) (StatusResult, error) {
	statuses := chooseStatuses(statusInputs{
		Provider:   cfg.Provider,
		HeadSHA:    t.headSHA,
		PRURL:      t.prURL,
		Changes:    touchedChanges(ctx, t.dir, t.baseRef, t.headSHA),
		Handoff:    t.handoff,
		HandoffURL: t.handoffURL,
		Record:     t.record,
		RecordURL:  t.recordURL,
	})
	return PublishCommitStatuses(ctx, root, cfg, t.headSHA, statuses)
}

// statusPublishLimit caps concurrent PR publications in a sync, as giteaCommentFetchLimit caps comment reads.
const statusPublishLimit = 8

// SyncCommitStatuses publishes the three statuses for every open, unmerged PR card with a head SHA whose branch has a
// local worktree, reading each handoff from that worktree's cache (D1 A). Only runStateSync calls it: a pickup,
// prune, board, MCP or UI sync never writes a status. The outcome is recorded for doctor (D7 A).
func SyncCommitStatuses(ctx context.Context, root string, cfg ForgeConfig, baseRef string, cards []BoardCard) (StatusResult, error) {
	worktrees := map[string]string{}
	for _, wt := range parseWorktrees(runGitOutput(ctx, root, "worktree", "list", "--porcelain")) {
		if b := strings.TrimSpace(wt.Branch); b != "" {
			worktrees[b] = wt.Path
		}
	}
	var targets []statusTarget
	for _, c := range cards {
		dir, ok := worktrees[c.Branch]
		if !isPRCard(c) || c.Column == "done" || c.Merged || strings.TrimSpace(c.HeadSHA) == "" || !ok {
			continue
		}
		targets = append(targets, statusTarget{dir: dir, baseRef: baseRef, headSHA: c.HeadSHA, prURL: c.URL,
			handoff: newestBranchHandoff(dir, c.Branch), handoffURL: c.HandoffURL, record: c.Review, recordURL: c.ReviewURL})
	}
	results := make([]StatusResult, len(targets))
	errs := make([]error, len(targets))
	var wg sync.WaitGroup
	slots := make(chan struct{}, statusPublishLimit)
	for i, t := range targets {
		wg.Go(func() {
			slots <- struct{}{}
			defer func() { <-slots }()
			results[i], errs[i] = publishTarget(ctx, root, cfg, t)
		})
	}
	wg.Wait()
	var total StatusResult
	var first error
	for i := range targets {
		total.add(results[i])
		if first == nil && errs[i] != nil {
			first = errs[i]
		}
	}
	return total, recordStatusOutcome(root, total, first)
}

// newestBranchHandoff is the newest handoff slot on branch in dir's cache, or nil.
func newestBranchHandoff(dir, branch string) *Handoff {
	st, err := Load(dir)
	if err != nil {
		return nil
	}
	var newest *Handoff
	consider := func(h *Handoff) {
		if h != nil && strings.TrimSpace(h.Branch) == branch && (newest == nil || h.UpdatedAt.After(newest.UpdatedAt)) {
			newest = h
		}
	}
	for _, h := range st.Handoffs {
		consider(h)
	}
	if len(st.Handoffs) == 0 {
		consider(st.LastHandoff)
	}
	return newest
}

// HandoffCommitStatuses publishes the three statuses after a posted handoff comment: the handoff just saved, the
// record from the cached card for that PR, and the head's OpenSpec changes read from root. The outcome is recorded
// for doctor (D7 A).
func HandoffCommitStatuses(ctx context.Context, root string, st *State, post HandoffPost, h Handoff) (StatusResult, error) {
	if st == nil || !post.Posted || strings.TrimSpace(post.HeadSHA) == "" {
		return StatusResult{}, nil
	}
	t := statusTarget{dir: root, baseRef: st.Git.DefaultRemoteRef, headSHA: post.HeadSHA, prURL: post.PRURL,
		handoff: &h, handoffURL: post.CommentURL}
	for _, c := range st.Cards {
		if isPRCard(c) && cardPRNumber(c) == post.Number && cardForgeProvider(c) == st.Forge.Provider && c.Review != nil {
			t.record = c.Review
			break
		}
	}
	res, err := publishTarget(ctx, root, st.Forge, t)
	return res, recordStatusOutcome(root, res, err)
}

// recordStatusOutcome persists a status error for doctor, or clears a recorded one after a clean publication. The
// cache is written only when that changes something.
func recordStatusOutcome(root string, res StatusResult, statusErr error) error {
	if statusErr == nil && res.Posted+res.Unchanged == 0 {
		return nil
	}
	cur, err := Load(root)
	if err != nil {
		return errors.Join(statusErr, err)
	}
	want := ""
	if statusErr != nil {
		want = statusErr.Error()
	}
	if cur.StatusWriteError == want {
		return statusErr
	}
	if _, err := Update(root, func(st *State) error {
		st.StatusWriteError = want
		return nil
	}); err != nil {
		return errors.Join(statusErr, fmt.Errorf("record status outcome: %w", err))
	}
	return statusErr
}

// String formats a publication's counts for a status line.
func (r StatusResult) String() string {
	return "posted " + strconv.Itoa(r.Posted) + ", unchanged " + strconv.Itoa(r.Unchanged) + ", errors " + strconv.Itoa(r.Errors)
}
