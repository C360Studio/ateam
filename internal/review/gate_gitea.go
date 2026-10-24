package review

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
)

// giteaSource reads a Gitea (API v1) repository over HTTPS with forge.go's
// guarded client: no redirects, a byte cap, a token sent only to the
// configured root.
type giteaSource struct {
	cfg  ForgeConfig
	root string
}

const (
	giteaPageSize = 50
	giteaMaxPages = 200
	// Issue comments are one unpaginated response, so list reads get more room
	// than a single evidence object.
	maxGateList = 16 << 20
)

func (s *giteaSource) get(ctx context.Context, name, path string, out any) (http.Header, error) {
	h, err := readForgePage(ctx, nil, s.cfg, s.root+path, maxGateList, out)
	if err != nil {
		return nil, fmt.Errorf("the read of %s failed (GET %s): %w", name, path, err)
	}
	return h, nil
}

func giteaShape(name, path, what string) error {
	return fmt.Errorf("the read of %s returned something other than what was asked for (GET %s): %s", name, path, what)
}

// giteaList reads every page of a list. It stops on an empty page or once
// X-Total-Count entries are in, and a count that disagrees with the entries
// read is unavailable. needTotal makes a missing X-Total-Count unavailable too.
func giteaList[T any](ctx context.Context, s *giteaSource, name, path string, q url.Values, needTotal bool) ([]T, int, error) {
	var all []T
	total := -1
	for page := 1; ; page++ {
		if page > giteaMaxPages {
			return nil, 0, fmt.Errorf("the read of %s stopped at %d pages and may be incomplete", name, giteaMaxPages)
		}
		q.Set("page", strconv.Itoa(page))
		q.Set("limit", strconv.Itoa(giteaPageSize))
		var items []T
		h, err := s.get(ctx, name, path+"?"+q.Encode(), &items)
		if err != nil {
			return nil, 0, err
		}
		if v := h.Get("X-Total-Count"); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil || n < 0 {
				return nil, 0, giteaShape(name, path, "X-Total-Count "+strconv.Quote(v))
			}
			total = n
		}
		if len(items) == 0 {
			break
		}
		all = append(all, items...)
		if total >= 0 && len(all) >= total {
			break
		}
	}
	if needTotal && total < 0 {
		return nil, 0, giteaShape(name, path, "no X-Total-Count")
	}
	if total >= 0 && len(all) != total {
		return nil, 0, fmt.Errorf("the read of %s is incomplete: it has %d of %d entries", name, len(all), total)
	}
	return all, total, nil
}

// upToDate maps Gitea branch protection onto the up-to-date rule. The branch
// read (any reader) names the rule in force and shows whether it requires the
// status check; only a repository admin can read the rule's
// block_on_outdated_branch. The job token Gitea Actions provides is not an
// admin token (owner ruling on #143: no extra secrets), so a 403 or 404 on the
// rule leaves the up-to-date setting unverified and the reader-level fields
// decide; any other failure is unavailable, never a pass.
func (s *giteaSource) upToDate(ctx context.Context, out io.Writer) (bool, error) {
	var branch struct {
		Protected         *bool     `json:"protected"`
		Protection        *string   `json:"effective_branch_protection_name"`
		EnableStatusCheck *bool     `json:"enable_status_check"`
		Contexts          *[]string `json:"status_check_contexts"`
	}
	if _, err := s.get(ctx, "the main branch", "/branches/main", &branch); err != nil {
		return false, err
	}
	if branch.Protected == nil || branch.Protection == nil || branch.EnableStatusCheck == nil || branch.Contexts == nil {
		return false, giteaShape("the main branch", "/branches/main", "protected, effective_branch_protection_name, enable_status_check or status_check_contexts is missing")
	}
	if !*branch.Protected {
		fmt.Fprintf(out, "merge-check: no branch protection in force on main requires %s\n", giteaRequired)
		return false, nil
	}
	// Gitea names the rule in force only to a repository admin; the job token
	// sees protected=true with an empty name, so the reader-level fields decide.
	if *branch.Protection == "" {
		lines, ok := giteaProtection("(hidden from this token)", *branch.EnableStatusCheck, *branch.Contexts, nil)
		return giteaPrint(out, lines, ok), nil
	}
	name := "branch protection " + *branch.Protection
	path := "/branch_protections/" + url.PathEscape(*branch.Protection)
	var p struct {
		EnableStatusCheck *bool     `json:"enable_status_check"`
		Contexts          *[]string `json:"status_check_contexts"`
		BlockOutdated     *bool     `json:"block_on_outdated_branch"`
	}
	_, err := s.get(ctx, name, path, &p)
	var status *forgeStatusError
	switch {
	case errors.As(err, &status) && (status.code == http.StatusForbidden || status.code == http.StatusNotFound):
		lines, ok := giteaProtection(*branch.Protection, *branch.EnableStatusCheck, *branch.Contexts, nil)
		return giteaPrint(out, lines, ok), nil
	case err != nil:
		return false, err
	}
	if p.EnableStatusCheck == nil || p.Contexts == nil || p.BlockOutdated == nil {
		return false, giteaShape(name, path, "enable_status_check, status_check_contexts or block_on_outdated_branch is missing")
	}
	lines, ok := giteaProtection(*branch.Protection, *p.EnableStatusCheck, *p.Contexts, p.BlockOutdated)
	return giteaPrint(out, lines, ok), nil
}

func giteaPrint(out io.Writer, lines []string, ok bool) bool {
	for _, l := range lines {
		fmt.Fprintln(out, "merge-check: "+l)
	}
	return ok
}

// giteaProtection holds when the rule requires the exact status check
// CI / Required (pull_request) (owner ruling 1 on #143: no glob) and blocks a
// merge from a branch behind main. A nil blockOutdated is a rule only an admin
// can read: it holds on the status check alone and says so.
func giteaProtection(rule string, enabled bool, contexts []string, blockOutdated *bool) ([]string, bool) {
	required := enabled && slices.Contains(contexts, giteaRequired)
	block := "unverified"
	if blockOutdated != nil {
		block = strconv.FormatBool(*blockOutdated)
	}
	lines := []string{fmt.Sprintf("rule type=branch_protection name=%s context=%s required=%t block_on_outdated_branch=%s", rule, giteaRequired, required, block)}
	if !required {
		listed, err := json.Marshal(contexts)
		if err != nil {
			listed = []byte("?")
		}
		return append(lines, fmt.Sprintf("no rule in force on main requires %s; branch protection %s has enable_status_check=%t status_check_contexts=%s", giteaRequired, rule, enabled, listed)), false
	}
	if blockOutdated == nil {
		return append(lines, "rule strict=unverified (branch protection needs a repo-admin token; the job token can see protected, status check and context only)"), true
	}
	if !*blockOutdated {
		return append(lines, fmt.Sprintf("FAIL: branch protection %s requires %s with block_on_outdated_branch=false; a head behind main could merge", rule, giteaRequired)), false
	}
	return lines, true
}

func (s *giteaSource) pullRequest(ctx context.Context, pr int) (gatePull, error) {
	name := fmt.Sprintf("pull request #%d for the review check", pr)
	path := fmt.Sprintf("/pulls/%d", pr)
	var p struct {
		Draft        *bool   `json:"draft"`
		ChangedFiles *int    `json:"changed_files"`
		Body         *string `json:"body"`
		Head         struct {
			SHA *string `json:"sha"`
		} `json:"head"`
		User struct {
			Type *string `json:"type"`
		} `json:"user"`
	}
	if _, err := s.get(ctx, name, path, &p); err != nil {
		return gatePull{}, err
	}
	if p.Draft == nil || p.ChangedFiles == nil || p.Head.SHA == nil || len(*p.Head.SHA) < 7 || p.User.Type == nil {
		return gatePull{}, giteaShape(name, path, "draft, changed_files, head.sha or user.type is missing")
	}
	body := ""
	if p.Body != nil {
		body = *p.Body
	}
	return gatePull{Draft: *p.Draft, Bot: *p.User.Type == "Bot", Body: body, Head: *p.Head.SHA, ChangedFiles: *p.ChangedFiles}, nil
}

func (s *giteaSource) files(ctx context.Context, pr int) ([]gateFile, error) {
	name := fmt.Sprintf("the files of pull request #%d", pr)
	path := fmt.Sprintf("/pulls/%d/files", pr)
	fs, _, err := giteaList[ghFile](ctx, s, name, path, url.Values{}, true)
	if err != nil {
		return nil, err
	}
	for _, f := range fs {
		if !ghFileOK(f) {
			return nil, giteaShape(name, path, "a file has no filename")
		}
	}
	return ghFiles(fs), nil
}

func (s *giteaSource) commits(ctx context.Context, pr int, pull gatePull) ([]gateCommit, int, error) {
	name := fmt.Sprintf("the commits of pull request #%d", pr)
	path := fmt.Sprintf("/pulls/%d/commits", pr)
	cs, total, err := giteaList[ghCommit](ctx, s, name, path, url.Values{"files": {"false"}, "verification": {"false"}}, true)
	if err != nil {
		return nil, 0, err
	}
	out := make([]gateCommit, len(cs))
	for i, c := range cs {
		if !ghCommitOK(c) {
			return nil, 0, giteaShape(name, path, "a commit has no sha or a parent has no sha")
		}
		out[i] = c.gate()
	}
	return giteaOldestFirst(out, pull.Head), total, nil
}

// giteaOldestFirst returns the PR's commits oldest first. Gitea lists them as
// git log does, newest first; a list that already ends at head is kept, and
// one that ends at head neither way fails the head check after it.
func giteaOldestFirst(commits []gateCommit, head string) []gateCommit {
	n := len(commits)
	if n > 1 && strings.EqualFold(commits[0].SHA, head) && !strings.EqualFold(commits[n-1].SHA, head) {
		commits = slices.Clone(commits)
		slices.Reverse(commits)
	}
	return commits
}

// comments reads the PR's comments; Gitea returns them in one response.
func (s *giteaSource) comments(ctx context.Context, pr int) ([]string, error) {
	name := fmt.Sprintf("the comments of pull request #%d", pr)
	path := fmt.Sprintf("/issues/%d/comments", pr)
	var cs []struct {
		Body *string `json:"body"`
	}
	if _, err := s.get(ctx, name, path, &cs); err != nil {
		return nil, err
	}
	bodies := make([]string, len(cs))
	for i, c := range cs {
		if c.Body != nil {
			bodies[i] = *c.Body
		}
	}
	return bodies, nil
}

// commit never classifies a Gitea commit (owner ruling 5 on #143).
func (s *giteaSource) commit(context.Context, int, string) (gateCommit, error) {
	return gateCommit{}, errCommitFilesUnproven
}

// labelExists reads the repository's labels; Gitea discards an unknown label
// in an issue filter, so a missing label must never reach that filter.
func (s *giteaSource) labelExists(ctx context.Context, label string) (bool, error) {
	ls, _, err := giteaList[struct {
		Name *string `json:"name"`
	}](ctx, s, "labels", "/labels", url.Values{}, false)
	if err != nil {
		return false, err
	}
	for _, l := range ls {
		if l.Name == nil {
			return false, giteaShape("labels", "/labels", "a label has no name")
		}
		if *l.Name == label {
			return true, nil
		}
	}
	return false, nil
}

type giteaIssue struct {
	Number      *int            `json:"number"`
	URL         *string         `json:"html_url"`
	Title       string          `json:"title"`
	PullRequest json.RawMessage `json:"pull_request"`
	Labels      []struct {
		Name string `json:"name"`
	} `json:"labels"`
}

func (s *giteaSource) openIssues(ctx context.Context, label string) ([]gateIssue, error) {
	name := "open " + label + " issues"
	q := url.Values{"type": {"issues"}, "state": {"open"}, "labels": {label}}
	is, _, err := giteaList[giteaIssue](ctx, s, name, "/issues", q, false)
	if err != nil {
		return nil, err
	}
	var out []gateIssue
	for _, i := range is {
		if i.Number == nil || i.URL == nil {
			return nil, giteaShape(name, "/issues", "an issue has no number or html_url")
		}
		if giteaLabelled(i, label) {
			out = append(out, gateIssue{Number: *i.Number, URL: *i.URL, Title: i.Title})
		}
	}
	return out, nil
}

// giteaLabelled filters client side: the label filter is not trusted to have
// been applied, and a pull request is not an issue here.
func giteaLabelled(i giteaIssue, label string) bool {
	if pr := strings.TrimSpace(string(i.PullRequest)); pr != "" && pr != "null" {
		return false
	}
	for _, l := range i.Labels {
		if l.Name == label {
			return true
		}
	}
	return false
}

type giteaTimelineEntry struct {
	RefAction string `json:"ref_action"`
	RefIssue  *struct {
		Number      int             `json:"number"`
		PullRequest json.RawMessage `json:"pull_request"`
		Repository  *struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
	} `json:"ref_issue"`
}

// closingRefs reads each open issue's timeline for the closing reference Gitea
// recorded from this pull request (owner ruling 3 on #143); Gitea has no
// closing-references field and the description is not parsed.
func (s *giteaSource) closingRefs(ctx context.Context, pr int, open []gateIssue) ([]string, error) {
	var urls []string
	for _, is := range open {
		name := fmt.Sprintf("the timeline of issue #%d", is.Number)
		path := fmt.Sprintf("/issues/%d/timeline", is.Number)
		entries, _, err := giteaList[giteaTimelineEntry](ctx, s, name, path, url.Values{}, false)
		if err != nil {
			return nil, err
		}
		if giteaCloses(entries, pr, s.cfg.Owner+"/"+s.cfg.Repo) {
			urls = append(urls, is.URL)
		}
	}
	return urls, nil
}

// giteaCloses reports a timeline entry in which pull request pr of repo
// closes the issue: ref_action closes (an edited-away reference reads
// neutered), from a pull request, in the same repository.
func giteaCloses(entries []giteaTimelineEntry, pr int, repo string) bool {
	for _, e := range entries {
		ref := e.RefIssue
		if e.RefAction != "closes" || ref == nil || ref.Number != pr || ref.Repository == nil || !strings.EqualFold(ref.Repository.FullName, repo) {
			continue
		}
		if p := strings.TrimSpace(string(ref.PullRequest)); p != "" && p != "null" {
			return true
		}
	}
	return false
}
