package state

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

// DefaultQueueMaxMilestoneIssues is Q5's threshold when .agents/ateam.json sets no queue.max_milestone_issues: the
// adopter milestone #416 measured as overloaded held 15 open issues.
const DefaultQueueMaxMilestoneIssues = 15

// MilestoneCount is one open milestone as the forge's milestones listing reports it.
type MilestoneCount struct {
	Title        string `json:"title"`
	OpenIssues   int    `json:"open_issues"`
	ClosedIssues int    `json:"closed_issues"`
}

// MilestoneSnapshot is the last open-milestones read for one forge. Milestones live beside the cards, not as cards,
// so the board never renders them and a forge with no snapshot reads as unavailable rather than as zero milestones.
type MilestoneSnapshot struct {
	Items []MilestoneCount `json:"items"`
	// Unavailable is why the last read gave no usable list; Q5 and Q6 are then not checked.
	Unavailable string `json:"unavailable,omitempty"`
}

// SyncForgeMilestones reads the forge's open milestones in one listing call (paged on Gitea). The tea fallback and
// a forge with no REST or gh path return an error, which the caller records as unavailable (#416 design D3).
func SyncForgeMilestones(ctx context.Context, root string, cfg ForgeConfig) ([]MilestoneCount, error) {
	switch cfg.Provider {
	case ForgeGitHub:
		return fetchGitHubMilestones(ctx, root, cfg)
	case ForgeGitea:
		if cfg.BaseURL == "" && cfg.Owner == "" && cfg.Repo == "" {
			return nil, errors.New("milestones need the Gitea REST API; the tea fallback lists none")
		}
		validated, err := normalizeForgeConfig(cfg)
		if err != nil {
			return nil, err
		}
		return fetchGiteaMilestones(ctx, validated)
	}
	return nil, fmt.Errorf("no milestones on forge %q", cfg.Provider)
}

func fetchGitHubMilestones(ctx context.Context, root string, cfg ForgeConfig) ([]MilestoneCount, error) {
	// {owner}/{repo} are filled by gh from the checkout's remote when no repository is configured.
	path := "repos/{owner}/{repo}/milestones?state=open&per_page=100"
	if cfg.Owner != "" || cfg.Repo != "" || cfg.BaseURL != "" {
		validated, err := normalizeForgeConfig(cfg)
		if err != nil {
			return nil, err
		}
		path = "repos/" + validated.Owner + "/" + validated.Repo + "/milestones?state=open&per_page=100"
	}
	if _, err := exec.LookPath("gh"); err != nil {
		return nil, fmt.Errorf("gh CLI not found")
	}
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	// --jq prints one compact object per milestone across every page, which avoids parsing --paginate's
	// concatenated arrays.
	cmd := ghCommand(cctx, ReadToken(), "api", "--paginate", path, "--jq", ".[] | {title, open_issues, closed_issues}")
	cmd.Dir = root
	cmd.WaitDelay = 100 * time.Millisecond
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gh api milestones: %w", err)
	}
	return decodeMilestoneStream(out)
}

// decodeMilestoneStream reads a stream of milestone objects, or one array of them.
func decodeMilestoneStream(data []byte) ([]MilestoneCount, error) {
	items := []MilestoneCount{}
	dec := json.NewDecoder(bytes.NewReader(data))
	for {
		var raw json.RawMessage
		if err := dec.Decode(&raw); err == io.EOF {
			return items, nil
		} else if err != nil {
			return nil, fmt.Errorf("parse milestones: %w", err)
		}
		raw = bytes.TrimSpace(raw)
		if len(raw) > 0 && raw[0] == '[' {
			var list []MilestoneCount
			if err := json.Unmarshal(raw, &list); err != nil {
				return nil, fmt.Errorf("parse milestones: %w", err)
			}
			items = append(items, list...)
			continue
		}
		var m MilestoneCount
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, fmt.Errorf("parse milestones: %w", err)
		}
		if m.Title == "" {
			return nil, errors.New("parse milestones: an entry has no title")
		}
		items = append(items, m)
	}
}

func fetchGiteaMilestones(ctx context.Context, cfg ForgeConfig) ([]MilestoneCount, error) {
	client := &http.Client{Timeout: 3 * time.Second}
	token := giteaToken(ReadToken())
	base := fmt.Sprintf("%s/api/v1/repos/%s/%s/milestones?state=open", strings.TrimRight(cfg.BaseURL, "/"), cfg.Owner, cfg.Repo)
	items := []MilestoneCount{}
	seen := map[string]bool{}
	// Paged like fetchGiteaPages: stop on an empty page, or on one that adds nothing in case the server ignores page.
	for page := 1; ; page++ {
		path := fmt.Sprintf("%s&page=%d&limit=%d", base, page, giteaPageSize)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
		if err != nil {
			return nil, err
		}
		if token != "" {
			req.Header.Set("Authorization", "token "+token)
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode >= 300 {
			return nil, fmt.Errorf("gitea API %s returned %d", path, resp.StatusCode)
		}
		var list []MilestoneCount
		if err := json.Unmarshal(body, &list); err != nil {
			return nil, fmt.Errorf("parse milestones: %w", err)
		}
		added := 0
		for _, m := range list {
			if !seen[m.Title] {
				seen[m.Title] = true
				items = append(items, m)
				added++
			}
		}
		if added == 0 {
			return items, nil
		}
	}
}

// SetForgeMilestones records one forge's milestones read; an error replaces the list with its reason, so a later
// queue check reports unavailable instead of trusting an older list.
func (st *State) SetForgeMilestones(provider ForgeProvider, items []MilestoneCount, err error) {
	if st.ForgeMilestones == nil {
		st.ForgeMilestones = map[ForgeProvider]*MilestoneSnapshot{}
	}
	if err != nil {
		st.ForgeMilestones[provider] = &MilestoneSnapshot{Unavailable: err.Error()}
		return
	}
	st.ForgeMilestones[provider] = &MilestoneSnapshot{Items: items}
}

// Queue rule identifiers (#416 design D2).
const (
	QueueBlockerClosed    = "Q1"
	QueueNoBlocker        = "Q2"
	QueueDecisionRecorded = "Q3"
	QueueUnplaced         = "Q4"
	QueueOverloaded       = "Q5"
	QueueFinished         = "Q6"
	// QueueUnread is a status:blocked issue whose comments were not read, so Q1 and Q2 cannot be decided.
	QueueUnread = "unread"
)

// QueueFinding is one queue problem: the rule, the issue or milestone it names, and a one-line message.
type QueueFinding struct {
	Rule      string `json:"rule"`
	Issue     int    `json:"issue,omitempty"`
	Milestone string `json:"milestone,omitempty"`
	Message   string `json:"message"`
}

// QueueResult is the queue check over the forge cache. Unavailable is set when nothing is synced or the milestones
// read did not succeed; the issue rules are still listed in the second case.
type QueueResult struct {
	Findings    []QueueFinding `json:"findings,omitempty"`
	Unavailable string         `json:"unavailable,omitempty"`
}

// ComputeQueue runs the queue checks on the active forge's cache. It reads only the cache, so the session-start
// hook and doctor can show it without a forge call.
func (st *State) ComputeQueue() QueueResult {
	if st == nil || (st.Forge.Provider != ForgeGitHub && st.Forge.Provider != ForgeGitea) || st.SyncedAtFor(st.Forge).IsZero() {
		return QueueResult{Unavailable: NextNotSynced}
	}
	limit := st.QueueMaxMilestoneIssues
	if limit <= 0 {
		limit = DefaultQueueMaxMilestoneIssues
	}
	return computeQueue(st.Cards, string(st.Forge.Provider), st.ForgeMilestones[st.Forge.Provider], limit)
}

func computeQueue(cards []BoardCard, forge string, ms *MilestoneSnapshot, maxOpen int) QueueResult {
	var res QueueResult
	openNums := map[int]bool{}
	var issues []BoardCard
	for _, c := range cards {
		if c.Forge != forge || c.Column == "done" {
			continue
		}
		// PRs are in the open set as computeNext counts them, so a Depends on: naming an open PR still blocks.
		openNums[c.Number] = true
		if c.Type != "pr" {
			issues = append(issues, c)
		}
	}
	sort.Slice(issues, func(i, j int) bool { return issues[i].Number < issues[j].Number })
	for _, c := range issues {
		switch {
		case hasLabel(c.Labels, "status:blocked") && c.CommentsUnread:
			res.Findings = append(res.Findings, QueueFinding{Rule: QueueUnread, Issue: c.Number,
				Message: "status:blocked, but its comments were not read, so its blocker is unknown"})
		case hasLabel(c.Labels, "status:blocked") && len(c.DependsOn) == 0:
			res.Findings = append(res.Findings, QueueFinding{Rule: QueueNoBlocker, Issue: c.Number,
				Message: "status:blocked, but names no readable blocker (no Depends on: #n line)"})
		case hasLabel(c.Labels, "status:blocked") && !anyOpen(c.DependsOn, openNums):
			res.Findings = append(res.Findings, QueueFinding{Rule: QueueBlockerClosed, Issue: c.Number,
				Message: "status:blocked, but every Depends on: issue (" + refList(c.DependsOn) + ") is closed"})
		}
		if hasLabel(c.Labels, "status:needs-decision") && strings.TrimSpace(c.Decision) != "" {
			res.Findings = append(res.Findings, QueueFinding{Rule: QueueDecisionRecorded, Issue: c.Number,
				Message: "status:needs-decision, but a decision is recorded: " + c.Decision})
		}
	}
	// Q4 reuses next's placement rule, so the two commands never disagree about what is unplaced.
	for _, n := range computeNext(cards, forge).Unplaced {
		res.Findings = append(res.Findings, QueueFinding{Rule: QueueUnplaced, Issue: n,
			Message: "unplaced: no milestone, or missing from its milestone's Order: line"})
	}
	if ms == nil || ms.Unavailable != "" {
		res.Unavailable = "milestones not read"
		if ms != nil {
			res.Unavailable += ": " + ms.Unavailable
		}
		return res
	}
	items := append([]MilestoneCount(nil), ms.Items...)
	sort.Slice(items, func(i, j int) bool { return items[i].Title < items[j].Title })
	for _, m := range items {
		switch {
		case m.OpenIssues > maxOpen:
			res.Findings = append(res.Findings, QueueFinding{Rule: QueueOverloaded, Milestone: m.Title,
				Message: fmt.Sprintf("%d open issues, over queue.max_milestone_issues %d", m.OpenIssues, maxOpen)})
		case m.OpenIssues == 0:
			res.Findings = append(res.Findings, QueueFinding{Rule: QueueFinished, Milestone: m.Title,
				Message: "open, with no open issues left: finished, not closed"})
		}
	}
	return res
}

func anyOpen(nums []int, open map[int]bool) bool {
	for _, n := range nums {
		if open[n] {
			return true
		}
	}
	return false
}

func refList(nums []int) string {
	parts := make([]string, len(nums))
	for i, n := range nums {
		parts[i] = "#" + strconv.Itoa(n)
	}
	return strings.Join(parts, ", ")
}

// Field is the pickup-line value of queue=.
func (r QueueResult) Field() string {
	if r.Unavailable != "" {
		return "unavailable"
	}
	return strconv.Itoa(len(r.Findings))
}

// Format renders the ateam state queue-check report.
func (r QueueResult) Format() string {
	var b strings.Builder
	for _, f := range r.Findings {
		target := f.Milestone
		if f.Issue != 0 {
			target = "#" + strconv.Itoa(f.Issue)
		}
		fmt.Fprintf(&b, "%s %s %s\n", f.Rule, target, f.Message)
	}
	if r.Unavailable != "" {
		fmt.Fprintf(&b, "unavailable: %s\n", r.Unavailable)
	} else if len(r.Findings) == 0 {
		b.WriteString("queue: no findings\n")
	}
	return b.String()
}

// DoctorLine is the advice line ateam doctor prints from the cache.
func (r QueueResult) DoctorLine() string {
	if r.Unavailable != "" {
		return fmt.Sprintf("queue: unavailable (%s); run ateam state queue-check", r.Unavailable)
	}
	return fmt.Sprintf("queue: %d finding(s); run ateam state queue-check", len(r.Findings))
}
