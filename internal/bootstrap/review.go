package bootstrap

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/c360studio/ateam/internal/review"
)

const reviewPolicyPath = ".agents/review-policy.json"

const developerContract = `# Developer contract

Read AGENTS.md and .agents/protocol.md fully. Confirm the configured developer route before implementation. Record every implementing run's identity, provider, exact model and reasoning effort, including escalations. Unknown models and unresolved inheritance are not eligible. Do not silently lower the developer model to make review possible.

Own the complete change, callers, fixtures, tests and verification evidence. An independent reviewer must write none of the deliverable. Supply the complete base-to-head diff, actual implementers and checks; fix findings yourself and request fresh review after any commit or policy change.

Model identity and reasoning effort are separate. A configured route is intent, not proof of the runtime model. If no approved reviewer outranks every implementer, approval is blocked pending an explicit, commit-bound owner exception. Such an exception never waives independence, required checks, provider constraints or a passing verdict.
`

const reviewerContract = `# Independent reviewer contract

Read AGENTS.md and .agents/protocol.md fully. Review in a separate run that wrote none of the deliverable. Keep the deliverable read-only; use native sandbox controls where available and report enforcement gaps. The author makes corrections. Do not edit the deliverable you grade.

Confirm the actual reviewer provider, exact model and reasoning effort. The reviewer must outrank every actual implementation model under the trusted policy; higher effort and a different provider do not prove superiority. Unknown identities and unresolved inheritance block approval. Optional cross-provider separation is an additional constraint. At the approved model ceiling, require an explicit owner exception bound to this head, reviewer and policy revision; it waives superiority only.

Inspect the complete base-to-head diff and affected callers, tests, fixtures and configuration. Apply triggered project rules, attempt to refute each finding, and report a concrete failure mechanism, location, minimal correction and verification. Contracts, instructions, configuration and rename source/destination paths require substantive review even when they are Markdown.

Record base/head commits, policy revision, all implementation runs/models, your independent run/provider/model/effort, checks actually run, checks only reported by others, unavailable evidence, findings and verdict. Approval requires no blocking/high findings and all required verification. Changed commits or policy invalidate previous evidence. Eligibility alone is not approval. Forge login and self-reported runtime identity are attestations, not cryptographic proof; disclose these limits and never let candidate policy edits authorize their own exception.

## Record

Post the record as one pull request comment. ` + "`task merge:check`" + ` accepts a comment whose ` + "`reviewed-by:`" + ` lines name only you, or, with no such line, whose ` + "`### Independent Cross-Agent Review`" + ` heading names no agent but you. The comment must designate exactly one commit of the pull request, written ` + "`Reviewed at <sha>`" + ` (7 to 40 hex characters); any other commit it designates the same way (such as a ` + "`Head: <sha>`" + ` line) makes it no record. Copy:

` + "```" + `text
reviewed-by: <agent> (<model>)
Reviewed at <commit-sha>
Verdict: APPROVE | REQUEST CHANGES
<findings, checks run, unavailable evidence, limitations>
` + "```" + `

A later commit that is neither a clean merge of ` + "`main`" + ` nor docs-only needs a new record naming the new commit.
`

// ConfigureReview installs a validated policy and its role instructions without
// replacing existing files. All conflicts are checked before the first write.
func ConfigureReview(root string, policyData []byte) ([]string, error) {
	policy, err := review.ParsePolicy(policyData)
	if err != nil {
		return nil, err
	}
	if err := supportedReviewProviders(policy); err != nil {
		return nil, err
	}
	if conflicts := unexpectedReviewFiles(root, &policy); len(conflicts) != 0 {
		return nil, fmt.Errorf("%s", strings.Join(conflicts, "; "))
	}
	files := reviewFiles(&policy)
	files[reviewPolicyPath] = string(policyData)
	paths := make([]string, 0, len(files))
	for rel := range files {
		paths = append(paths, rel)
	}
	sort.Strings(paths)
	var missing []string
	for _, rel := range paths {
		if err := rejectReviewSymlinks(root, rel); err != nil {
			return nil, err
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if os.IsNotExist(err) {
			missing = append(missing, rel)
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", rel, err)
		}
		if !matchingReviewFile(rel, data, files[rel]) {
			return nil, fmt.Errorf("%s: existing custom content conflicts; review and reconcile it explicitly", rel)
		}
	}
	var written []string
	for _, rel := range missing {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			return written, fmt.Errorf("mkdir %s: %w", rel, err)
		}
		file, err := os.OpenFile(abs, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if err != nil {
			return written, fmt.Errorf("create %s: %w", rel, err)
		}
		written = append(written, rel)
		_, writeErr := file.WriteString(files[rel])
		closeErr := file.Close()
		if writeErr != nil {
			return written, fmt.Errorf("write %s: %w", rel, writeErr)
		}
		if closeErr != nil {
			return written, fmt.Errorf("close %s: %w", rel, closeErr)
		}
	}
	return written, nil
}

func rejectReviewSymlinks(root, rel string) error {
	path := root
	for _, part := range strings.Split(rel, "/") {
		path = filepath.Join(path, part)
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("inspect %s: %w", rel, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s: symbolic links are not supported for generated review files", rel)
		}
	}
	return nil
}

func reviewFiles(policy *review.Policy) map[string]string {
	files := map[string]string{
		".agents/contracts/developer.md": developerContract,
		".agents/contracts/reviewer.md":  reviewerContract,
	}
	if policy == nil {
		return files
	}
	for _, item := range []struct {
		role  string
		route review.Route
	}{{"developer", policy.Routes.Developer}, {"reviewer", policy.Routes.Reviewer}} {
		model, _ := policy.Model(item.route.Model)
		rel := ".agents/routes/" + item.role + ".md"
		files[rel] = fmt.Sprintf("# Configured %s route\n\nProvider: %s\nModel: %s\nReasoning effort: %s\n\nRead .agents/contracts/%s.md fully. Explicitly select this model and effort in the native client before starting a separate %s run. Stop if the runtime cannot honor this route. This card does not launch a model or prove runtime identity.\n", item.role, strconv.Quote(model.Provider), strconv.Quote(item.route.Model), strconv.Quote(item.route.Effort), item.role, item.role)
		switch model.Provider {
		case "claude":
			files[".claude/skills/ateam-"+item.role+"/SKILL.md"] = "---\nname: ateam-" + item.role + "\ndescription: Use the configured " + item.role + " route.\n---\n\nRead [" + rel + "](../../../" + rel + ") fully and follow its canonical contract. This instruction does not select the runtime model.\n"
		case "gemini":
			files[".gemini/commands/ateam-"+item.role+".toml"] = "description = \"Use the configured " + item.role + " route.\"\nprompt = \"Read " + rel + " fully and follow its canonical contract. This instruction does not select the runtime model.\"\n"
		}
	}
	return files
}

func checkReviewSetup(root string) []string {
	if err := rejectReviewSymlinks(root, reviewPolicyPath); err != nil {
		return []string{err.Error()}
	}
	data, err := os.ReadFile(filepath.Join(root, reviewPolicyPath))
	if err != nil && !os.IsNotExist(err) {
		return []string{fmt.Sprintf("%s: %v", reviewPolicyPath, err)}
	}
	var policy *review.Policy
	if err == nil {
		parsed, err := review.ParsePolicy(data)
		if err != nil {
			return []string{fmt.Sprintf("%s: %v", reviewPolicyPath, err)}
		}
		if err := supportedReviewProviders(parsed); err != nil {
			return []string{fmt.Sprintf("%s: %v", reviewPolicyPath, err)}
		}
		policy = &parsed
	}
	out := unexpectedReviewFiles(root, policy)
	for rel, expected := range reviewFiles(policy) {
		if err := rejectReviewSymlinks(root, rel); err != nil {
			out = append(out, err.Error())
			continue
		}
		actual, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil || !matchingReviewFile(rel, actual, expected) {
			out = append(out, rel+": missing or drifted review instructions")
		}
	}
	sort.Strings(out)
	return out
}

// Inspect every reserved role path, including pointers for providers that are
// no longer selected and orphan cards left after policy removal. Never delete
// these files: an owner must reconcile stale or custom instructions explicitly.
func unexpectedReviewFiles(root string, policy *review.Policy) []string {
	expected := reviewFiles(policy)
	var out []string
	for _, role := range []string{"developer", "reviewer"} {
		for _, rel := range []string{
			".agents/routes/" + role + ".md",
			".claude/skills/ateam-" + role + "/SKILL.md",
			".gemini/commands/ateam-" + role + ".toml",
		} {
			if _, selected := expected[rel]; selected {
				continue
			}
			_, err := os.Lstat(filepath.Join(root, filepath.FromSlash(rel)))
			if err == nil {
				out = append(out, rel+": unexpected review instructions for the current policy; reconcile explicitly")
			} else if !os.IsNotExist(err) {
				out = append(out, fmt.Sprintf("inspect %s: %v", rel, err))
			}
		}
	}
	return out
}

// Project-specific additions may extend contracts while retaining the canonical obligations.
func matchingReviewFile(rel string, actual []byte, expected string) bool {
	if strings.HasPrefix(rel, ".agents/contracts/") {
		for _, paragraph := range strings.Split(strings.TrimSpace(expected), "\n\n") {
			if !bytes.Contains(actual, []byte(paragraph)) {
				return false
			}
		}
		return true
	}
	return bytes.Equal(actual, []byte(expected))
}

func supportedReviewProviders(policy review.Policy) error {
	for _, route := range []review.Route{policy.Routes.Developer, policy.Routes.Reviewer} {
		model, _ := policy.Model(route.Model)
		switch model.Provider {
		case "claude", "codex", "gemini":
		default:
			return fmt.Errorf("unsupported review route provider %q (use claude, codex or gemini)", model.Provider)
		}
	}
	return nil
}
