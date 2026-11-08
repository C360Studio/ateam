package bootstrap

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/c360studio/ateam/internal/review"
	"github.com/c360studio/ateam/internal/state"
)

// checkTeamSettings fails doctor --check on an invalid .agents/ateam.json; a missing file reports nothing.
func checkTeamSettings(root string) []string {
	if _, _, err := state.LoadSettings(root); err != nil {
		return []string{err.Error()}
	}
	return nil
}

// CheckMergeMethodDocs returns a doctor warning when .agents/ateam.json sets merge_method but
// .agents/protocol.md does not name the matching `gh pr merge` command: changing the key does not rewrite
// committed docs (D2). It says nothing without the key, without a protocol, or under review_record: aip,
// where no PR is merged.
func CheckMergeMethodDocs(root string) []string {
	s, ok, err := state.LoadSettings(root)
	if err != nil || !ok || s.MergeMethod == "" {
		return nil
	}
	if record, err := policyReviewRecord(root); err != nil || record == review.RecordAIP {
		return nil
	}
	data, err := os.ReadFile(filepath.Join(root, ".agents", "protocol.md"))
	if err != nil {
		return nil
	}
	want := "gh pr merge --" + s.MergeMethod + " --auto"
	if strings.Contains(string(data), want) {
		return nil
	}
	return []string{fmt.Sprintf(".agents/protocol.md does not name `%s` for merge_method %q in %s; run `ateam init --force` or edit its Close step", want, s.MergeMethod, state.SettingsPath)}
}
