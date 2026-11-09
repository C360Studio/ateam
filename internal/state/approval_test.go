package state

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// spec: openspec/specs/harness/spec.md#Requirement: OpenSpec queue and stray Hold detection
func TestApprovalDigest(t *testing.T) {
	const empty = "# C\n\nApproved-by:\n\nWhy\n"
	// Pinned from the shell, an oracle independent of this package:
	//   printf 'proposal.md 10\n# C\n\n\nWhy\n' | shasum -a 256 | cut -c1-12
	//   printf 'proposal.md 10\n# C\n\n\nWhy\ndesign.md 3\nD.\n' | shasum -a 256 | cut -c1-12
	if got := ApprovalDigest([]byte(empty), nil); got != "dc711ee07ab1" {
		t.Fatalf("digest without design.md = %s, want dc711ee07ab1", got)
	}
	if got := ApprovalDigest([]byte(empty), []byte("D.\n")); got != "62fb9eca2544" {
		t.Fatalf("digest with design.md = %s, want 62fb9eca2544", got)
	}
	base := ApprovalDigest([]byte(empty), []byte("D.\n"))
	same := map[string][2]string{
		"line filled":            {"# C\n\nApproved-by: Ann (2026-11-09)\n\nWhy\n", "D.\n"},
		"line filled with token": {"# C\n\nApproved-by: Ann (2026-11-09) sha256:000000000000\n\nWhy\n", "D.\n"},
		"CRLF line endings":      {strings.ReplaceAll(empty, "\n", "\r\n"), "D.\r\n"},
	}
	for name, files := range same {
		if got := ApprovalDigest([]byte(files[0]), []byte(files[1])); got != base {
			t.Errorf("%s: digest %s, want %s", name, got, base)
		}
	}
	differ := map[string][2]string{
		"proposal edited":        {"# C\n\nApproved-by:\n\nWhy.\n", "D.\n"},
		"design edited":          {empty, "D!\n"},
		"fenced example edited":  {"# C\n\n```\nApproved-by: x\n```\nApproved-by:\n\nWhy\n", "D.\n"},
		"design emptied":         {empty, ""},
		"bytes moved across two": {"# C\n\nApproved-by:\n\nWhy\nD", ".\n"},
	}
	for name, files := range differ {
		if got := ApprovalDigest([]byte(files[0]), []byte(files[1])); got == base {
			t.Errorf("%s: digest unchanged at %s", name, got)
		}
	}
	if ApprovalDigest([]byte(empty), nil) == ApprovalDigest([]byte(empty), []byte{}) {
		t.Error("an empty design.md digests like an absent one; adding it must make an approval stale")
	}
	// The real header follows a fenced example; filling it must not change the digest.
	fenced := "```\nApproved-by: <name>\n```\nApproved-by:\n"
	if ApprovalDigest([]byte(fenced), nil) != ApprovalDigest([]byte(strings.Replace(fenced, "Approved-by:\n", "Approved-by: Ann (2026-11-09)\n", 1)), nil) {
		t.Error("filling the unfenced header after a fenced example changed the digest")
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: OpenSpec queue and stray Hold detection
func TestParseApproval(t *testing.T) {
	for _, tc := range []struct{ value, name, digest string }{
		{"", "", ""},
		{"Ann (2026-11-09)", "Ann (2026-11-09)", ""},
		{"Ann (2026-11-09) sha256:3f9a1c2b4d5e", "Ann (2026-11-09)", "3f9a1c2b4d5e"},
		{"  Ann Owner (2026-11-09)   sha256:3f9a1c2b4d5e  ", "Ann Owner (2026-11-09)", "3f9a1c2b4d5e"},
		{"sha256:3f9a1c2b4d5e", "", "3f9a1c2b4d5e"},
		{"Ann (2026-11-09) sha256:XYZ", "Ann (2026-11-09) sha256:XYZ", ""},
		{"Ann sha256:3f9a1c2b4d5e (2026-11-09)", "Ann sha256:3f9a1c2b4d5e (2026-11-09)", ""},
	} {
		if name, digest := parseApproval(tc.value); name != tc.name || digest != tc.digest {
			t.Errorf("parseApproval(%q) = %q, %q; want %q, %q", tc.value, name, digest, tc.name, tc.digest)
		}
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: OpenSpec queue and stray Hold detection
func TestCheckOpenSpecApprovalStates(t *testing.T) {
	t.Setenv("PATH", gitOnlyPath(t)) // git, but no host openspec validating these fixtures
	const (
		proposal = "# Change\n\nApproved-by:\n\n## Why\n"
		design   = "# Design\n\nD1 A.\n"
		tasks    = "## 1. Tasks\n\n- [ ] 1.1 Do it\n"
	)
	digest := ApprovalDigest([]byte(proposal), []byte(design))
	bound := strings.Replace(proposal, "Approved-by:", "Approved-by: Ann (2026-11-09) sha256:"+digest, 1)
	unbound := strings.Replace(proposal, "Approved-by:", "Approved-by: Ann (2026-11-09)", 1)

	repo := t.TempDir()
	gitIn(t, repo, "init", "-q", "-b", "main")
	writeRepoFile(t, repo, "openspec/config.yaml", "schema: spec-driven\n")
	for _, id := range []string{"ok", "edited", "dirty", "loose"} {
		writeRepoFile(t, repo, "openspec/changes/"+id+"/tasks.md", tasks)
		writeRepoFile(t, repo, "openspec/changes/"+id+"/design.md", design)
		writeRepoFile(t, repo, "openspec/changes/"+id+"/proposal.md", bound)
	}
	writeRepoFile(t, repo, "openspec/changes/loose/proposal.md", unbound)
	writeRepoFile(t, repo, "openspec/changes/archive/2026-01-01-old/proposal.md", unbound)
	writeRepoFile(t, repo, "openspec/changes/archive/2026-01-01-old/tasks.md", tasks)
	gitIn(t, repo, "add", "-A")
	gitIn(t, repo, "commit", "-q", "-m", "approve")

	// Only the unbound in-flight change warns; the archived one is never checked.
	out, code, err := CheckOpenSpec(context.Background(), repo)
	if code != 0 || err != nil {
		t.Fatalf("all bound or unbound: code=%d err=%v out=%s", code, err, out)
	}
	wantWarn := "warning: openspec/changes/loose/proposal.md: approval unbound (Approved-by: has no sha256: digest); re-approve with sha256:" + digest + "\n"
	if !strings.Contains(out, wantWarn) || strings.Count(out, "warning:") != 1 {
		t.Fatalf("want exactly the loose warning %q, got:\n%s", wantWarn, out)
	}

	writeRepoFile(t, repo, "openspec/changes/edited/design.md", design+"D2 B.\n")
	gitIn(t, repo, "commit", "-q", "-am", "edit design")
	sha := gitIn(t, repo, "rev-parse", "--short=7", "HEAD")
	writeRepoFile(t, repo, "openspec/changes/dirty/proposal.md", bound+"\nOne more line.\n")

	out, code, err = CheckOpenSpec(context.Background(), repo)
	if code != 1 || err == nil {
		t.Fatalf("stale approvals: code=%d err=%v, want exit 1; out=%s", code, err, out)
	}
	editedNow := ApprovalDigest([]byte(bound), []byte(design+"D2 B.\n"))
	dirtyNow := ApprovalDigest([]byte(bound+"\nOne more line.\n"), []byte(design))
	for _, want := range []string{
		"openspec/changes/edited/proposal.md: approval sha256:" + digest + " is stale; design.md changed since (last commit " + sha + "); re-approve with sha256:" + editedNow + "\n",
		"openspec/changes/dirty/proposal.md: approval sha256:" + digest + " is stale; proposal.md changed since (uncommitted); re-approve with sha256:" + dirtyNow + "\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "openspec/changes/ok/") {
		t.Errorf("the approved change is reported:\n%s", out)
	}

	changes, err := ScanOpenSpecQueue(repo)
	if err != nil {
		t.Fatal(err)
	}
	queue, _ := FormatOpenSpecQueue(changes)
	for _, want := range []string{
		"approved sha256:" + digest,
		"stale approval (design.md edited at " + sha + ")",
		"stale approval (proposal.md uncommitted)",
		"approval unbound",
		"digest   sha256:" + editedNow + " is the current digest to approve with",
	} {
		if !strings.Contains(queue, want) {
			t.Errorf("queue missing %q:\n%s", want, queue)
		}
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: OpenSpec queue and stray Hold detection
func TestQueuePrintsDigestForUnapprovedChange(t *testing.T) {
	dir := t.TempDir()
	const proposal = "# Change\n\nApproved-by:\n"
	writeRepoFile(t, dir, "openspec/changes/c1/proposal.md", proposal)
	writeRepoFile(t, dir, "openspec/changes/c1/tasks.md", "## 1. Approval\n\n- [ ] 1.1 Hold: OPEN-Q proposal awaits owner approval.\n")
	changes, err := ScanOpenSpecQueue(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := changes[0].Approval.State; got != ApprovalUnapproved {
		t.Fatalf("state = %q, want unapproved", got)
	}
	out, _ := FormatOpenSpecQueue(changes)
	if want := "sha256:" + ApprovalDigest([]byte(proposal), nil) + " is the current digest"; !strings.Contains(out, want) {
		t.Fatalf("queue missing %q:\n%s", want, out)
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: OpenSpec queue and stray Hold detection
func TestApproveEdits(t *testing.T) {
	date := time.Date(2026, 11, 9, 23, 0, 0, 0, time.UTC)
	const design = "# Design\n"
	for _, tc := range []struct {
		name, proposal, tasks, wantTasks string
	}{
		{name: "hold leads the task",
			proposal:  "# C\n\nApproved-by:\n",
			tasks:     "## 1. Approval\n\n- [ ] 1.1 Hold: OPEN-Q proposal awaits owner approval. Owner approves (#1).\n- [ ] 1.2 Next\n",
			wantTasks: "## 1. Approval\n\n- [ ] 1.1 Owner approves (#1).\n- [ ] 1.2 Next\n"},
		{name: "hold trails the task",
			proposal:  "# C\n\nApproved-by:\n",
			tasks:     "## 1. Approval\n\n- [ ] 1.1 Owner approves (#1). Hold: OPEN-Q proposal awaits owner approval\n",
			wantTasks: "## 1. Approval\n\n- [ ] 1.1 Owner approves (#1).\n"},
		{name: "hold on its own line, CRLF",
			proposal:  "```\r\nApproved-by: <name>\r\n```\r\nApproved-by:\r\n",
			tasks:     "## 1. Approval\r\n\r\n- [ ] 1.1 Owner approves\r\n  Hold: OPEN-Q proposal awaits owner approval\r\n- [x] 1.2 Done\r\n",
			wantTasks: "## 1. Approval\r\n\r\n- [ ] 1.1 Owner approves\r\n- [x] 1.2 Done\r\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := ApprovalDigest([]byte(tc.proposal), []byte(design))
			p, tasks, err := ApproveEdits([]byte(tc.proposal), []byte(design), []byte(tc.tasks), " Ann Owner ", date)
			if err != nil {
				t.Fatal(err)
			}
			if string(tasks) != tc.wantTasks {
				t.Fatalf("tasks = %q, want %q", tasks, tc.wantTasks)
			}
			value, _ := ParseProposal(p)
			name, digest := parseApproval(value)
			if name != "Ann Owner (2026-11-09)" || digest != before {
				t.Fatalf("read back %q, %q; want Ann Owner (2026-11-09), %s", name, digest, before)
			}
			if after := ApprovalDigest(p, []byte(design)); after != before {
				t.Fatalf("digest moved from %s to %s", before, after)
			}
			if a := approvalOf(p, []byte(design)); a.State != ApprovalApproved {
				t.Fatalf("approval after ApproveEdits = %+v, want approved", a)
			}
			if _, _, err := ParseTasksMarkdown("tasks.md", tasks); err != nil {
				t.Fatal(err)
			}
		})
	}
	if _, _, err := ApproveEdits([]byte("Approved-by:\n"), nil, []byte("## 1. T\n\n- [ ] 1.1 Do it\n"), "Ann", date); err == nil {
		t.Error("tasks.md without the OPEN-Q hold: want an error")
	}
	if _, _, err := ApproveEdits([]byte("# C\n```\nApproved-by:\n```\n"), nil, []byte("- [ ] 1.1 Hold: OPEN-Q proposal awaits owner approval\n"), "Ann", date); err == nil {
		t.Error("proposal.md with only a fenced Approved-by: want an error")
	}
}

// spec: openspec/specs/harness/spec.md#Requirement: OpenSpec queue and stray Hold detection
func FuzzApproveEditsKeepsDigest(f *testing.F) {
	f.Add("# C\n\n", "\n## Why\n", "Ann")
	f.Add("```\nApproved-by: x\n```\n", "", "Bob")
	f.Add("~~~\n", "\n~~~\n", "Cy") // header fenced: ApproveEdits must refuse
	f.Add("a\r\n", "b\r\n", "D")
	f.Fuzz(func(t *testing.T, head, tail, name string) {
		if strings.ContainsAny(name, "\r\n") || strings.TrimSpace(name) == "" {
			t.Skip()
		}
		proposal := []byte(head + "Approved-by:\n" + tail)
		tasks := []byte("- [ ] 1.1 Hold: OPEN-Q proposal awaits owner approval\n")
		p, _, err := ApproveEdits(proposal, nil, tasks, name, time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC))
		if err != nil {
			return // no unfenced header to fill
		}
		if a := approvalOf(p, nil); a.State != ApprovalApproved || a.Digest != ApprovalDigest(proposal, nil) {
			t.Fatalf("approval after ApproveEdits = %+v, want approved at %s", a, ApprovalDigest(proposal, nil))
		}
	})
}

// gitOnlyPath is a PATH holding only a link to git, so a test can hide a host openspec while keeping git.
func gitOnlyPath(t *testing.T) string {
	t.Helper()
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	if err := os.Symlink(git, filepath.Join(dir, "git")); err != nil {
		t.Fatal(err)
	}
	return dir
}
