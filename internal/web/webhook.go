package web

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/c360studio/ateam/internal/state"
)

type webhookLimiter struct {
	mu     sync.Mutex
	hits   map[string][]time.Time
	limit  int
	window time.Duration
}

func newWebhookLimiter(limit int, window time.Duration) *webhookLimiter {
	return &webhookLimiter{
		hits:   make(map[string][]time.Time),
		limit:  limit,
		window: window,
	}
}

func (l *webhookLimiter) allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	cutoff := now.Add(-l.window)
	valid := l.hits[key][:0]
	for _, t := range l.hits[key] {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	if len(valid) >= l.limit {
		l.hits[key] = valid
		return false
	}
	l.hits[key] = append(valid, now)
	return true
}

// VerifyWebhookSignature validates a GitHub (sha256=<hex>) or Gitea (<hex>) HMAC-SHA256 signature in constant time.
func VerifyWebhookSignature(payload, secret []byte, sigHeader string) bool {
	if len(secret) == 0 || strings.TrimSpace(sigHeader) == "" {
		return false
	}
	cleaned := strings.TrimSpace(sigHeader)
	cleaned = strings.TrimPrefix(cleaned, "sha256=")
	expected, err := hex.DecodeString(cleaned)
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write(payload)
	return hmac.Equal(mac.Sum(nil), expected)
}

type webhookEventPayload struct {
	Action string `json:"action"`
	Issue  *struct {
		Number  int    `json:"number"`
		Title   string `json:"title"`
		State   string `json:"state"`
		HTMLURL string `json:"html_url"`
		Labels  []struct {
			Name string `json:"name"`
		} `json:"labels"`
	} `json:"issue,omitempty"`
	PullRequest *struct {
		Number  int    `json:"number"`
		Title   string `json:"title"`
		State   string `json:"state"`
		Draft   bool   `json:"draft"`
		Merged  bool   `json:"merged"`
		HTMLURL string `json:"html_url"`
		Head    struct {
			Ref string `json:"ref"`
			SHA string `json:"sha,omitempty"`
		} `json:"head"`
		Labels []struct {
			Name string `json:"name"`
		} `json:"labels"`
	} `json:"pull_request,omitempty"`
}

func (s *Server) handleWebhook(forge string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		clientIP := extractClientIP(r.RemoteAddr)
		if !s.limiter.allow(clientIP, time.Now()) {
			writeError(w, http.StatusTooManyRequests, "webhook rate limit exceeded (60 req/min)")
			return
		}

		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			writeError(w, http.StatusBadRequest, "failed to read webhook body")
			return
		}

		secret := strings.TrimSpace(s.webhookSecret)
		if secret == "" {
			secret = strings.TrimSpace(os.Getenv("ATEAM_WEBHOOK_SECRET"))
		}
		if secret == "" {
			if !s.allowUnsignedWebhooks && strings.TrimSpace(os.Getenv("ATEAM_ALLOW_UNSIGNED_WEBHOOKS")) != "1" {
				writeError(w, http.StatusUnauthorized, "webhook secret is required (set ATEAM_WEBHOOK_SECRET)")
				return
			}
		} else {
			sig := r.Header.Get("X-Gitea-Signature")
			if sig == "" {
				sig = r.Header.Get("X-Hub-Signature-256")
			}
			if !VerifyWebhookSignature(body, []byte(secret), sig) {
				writeError(w, http.StatusUnauthorized, "invalid webhook HMAC-SHA256 signature")
				return
			}
		}

		var ev webhookEventPayload
		if err := json.Unmarshal(body, &ev); err != nil {
			writeError(w, http.StatusBadRequest, "invalid webhook JSON")
			return
		}

		s.mu.Lock()
		defer s.mu.Unlock()

		st, err := state.Refresh(r.Context(), s.repoRoot, false)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		var updated *state.BoardCard
		if card, ok := cardFromWebhook(ev, forge); ok {
			if _, err := state.Update(s.repoRoot, func(cur *state.State) error {
				c, err := cur.UpsertCard(card)
				if err != nil {
					return err
				}
				updated = &c
				return nil
			}); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
		}

		mergeAllowed := len(st.Holds) == 0
		writeJSON(w, http.StatusOK, map[string]any{
			"forge":         forge,
			"card":          updated,
			"merge_allowed": mergeAllowed,
			"active_holds":  len(st.Holds),
		})
	}
}

func cardFromWebhook(ev webhookEventPayload, forge string) (state.BoardCard, bool) {
	prefix := "GH"
	if forge == "gitea" {
		prefix = "GT"
	}
	if ev.PullRequest != nil && ev.PullRequest.Number > 0 {
		pr := ev.PullRequest
		var labels []string
		for _, l := range pr.Labels {
			if l.Name != "" {
				labels = append(labels, l.Name)
			}
		}
		col := "in-review"
		if pr.Merged || strings.EqualFold(pr.State, "closed") {
			col = "done"
		} else if pr.Draft {
			col = "in-progress"
		}
		return state.BoardCard{
			ID:      fmt.Sprintf("%s-PR-%d", prefix, pr.Number),
			Number:  pr.Number,
			Title:   pr.Title,
			Column:  col,
			Type:    "pr",
			Merged:  pr.Merged || strings.EqualFold(pr.State, "merged"),
			HeadSHA: strings.TrimSpace(pr.Head.SHA),
			Labels:  labels,
			Branch:  pr.Head.Ref,
			Forge:   forge,
			URL:     pr.HTMLURL,
		}, true
	}
	if ev.Issue != nil && ev.Issue.Number > 0 {
		iss := ev.Issue
		var labels []string
		col := "backlog"
		for _, l := range iss.Labels {
			if l.Name != "" {
				labels = append(labels, l.Name)
			}
			switch strings.ToLower(l.Name) {
			case "status:needs-decision":
				col = "needs-decision"
			case "status:blocked":
				col = "blocked"
			}
		}
		if strings.EqualFold(iss.State, "closed") {
			col = "done"
		}
		return state.BoardCard{
			ID:     fmt.Sprintf("%s-%d", prefix, iss.Number),
			Number: iss.Number,
			Title:  iss.Title,
			Column: col,
			Type:   "issue",
			Labels: labels,
			Forge:  forge,
			URL:    iss.HTMLURL,
		}, true
	}
	return state.BoardCard{}, false
}

func extractClientIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(remoteAddr))
	if err == nil && host != "" {
		return host
	}
	if remoteAddr != "" {
		return remoteAddr
	}
	return "local"
}
