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

` + "`AGENTS.md`" + ` is loaded by your host; read it only if it is not in context. From ` + "`.agents/protocol.md`" + `, read § 1, § 2, § 3, *Test-to-spec attribution* and § 4. If your protocol has no such numbered sections, read all of it. When the review policy configures routes, confirm the developer route with ` + "`ateam review route`" + ` before implementation; a policy declaring only tiers and bands has no route to confirm. Record every implementing run's identity, provider, model as ` + "`.agents/review-policy.json`" + ` lists it (for example ` + "`opus`" + `) and reasoning effort, including escalations. The merge gate ranks an unmapped model or effort only against another unmapped run; on a configured route, unknown models and unresolved inheritance are not eligible. Do not silently lower the developer model to make review possible.

Own the complete change, callers, fixtures, tests and verification evidence. An independent reviewer must write none of the deliverable. Supply the complete base-to-head diff, actual implementers and checks; fix findings yourself and request a fresh review after any commit the reviewer contract's Record section does not exempt, or, on the configured ` + "`ateam review check`" + ` path, after any commit or policy change.

When a coordinator started you, never ask the owner a question in your own session. Post the question as one comment on your pull request, with the options and your recommendation; on the design track, put the proposal's open questions in ` + "`" + `design.md` + "`" + ` § Open Questions instead, with the same options and recommendation. Hand off with ` + "`" + `ateam state handoff` + "`" + ` naming that comment or section as the blocker; send the coordinator the wake-up line from your launch prompt with the reason ` + "`question`" + `; and wait for its message before you resume. When the coordinator tells you to wait, for example before arming auto-merge, hand off with the reason ` + "`" + `holding` + "`" + `, which asks the owner nothing.

Write ` + "`Approved-by:`" + ` only on the owner's own words, given in your conversation or passed on unchanged by the coordinator; a summary of the answer, or an approval naming a digest other than the one ` + "`task spec:queue`" + ` prints, is not one, so ask again. Fill it with the owner's name, the date and that digest, remove ` + "`Hold: OPEN-Q proposal awaits owner approval`" + ` in the same commit, and quote the owner's words and where they came from in the commit message. When your host refuses the edit, do not retry it: hand off as ` + "`blocked`" + ` naming the refusal, so the coordinator offers the owner only the other routes.

Hand off at ready-for-review (the reason ` + "`ready`" + `) only once every required check at the pull request head has passed, apart from ` + "`Merge check`" + `, which stays absent until the reviewer writes its ` + "`reviewed-by:`" + ` line. While one is still running, wait for it; when one fails, fix it and push, or hand off as ` + "`blocked`" + ` naming the check.

When a coordinator started you, arm auto-merge only when its message tells you to. After the approving review record and the archive commit, hand off with the reason ` + "`holding`" + ` and wait. On its message, merge ` + "`main`" + ` into your branch if it is behind, never by rebase, and arm once the required checks pass; if that merge touched one of your pull request's files, the review record no longer counts, so hand off ` + "`ready`" + ` for a new review instead.

Record model and reasoning effort separately; the merge gate ranks a run by model tier, then effort band, and needs a reviewer not weaker than every implementing run. A configured route is intent, not proof of the runtime model. On the configured ` + "`ateam review check`" + ` path, if no approved reviewer outranks every implementer, approval is blocked pending an explicit, commit-bound owner exception. Such an exception never waives independence, required checks, provider constraints or a passing verdict.

Write what the owner must see where the coordinator reads it. An action only the owner can take goes in the handoff Goal, named as owed by the owner (for example ` + "`" + `owed by the owner: delete scratch repository <name>` + "`" + `), never only in Constraints. The handoff names the change's track and, for ` + "`" + `quick` + "`" + `, the one-line reason no design trigger applies. State what was checked, not a conclusion: never write "no gate file touched"; quote the line ` + "`" + `scripts/gate-files.sh <pr>` + "`" + ` printed and the commit it ran at. When you rewrite your pull request body, keep any ` + "`" + `owner-prompts:` + "`" + ` line you find, unchanged.
`

const reviewerContract = reviewerContractBody + reviewerRecordPRComment

const reviewerContractBody = `# Independent reviewer contract

` + "`AGENTS.md`" + ` is loaded by your host; read it only if it is not in context. From ` + "`.agents/protocol.md`" + `, read § 1, § 2, § 3, *Test-to-spec attribution* and § 4. If your protocol has no such numbered sections, read all of it. Review in a separate run that wrote none of the deliverable. Keep the deliverable read-only; use native sandbox controls where available and report enforcement gaps. The author makes corrections. Do not edit the deliverable you grade.

Confirm the actual reviewer provider, model and reasoning effort. The merge gate requires your run to be not weaker than every implementation run: a stronger model tier, or the same tier at an equal or higher effort band; a model or effort the policy does not map passes as unranked only against another unranked run, so name the model as ` + "`.agents/review-policy.json`" + ` lists it (for example ` + "`opus`" + `) and the effort. Cross-provider separation applies only when the policy sets ` + "`cross_provider`" + `. On the configured ` + "`ateam review check`" + ` path, the reviewer must outrank every implementation model, unknown identities and unresolved inheritance block approval, and at the approved model ceiling an explicit owner exception bound to this head, reviewer and policy revision waives superiority only.

On a first review, inspect the complete base-to-head diff and affected callers, tests, fixtures and configuration. Apply triggered project rules, attempt to refute each finding, and report a concrete failure mechanism, location, minimal correction and verification. Contracts, instructions, configuration and rename source/destination paths require substantive review even when they are Markdown.

A finding blocks approval only when it is one of: a regression against the base; silent data or identifier corruption reachable from a supported input; a spec or contract violation you show with a reproduction or a traced path; a breach of a rule ` + "`AGENTS.md`" + ` never cuts (security, trust-boundary validation, error handling that prevents data loss, tests on non-trivial logic); or missing or failing required verification. A repository's own rules may raise this bar. For each finding, say whether an existing check already catches it: a caught finding does not block unless the spec makes the earlier refusal a promise and you say why it matters. List every other finding under ` + "`Non-blocking:`" + `; it does not change the verdict, and the author files each one as a follow-up issue before merge.

From the second round on, first confirm that each finding of your previous record is closed, then review the commits since that record (` + "`<record-sha>..<head>`" + `) and their callers; review the full base-to-head diff again only when those commits touch files outside the diff you last reviewed, or when asked. When another reviewer's verdict on this pull request conflicts with yours, quote it in your record as dissent and answer each of its findings; the latest record ` + "`task merge:check`" + ` reads decides.

For each test oracle, ask where its expected values came from: literals, a hand derivation shown in the test, or a named external source qualify; a helper that mirrors the code under test does not, however many mutants it kills. A claim in a title, spec, comment or record cites its artifact: ` + "`" + `measured` + "`" + ` names the artifact, otherwise say ` + "`" + `modelled` + "`" + ` or ` + "`" + `compile-checked` + "`" + `; reject ` + "`" + `verified` + "`" + ` or ` + "`" + `proves` + "`" + ` without an artifact.

Record base/head commits, policy revision, all implementation runs/models, your independent run/provider/model/effort, checks actually run, checks only reported by others, unavailable evidence, findings and verdict. Approval requires no blocking finding and all required verification. A later commit the Record section does not exempt invalidates previous evidence; on the configured ` + "`ateam review check`" + ` path, any changed commit or policy does. Eligibility alone is not approval. Forge login and self-reported runtime identity are attestations, not cryptographic proof; disclose these limits and never let candidate policy edits authorize their own exception.

`

// reviewerRecordPRComment is the contract's record section under the default review_record.
const reviewerRecordPRComment = `## Record

Post the record as one pull request comment. ` + "`task merge:check`" + ` accepts a comment whose ` + "`reviewed-by:`" + ` lines name only you, or, with no such line, whose ` + "`### Independent Cross-Agent Review`" + ` heading names no agent but you. The comment must designate exactly one commit of the pull request, written ` + "`Reviewed at <sha>`" + ` (7 to 40 hex characters); any other commit it designates the same way (such as a ` + "`Head: <sha>`" + ` line) makes it no record. The latest record must carry exactly one ` + "`Verdict: APPROVE`" + ` line; ` + "`Verdict: REQUEST CHANGES`" + `, no ` + "`Verdict:`" + ` line, or more than one fails the check until a later approving record. Lines inside ` + "```" + ` or ~~~ code fences are ignored, so a quoted template or old record never counts. Copy:

` + "```" + `text
reviewed-by: <agent> (<model>, <effort>)
Reviewed at <commit-sha>
Verdict: APPROVE | REQUEST CHANGES
<findings, checks run, unavailable evidence, limitations>
` + "```" + `

On GitHub, a later commit that is neither a clean merge of ` + "`main`" + ` nor docs-only needs a new record naming the new commit. On Gitea, every later commit needs one, since the gate cannot yet prove a Gitea commit's file list complete.

The record alone does not clear the gate. After ` + "`Verdict: APPROVE`" + `, the pull request description also needs your ` + "`reviewed-by: <agent> (<model>, <effort>)`" + ` line. After either verdict, refresh ` + "`Merge check`" + ` and confirm its result at the commit you reviewed: a comment starts no run, so a rejection after an earlier approval at the same head needs a rerun of the whole check, observed failing. If you cannot edit the description or rerun the check, report that instead of claiming it changed. The ` + "`review-pr`" + ` skill § 5 gives the commands.
`

// ConfigureReview installs a validated policy and its role instructions without
// replacing existing files. All conflicts are checked before the first write.
func ConfigureReview(root string, policyData []byte) ([]string, error) {
	parsed, err := review.ParsePolicy(policyData)
	if err != nil {
		return nil, err
	}
	// A tiers-only policy (ADR 0002) configures no routes, so it installs no route cards.
	var policy *review.Policy
	if parsed.HasRoutes() {
		if err := supportedReviewProviders(parsed); err != nil {
			return nil, err
		}
		policy = &parsed
	}
	if conflicts := unexpectedReviewFiles(root, policy); len(conflicts) != 0 {
		return nil, fmt.Errorf("%s", strings.Join(conflicts, "; "))
	}
	files := reviewFiles(policy, parsed.ReviewRecord)
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

// reviewFiles lists the generated role instructions for a policy (nil when it
// configures no routes) and its review_record value.
func reviewFiles(policy *review.Policy, record string) map[string]string {
	files := map[string]string{
		".agents/contracts/architect.md": architectContract,
		".agents/contracts/developer.md": developerContract,
		".agents/contracts/reviewer.md":  reviewerContractFor(record),
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

// CheckDeprecatedPolicyKeys returns a doctor warning while a valid review policy still sets
// required_checks, so adopters rename it before the alias is removed (#336). An invalid or
// absent policy says nothing here; checkReviewSetup reports it.
func CheckDeprecatedPolicyKeys(root string) []string {
	data, err := os.ReadFile(filepath.Join(root, reviewPolicyPath))
	if err != nil {
		return nil
	}
	if _, err := review.ParsePolicy(data); err != nil || !review.PolicyUsesDeprecatedKeys(data) {
		return nil
	}
	return []string{reviewPolicyPath + ": required_checks is deprecated; rename it to reviewer_checks"}
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
	record := ""
	if err == nil {
		parsed, err := review.ParsePolicy(data)
		if err != nil {
			return []string{fmt.Sprintf("%s: %v", reviewPolicyPath, err)}
		}
		record = parsed.ReviewRecord
		if parsed.HasRoutes() {
			if err := supportedReviewProviders(parsed); err != nil {
				return []string{fmt.Sprintf("%s: %v", reviewPolicyPath, err)}
			}
			policy = &parsed
		}
	}
	out := unexpectedReviewFiles(root, policy)
	for rel, expected := range reviewFiles(policy, record) {
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
	expected := reviewFiles(policy, "")
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
