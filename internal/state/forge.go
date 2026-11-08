package state

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/c360studio/ateam/internal/review"
)

// ForgeProvider identifies the issue/PR hosting provider.
type ForgeProvider string

// Supported forge providers.
const (
	ForgeGitHub ForgeProvider = "github"
	ForgeGitea  ForgeProvider = "gitea"
	ForgeLocal  ForgeProvider = "local"
)

// ForgeConfig holds resolved repository forge connection metadata.
type ForgeConfig struct {
	Provider ForgeProvider `json:"provider"`
	BaseURL  string        `json:"base_url,omitempty"`
	Owner    string        `json:"owner,omitempty"`
	Repo     string        `json:"repo,omitempty"`
}

// ForgeSnapshot is a forge configuration set aside while a committed list replaces it (#333).
type ForgeSnapshot struct {
	Forge  ForgeConfig   `json:"forge"`
	Forges []ForgeConfig `json:"forges,omitempty"`
}

// StandardLabels defines the canonical SemEngine/ATeam workflow labels seeded by ateam init.
var StandardLabels = []struct {
	Name        string
	Color       string
	Description string
}{
	{Name: "type:epic", Color: "5319e7", Description: "Tracking epic issue"},
	{Name: "status:needs-decision", Color: "d93f0b", Description: "Awaiting owner architectural or scope decision"},
	{Name: "status:blocked", Color: "b60205", Description: "Blocked by another issue, PR, or external dependency"},
	{Name: "class:flake", Color: "fbca04", Description: "Known test flake that blocks merges until resolved"},
}

var sshRemoteRe = regexp.MustCompile(`^(?:ssh://)?git@([^:/]+)[:/]([^/]+)/([^/]+?)(?:\.git)?$`)

// DetectForge determines whether a repo uses GitHub, Gitea, or local state.
func DetectForge(_ context.Context, _ string, remoteURL string) ForgeConfig {
	explicit := strings.ToLower(strings.TrimSpace(os.Getenv("ATEAM_FORGE")))
	giteaURL := strings.TrimRight(strings.TrimSpace(os.Getenv("GITEA_URL")), "/")

	host, owner, repo := ParseGitRemote(remoteURL)
	if explicit == "gitea" || (giteaURL != "" && explicit != "github") || strings.Contains(strings.ToLower(host), "gitea") {
		base := giteaURL
		if base == "" && host != "" {
			base = "https://" + host
		}
		return ForgeConfig{
			Provider: ForgeGitea,
			BaseURL:  base,
			Owner:    owner,
			Repo:     repo,
		}
	}
	if explicit == "github" || strings.Contains(strings.ToLower(host), "github.com") {
		base := strings.TrimRight(strings.TrimSpace(os.Getenv("GITHUB_API_URL")), "/")
		if base == "" {
			base = "https://api.github.com"
		}
		return ForgeConfig{
			Provider: ForgeGitHub,
			BaseURL:  base,
			Owner:    owner,
			Repo:     repo,
		}
	}
	if host != "" && owner != "" && repo != "" {
		// Default self-hosted non-github git remotes to Gitea-compatible v1 API.
		return ForgeConfig{
			Provider: ForgeGitea,
			BaseURL:  "https://" + host,
			Owner:    owner,
			Repo:     repo,
		}
	}
	return ForgeConfig{Provider: ForgeLocal}
}

// ParseGitRemote extracts host, owner, and repo name from SSH or HTTPS git remote URLs.
func ParseGitRemote(remoteURL string) (host, owner, repo string) {
	remoteURL = strings.TrimSpace(remoteURL)
	if remoteURL == "" {
		return "", "", ""
	}
	if m := sshRemoteRe.FindStringSubmatch(remoteURL); len(m) == 4 {
		return m[1], m[2], strings.TrimSuffix(m[3], ".git")
	}
	u, err := url.Parse(remoteURL)
	if err != nil || u.Host == "" {
		return "", "", ""
	}
	host = u.Host
	// An ssh:// port is the SSH port, not the web port the API URL is built from (#291).
	if u.Scheme == "ssh" && u.Port() != "" {
		host = strings.TrimSuffix(host, ":"+u.Port())
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 {
		return host, "", ""
	}
	return host, parts[len(parts)-2], strings.TrimSuffix(parts[len(parts)-1], ".git")
}

// MergeConfiguredForges combines the primary detected forge with any saved forge configs without duplicating providers.
func MergeConfiguredForges(primary ForgeConfig, saved []ForgeConfig) []ForgeConfig {
	var out []ForgeConfig
	seen := make(map[ForgeProvider]int)

	add := func(cfg ForgeConfig) {
		p := ForgeProvider(strings.ToLower(strings.TrimSpace(string(cfg.Provider))))
		if p != ForgeGitHub && p != ForgeGitea {
			return
		}
		cfg.Provider = p
		cfg.BaseURL = strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
		cfg.Owner = strings.TrimSpace(cfg.Owner)
		cfg.Repo = strings.TrimSpace(cfg.Repo)
		if cfg.Provider == ForgeGitHub && cfg.BaseURL == "" {
			cfg.BaseURL = "https://api.github.com"
		}
		if idx, ok := seen[p]; ok {
			if cfg.Owner != "" && cfg.Repo != "" {
				out[idx] = cfg
			}
			return
		}
		seen[p] = len(out)
		out = append(out, cfg)
	}

	add(primary)
	for _, s := range saved {
		add(s)
	}
	return out
}

// normalizeForgeConfig validates persisted settings before they can select a remote target.
func normalizeForgeConfig(cfg ForgeConfig) (ForgeConfig, error) {
	cfg.Provider = ForgeProvider(strings.ToLower(strings.TrimSpace(string(cfg.Provider))))
	if cfg.Provider != ForgeGitHub && cfg.Provider != ForgeGitea {
		return ForgeConfig{}, fmt.Errorf("forge provider must be github or gitea")
	}
	cfg.BaseURL = strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	cfg.Owner = strings.TrimSpace(cfg.Owner)
	cfg.Repo = strings.TrimSpace(cfg.Repo)
	for _, component := range []string{cfg.Owner, cfg.Repo} {
		if component == "" || component == "." || component == ".." || !forgePathComponent.MatchString(component) {
			return ForgeConfig{}, fmt.Errorf("forge owner and repo must be nonempty path components containing letters, digits, dot, underscore or hyphen")
		}
	}
	if cfg.Provider == ForgeGitHub && cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.github.com"
	}
	u, err := url.Parse(cfg.BaseURL)
	if err != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" || strings.Contains(cfg.BaseURL, "#") {
		return ForgeConfig{}, fmt.Errorf("forge base_url must be an HTTP(S) URL without credentials, query or fragment")
	}
	if cfg.Provider == ForgeGitHub && cfg.BaseURL != "https://api.github.com" {
		return ForgeConfig{}, fmt.Errorf("custom GitHub API URLs are not supported")
	}
	return cfg, nil
}

var forgePathComponent = regexp.MustCompile(`^[A-Za-z0-9_.-]+$`)

// SyncForgeBoard fetches issues and PR claims from GitHub or Gitea and converts them into BoardCards.
func SyncForgeBoard(ctx context.Context, root string, cfg ForgeConfig) ([]BoardCard, error) {
	var cards []BoardCard
	var err error
	switch cfg.Provider {
	case ForgeGitea:
		cards, err = syncGiteaBoard(ctx, root, cfg)
	case ForgeGitHub:
		cards, err = syncGitHubBoard(ctx, root, cfg)
	default:
		return nil, nil
	}
	// A record's model and effort reach pickup and session-start, so only the
	// checkout's policy spelling of them is cached (#295); without a readable
	// policy they are all unlisted.
	var policy review.Policy
	if data, readErr := os.ReadFile(filepath.Join(root, ".agents", "review-policy.json")); readErr == nil {
		if p, parseErr := review.ParsePolicy(data); parseErr == nil {
			policy = p
		}
	}
	for i := range cards {
		if cards[i].Review != nil {
			rec := policy.Canonical(*cards[i].Review)
			cards[i].Review = &rec
		}
	}
	return cards, err
}

type forgeIssuePayload struct {
	Number        int    `json:"number"`
	Title         string `json:"title"`
	Body          string `json:"body,omitempty"`
	State         string `json:"state"`
	HTMLURL       string `json:"html_url"`
	URL           string `json:"url"`
	Draft         bool   `json:"draft"`
	Merged        bool   `json:"merged,omitempty"`
	MergedAt      string `json:"mergedAt,omitempty"`
	GiteaMergedAt string `json:"merged_at,omitempty"`
	HeadRef       string `json:"headRefName"`
	HeadRefOid    string `json:"headRefOid,omitempty"`
	Labels        []struct {
		Name string `json:"name"`
	} `json:"labels"`
	Milestone *struct {
		Title string `json:"title"`
	} `json:"milestone"`
	PullRequest *struct {
		Draft  bool `json:"draft"`
		Merged bool `json:"merged,omitempty"`
	} `json:"pull_request,omitempty"`
	Head *struct {
		Ref string `json:"ref"`
		SHA string `json:"sha,omitempty"`
	} `json:"head,omitempty"`
	Comments      json.RawMessage `json:"comments,omitempty"`
	CommentBodies []string        `json:"-"`
	// Review, ReviewURL and HandoffURL are filled only for an open PR whose branch has a local worktree.
	Review     *review.RecordComment `json:"-"`
	ReviewURL  string                `json:"-"`
	HandoffURL string                `json:"-"`
}

// forgeComment is one issue or PR comment: gh names its link url, Gitea html_url.
type forgeComment struct {
	Body    string `json:"body"`
	URL     string `json:"url"`
	HTMLURL string `json:"html_url"`
}

func (c forgeComment) link() string {
	if c.HTMLURL != "" {
		return c.HTMLURL
	}
	return c.URL
}

// setPRComments fills the review record and the record and handoff-block comment links from a PR's comments,
// oldest first.
func (it *forgeIssuePayload) setPRComments(comments []forgeComment) {
	it.Review, it.ReviewURL = latestReview(comments)
	for i := len(comments) - 1; i >= 0; i-- {
		if isHandoffComment(comments[i].Body) {
			it.HandoffURL = comments[i].link()
			return
		}
	}
}

// isHandoffComment reports whether a comment body is a FormatFiveFieldPrompt block.
func isHandoffComment(body string) bool {
	body = strings.ReplaceAll(strings.TrimSpace(body), "\r\n", "\n")
	return strings.HasPrefix(body, "Goal: ") && strings.Contains(body, "\nChecks to run: ")
}

// worktreeBranches lists the branches checked out in root's worktrees: the PRs
// whose review record a sync fetches, so the cost follows this machine's open
// work, not the repository's (#295).
func worktreeBranches(ctx context.Context, root string) map[string]bool {
	branches := map[string]bool{}
	for _, wt := range parseWorktrees(runGitOutput(ctx, root, "worktree", "list", "--porcelain")) {
		if b := strings.TrimSpace(wt.Branch); b != "" {
			branches[b] = true
		}
	}
	return branches
}

// latestReview returns the newest review record among comments, oldest
// first, as the merge check keeps the latest, and its link; nil when none is a record.
func latestReview(comments []forgeComment) (*review.RecordComment, string) {
	for i := len(comments) - 1; i >= 0; i-- {
		if rec, ok := review.ParseRecordComment(comments[i].Body); ok {
			return &rec, comments[i].link()
		}
	}
	return nil, ""
}

func parseIssueComments(raw json.RawMessage) ([]string, int) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return nil, 0
	}
	if trimmed[0] == '[' {
		var list []struct {
			Body string `json:"body"`
		}
		if err := json.Unmarshal(trimmed, &list); err == nil {
			out := make([]string, 0, len(list))
			for _, c := range list {
				out = append(out, c.Body)
			}
			return out, len(out)
		}
	}
	var count int
	if err := json.Unmarshal(trimmed, &count); err == nil {
		return nil, count
	}
	return nil, 0
}

func syncGiteaBoard(ctx context.Context, root string, cfg ForgeConfig) ([]BoardCard, error) {
	if cfg.BaseURL != "" || cfg.Owner != "" || cfg.Repo != "" {
		validated, err := normalizeForgeConfig(cfg)
		if err != nil {
			return nil, err
		}
		return fetchGiteaRESTCards(ctx, root, validated)
	}
	// Fallback to tea CLI if installed.
	// shortcut: the tea fallback lists only tea's default --limit (30) of issues and PRs; page it like the REST path if tea-only setups need older open work.
	if _, err := exec.LookPath("tea"); err == nil {
		return fetchTeaCLICards(ctx, root)
	}
	return nil, fmt.Errorf("gitea sync unavailable for %s/%s", cfg.Owner, cfg.Repo)
}

func fetchGiteaRESTCards(ctx context.Context, root string, cfg ForgeConfig) ([]BoardCard, error) {
	client := &http.Client{Timeout: 3 * time.Second}
	token := giteaToken(ReadToken())
	baseURL := strings.TrimRight(cfg.BaseURL, "/")

	repoAPI := fmt.Sprintf("%s/api/v1/repos/%s/%s", baseURL, cfg.Owner, cfg.Repo)
	endpoints := []struct {
		path string
		// closedSort orders the capped closed page; the issues endpoint has no sort parameter, so closed issues stay newest-created first.
		closedSort string
		isPR       bool
	}{
		{path: repoAPI + "/issues?type=issues"},
		{path: repoAPI + "/pulls", closedSort: "&sort=recentupdate", isPR: true},
	}
	// Open work is paged to exhaustion in creation order, so an item updated mid-paging cannot jump to a page already
	// read. An item that closes mid-paging still shifts later items up a page, so one of them can be missed until the
	// next sync (#162). Closed work is capped at closedItemCap.
	opens := make([][]forgeIssuePayload, len(endpoints))
	closeds := make([][]forgeIssuePayload, len(endpoints))
	var fetches []func(context.Context) error
	for i, endpoint := range endpoints {
		fetches = append(fetches,
			func(ctx context.Context) error {
				items, err := fetchGiteaPages(ctx, client, token, endpoint.path+queryJoin(endpoint.path)+"state=open")
				opens[i] = items
				return err
			},
			func(ctx context.Context) error {
				items, err := fetchGiteaPage(ctx, client, token, fmt.Sprintf("%s%sstate=closed&limit=%d%s", endpoint.path, queryJoin(endpoint.path), closedItemCap, endpoint.closedSort))
				closeds[i] = items
				return err
			})
	}
	if err := runConcurrently(ctx, fetches...); err != nil {
		return nil, err
	}

	merged := make([][]forgeIssuePayload, len(endpoints))
	var wg sync.WaitGroup
	slots := make(chan struct{}, giteaCommentFetchLimit)
	branches := worktreeBranches(ctx, root)
	for i, endpoint := range endpoints {
		merged[i] = mergeOpenClosed(opens[i], closeds[i])
		if endpoint.isPR {
			// Gitea serves a pull request's comments on the issue comments URL.
			for j := range merged[i] {
				item := &merged[i][j]
				if !strings.EqualFold(item.State, "open") || item.Head == nil || !branches[item.Head.Ref] {
					continue
				}
				commentsURL := fmt.Sprintf("%s/issues/%d/comments", repoAPI, item.Number)
				wg.Go(func() {
					slots <- struct{}{}
					defer func() { <-slots }()
					item.setPRComments(fetchGiteaIssueComments(ctx, client, token, commentsURL))
				})
			}
			continue
		}
		for j := range merged[i] {
			item := &merged[i][j]
			if _, count := parseIssueComments(item.Comments); count == 0 {
				continue
			}
			commentsURL := fmt.Sprintf("%s/issues/%d/comments", repoAPI, item.Number)
			wg.Go(func() {
				slots <- struct{}{}
				defer func() { <-slots }()
				for _, c := range fetchGiteaIssueComments(ctx, client, token, commentsURL) {
					item.CommentBodies = append(item.CommentBodies, c.Body)
				}
			})
		}
	}
	wg.Wait()

	var all []BoardCard
	for i, endpoint := range endpoints {
		for _, item := range merged[i] {
			all = append(all, mapForgeItemToCard(item, endpoint.isPR, "gitea"))
		}
	}
	return all, nil
}

// giteaCommentFetchLimit caps concurrent per-issue comment fetches: one at a time could spend the 3s pickup budget on a
// repository with many discussed issues, and one request per issue at once could flood a small Gitea server (#162).
const giteaCommentFetchLimit = 8

// closedItemCap is how many of the most recently updated closed issues and pull requests a forge sync keeps; open items are never capped.
const closedItemCap = 50

// giteaPageSize is the limit asked for per open-list page; the server may clamp it lower ([api] MAX_RESPONSE_ITEMS).
const giteaPageSize = 50

func queryJoin(path string) string {
	if strings.Contains(path, "?") {
		return "&"
	}
	return "?"
}

// runConcurrently runs every fn at once, joins them all, and returns the first failure; that failure cancels the others' context.
func runConcurrently(ctx context.Context, fns ...func(context.Context) error) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var (
		wg    sync.WaitGroup
		once  sync.Once
		first error
	)
	for _, fn := range fns {
		wg.Go(func() {
			if err := fn(ctx); err != nil {
				once.Do(func() {
					first = err
					cancel()
				})
			}
		})
	}
	wg.Wait()
	return first
}

// mergeOpenClosed keeps one item per number. The closed copy wins: an item listed in both was closed between the two reads.
func mergeOpenClosed(open, closed []forgeIssuePayload) []forgeIssuePayload {
	isClosed := make(map[int]bool, len(closed))
	for _, it := range closed {
		isClosed[it.Number] = true
	}
	merged := make([]forgeIssuePayload, 0, len(open)+len(closed))
	for _, it := range open {
		if !isClosed[it.Number] {
			merged = append(merged, it)
		}
	}
	return append(merged, closed...)
}

// fetchGiteaPages reads an open-state listing page by page. It stops on an empty page, not a short one, because the
// server may clamp the page size; it also stops on a page that adds no new numbers, in case the server ignores page.
func fetchGiteaPages(ctx context.Context, client *http.Client, token, base string) ([]forgeIssuePayload, error) {
	var all []forgeIssuePayload
	seen := make(map[int]bool)
	for page := 1; ; page++ {
		items, err := fetchGiteaPage(ctx, client, token, fmt.Sprintf("%s&page=%d&limit=%d", base, page, giteaPageSize))
		if err != nil {
			return nil, err
		}
		if len(items) == 0 {
			return all, nil
		}
		added := 0
		for _, it := range items {
			if !seen[it.Number] {
				seen[it.Number] = true
				all = append(all, it)
				added++
			}
		}
		if added == 0 {
			return all, nil
		}
	}
}

func fetchGiteaPage(ctx context.Context, client *http.Client, token, path string) ([]forgeIssuePayload, error) {
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
	var items []forgeIssuePayload
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func fetchGiteaIssueComments(ctx context.Context, client *http.Client, token, commentsURL string) []forgeComment {
	cctx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(cctx, http.MethodGet, commentsURL, nil)
	if err != nil {
		return nil
	}
	if token != "" {
		req.Header.Set("Authorization", "token "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil
	}
	var comments []forgeComment
	if json.Unmarshal(raw, &comments) != nil {
		return nil
	}
	return comments
}

func fetchTeaCLICards(ctx context.Context, root string) ([]BoardCard, error) {
	cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cctx, "tea", "issues", "list", "--output", "json")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var items []forgeIssuePayload
	if err := json.Unmarshal(out, &items); err != nil {
		return nil, err
	}
	var cards []BoardCard
	for _, it := range items {
		cards = append(cards, mapForgeItemToCard(it, false, "gitea"))
	}
	prCmd := exec.CommandContext(cctx, "tea", "pulls", "list", "--output", "json")
	prCmd.Dir = root
	prOut, err := prCmd.Output()
	if err != nil {
		return nil, err
	}
	var prs []forgeIssuePayload
	if err := json.Unmarshal(prOut, &prs); err != nil {
		return nil, err
	}
	for _, pr := range prs {
		cards = append(cards, mapForgeItemToCard(pr, true, "gitea"))
	}
	return cards, nil
}

// ghCommentTimeout bounds the open-issues-with-comments call when the caller set no deadline. A variable so a test
// can prove comment expansion without racing the bound on a loaded machine (#188).
var ghCommentTimeout = 2500 * time.Millisecond

// ghCommentBudget is the sub-timeout for the open-issues-with-comments call: half of any parent deadline, so the
// comment-free fallback keeps the other half, and never more than ghCommentTimeout, so a hung call under a long
// deadline (the UI's 10s sync) still falls back quickly. On this repository the call takes about 0.8s, which the old
// fixed 750ms cap under the 3s pickup budget cut off on most syncs (#162).
func ghCommentBudget(ctx context.Context) time.Duration {
	deadline, ok := ctx.Deadline()
	if !ok {
		return ghCommentTimeout
	}
	return min(time.Until(deadline)/2, ghCommentTimeout)
}

func syncGitHubBoard(ctx context.Context, root string, cfg ForgeConfig) ([]BoardCard, error) {
	var repoArgs []string
	if cfg.Owner != "" || cfg.Repo != "" || cfg.BaseURL != "" {
		validated, err := normalizeForgeConfig(cfg)
		if err != nil {
			return nil, err
		}
		repoArgs = []string{"--repo", "github.com/" + validated.Owner + "/" + validated.Repo}
	}
	if _, err := exec.LookPath("gh"); err != nil {
		return nil, fmt.Errorf("gh CLI not found")
	}
	var cards []BoardCard

	// G-06: Attempt issue fetch with comments under a bounded sub-timeout, leaving the rest for the comment-free fallback.
	issueTimeout := ghCommentBudget(ctx)
	// shortcut: open lists stop at 1000 items with no caveat; page them or record a sync caveat when a repo nears 1000 open issues or PRs.
	const openLimit = "1000"
	closedLimit := strconv.Itoa(closedItemCap)
	// Closed lists are ordered by last update, so an old item that just closed or merged stays inside the cap. They come
	// from GitHub search, whose index lags: a just-merged PR can be missing for a sync, labelling its worktree `no PR`.
	const closedSort = "sort:updated-desc"
	ghList := func(ctx context.Context, timeout time.Duration, args ...string) ([]byte, error) {
		cctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		cmd := ghCommand(cctx, ReadToken(), append(args, repoArgs...)...)
		cmd.Dir = root
		cmd.WaitDelay = 100 * time.Millisecond
		return cmd.Output()
	}
	type ghPR struct {
		forgeIssuePayload
		IsDraft bool `json:"isDraft"`
	}
	const prFields = "number,title,state,isDraft,headRefName,headRefOid,mergedAt,labels,url"
	var openIssues, closedIssues []forgeIssuePayload
	var openPRs, closedPRs []ghPR
	// The four lists run at once, so the sync costs the slowest call rather than the sum (the pickup budget is 3s per forge).
	err := runConcurrently(ctx,
		func(ctx context.Context) error {
			out, err := ghList(ctx, issueTimeout, "issue", "list", "--state", "open", "--limit", openLimit, "--json", "number,title,state,labels,milestone,url,body,comments")
			if err != nil {
				out, err = ghList(ctx, 5*time.Second, "issue", "list", "--state", "open", "--limit", openLimit, "--json", "number,title,state,labels,milestone,url,body")
				if err != nil {
					return fmt.Errorf("gh issue list failed: %w", err)
				}
			}
			if err := json.Unmarshal(out, &openIssues); err != nil {
				return fmt.Errorf("parse gh issue list JSON: %w", err)
			}
			return nil
		},
		// Closed issues skip comment expansion: their decision summaries are not needed.
		func(ctx context.Context) error {
			out, err := ghList(ctx, 5*time.Second, "issue", "list", "--state", "closed", "--limit", closedLimit, "--search", closedSort, "--json", "number,title,state,labels,milestone,url,body")
			if err != nil {
				return fmt.Errorf("gh issue list failed: %w", err)
			}
			if err := json.Unmarshal(out, &closedIssues); err != nil {
				return fmt.Errorf("parse gh issue list JSON: %w", err)
			}
			return nil
		},
		func(ctx context.Context) error {
			out, err := ghList(ctx, 5*time.Second, "pr", "list", "--state", "open", "--limit", openLimit, "--json", prFields)
			if err != nil {
				return fmt.Errorf("gh pr list failed: %w", err)
			}
			if err := json.Unmarshal(out, &openPRs); err != nil {
				return fmt.Errorf("parse gh pr list JSON: %w", err)
			}
			return nil
		},
		func(ctx context.Context) error {
			out, err := ghList(ctx, 5*time.Second, "pr", "list", "--state", "closed", "--limit", closedLimit, "--search", closedSort, "--json", prFields)
			if err != nil {
				return fmt.Errorf("gh pr list failed: %w", err)
			}
			if err := json.Unmarshal(out, &closedPRs); err != nil {
				return fmt.Errorf("parse gh pr list JSON: %w", err)
			}
			return nil
		},
	)
	if err != nil {
		return nil, err
	}
	// Review records: one comments call per open PR with a local worktree, each under the comment budget; a call
	// that fails or runs out of budget leaves that card without a record until the next sync.
	branches := worktreeBranches(ctx, root)
	commentTimeout := ghCommentBudget(ctx)
	var wg sync.WaitGroup
	slots := make(chan struct{}, giteaCommentFetchLimit)
	for i := range openPRs {
		pr := &openPRs[i]
		if !branches[pr.HeadRef] {
			continue
		}
		wg.Go(func() {
			slots <- struct{}{}
			defer func() { <-slots }()
			out, err := ghList(ctx, commentTimeout, "pr", "view", strconv.Itoa(pr.Number), "--json", "comments")
			if err != nil {
				return
			}
			var view struct {
				Comments []forgeComment `json:"comments"`
			}
			if json.Unmarshal(out, &view) != nil {
				return
			}
			pr.setPRComments(view.Comments)
		})
	}
	wg.Wait()
	for _, it := range mergeOpenClosed(openIssues, closedIssues) {
		cards = append(cards, mapForgeItemToCard(it, false, "github"))
	}
	prPayloads := func(prs []ghPR) []forgeIssuePayload {
		out := make([]forgeIssuePayload, 0, len(prs))
		for _, pr := range prs {
			pr.Draft = pr.IsDraft
			out = append(out, pr.forgeIssuePayload)
		}
		return out
	}
	for _, pr := range mergeOpenClosed(prPayloads(openPRs), prPayloads(closedPRs)) {
		cards = append(cards, mapForgeItemToCard(pr, true, "github"))
	}
	return cards, nil
}

func mapForgeItemToCard(it forgeIssuePayload, isPR bool, forgeName string) BoardCard {
	var labels []string
	for _, l := range it.Labels {
		if l.Name != "" {
			labels = append(labels, l.Name)
		}
	}
	branch := it.HeadRef
	if branch == "" && it.Head != nil {
		branch = it.Head.Ref
	}
	headSHA := strings.TrimSpace(it.HeadRefOid)
	if headSHA == "" && it.Head != nil {
		headSHA = strings.TrimSpace(it.Head.SHA)
	}
	draft := it.Draft || (it.PullRequest != nil && it.PullRequest.Draft)
	merged := isPR && (it.Merged || (it.PullRequest != nil && it.PullRequest.Merged) || strings.EqualFold(it.State, "merged") || strings.TrimSpace(it.MergedAt) != "" || strings.TrimSpace(it.GiteaMergedAt) != "")
	cardType := "issue"
	if isPR {
		cardType = "pr"
	}
	for _, l := range labels {
		if l == "type:epic" {
			cardType = "epic"
		}
	}
	col := classifyCardColumn(strings.ToLower(it.State), isPR, draft, labels)
	ms := ""
	if it.Milestone != nil {
		ms = it.Milestone.Title
	}
	u := it.HTMLURL
	if u == "" {
		u = it.URL
	}
	prefix := "GH"
	if forgeName == "gitea" {
		prefix = "GT"
	}
	if isPR {
		prefix += "-PR"
	}
	return BoardCard{
		ID:         fmt.Sprintf("%s-%d", prefix, it.Number),
		Number:     it.Number,
		Title:      it.Title,
		Column:     col,
		Type:       cardType,
		Merged:     merged,
		HeadSHA:    headSHA,
		Labels:     labels,
		Agent:      inferAgentFromBranch(branch),
		Branch:     branch,
		Milestone:  ms,
		Decision:   extractDecisionSummary(it, labels),
		Review:     it.Review,
		ReviewURL:  it.ReviewURL,
		HandoffURL: it.HandoffURL,
		Forge:      forgeName,
		URL:        u,
		UpdatedAt:  time.Now().UTC(),
	}
}

func extractDecisionSummary(it forgeIssuePayload, labels []string) string {
	bodies, _ := parseIssueComments(it.Comments)
	bodies = append(bodies, it.CommentBodies...)
	for i := len(bodies) - 1; i >= 0; i-- {
		if line := findDecisionLine(bodies[i]); line != "" {
			return line
		}
	}
	if line := findDecisionLine(it.Body); line != "" {
		return line
	}
	for _, l := range labels {
		if strings.HasPrefix(strings.ToLower(l), "decision:") {
			return l
		}
	}
	return ""
}

func findDecisionLine(text string) string {
	for _, raw := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(raw)
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "decision:") || strings.HasPrefix(lower, "ruling:") {
			if len(line) > 120 {
				line = line[:120]
			}
			return line
		}
	}
	return ""
}

func classifyCardColumn(stateStr string, isPR, draft bool, labels []string) string {
	if stateStr == "closed" || stateStr == "merged" {
		return "done"
	}
	for _, l := range labels {
		switch strings.ToLower(l) {
		case "status:needs-decision":
			return "needs-decision"
		case "status:blocked":
			return "blocked"
		}
	}
	if isPR {
		if draft {
			return "in-progress"
		}
		return "in-review"
	}
	return "backlog"
}

// HandoffPost is what posting a handoff comment learned about the branch's open PR; the PR fields feed its commit
// statuses (#302) and are empty when the forge did not say.
type HandoffPost struct {
	Posted     bool
	Existing   bool // the PR's newest handoff comment already had this body, so nothing was sent (#320)
	Number     int
	HeadSHA    string
	PRURL      string
	CommentURL string
}

// ErrPostUncertain marks a handoff comment send that timed out: the forge may hold the comment or not (#320).
var ErrPostUncertain = errors.New("outcome unknown")

// handoffCommentReadTimeout bounds the pre-post comment read, apart from the post's own budget (#320).
const handoffCommentReadTimeout = 2 * time.Second

// handoffPostTimeout bounds the handoff comment post. A variable so a test can time a send out quickly (#320).
var handoffPostTimeout = 5 * time.Second

func isTimeout(err error) bool {
	var ne net.Error
	return errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &ne) && ne.Timeout())
}

// newestHandoffMatches returns the newest five-field handoff comment when its body equals body. Only the newest
// counts, so a deliberate A, B, A sequence posts the second A (#320).
func newestHandoffMatches(comments []forgeComment, body string) (forgeComment, bool) {
	norm := func(s string) string { return strings.ReplaceAll(strings.TrimSpace(s), "\r\n", "\n") }
	for i := len(comments) - 1; i >= 0; i-- {
		if isHandoffComment(comments[i].Body) {
			return comments[i], norm(comments[i].Body) == norm(body)
		}
	}
	return forgeComment{}, false
}

// ghNewestHandoffMatches reads the branch PR's comments under handoffCommentReadTimeout; a failed read matches nothing.
func ghNewestHandoffMatches(ctx context.Context, root, branch, body string) (forgeComment, bool) {
	rctx, cancel := context.WithTimeout(ctx, handoffCommentReadTimeout)
	defer cancel()
	read := ghCommand(rctx, ReadToken(), "pr", "view", branch, "--json", "comments")
	read.Dir = root
	out, err := read.Output()
	var view struct {
		Comments []forgeComment `json:"comments"`
	}
	if err != nil || json.Unmarshal(out, &view) != nil {
		return forgeComment{}, false
	}
	return newestHandoffMatches(view.Comments, body)
}

// PostPRHandoffComment posts the 5-field handoff block to the open PR for branch on GitHub or Gitea. When the
// PR's newest handoff comment already has this body it sends nothing and sets Existing; a failed comment read
// posts anyway. A send that times out returns an error wrapping ErrPostUncertain (#320).
func PostPRHandoffComment(ctx context.Context, root string, cfg ForgeConfig, branch, body string) (HandoffPost, error) {
	var post HandoffPost
	branch = strings.TrimSpace(branch)
	body = strings.TrimSpace(body)
	if branch == "" || body == "" {
		return post, nil
	}
	switch cfg.Provider {
	case ForgeGitHub:
		if _, err := exec.LookPath("gh"); err != nil {
			return post, fmt.Errorf("gh CLI not found")
		}
		if c, ok := ghNewestHandoffMatches(ctx, root, branch, body); ok {
			post.Posted, post.Existing, post.CommentURL = true, true, c.link()
		}
		cctx, cancel := context.WithTimeout(ctx, handoffPostTimeout)
		defer cancel()
		if !post.Existing {
			cmd := ghCommand(cctx, WriteToken(), "pr", "comment", branch, "--body", body)
			cmd.Dir = root
			var stderr bytes.Buffer
			cmd.Stderr = &stderr
			out, err := cmd.Output()
			if err != nil {
				// A deadline that stopped gh from starting sent nothing; only a started send is uncertain.
				if cmd.Process != nil && errors.Is(cctx.Err(), context.DeadlineExceeded) {
					return post, fmt.Errorf("gh pr comment %s timed out: %w", branch, ErrPostUncertain)
				}
				return post, fmt.Errorf("gh pr comment %s: %w (%s)", branch, err, strings.TrimSpace(stderr.String()))
			}
			post.Posted = true
			// gh pr comment prints the new comment's URL on stdout.
			if lines := strings.Fields(string(out)); len(lines) > 0 && strings.HasPrefix(lines[len(lines)-1], "https://") {
				post.CommentURL = lines[len(lines)-1]
			}
		}
		view := ghCommand(cctx, ReadToken(), "pr", "view", branch, "--json", "number,headRefOid,url")
		view.Dir = root
		if viewOut, err := view.Output(); err == nil {
			var pr struct {
				Number     int    `json:"number"`
				HeadRefOid string `json:"headRefOid"`
				URL        string `json:"url"`
			}
			if json.Unmarshal(viewOut, &pr) == nil {
				post.Number, post.HeadSHA, post.PRURL = pr.Number, pr.HeadRefOid, pr.URL
			}
		}
		return post, nil

	case ForgeGitea:
		if cfg.BaseURL == "" || cfg.Owner == "" || cfg.Repo == "" {
			return post, fmt.Errorf("gitea PR comment requires BaseURL, Owner, and Repo")
		}
		client := &http.Client{Timeout: handoffPostTimeout}
		token := giteaToken(WriteToken())
		pullsURL := fmt.Sprintf("%s/api/v1/repos/%s/%s/pulls?state=open", strings.TrimRight(cfg.BaseURL, "/"), cfg.Owner, cfg.Repo)
		prs, err := fetchGiteaPages(ctx, client, token, pullsURL)
		if err != nil {
			return post, fmt.Errorf("gitea pulls: %w", err)
		}
		for _, pr := range prs {
			ref := pr.HeadRef
			if ref == "" && pr.Head != nil {
				ref = pr.Head.Ref
			}
			if ref == branch {
				post.Number, post.PRURL = pr.Number, pr.HTMLURL
				if pr.Head != nil {
					post.HeadSHA = pr.Head.SHA
				}
				break
			}
		}
		if post.Number == 0 {
			return HandoffPost{}, nil
		}
		commentURL := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d/comments", strings.TrimRight(cfg.BaseURL, "/"), cfg.Owner, cfg.Repo, post.Number)
		// fetchGiteaIssueComments returns nil on any failure, which posts as before.
		if c, ok := newestHandoffMatches(fetchGiteaIssueComments(ctx, client, giteaToken(ReadToken()), commentURL), body); ok {
			post.Posted, post.Existing, post.CommentURL = true, true, c.link()
			return post, nil
		}
		payload, _ := json.Marshal(map[string]string{"body": body})
		postReq, err := http.NewRequestWithContext(ctx, http.MethodPost, commentURL, bytes.NewReader(payload))
		if err != nil {
			return HandoffPost{}, err
		}
		postReq.Header.Set("Content-Type", "application/json")
		if token != "" {
			postReq.Header.Set("Authorization", "token "+token)
		}
		// A context already done here sends nothing, so its error is not a timed-out send.
		if err := ctx.Err(); err != nil {
			return HandoffPost{}, fmt.Errorf("gitea post comment not sent: %w", err)
		}
		postResp, err := client.Do(postReq)
		if err != nil {
			if isTimeout(err) {
				return HandoffPost{}, fmt.Errorf("gitea post comment: %v: %w", err, ErrPostUncertain)
			}
			return HandoffPost{}, err
		}
		defer postResp.Body.Close()
		if postResp.StatusCode >= 300 {
			return HandoffPost{}, fmt.Errorf("gitea post comment returned %d", postResp.StatusCode)
		}
		post.Posted = true
		var created forgeComment
		if raw, err := io.ReadAll(io.LimitReader(postResp.Body, 1<<20)); err == nil && json.Unmarshal(raw, &created) == nil {
			post.CommentURL = created.link()
		}
		return post, nil

	default:
		return post, nil
	}
}

// BootstrapForgeLabels seeds the 4 standard workflow labels on GitHub or Gitea.
func BootstrapForgeLabels(ctx context.Context, root string, cfg ForgeConfig) ([]string, error) {
	var created []string
	switch cfg.Provider {
	case ForgeGitea:
		client := &http.Client{Timeout: 5 * time.Second}
		token := giteaToken(WriteToken())
		if cfg.BaseURL == "" || cfg.Owner == "" || cfg.Repo == "" {
			return nil, fmt.Errorf("gitea label bootstrap requires BaseURL, Owner, and Repo")
		}
		endpoint := fmt.Sprintf("%s/api/v1/repos/%s/%s/labels", strings.TrimRight(cfg.BaseURL, "/"), cfg.Owner, cfg.Repo)
		for _, lbl := range StandardLabels {
			payload, _ := json.Marshal(map[string]string{
				"name":        lbl.Name,
				"color":       "#" + lbl.Color,
				"description": lbl.Description,
			})
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
			if err != nil {
				return created, err
			}
			req.Header.Set("Content-Type", "application/json")
			if token != "" {
				req.Header.Set("Authorization", "token "+token)
			}
			resp, err := client.Do(req)
			if err != nil {
				return created, err
			}
			resp.Body.Close()
			if resp.StatusCode >= 300 && resp.StatusCode != http.StatusConflict && resp.StatusCode != http.StatusUnprocessableEntity {
				return created, fmt.Errorf("gitea create label %s returned %d", lbl.Name, resp.StatusCode)
			}
			created = append(created, lbl.Name)
		}
		return created, nil

	case ForgeGitHub:
		for _, lbl := range StandardLabels {
			cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			cmd := ghCommand(cctx, WriteToken(), "label", "create", lbl.Name, "--color", lbl.Color, "--description", lbl.Description, "--force")
			cmd.Dir = root
			err := cmd.Run()
			cancel()
			if err != nil {
				return created, fmt.Errorf("gh label create %s: %w", lbl.Name, err)
			}
			created = append(created, lbl.Name)
		}
		return created, nil

	default:
		return nil, fmt.Errorf("no remote forge configured (provider=%s)", cfg.Provider)
	}
}
