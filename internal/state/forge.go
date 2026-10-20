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
	"strings"
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
	Number  int    `json:"number"`
	Title   string `json:"title"`
	Body    string `json:"body,omitempty"`
	State   string `json:"state"`
	HTMLURL string `json:"html_url"`
	URL     string `json:"url"`
	Draft   bool   `json:"draft"`
	HeadRef string `json:"headRefName"`
	Labels  []struct {
		Name string `json:"name"`
	} `json:"labels"`
	Milestone *struct {
		Title string `json:"title"`
	} `json:"milestone"`
	PullRequest *struct {
		Draft bool `json:"draft"`
	} `json:"pull_request,omitempty"`
	Head *struct {
		Ref string `json:"ref"`
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
	if _, err := exec.LookPath("tea"); err == nil {
		return fetchTeaCLICards(ctx, root)
	}
	return nil, fmt.Errorf("gitea sync unavailable for %s/%s", cfg.Owner, cfg.Repo)
}

func fetchGiteaRESTCards(ctx context.Context, cfg ForgeConfig) ([]BoardCard, error) {
	client := &http.Client{Timeout: 3 * time.Second}
	token := strings.TrimSpace(os.Getenv("GITEA_TOKEN"))
	if token == "" {
		token = strings.TrimSpace(os.Getenv("ATEAM_GITEA_TOKEN"))
	}
	baseURL := strings.TrimRight(cfg.BaseURL, "/")

	var all []BoardCard
	for _, endpoint := range []struct {
		path string
		isPR bool
	}{
		{path: fmt.Sprintf("%s/api/v1/repos/%s/%s/issues?type=issues&state=all&limit=50", baseURL, cfg.Owner, cfg.Repo), isPR: false},
		{path: fmt.Sprintf("%s/api/v1/repos/%s/%s/pulls?state=all&limit=50", baseURL, cfg.Owner, cfg.Repo), isPR: true},
	} {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.path, nil)
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
			return nil, fmt.Errorf("gitea API %s returned %d", endpoint.path, resp.StatusCode)
		}
		var items []forgeIssuePayload
		if err := json.Unmarshal(body, &items); err != nil {
			return nil, err
		}
		for _, item := range items {
			if !endpoint.isPR {
				if _, count := parseIssueComments(item.Comments); count > 0 {
					commentsURL := fmt.Sprintf("%s/api/v1/repos/%s/%s/issues/%d/comments", baseURL, cfg.Owner, cfg.Repo, item.Number)
					item.CommentBodies = fetchGiteaIssueComments(ctx, client, token, commentsURL)
				}
			}
			all = append(all, mapForgeItemToCard(item, endpoint.isPR, "gitea"))
		}
	}
	return all, nil
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

	// G-06: Attempt issue fetch with comments under a bounded sub-timeout that leaves at least 2/3 of any parent deadline for comment-free fallback + PR queries.
	issueTimeout := 2500 * time.Millisecond
	if deadline, ok := ctx.Deadline(); ok {
		issueTimeout = 750 * time.Millisecond
		if sub := time.Until(deadline) / 3; sub > 0 && sub < issueTimeout {
			issueTimeout = sub
		}
	}
	issueCtx, issueCancel := context.WithTimeout(ctx, issueTimeout)
	issueCmd := exec.CommandContext(issueCtx, "gh", "issue", "list", "--state", "all", "--limit", "50", "--json", "number,title,state,labels,milestone,url,body,comments")
	issueCmd.Args = append(issueCmd.Args, repoArgs...)
	issueCmd.Dir = root
	issueCmd.WaitDelay = 100 * time.Millisecond
	issueOut, err := issueCmd.Output()
	issueCancel()
	if err != nil {
		fallbackCtx, fallbackCancel := context.WithTimeout(ctx, 5*time.Second)
		fallbackCmd := exec.CommandContext(fallbackCtx, "gh", "issue", "list", "--state", "all", "--limit", "50", "--json", "number,title,state,labels,milestone,url,body")
		fallbackCmd.Args = append(fallbackCmd.Args, repoArgs...)
		fallbackCmd.Dir = root
		fallbackCmd.WaitDelay = 100 * time.Millisecond
		issueOut, err = fallbackCmd.Output()
		fallbackCancel()
		if err != nil {
			return nil, fmt.Errorf("gh issue list failed: %w", err)
		}
	}

	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var issues []forgeIssuePayload
	if err := json.Unmarshal(issueOut, &issues); err != nil {
		return nil, fmt.Errorf("parse gh issue list JSON: %w", err)
	}
	for _, it := range issues {
		cards = append(cards, mapForgeItemToCard(it, false, "github"))
	}

	prCmd := exec.CommandContext(cctx, "gh", "pr", "list", "--state", "all", "--limit", "50", "--json", "number,title,state,isDraft,headRefName,labels,url")
	prCmd.Args = append(prCmd.Args, repoArgs...)
	prCmd.Dir = root
	prCmd.WaitDelay = 100 * time.Millisecond
	prOut, err := prCmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gh pr list failed: %w", err)
	}
	var prs []struct {
		forgeIssuePayload
		IsDraft bool `json:"isDraft"`
	}
	if err := json.Unmarshal(prOut, &prs); err != nil {
		return nil, fmt.Errorf("parse gh pr list JSON: %w", err)
	}
	for _, pr := range prs {
		pr.Draft = pr.IsDraft
		cards = append(cards, mapForgeItemToCard(pr.forgeIssuePayload, true, "github"))
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
	draft := it.Draft || (it.PullRequest != nil && it.PullRequest.Draft)
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
		cmd := exec.CommandContext(cctx, "gh", "pr", "comment", branch, "--body", body)
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
		token := strings.TrimSpace(os.Getenv("GITEA_TOKEN"))
		if token == "" {
			token = strings.TrimSpace(os.Getenv("ATEAM_GITEA_TOKEN"))
		}
		pullsURL := fmt.Sprintf("%s/api/v1/repos/%s/%s/pulls?state=open&limit=50", strings.TrimRight(cfg.BaseURL, "/"), cfg.Owner, cfg.Repo)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, pullsURL, nil)
		if err != nil {
			return false, err
		}
		if token != "" {
			req.Header.Set("Authorization", "token "+token)
		}
		resp, err := client.Do(req)
		if err != nil {
			return false, err
		}
		rawPulls, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode >= 300 {
			return false, fmt.Errorf("gitea pulls API returned %d", resp.StatusCode)
		}
		var prs []forgeIssuePayload
		if err := json.Unmarshal(rawPulls, &prs); err != nil {
			return false, err
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
		token := strings.TrimSpace(os.Getenv("GITEA_TOKEN"))
		if token == "" {
			token = strings.TrimSpace(os.Getenv("ATEAM_GITEA_TOKEN"))
		}
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
			cmd := exec.CommandContext(cctx, "gh", "label", "create", lbl.Name, "--color", lbl.Color, "--description", lbl.Description, "--force")
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
