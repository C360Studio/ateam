package state

import (
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// Reasons NextResult.None carries when no issue is named (#418, design D2, D3, D5).
const (
	NextNotSynced       = "not synced"
	NextNoOrdered       = "no ordered milestone"
	NextAmbiguous       = "milestone order ambiguous"
	NextCycle           = "milestone order cycle"
	NextDepsUnread      = "dependencies unread"
	nextAllSkippedShape = "all of %s skipped"
)

// NextResult is the one shared "next" issue computed from the forge cache: the first startable issue in the earliest
// ordered milestone, why each earlier issue was passed over, and the open issues no Order: line places (#418).
type NextResult struct {
	Milestone string     `json:"milestone,omitempty"`
	Number    int        `json:"number,omitempty"`
	Title     string     `json:"title,omitempty"`
	None      string     `json:"none,omitempty"`
	Skipped   []NextSkip `json:"skipped,omitempty"`
	// Unplaced holds open issues with no milestone, or in an ordered milestone but missing from its Order: line.
	Unplaced []int `json:"unplaced,omitempty"`
	// NoOrder names milestones with open issues and no Order: line.
	NoOrder []string `json:"no_order,omitempty"`
}

// NextSkip is one ordered issue passed over and why.
type NextSkip struct {
	Number int    `json:"number"`
	Reason string `json:"reason"`
}

var (
	// Order: may close a prose line (the milestone descriptions this convention came from do), so it is matched
	// anywhere, but only when an issue list follows it.
	orderLineRE = regexp.MustCompile(`(?i)\border:\s*(#\d+(?:\s*(?:->|→|,)\s*#\d+)*)`)
	// After: names a milestone title, which has spaces, so it must own its line.
	afterLineRE   = regexp.MustCompile(`(?im)^[\s>*_-]*after:[*_]*\s*(.+?)\s*$`)
	dependsLineRE = regexp.MustCompile(`(?im)^[\s>*_-]*depends on:[*_]*\s*(#\d+(?:\s*,\s*#\d+)*)`)
	claimRE       = regexp.MustCompile(`(?i)\b(?:close[sd]?|fix(?:e[sd])?|resolve[sd]?|addresses):?\s+#(\d+)\b`)
	issueRefRE    = regexp.MustCompile(`#(\d+)`)
)

func issueRefs(s string) []int {
	var out []int
	for _, m := range issueRefRE.FindAllStringSubmatch(s, -1) {
		if n, err := strconv.Atoi(m[1]); err == nil && !slices.Contains(out, n) {
			out = append(out, n)
		}
	}
	return out
}

// parseOrder returns the issue numbers of a description's first Order: line, in order; ok is false without one.
func parseOrder(desc string) ([]int, bool) {
	m := orderLineRE.FindStringSubmatch(desc)
	if m == nil {
		return nil, false
	}
	return issueRefs(m[1]), true
}

// parseAfter returns the milestone title a description's first After: line names, or "".
func parseAfter(desc string) string {
	m := afterLineRE.FindStringSubmatch(desc)
	if m == nil {
		return ""
	}
	return strings.TrimRight(m[1], ".")
}

// parseDependsOn returns the issues every Depends on: line across texts names.
func parseDependsOn(texts ...string) []int {
	var out []int
	for _, t := range texts {
		for _, m := range dependsLineRE.FindAllStringSubmatch(t, -1) {
			for _, n := range issueRefs(m[1]) {
				if !slices.Contains(out, n) {
					out = append(out, n)
				}
			}
		}
	}
	return out
}

// parseClaims returns the issues a PR body names with a closing keyword or Addresses.
func parseClaims(body string) []int {
	var out []int
	for _, m := range claimRE.FindAllStringSubmatch(body, -1) {
		if n, err := strconv.Atoi(m[1]); err == nil && !slices.Contains(out, n) {
			out = append(out, n)
		}
	}
	return out
}

// ComputeNext names the next issue from the cached cards of the active forge. It reads only the cache, so the
// session-start hook can show it; it is only as fresh as the last sync.
func (st *State) ComputeNext() NextResult {
	if st == nil || (st.Forge.Provider != ForgeGitHub && st.Forge.Provider != ForgeGitea) || st.SyncedAtFor(st.Forge).IsZero() {
		return NextResult{None: NextNotSynced}
	}
	return computeNext(st.Cards, string(st.Forge.Provider))
}

func computeNext(cards []BoardCard, forge string) NextResult {
	var res NextResult
	issues := map[int]BoardCard{}
	openNums := map[int]bool{}
	claimedBy := map[int]int{}
	descByMilestone := map[string]string{}
	descFrom := map[string]int{}
	for _, c := range cards {
		if c.Forge != forge || c.Column == "done" {
			continue
		}
		openNums[c.Number] = true
		if c.Type == "pr" {
			for _, n := range c.Claims {
				if prev, ok := claimedBy[n]; !ok || c.Number < prev {
					claimedBy[n] = c.Number
				}
			}
			continue
		}
		issues[c.Number] = c
		// Every card of a milestone carries its description; the lowest-numbered card's copy wins so a sync that
		// raced an edit gives the same answer whatever the card order.
		if c.Milestone != "" {
			if prev, ok := descFrom[c.Milestone]; !ok || c.Number < prev {
				descFrom[c.Milestone] = c.Number
				descByMilestone[c.Milestone] = c.MilestoneDescription
			}
		}
	}

	orders := map[string][]int{}
	var titles []string
	for title := range descByMilestone {
		titles = append(titles, title)
	}
	sort.Strings(titles)
	for _, title := range titles {
		if order, ok := parseOrder(descByMilestone[title]); ok {
			orders[title] = order
		} else {
			res.NoOrder = append(res.NoOrder, title)
		}
	}
	nums := make([]int, 0, len(issues))
	for n := range issues {
		nums = append(nums, n)
	}
	sort.Ints(nums)
	for _, n := range nums {
		c := issues[n]
		if c.Milestone == "" {
			res.Unplaced = append(res.Unplaced, n)
		} else if order, ok := orders[c.Milestone]; ok && !slices.Contains(order, n) {
			res.Unplaced = append(res.Unplaced, n)
		}
	}

	res.Milestone, res.None = earliestMilestone(orders, issues, descByMilestone)
	if res.None != "" {
		return res
	}

	for _, n := range orders[res.Milestone] {
		c, open := issues[n]
		reason := ""
		switch {
		case !open:
			reason = "closed"
		case c.Milestone != res.Milestone:
			reason = "in milestone " + c.Milestone
		case hasLabel(c.Labels, "status:needs-decision"):
			reason = "status:needs-decision"
		case hasLabel(c.Labels, "status:blocked"):
			reason = "status:blocked"
		case claimedBy[n] != 0:
			reason = fmt.Sprintf("claimed by #%d", claimedBy[n])
		case c.CommentsUnread:
			// A dependency in an unread comment would hand out blocked work, so nothing is named (design D5).
			res.None = NextDepsUnread
			return res
		default:
			for _, d := range c.DependsOn {
				if openNums[d] {
					reason = fmt.Sprintf("waits on #%d", d)
					break
				}
			}
		}
		if reason == "" {
			res.Number, res.Title = n, c.Title
			return res
		}
		res.Skipped = append(res.Skipped, NextSkip{Number: n, Reason: reason})
	}
	res.None = fmt.Sprintf(nextAllSkippedShape, res.Milestone)
	return res
}

// earliestMilestone returns the one ordered milestone whose After: names no other unfinished ordered milestone, or
// the reason there is not exactly one (design D2).
func earliestMilestone(orders map[string][]int, issues map[int]BoardCard, descs map[string]string) (string, string) {
	// A milestone whose Order: line has no open issue left is finished and drops out, so one naming it in After:
	// becomes startable.
	var candidates []string
	for title, order := range orders {
		for _, n := range order {
			if _, ok := issues[n]; ok {
				candidates = append(candidates, title)
				break
			}
		}
	}
	if len(candidates) == 0 {
		return "", NextNoOrdered
	}
	var roots []string
	for _, title := range candidates {
		after := parseAfter(descs[title])
		// A milestone after itself waits forever, so it counts as a loop, not a root.
		waits := slices.ContainsFunc(candidates, func(other string) bool { return strings.EqualFold(other, after) })
		if !waits {
			roots = append(roots, title)
		}
	}
	switch {
	case len(roots) == 0:
		return "", NextCycle
	case len(roots) > 1:
		return "", NextAmbiguous
	}
	return roots[0], ""
}

func hasLabel(labels []string, want string) bool {
	for _, l := range labels {
		if strings.EqualFold(l, want) {
			return true
		}
	}
	return false
}

// Field is the pickup-line value of next=.
func (r NextResult) Field() string {
	if r.Number != 0 {
		return fmt.Sprintf("#%d", r.Number)
	}
	return "none (" + r.None + ")"
}

// Format renders the ateam state next report.
func (r NextResult) Format() string {
	var b strings.Builder
	if r.Milestone != "" {
		fmt.Fprintf(&b, "milestone: %s\n", r.Milestone)
	}
	if r.Number != 0 {
		fmt.Fprintf(&b, "next: #%d %s\n", r.Number, r.Title)
	} else {
		fmt.Fprintf(&b, "next: none (%s)\n", r.None)
	}
	for _, s := range r.Skipped {
		fmt.Fprintf(&b, "skipped: #%d %s\n", s.Number, s.Reason)
	}
	if len(r.Unplaced) > 0 {
		parts := make([]string, len(r.Unplaced))
		for i, n := range r.Unplaced {
			parts[i] = "#" + strconv.Itoa(n)
		}
		fmt.Fprintf(&b, "unplaced: %s\n", strings.Join(parts, " "))
	}
	for _, m := range r.NoOrder {
		fmt.Fprintf(&b, "no order: %s\n", m)
	}
	return b.String()
}
