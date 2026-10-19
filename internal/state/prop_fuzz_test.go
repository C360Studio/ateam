package state

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/quick"
	"unicode/utf8"
)

// FuzzScanHoldsMarkdown exercises ScanHolds and ParseTasksMarkdown across valid, stray-hold,
// and invalid-UTF-8 Markdown task files, comparing against an independent line-grammar oracle.
// spec: openspec/specs/harness/spec.md#Requirement: OpenSpec queue and stray Hold detection
func FuzzScanHoldsMarkdown(f *testing.F) {
	f.Add([]byte("# Change\n\n## 1. Tasks\n- [ ] 1.1 Open task\n  Hold: waiting on #42\n"))
	f.Add([]byte("# Change\n\n## 1. Tasks\n- [x] 1.1 Done task\n  Hold: resolved\n"))
	f.Add([]byte("# Change\n\n## 1. Tasks\n- [~] 1.1 Wontdo task\n  Hold: skipped\n"))
	f.Add([]byte("# Change\n\n## 1. Tasks\nHold: stray hold outside task\n- [ ] 1.1 Open task\n"))
	f.Add([]byte{0xff, 0xfe, 0xfd})

	f.Fuzz(func(t *testing.T, raw []byte) {
		root := t.TempDir()
		tasksPath := filepath.Join(root, "openspec", "changes", "c1", "tasks.md")
		if err := os.MkdirAll(filepath.Dir(tasksPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(tasksPath, raw, 0o600); err != nil {
			t.Fatal(err)
		}

		holds, err := ScanHolds(root)
		if !utf8.Valid(raw) {
			if err == nil {
				t.Fatalf("expected error for invalid UTF-8 input")
			}
			return
		}

		wantOpenHolds, wantStray := oracleTasksMarkdown(string(raw))
		if wantStray {
			if err == nil {
				t.Fatalf("expected ScanHolds to fail closed on stray Hold: outside task block\ninput:\n%s", string(raw))
			}
			return
		}
		if err != nil {
			t.Fatalf("unexpected ScanHolds error: %v", err)
		}
		if len(holds) != wantOpenHolds {
			t.Fatalf("ScanHolds count = %d, oracle want %d\ninput:\n%s", len(holds), wantOpenHolds, string(raw))
		}
	})
}

// oracleTasksMarkdown is an independent line-oriented reference model for tasks.md Hold: rules.
func oracleTasksMarkdown(doc string) (openHoldTasks int, hasStrayHold bool) {
	normalized := strings.ReplaceAll(doc, "\r\n", "\n")
	var (
		belowFirstHeading bool
		inTask            bool
		taskIsOpen        bool
		taskDepth         int
		taskHasHold       bool
	)
	flushTask := func() {
		if inTask && taskIsOpen && taskHasHold {
			openHoldTasks++
		}
		inTask = false
		taskIsOpen = false
		taskHasHold = false
	}

	for _, line := range strings.Split(normalized, "\n") {
		trimmedLeft := strings.TrimLeft(line, " \t")
		depth := len(line) - len(trimmedLeft)

		if m := taskLineRe.FindStringSubmatch(line); len(m) == 4 {
			flushTask()
			inTask = true
			taskIsOpen = m[2] == " " || m[2] == "~"
			taskDepth = depth
		} else if inTask && strings.TrimSpace(line) != "" && depth <= taskDepth {
			flushTask()
		}

		if !inTask {
			if belowFirstHeading && strings.Contains(line, "Hold:") {
				hasStrayHold = true
			}
		} else if strings.Contains(line, "Hold:") {
			taskHasHold = true
		}

		if strings.HasPrefix(line, "## ") {
			belowFirstHeading = true
		}
	}
	flushTask()
	return openHoldTasks, hasStrayHold
}

// FuzzReadFastGitHeadRefs exercises ReadFastGitHead across loose refs, packed-refs, detached HEADs,
// and linked worktrees (`gitdir:` + `commondir`).
// spec: openspec/specs/harness/spec.md#Requirement: Fast Git identity across worktrees
func FuzzReadFastGitHeadRefs(f *testing.F) {
	f.Add("refs/heads/gemini/test", "0123456789abcdef0123456789abcdef01234567", true, true)
	f.Add("refs/heads/main", "fedcba9876543210fedcba9876543210fedcba98", false, false)
	f.Add("", "abcdef0123456789abcdef0123456789abcdef01", false, true)

	f.Fuzz(func(t *testing.T, refName, sha string, useWorktree, usePacked bool) {
		cleanSHA := sanitizeHexSHA(sha)
		cleanRef := sanitizeRefName(refName)

		root := t.TempDir()
		gitDir := filepath.Join(root, ".git")
		commonDir := gitDir
		workRoot := root

		if useWorktree {
			commonDir = filepath.Join(root, "main-repo", ".git")
			gitDir = filepath.Join(commonDir, "worktrees", "wt1")
			workRoot = filepath.Join(root, "wt1")
			if err := os.MkdirAll(gitDir, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(workRoot, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(workRoot, ".git"), []byte("gitdir: "+gitDir+"\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(gitDir, "commondir"), []byte("../..\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		} else if err := os.MkdirAll(gitDir, 0o755); err != nil {
			t.Fatal(err)
		}

		if cleanRef == "" {
			// Detached HEAD
			if err := os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte(cleanSHA+"\n"), 0o600); err != nil {
				t.Fatal(err)
			}
		} else {
			if err := os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: "+cleanRef+"\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if usePacked {
				packed := fmt.Sprintf("# pack-refs with: peeled fully-peeled sorted\n%s %s\n", cleanSHA, cleanRef)
				if err := os.WriteFile(filepath.Join(commonDir, "packed-refs"), []byte(packed), 0o600); err != nil {
					t.Fatal(err)
				}
			} else {
				refFile := filepath.Join(commonDir, filepath.FromSlash(cleanRef))
				if err := os.MkdirAll(filepath.Dir(refFile), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(refFile, []byte(cleanSHA+"\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
		}

		gotBranch, gotSHA := ReadFastGitHead(workRoot)
		wantBranch := "HEAD"
		if cleanRef != "" {
			wantBranch = strings.TrimPrefix(cleanRef, "refs/heads/")
		}
		wantSHA := cleanSHA[:7]
		if gotBranch != wantBranch || gotSHA != wantSHA {
			t.Fatalf("ReadFastGitHead() = (%q, %q), want (%q, %q)", gotBranch, gotSHA, wantBranch, wantSHA)
		}
	})
}

func sanitizeHexSHA(in string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(in) {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') {
			b.WriteRune(r)
		}
	}
	for b.Len() < 40 {
		b.WriteString("0123456789abcdef")
	}
	return b.String()[:40]
}

func sanitizeRefName(in string) string {
	trimmed := strings.TrimSpace(in)
	if trimmed == "" {
		return ""
	}
	var segs []string
	for _, part := range strings.Split(trimmed, "/") {
		var clean strings.Builder
		for _, r := range part {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
				clean.WriteRune(r)
			}
		}
		if clean.Len() > 0 {
			segs = append(segs, clean.String())
		}
	}
	if len(segs) == 0 {
		return "refs/heads/main"
	}
	if segs[0] == "refs" && len(segs) >= 3 && segs[1] == "heads" {
		return strings.Join(segs, "/")
	}
	return "refs/heads/" + strings.Join(segs, "/")
}

// TestPropMergeForgeCardsHistory checks State.MergeForgeCards across generated histories of
// local, github, and gitea board operations against an independent test-owned reference model.
// spec: openspec/specs/harness/spec.md#Requirement: Dual-forge GitHub and Gitea state sync
func TestPropMergeForgeCardsHistory(t *testing.T) {
	var assertionsRun int

	prop := func(seed int64, steps uint8) bool {
		rng := rand.New(rand.NewSource(seed))
		numSteps := int(steps%12) + 3 // 3..14 steps per history

		var st State
		// Independent reference model: forge -> ordered slice of cards
		refModel := map[string][]BoardCard{
			"local":  {},
			"github": {},
			"gitea":  {},
		}

		for step := 0; step < numSteps; step++ {
			op := rng.Intn(3)
			switch op {
			case 0: // Add or replace local cards
				id := fmt.Sprintf("local-%d", rng.Intn(5)+1)
				card := BoardCard{ID: id, Title: fmt.Sprintf("Local %d", step), Column: "backlog", Forge: "local"}
				st.Cards = append(filterOutForgeAndID(st.Cards, "local", id), card)
				refModel["local"] = append(filterOutForgeAndID(refModel["local"], "local", id), card)
			case 1: // Sync GitHub batch
				batchSize := rng.Intn(4)
				var batch []BoardCard
				for i := 0; i < batchSize; i++ {
					batch = append(batch, BoardCard{
						ID:     fmt.Sprintf("gh-%d", i+1),
						Title:  fmt.Sprintf("GH step %d item %d", step, i+1),
						Column: "in-progress",
						Forge:  "github",
					})
				}
				st.MergeForgeCards(batch, ForgeGitHub)
				refModel["github"] = append([]BoardCard(nil), batch...)
			case 2: // Sync Gitea batch
				batchSize := rng.Intn(4)
				var batch []BoardCard
				for i := 0; i < batchSize; i++ {
					batch = append(batch, BoardCard{
						ID:     fmt.Sprintf("gt-%d", i+1),
						Title:  fmt.Sprintf("Gitea step %d item %d", step, i+1),
						Column: "in-review",
						Forge:  "gitea",
					})
				}
				st.MergeForgeCards(batch, ForgeGitea)
				refModel["gitea"] = append([]BoardCard(nil), batch...)
			}

			// Compare state against independent reference model
			want := make(map[string]BoardCard)
			for _, cards := range refModel {
				for _, c := range cards {
					want[c.Forge+":"+c.ID] = c
				}
			}
			if len(st.Cards) != len(want) {
				return false
			}
			for _, gotCard := range st.Cards {
				w, ok := want[gotCard.Forge+":"+gotCard.ID]
				if !ok || w.Title != gotCard.Title || w.Column != gotCard.Column {
					return false
				}
			}
			assertionsRun++
		}
		return true
	}

	cfg := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(1320)),
	}
	if err := quick.Check(prop, cfg); err != nil {
		t.Fatalf("MergeForgeCards property failed: %v", err)
	}
	if assertionsRun < 300 {
		t.Fatalf("assertion activation check failed: only %d step assertions executed, want >= 300", assertionsRun)
	}
}

func filterOutForgeAndID(cards []BoardCard, forge, id string) []BoardCard {
	var out []BoardCard
	for _, c := range cards {
		if c.Forge == forge && c.ID == id {
			continue
		}
		out = append(out, c)
	}
	return out
}
