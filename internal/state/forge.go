package state

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
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
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 {
		return u.Host, "", ""
	}
	return u.Host, parts[len(parts)-2], strings.TrimSuffix(parts[len(parts)-1], ".git")
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
	switch cfg.Provider {
	case ForgeGitea:
		return syncGiteaBoard(ctx, root, cfg)
	case ForgeGitHub:
		return syncGitHubBoard(ctx, root, cfg)
	default:
		return nil, nil
	}
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
		return fetchGiteaRESTCards(ctx, validated)
	}
	// Fallback to tea CLI if installed.
	// shortcut: the tea fallback lists only tea's default --limit (30) of issues and PRs; page it like the REST path if tea-only setups need older open work.
	if _, err := exec.LookPath("tea"); err == nil {
		return fetchTeaCLICards(ctx, root)
	}
	return nil, fmt.Errorf("gitea sync unavailable for %s/%s", cfg.Owner, cfg.Repo)
}

func fetchGiteaRESTCards(ctx context.Context, cfg ForgeConfig) ([]BoardCard, error) {
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
	for i, endpoint := range endpoints {
		merged[i] = mergeOpenClosed(opens[i], closeds[i])
		if endpoint.isPR {
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
				item.CommentBodies = fetchGiteaIssueComments(ctx, client, token, commentsURL)
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

func fetchGiteaIssueComments(ctx context.Context, client *http.Client, token, commentsURL string) []string {
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
	bodies, _ := parseIssueComments(raw)
	return bodies
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
// comment-free fallback keeps the other half, or ghCommentTimeout with no deadline. On this repository the call takes
// about 0.8s, which a fixed 750ms cap under the 3s pickup budget cut off on most syncs (#162).
func ghCommentBudget(ctx context.Context) time.Duration {
	deadline, ok := ctx.Deadline()
	if !ok {
		return ghCommentTimeout
	}
	return time.Until(deadline) / 2
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
		ID:        fmt.Sprintf("%s-%d", prefix, it.Number),
		Number:    it.Number,
		Title:     it.Title,
		Column:    col,
		Type:      cardType,
		Merged:    merged,
		HeadSHA:   headSHA,
		Labels:    labels,
		Agent:     inferAgentFromBranch(branch),
		Branch:    branch,
		Milestone: ms,
		Decision:  extractDecisionSummary(it, labels),
		Forge:     forgeName,
		URL:       u,
		UpdatedAt: time.Now().UTC(),
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

// PostPRHandoffComment posts the 5-field handoff block to the open PR for branch on GitHub or Gitea.
func PostPRHandoffComment(ctx context.Context, root string, cfg ForgeConfig, branch, body string) (bool, error) {
	branch = strings.TrimSpace(branch)
	body = strings.TrimSpace(body)
	if branch == "" || body == "" {
		return false, nil
	}
	switch cfg.Provider {
	case ForgeGitHub:
		if _, err := exec.LookPath("gh"); err != nil {
			return false, fmt.Errorf("gh CLI not found")
		}
		cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		cmd := ghCommand(cctx, WriteToken(), "pr", "comment", branch, "--body", body)
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			return false, fmt.Errorf("gh pr comment %s: %w (%s)", branch, err, strings.TrimSpace(string(out)))
		}
		return true, nil

	case ForgeGitea:
		if cfg.BaseURL == "" || cfg.Owner == "" || cfg.Repo == "" {
			return false, fmt.Errorf("gitea PR comment requires BaseURL, Owner, and Repo")
		}
		client := &http.Client{Timeout: 5 * time.Second}
		token := giteaToken(WriteToken())
		pullsURL := fmt.Sprintf("%s/api/v1/repos/%s/%s/pulls?state=open", strings.TrimRight(cfg.BaseURL, "/"), cfg.Owner, cfg.Repo)
		prs, err := fetchGiteaPages(ctx, client, token, pullsURL)
		if err != nil {
			return false, fmt.Errorf("gitea pulls: %w", err)
		}
		prNum := 0
		for _, pr := range prs {
			ref := pr.HeadRef
			if ref == "" && pr.Head != nil {
				ref = pr.Head.Ref
			}
			if ref == branch {
				prNum = pr.Number
				break
			}
		}
		if prNum == 0 {
			return false, nil
		}
		commentURL := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d/comments", strings.TrimRight(cfg.BaseURL, "/"), cfg.Owner, cfg.Repo, prNum)
		payload, _ := json.Marshal(map[string]string{"body": body})
		postReq, err := http.NewRequestWithContext(ctx, http.MethodPost, commentURL, bytes.NewReader(payload))
		if err != nil {
			return false, err
		}
		postReq.Header.Set("Content-Type", "application/json")
		if token != "" {
			postReq.Header.Set("Authorization", "token "+token)
		}
		postResp, err := client.Do(postReq)
		if err != nil {
			return false, err
		}
		postResp.Body.Close()
		if postResp.StatusCode >= 300 {
			return false, fmt.Errorf("gitea post comment returned %d", postResp.StatusCode)
		}
		return true, nil

	default:
		return false, nil
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
