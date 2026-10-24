package review

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"slices"
	"strconv"
	"strings"
)

// ghSource reads GitHub through the gh CLI with the argv scripts/merge-check.sh
// used, {owner}/{repo} placeholders included, so gh resolves the repository
// from GH_REPO or the checkout.
// shortcut: GitHub reads exec gh so the fake-gh suite stays the parity oracle; move to REST when that suite is ported to httptest
type ghSource struct{}

// gh runs one read and returns its stdout; a failed run is unavailable.
func (ghSource) gh(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "gh", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("the read of %s failed (gh %s): %s", name, strings.Join(args, " "), strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

func ghShape(name string, args []string, answer []byte) error {
	const shown = 2000
	if len(answer) > shown {
		answer = append(answer[:shown:shown], "..."...)
	}
	return fmt.Errorf("the read of %s returned something other than what was asked for (gh %s): %s", name, strings.Join(args, " "), answer)
}

// ghValues splits gh output into its JSON values: gh --paginate prints the
// pages joined into one list or as lists one after the other.
func ghValues(answer []byte) ([]json.RawMessage, error) {
	dec := json.NewDecoder(bytes.NewReader(answer))
	var values []json.RawMessage
	for {
		var v json.RawMessage
		err := dec.Decode(&v)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		values = append(values, v)
	}
	if len(values) == 0 {
		return nil, errors.New("no JSON value")
	}
	return values, nil
}

// ghRead runs a read whose answer is one JSON value decoded into out.
func (s ghSource) ghRead(ctx context.Context, name string, out any, args ...string) error {
	answer, err := s.gh(ctx, name, args...)
	if err != nil {
		return err
	}
	values, err := ghValues(answer)
	if err != nil || len(values) != 1 || json.Unmarshal(values[0], out) != nil {
		return ghShape(name, args, answer)
	}
	return nil
}

// ghList runs a paginated read whose every page is a list, joined into out's
// element type; check rejects an entry.
func ghList[T any](ctx context.Context, s ghSource, name string, check func(T) bool, args ...string) ([]T, error) {
	answer, err := s.gh(ctx, name, args...)
	if err != nil {
		return nil, err
	}
	values, err := ghValues(answer)
	if err != nil {
		return nil, ghShape(name, args, answer)
	}
	var all []T
	for _, v := range values {
		var page []T
		if !isJSONArray(v) || json.Unmarshal(v, &page) != nil {
			return nil, ghShape(name, args, answer)
		}
		for _, item := range page {
			if !check(item) {
				return nil, ghShape(name, args, answer)
			}
		}
		all = append(all, page...)
	}
	return all, nil
}

func isJSONArray(v json.RawMessage) bool {
	return len(bytes.TrimSpace(v)) > 0 && bytes.TrimSpace(v)[0] == '['
}

func isJSONObject(v json.RawMessage) bool {
	return len(bytes.TrimSpace(v)) > 0 && bytes.TrimSpace(v)[0] == '{'
}

// jqString renders a JSON value as jq's tostring does for the scalars read here.
func jqString(v json.RawMessage) string {
	var s string
	if json.Unmarshal(v, &s) == nil {
		return s
	}
	if len(v) == 0 {
		return "null"
	}
	return string(v)
}

type ghRule struct {
	Type       json.RawMessage `json:"type"`
	RulesetID  json.RawMessage `json:"ruleset_id"`
	Parameters json.RawMessage `json:"parameters"`
}

type ghRuleParams struct {
	Strict json.RawMessage `json:"strict_required_status_checks_policy"`
	Checks []struct {
		Context any `json:"context"`
	} `json:"required_status_checks"`
}

// params reads a rule's parameters; parameters that are not this shape name
// no check and no strict setting, so the rule cannot satisfy the up-to-date rule.
func (r ghRule) params() ghRuleParams {
	var p ghRuleParams
	if err := json.Unmarshal(r.Parameters, &p); err != nil {
		return ghRuleParams{}
	}
	return p
}

func (p ghRuleParams) contexts() []string {
	var out []string
	for _, c := range p.Checks {
		if s, ok := c.Context.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func (s ghSource) upToDate(ctx context.Context, out io.Writer) (bool, error) {
	const check = "Required"
	name := "the rules in force on main"
	args := []string{"api", "repos/{owner}/{repo}/rules/branches/main"}
	var raw []json.RawMessage
	if err := s.ghRead(ctx, name, &raw, args...); err != nil {
		return false, err
	}
	var all, found []ghRule
	for _, v := range raw {
		var r ghRule
		if !isJSONObject(v) || json.Unmarshal(v, &r) != nil {
			return false, ghShape(name, args, v)
		}
		all = append(all, r)
		if jqString(r.Type) == "required_status_checks" && slices.Contains(r.params().contexts(), check) {
			found = append(found, r)
		}
	}
	if len(found) == 0 {
		var listed []string
		for _, r := range all {
			b, err := json.Marshal(struct {
				Type      json.RawMessage `json:"type"`
				RulesetID json.RawMessage `json:"ruleset_id"`
			}{r.Type, r.RulesetID})
			if err != nil {
				return false, err
			}
			listed = append(listed, string(b))
		}
		fmt.Fprintf(out, "merge-check: no rule in force on main requires %s; rules found: [%s]\n", check, strings.Join(listed, ","))
		return false, nil
	}
	ok := false
	for _, r := range found {
		id := jqString(r.RulesetID)
		if id == "" || strings.Trim(id, "0123456789") != "" {
			return false, gateDie("a rule requiring %s names ruleset_id '%s', not a number", check, id)
		}
		var ruleset struct {
			Enforcement *string `json:"enforcement"`
		}
		rargs := []string{"api", "repos/{owner}/{repo}/rulesets/" + id}
		if err := s.ghRead(ctx, "ruleset "+id, &ruleset, rargs...); err != nil {
			return false, err
		}
		if ruleset.Enforcement == nil {
			return false, ghShape("ruleset "+id, rargs, []byte("enforcement is not a string"))
		}
		strict := jqString(r.params().Strict)
		fmt.Fprintf(out, "merge-check: rule type=required_status_checks context=%s strict_required_status_checks_policy=%s ruleset_id=%s enforcement=%s\n", check, strict, id, *ruleset.Enforcement)
		if strict == "true" && *ruleset.Enforcement == "active" {
			ok = true
		}
	}
	if !ok {
		fmt.Fprintf(out, "merge-check: FAIL: no active ruleset requires %s with strict_required_status_checks_policy=true; a head behind main could merge\n", check)
	}
	return ok, nil
}

func (s ghSource) pullRequest(ctx context.Context, pr int) (gatePull, error) {
	name := fmt.Sprintf("pull request #%d for the review check", pr)
	args := []string{"api", fmt.Sprintf("repos/{owner}/{repo}/pulls/%d", pr)}
	var p struct {
		Draft        *bool   `json:"draft"`
		ChangedFiles *int    `json:"changed_files"`
		Commits      *int    `json:"commits"`
		Body         *string `json:"body"`
		Head         struct {
			SHA *string `json:"sha"`
		} `json:"head"`
		User struct {
			Type *string `json:"type"`
		} `json:"user"`
	}
	if err := s.ghRead(ctx, name, &p, args...); err != nil {
		return gatePull{}, err
	}
	if p.Draft == nil || p.ChangedFiles == nil || p.Commits == nil || p.Head.SHA == nil || len(*p.Head.SHA) < 7 || p.User.Type == nil {
		return gatePull{}, ghShape(name, args, []byte("draft, changed_files, commits, head.sha or user.type is missing"))
	}
	body := ""
	if p.Body != nil {
		body = *p.Body
	}
	return gatePull{Draft: *p.Draft, Bot: *p.User.Type == "Bot", Body: body, Head: *p.Head.SHA, ChangedFiles: *p.ChangedFiles, Commits: *p.Commits}, nil
}

type ghFile struct {
	Filename         *string `json:"filename"`
	PreviousFilename *string `json:"previous_filename"`
}

func ghFileOK(f ghFile) bool { return f.Filename != nil }

func ghFiles(fs []ghFile) []gateFile {
	out := make([]gateFile, len(fs))
	for i, f := range fs {
		out[i].Name = *f.Filename
		if f.PreviousFilename != nil {
			out[i].Previous = *f.PreviousFilename
		}
	}
	return out
}

func (s ghSource) files(ctx context.Context, pr int) ([]gateFile, error) {
	fs, err := ghList(ctx, s, fmt.Sprintf("the files of pull request #%d", pr), ghFileOK,
		"api", "--paginate", fmt.Sprintf("repos/{owner}/{repo}/pulls/%d/files?per_page=100", pr))
	if err != nil {
		return nil, err
	}
	return ghFiles(fs), nil
}

type ghCommit struct {
	SHA     *string `json:"sha"`
	Parents []struct {
		SHA *string `json:"sha"`
	} `json:"parents"`
	Commit *struct {
		Message *string `json:"message"`
	} `json:"commit"`
}

func ghCommitOK(c ghCommit) bool {
	if c.SHA == nil || len(*c.SHA) < 7 {
		return false
	}
	for _, p := range c.Parents {
		if p.SHA == nil {
			return false
		}
	}
	return true
}

func (c ghCommit) gate() gateCommit {
	msg := ""
	if c.Commit != nil && c.Commit.Message != nil {
		msg = *c.Commit.Message
	}
	return gateCommit{SHA: *c.SHA, Parents: len(c.Parents), Message: msg}
}

func (s ghSource) commits(ctx context.Context, pr int, pull gatePull) ([]gateCommit, int, error) {
	cs, err := ghList(ctx, s, fmt.Sprintf("the commits of pull request #%d", pr), ghCommitOK,
		"api", "--paginate", fmt.Sprintf("repos/{owner}/{repo}/pulls/%d/commits?per_page=100", pr))
	if err != nil {
		return nil, 0, err
	}
	out := make([]gateCommit, len(cs))
	for i, c := range cs {
		out[i] = c.gate()
	}
	return out, pull.Commits, nil
}

func (s ghSource) comments(ctx context.Context, pr int) ([]string, error) {
	type comment struct {
		Body *string `json:"body"`
	}
	cs, err := ghList(ctx, s, fmt.Sprintf("the comments of pull request #%d", pr), func(comment) bool { return true },
		"api", "--paginate", fmt.Sprintf("repos/{owner}/{repo}/issues/%d/comments?per_page=100", pr))
	if err != nil {
		return nil, err
	}
	var bodies []string
	for _, c := range cs {
		if c.Body != nil {
			bodies = append(bodies, *c.Body)
		} else {
			bodies = append(bodies, "")
		}
	}
	return bodies, nil
}

type ghCommitPage struct {
	ghCommit
	Files []struct {
		ghFile
		Additions *float64 `json:"additions"`
		Deletions *float64 `json:"deletions"`
	} `json:"files"`
	ChangedFiles json.RawMessage `json:"changed_files"`
	Stats        json.RawMessage `json:"stats"`
}

// commit reads one commit's pages of files. GitHub returns at most 300 files a
// page and 3,000 in all, so a single full page, the ceiling, or totals that
// disagree with the list may hide files: each is unavailable, never a pass.
func (s ghSource) commit(ctx context.Context, pr int, sha string) (gateCommit, error) {
	name := fmt.Sprintf("commit %s of pull request #%d", sha, pr)
	args := []string{"api", "--paginate", fmt.Sprintf("repos/{owner}/{repo}/commits/%s?per_page=100", sha)}
	answer, err := s.gh(ctx, name, args...)
	if err != nil {
		return gateCommit{}, err
	}
	values, err := ghValues(answer)
	if err != nil {
		return gateCommit{}, ghShape(name, args, answer)
	}
	if len(values) == 1 && isJSONArray(values[0]) {
		if json.Unmarshal(values[0], &values) != nil {
			return gateCommit{}, ghShape(name, args, answer)
		}
	}
	var pages []ghCommitPage
	for _, v := range values {
		var p ghCommitPage
		if !isJSONObject(v) || json.Unmarshal(v, &p) != nil || p.SHA == nil || !ghCommitOK(p.ghCommit) || p.Files == nil {
			return gateCommit{}, ghShape(name, args, answer)
		}
		for _, f := range p.Files {
			if !ghFileOK(f.ghFile) {
				return gateCommit{}, ghShape(name, args, answer)
			}
		}
		if len(pages) > 0 && !strings.EqualFold(*p.SHA, *pages[0].SHA) {
			return gateCommit{}, ghShape(name, args, answer)
		}
		pages = append(pages, p)
	}
	if len(pages) == 0 {
		return gateCommit{}, ghShape(name, args, answer)
	}
	c := pages[0].gate()
	var adds, dels float64
	for _, p := range pages {
		for _, f := range p.Files {
			c.Files = append(c.Files, gateFile{Name: *f.Filename})
			if f.PreviousFilename != nil {
				c.Files[len(c.Files)-1].Previous = *f.PreviousFilename
			}
			if f.Additions != nil {
				adds += *f.Additions
			}
			if f.Deletions != nil {
				dels += *f.Deletions
			}
		}
	}
	if !ghCommitComplete(pages[0], len(pages), len(c.Files), adds, dels) {
		return gateCommit{}, fmt.Errorf("the file list of commit %s of pull request #%d has %d files and may be incomplete", sha, pr, len(c.Files))
	}
	return c, nil
}

func ghCommitComplete(first ghCommitPage, pages, files int, adds, dels float64) bool {
	if files >= 3000 || (pages == 1 && len(first.Files) == 300) {
		return false
	}
	if cf := bytes.TrimSpace(first.ChangedFiles); len(cf) > 0 && string(cf) != "null" {
		n, err := strconv.Atoi(string(cf))
		if err != nil || n != files {
			return false
		}
	}
	if st := bytes.TrimSpace(first.Stats); len(st) > 0 && string(st) != "null" {
		var stats struct {
			Additions *float64 `json:"additions"`
			Deletions *float64 `json:"deletions"`
		}
		if !isJSONObject(st) || json.Unmarshal(st, &stats) != nil || stats.Additions == nil || stats.Deletions == nil ||
			*stats.Additions != adds || *stats.Deletions != dels {
			return false
		}
	}
	return true
}

// labelExists reads the whole label list, not a search: gh 2.97 prints
// nothing at all, not [], for a search that matches no label.
func (s ghSource) labelExists(ctx context.Context, label string) (bool, error) {
	var labels []struct {
		Name *string `json:"name"`
	}
	args := []string{"label", "list", "--json", "name", "--limit", strconv.Itoa(gateLimit)}
	if err := s.ghRead(ctx, "labels", &labels, args...); err != nil {
		return false, err
	}
	for _, l := range labels {
		if l.Name == nil {
			return false, ghShape("labels", args, []byte("a label has no name"))
		}
		if *l.Name == label {
			return true, nil
		}
	}
	if len(labels) >= gateLimit {
		return false, fmt.Errorf("the label list returned %d labels and may be incomplete", gateLimit)
	}
	return false, nil
}

func (s ghSource) openIssues(ctx context.Context, label string) ([]gateIssue, error) {
	name := "open " + label + " issues"
	args := []string{"issue", "list", "--label", label, "--state", "open", "--json", "number,url,title", "--limit", strconv.Itoa(gateLimit)}
	var issues []struct {
		Number *int    `json:"number"`
		URL    *string `json:"url"`
		Title  string  `json:"title"`
	}
	if err := s.ghRead(ctx, name, &issues, args...); err != nil {
		return nil, err
	}
	out := make([]gateIssue, 0, len(issues))
	for _, is := range issues {
		if is.Number == nil || is.URL == nil {
			return nil, ghShape(name, args, []byte("an issue has no number or url"))
		}
		out = append(out, gateIssue{Number: *is.Number, URL: *is.URL, Title: is.Title})
	}
	if len(out) >= gateLimit {
		return nil, gateDie("FAIL: the list of open %s issues returned %d of %d asked for and may be incomplete", label, len(out), gateLimit)
	}
	return out, nil
}

func (s ghSource) closingRefs(ctx context.Context, pr int, _ []gateIssue) ([]string, error) {
	name := fmt.Sprintf("pull request #%d", pr)
	args := []string{"pr", "view", strconv.Itoa(pr), "--json", "closingIssuesReferences"}
	var v struct {
		Refs *[]struct {
			URL *string `json:"url"`
		} `json:"closingIssuesReferences"`
	}
	if err := s.ghRead(ctx, name, &v, args...); err != nil {
		return nil, err
	}
	if v.Refs == nil {
		return nil, ghShape(name, args, []byte("closingIssuesReferences is not a list"))
	}
	var urls []string
	for _, r := range *v.Refs {
		if r.URL == nil {
			return nil, ghShape(name, args, []byte("a closing reference has no url"))
		}
		urls = append(urls, *r.URL)
	}
	return urls, nil
}
