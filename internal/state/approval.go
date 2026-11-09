package state

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/c360studio/ateam/internal/testspec"
)

// Approval states a change's proposal.md `Approved-by:` line can be in (#272 D3).
const (
	ApprovalUnapproved = "unapproved" // the line is empty or absent
	ApprovalApproved   = "approved"   // filled, and its digest matches proposal.md and design.md
	ApprovalStale      = "stale"      // filled with a digest that no longer matches
	ApprovalUnbound    = "unbound"    // filled without a digest (D4 A: warns, never fails)
)

// Approval is what a change's `Approved-by:` line says and whether it still holds.
type Approval struct {
	Name    string `json:"name,omitempty"`    // approver and date, without the digest token
	Digest  string `json:"digest,omitempty"`  // the 12 hex the line carries
	Current string `json:"current,omitempty"` // the 12 hex of proposal.md and design.md now
	State   string `json:"state,omitempty"`
	// File and At are set on stale only: the file that changed and the last commit touching it (sha7) or
	// "uncommitted".
	File string `json:"file,omitempty"`
	At   string `json:"at,omitempty"`
}

var digestTokenRe = regexp.MustCompile(`^sha256:([0-9a-f]+)$`)

// approvalLine returns the index of the first `Approved-by:` line outside code fences, or -1. A candidate line is
// unfenced when appending it to the lines before it adds one line to what testspec.EachUnfencedLine visits.
func approvalLine(lines []string) int {
	for i, line := range lines {
		if !strings.HasPrefix(strings.TrimSpace(line), "Approved-by:") {
			continue
		}
		if i == 0 {
			return 0
		}
		before, with := 0, 0
		testspec.EachUnfencedLine(strings.Join(lines[:i], "\n"), func(string) { before++ })
		testspec.EachUnfencedLine(strings.Join(lines[:i+1], "\n"), func(string) { with++ })
		if with > before {
			return i
		}
	}
	return -1
}

// ApprovalDigest is the 12-hex sha256 an approval binds to: proposal.md without its `Approved-by:` line, then
// design.md when present (nil when absent), each LF-normalized and length-prefixed (#272 D1 A).
func ApprovalDigest(proposal, design []byte) string {
	lines := strings.Split(strings.ReplaceAll(string(proposal), "\r\n", "\n"), "\n")
	if i := approvalLine(lines); i >= 0 {
		lines = append(lines[:i:i], lines[i+1:]...)
	}
	h := sha256.New()
	p := strings.Join(lines, "\n")
	fmt.Fprintf(h, "proposal.md %d\n%s", len(p), p)
	if design != nil {
		d := strings.ReplaceAll(string(design), "\r\n", "\n")
		fmt.Fprintf(h, "design.md %d\n%s", len(d), d)
	}
	return hex.EncodeToString(h.Sum(nil))[:12]
}

// parseApproval splits a trailing `sha256:<hex>` token off an `Approved-by:` value (D2 A).
func parseApproval(value string) (name, digest string) {
	value = strings.TrimSpace(value)
	i := strings.LastIndexAny(value, " \t")
	if m := digestTokenRe.FindStringSubmatch(value[i+1:]); m != nil {
		return strings.TrimSpace(value[:i+1]), m[1]
	}
	return value, ""
}

// approvalOf classifies a change's approval from its proposal.md and design.md (nil when absent).
func approvalOf(proposal, design []byte) Approval {
	value, _ := ParseProposal(proposal)
	a := Approval{Current: ApprovalDigest(proposal, design)}
	a.Name, a.Digest = parseApproval(value)
	switch {
	case a.Name == "" && a.Digest == "":
		a.State = ApprovalUnapproved
	case a.Digest == "":
		a.State = ApprovalUnbound
	case a.Digest == a.Current:
		a.State = ApprovalApproved
	default:
		a.State = ApprovalStale
	}
	return a
}

// staleSource names the file a stale approval's change touched and where: "uncommitted" when the working tree
// differs from HEAD, else the sha7 of the last commit touching it. It runs git only on the stale path (D3).
// design.md is always inspected: a deleted design makes the approval stale too, and git reports an absent path.
func staleSource(ctx context.Context, root, dir string) (file, at string) {
	files := []string{"proposal.md", "design.md"}
	for _, f := range files {
		out, ok := gitOutput(ctx, root, "status", "--porcelain", "--", path.Join(dir, f))
		if !ok {
			return strings.Join(files, " or "), "uncommitted" // not a git checkout: nothing is committed
		}
		if strings.TrimSpace(string(out)) != "" {
			return f, "uncommitted"
		}
	}
	args := []string{"log", "-1", "--format=%H", "--name-only", "--"}
	for _, f := range files {
		args = append(args, path.Join(dir, f))
	}
	out, ok := gitOutput(ctx, root, args...)
	fields := strings.Fields(string(out))
	if !ok || len(fields) < 2 {
		return strings.Join(files, " or "), "uncommitted"
	}
	// The newest commit can touch both; design.md is named only when proposal.md is not among them.
	file = path.Base(fields[len(fields)-1])
	for _, f := range fields[1:] {
		if path.Base(f) == "proposal.md" {
			file = "proposal.md"
		}
	}
	return file, sha7(fields[0])
}

// approvalHold is the hold the protocol writes on a change's first task until the owner approves.
const approvalHold = "Hold: OPEN-Q proposal awaits owner approval"

// ApproveEdits fills proposal.md's first unfenced `Approved-by:` line in the bound form
// `Approved-by: <name> (<YYYY-MM-DD>) sha256:<12 hex>` and removes the approval hold from tasks.md, the one
// owner edit that approves a change. It is pure so #181's handler can apply it to blobs it fetched; design is nil
// when the change has no design.md. It errors when either line is missing; other OPEN-Q holds are owner
// decisions and stay.
func ApproveEdits(proposal, design, tasks []byte, name string, date time.Time) (newProposal, newTasks []byte, err error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, nil, fmt.Errorf("approve: empty approver name")
	}
	pLines := strings.Split(string(proposal), "\n")
	i := approvalLine(pLines)
	if i < 0 {
		return nil, nil, fmt.Errorf("approve: proposal.md has no Approved-by: line outside code fences")
	}
	_, holds, err := ParseTasksMarkdown("tasks.md", tasks)
	if err != nil {
		return nil, nil, fmt.Errorf("approve: %w", err)
	}
	holdLine := 0
	for _, h := range holds {
		if strings.HasPrefix(h.Text, approvalHold) {
			holdLine = h.Line
			break
		}
	}
	if holdLine == 0 {
		return nil, nil, fmt.Errorf("approve: tasks.md has no open %q line", approvalHold)
	}

	line := pLines[i]
	indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
	cr := ""
	if strings.HasSuffix(line, "\r") {
		cr = "\r"
	}
	pLines[i] = fmt.Sprintf("%sApproved-by: %s (%s) sha256:%s%s", indent, name, date.Format("2006-01-02"), ApprovalDigest(proposal, design), cr)

	tLines := strings.Split(string(tasks), "\n")
	raw := tLines[holdLine-1]
	cr = ""
	if strings.HasSuffix(raw, "\r") {
		raw, cr = strings.TrimSuffix(raw, "\r"), "\r"
	}
	pos := strings.Index(raw, approvalHold)
	if pos < 0 {
		return nil, nil, fmt.Errorf("approve: tasks.md line %d does not carry %q verbatim", holdLine, approvalHold)
	}
	rest := strings.TrimPrefix(raw[pos+len(approvalHold):], ".") // the hold leading a task's text ends in a period
	kept := strings.TrimRight(raw[:pos], " \t")
	if r := strings.TrimSpace(rest); r != "" {
		kept += " " + r
	}
	if strings.TrimSpace(kept) == "" {
		tLines = append(tLines[:holdLine-1:holdLine-1], tLines[holdLine:]...)
	} else {
		tLines[holdLine-1] = kept + cr
	}
	return []byte(strings.Join(pLines, "\n")), []byte(strings.Join(tLines, "\n")), nil
}
