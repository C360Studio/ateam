package review

import "strings"

// RecordComment is what a review record PR comment says, read with the merge
// check's grammar (#295). Its text is untrusted model output: only these
// fields leave it, never the body.
type RecordComment struct {
	// SHA is the reviewed commit as written, lower case; it may be abbreviated.
	SHA string `json:"sha"`
	// Verdict is the one unfenced Verdict: line's value, or "" when the record
	// has none or several, which never approves.
	Verdict string `json:"verdict,omitempty"`
	// Agent is "" for an Independent Cross-Agent Review heading naming none.
	Agent  string `json:"agent,omitempty"`
	Model  string `json:"model,omitempty"`
	Effort string `json:"effort,omitempty"`
}

// ParseRecordComment reads a review record comment. It is a record when it
// names one reviewing agent (or an anonymous review heading) and designates one
// reviewed commit; several designations count as one when each is a prefix of
// the longest, the abbreviated and the full SHA of a commit.
func ParseRecordComment(body string) (RecordComment, bool) {
	agent, ok := recordAgent(body)
	if !ok {
		return RecordComment{}, false
	}
	toks := gateDesignatedTokens(body)
	if len(toks) == 0 {
		return RecordComment{}, false
	}
	sha := toks[0]
	for _, t := range toks[1:] {
		if len(t) > len(sha) {
			sha = t
		}
	}
	for _, t := range toks {
		if !strings.HasPrefix(sha, t) {
			return RecordComment{}, false
		}
	}
	rec := RecordComment{SHA: sha, Agent: agent}
	if verdicts := recordVerdicts(body); len(verdicts) == 1 {
		rec.Verdict = verdicts[0]
	}
	// The model and effort come from the first unfenced line that starts with
	// reviewed-by: naming the record's agent.
	for _, l := range unfencedLines(body) {
		t := strings.TrimSpace(l)
		if agent == "" || len(t) < len("reviewed-by:") || !strings.EqualFold(t[:len("reviewed-by:")], "reviewed-by:") {
			continue
		}
		if runs := gateRuns(t[len("reviewed-by:"):]); len(runs) > 0 {
			if r, err := ParseRunSpec(runs[0]); err == nil && r.Provider == agent {
				rec.Model, rec.Effort = r.Model, r.Effort
				break
			}
		}
	}
	return rec, true
}
