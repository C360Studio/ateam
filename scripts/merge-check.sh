#!/usr/bin/env bash
# Merge check (openspec spec harness, "Merge gate and CI workflow enforcement").
# Reads GitHub when it runs, so a re-run sees the state of that moment.
#
#   1. The rules in force on main must require the check `Required` with the
#      strict (up-to-date) setting on, from a ruleset whose enforcement is active.
#   2. On a pull-request run: while an issue labelled class:flake is open, fail
#      unless this pull request's closing references include every open one.
#      There is no waiver: no comment, label or variable exempts a pull request.
#   3. On a pull-request run, before rule 2: a code pull request that is not a
#      draft fails unless its description has one implemented-by: line naming
#      claude, codex, or gemini (not read for a bot's) and one reviewed-by:
#      line naming one of them (another agent only when .agents/review-policy.json
#      sets cross_provider), each line naming its model in parentheses after the
#      agent (e.g. implemented-by: claude (opus)). Once that policy exists, each
#      model is followed by its effort (claude (sonnet, medium, opus, high)) and
#      ateam review compare must find the reviewer not weaker by tier, then effort
#      band. A PR comment must also contain a review record (reviewed-by:
#      <agent> or a ### Independent Cross-Agent Review heading) designating a
#      single reviewed commit (e.g. reviewed at <sha>, re-reviewed at <sha>,
#      reviewed [head] commit <sha>, or reviewed-commit: <sha>) in the PR's
#      complete commit list with no later content commit (allowing only a
#      merge of main whose non-merge diff touches none of the PR's changed files,
#      or a documents-only commit); a draft's findings are printed and fail nothing.
#
# Usage: merge-check.sh <pr-number>
#   The kind of run comes from GITHUB_EVENT_NAME only when GITHUB_ACTIONS is
#   true (pull_request: a pull-request run; push: a push run, which checks rule 1
#   only). Outside Actions GITHUB_EVENT_NAME is not read and every run is a
#   pull-request run. Run it locally as `task merge:check -- <n>` immediately
#   before merging, in a shell where GITHUB_ACTIONS is not set.
#
# A read that fails, or an answer that is not the shape asked for, exits 2
# naming the read; it is never taken as "no known flake". Needs gh and jq.
set -uo pipefail

readonly label=class:flake
readonly limit=100
readonly check=Required

die() { echo "merge-check: $*" >&2; exit 1; }

if [ "${GITHUB_ACTIONS:-}" = true ]; then
  case "${GITHUB_EVENT_NAME:-}" in
    pull_request) kind=pull-request how="GitHub Actions, pull_request event" ;;
    push) kind=push how="GitHub Actions, push event" ;;
    "") echo "merge-check: inside GitHub Actions with GITHUB_EVENT_NAME unset"; die "the event is missing" ;;
    *)
      echo "merge-check: inside GitHub Actions on event ${GITHUB_EVENT_NAME}"
      die "event ${GITHUB_EVENT_NAME} has no rule here; add one before the workflow triggers on it" ;;
  esac
else
  kind=pull-request how="outside GitHub Actions; GITHUB_EVENT_NAME not read"
fi

pr=${1:-}
if [ "$kind" = pull-request ]; then
  echo "merge-check: pull-request run for #${pr:-?} (${how})"
  [ -n "$pr" ] || die "a pull request number is required: merge-check.sh <pr-number>"
  case "$pr" in *[!0-9]* | 0*) die "pull request number '$pr' is not a positive integer" ;; esac
else
  echo "merge-check: push run (${how}); the known-flake check does not apply, the up-to-date rule is checked"
  echo "merge-check: the review check does not apply to a push run; no pull request is read"
fi

# read NAME SHAPE CMD...: run a gh read and keep its stdout in $answer when it
# succeeded and jq accepts SHAPE; otherwise exit 2 naming the read.
answer=
errfile=$(mktemp)
# The PR file and commit lists go to jq through these files, not argv: with
# patches and full commit objects they exceed Linux's 128 KiB per-argument
# limit (#122).
filesfile=$(mktemp)
commitsfile=$(mktemp)
trap 'rm -f "$errfile" "$filesfile" "$commitsfile"' EXIT
read_gh() {
  local name=$1 shape=$2
  shift 2
  if ! answer=$("$@" </dev/null 2>"$errfile"); then
    echo "merge-check: unavailable: the read of ${name} failed ($*): $(cat "$errfile")" >&2
    exit 2
  fi
  if ! printf '%s' "$answer" | jq -e "$shape" >/dev/null 2>&1; then
    echo "merge-check: unavailable: the read of ${name} returned something other than what was asked for ($*): ${answer}" >&2
    exit 2
  fi
}

failed=0

# 1. Up-to-date rule. GitHub answers which rules apply to main; only the fields
# that carry the decision are read.
read_gh "the rules in force on main" 'type == "array" and all(.[]; type == "object")' \
  gh api 'repos/{owner}/{repo}/rules/branches/main'
rules=$answer
found=$(printf '%s' "$rules" | jq -c --arg c "$check" '[.[]
  | select(.type == "required_status_checks")
  | select([.parameters.required_status_checks[]?.context] | index($c))
  | {ruleset_id, strict: .parameters.strict_required_status_checks_policy}]')
if [ "$(printf '%s' "$found" | jq 'length')" -eq 0 ]; then
  echo "merge-check: no rule in force on main requires ${check}; rules found: $(printf '%s' "$rules" | jq -c '[.[] | {type, ruleset_id}]')"
  failed=1
else
  ok=0
  while IFS=$'\t' read -r id strict; do
    case "$id" in '' | *[!0-9]*) die "a rule requiring ${check} names ruleset_id '${id}', not a number" ;; esac
    read_gh "ruleset ${id}" 'type == "object" and (.enforcement | type == "string")' \
      gh api "repos/{owner}/{repo}/rulesets/${id}"
    enforcement=$(printf '%s' "$answer" | jq -r '.enforcement')
    echo "merge-check: rule type=required_status_checks context=${check} strict_required_status_checks_policy=${strict} ruleset_id=${id} enforcement=${enforcement}"
    if [ "$strict" = true ] && [ "$enforcement" = active ]; then ok=1; fi
  done < <(printf '%s' "$found" | jq -r '.[] | [(.ruleset_id | tostring), (.strict | tostring)] | @tsv')
  if [ "$ok" -ne 1 ]; then
    echo "merge-check: FAIL: no active ruleset requires ${check} with strict_required_status_checks_policy=true; a head behind main could merge"
    failed=1
  fi
fi

if [ "$kind" = push ]; then
  [ "$failed" -eq 0 ] || { echo "merge-check: FAILED"; exit 1; }
  echo "merge-check: ok"
  exit 0
fi

# Review check. The pull request, then every page of its changed files: gh prints
# the pages joined into one list or as lists one after the other, and both are
# read the same way.
read_gh "pull request #${pr} for the review check" \
  'type == "object" and (.draft | type) == "boolean" and (.changed_files | type) == "number" and (.commits | type) == "number" and (.head.sha | type) == "string" and (.head.sha | length) >= 7 and (.body == null or (.body | type) == "string") and (.user.type | type) == "string"' \
  gh api "repos/{owner}/{repo}/pulls/${pr}"
pull=$answer
read_gh "the files of pull request #${pr}" 'type == "array"' \
  gh api --paginate "repos/{owner}/{repo}/pulls/${pr}/files?per_page=100"
# The shape check above sees only the last page; every page is checked here.
if ! files=$(printf '%s' "$answer" | jq -c -s 'if all(.[]; type == "array" and all(.[]; type == "object" and (.filename | type) == "string" and (.previous_filename == null or (.previous_filename | type) == "string"))) then add // [] else error("not pages of files") end'); then
  echo "merge-check: unavailable: the read of the files of pull request #${pr} returned something other than what was asked for: ${answer}" >&2
  exit 2
fi
if ! printf '%s' "$files" > "$filesfile"; then
  echo "merge-check: unavailable: could not write the file list of pull request #${pr} to a temp file" >&2
  exit 2
fi
reported=$(printf '%s' "$pull" | jq '.changed_files')
draft=$(printf '%s' "$pull" | jq '.draft')
entries=$(printf '%s' "$files" | jq 'length')
# A failed jq leaves an empty number, which a test would not compare.
case "$entries:$reported:$draft" in
  [0-9]*:[0-9]*:true | [0-9]*:[0-9]*:false) ;;
  *) echo "merge-check: unavailable: jq failed reading the file count or draft flag of pull request #${pr}" >&2; exit 2 ;;
esac
# Compared in jq: a shell test on a count that is not an integer errors, and an
# error would read as "the counts match".
if ! printf '%s' "$pull" | jq -e --argjson n "$entries" '.changed_files == $n' >/dev/null; then
  echo "merge-check: unavailable: the file list of pull request #${pr} is incomplete: it has ${entries} of ${reported} files" >&2
  exit 2
fi
# A document name ends in .md (or starts with openspec/ and ends in .md, .yml,
# .yaml, or .json) and is not under .claude/agents/ or .agents/contracts/;
# every other name is a code name (tightening semengine/scripts/merge-check.sh
# and semengine AGENTS.md:163 so scripts/lockfiles under openspec/ require review).
# A renamed file's previous name counts too. A failed jq leaves an empty list,
# which would read as documents only. The names stay a JSON list: printed one
# per line, a name that is empty or only line feeds would vanish.
if ! code=$(printf '%s' "$files" | jq -c '[.[] | .filename, (.previous_filename // empty)]
  | map(select(((endswith(".md") or (startswith("openspec/") and (endswith(".yml") or endswith(".yaml") or endswith(".json")))) and (startswith(".claude/agents/") | not) and (startswith(".agents/contracts/") | not)) | not))
  | unique'); then
  echo "merge-check: unavailable: jq failed sorting the changed files of pull request #${pr} into code names" >&2
  exit 2
fi
if [ "$code" = "[]" ]; then
  echo "merge-check: pull request #${pr} is documents only (${reported} changed files); the review check passes"
else
  echo "merge-check: pull request #${pr} is a code pull request; its code files ($(printf '%s' "$code" | jq length), up to ten shown):"
  printf '%s' "$code" | jq -r '.[:10][] | "merge-check: code file: \(.)"'
  # The two lines: each starts in the first column; a carriage return before the
  # line feed is dropped. An agent name is claude, codex, or gemini in any letter
  # case, with no letter or digit directly before or after it. The model is the
  # parenthesised group directly after the first agent name (spaces and backticks
  # allowed between), with a character other than a space in it. Nothing else is read.
  if ! lines=$(printf '%s' "$pull" | jq -r '
    def agents: [scan("(?i)(?:^|[^\\p{L}\\p{N}])(claude|codex|gemini)(?=[^\\p{L}\\p{N}]|$)") | .[0] | ascii_downcase] | unique | join(" ");
    def model: if test("^[ \\t]*`?(?i:claude|codex|gemini)`?[ \\t]*\\([^)]*[^\\s)][^)]*\\)") then "yes" else "no" end;
    [(.body // "") | split("\n")[] | rtrimstr("\r")] as $l
    | ($l | map(select(startswith("implemented-by:")))) as $i
    | ($l | map(select(startswith("reviewed-by:")))) as $r
    | [.user.type, ($i | length), ($i[0] // "" | ltrimstr("implemented-by:") | agents),
       ($r | length), ($r[0] // "" | ltrimstr("reviewed-by:") | agents),
       ($i[0] // "" | ltrimstr("implemented-by:") | model), ($r[0] // "" | ltrimstr("reviewed-by:") | model)] | map(tostring) | join("|")'); then
    echo "merge-check: unavailable: jq failed reading the description of pull request #${pr}" >&2
    exit 2
  fi
  IFS='|' read -r author icount iagents rcount ragents imodel rmodel <<<"$lines"
  case "$icount:$rcount" in
    *[!0-9:]* | :* | *:) echo "merge-check: unavailable: the description of pull request #${pr} was read as '${lines}'" >&2; exit 2 ;;
  esac

  # ADR 0002: once the policy file exists, both lines carry a native effort after
  # each model and ateam review compare decides strength. The different-agent rule
  # applies only when the policy sets cross_provider to true.
  policy=.agents/review-policy.json
  cross=false
  effort_hint=
  if [ -e "$policy" ]; then
    effort_hint=", <effort>"
    if ! cross=$(jq -r 'if .cross_provider == true then "true" else "false" end' "$policy"); then
      echo "merge-check: unavailable: jq failed reading cross_provider from ${policy}" >&2
      exit 2
    fi
  fi
  # The single implementing agent when implemented-by: names one and cross_provider
  # is on; empty when any agent may review: cross_provider off, a bot's pull
  # request, more than one agent named, or no valid line.
  impl_single=
  others=
  wantline="reviewed-by: <claude, codex, or gemini> (<model>${effort_hint})"
  if [ "$cross" = true ] && [ "$author" != Bot ] && [ "$icount" -eq 1 ]; then
    case "$iagents" in
      claude) impl_single=claude others="codex or gemini" wantline="reviewed-by: codex (or reviewed-by: gemini)" ;;
      codex)  impl_single=codex  others="claude or gemini" wantline="reviewed-by: claude (or reviewed-by: gemini)" ;;
      gemini) impl_single=gemini others="claude or codex"  wantline="reviewed-by: claude (or reviewed-by: codex)" ;;
    esac
  fi
  implline="implemented-by: claude (<model or persona>), implemented-by: codex (<model or persona>), or implemented-by: gemini (<model or persona>)"
  findings=0
  finding() { echo "merge-check: review finding: $1; the line to write: $2"; findings=$((findings + 1)); }
  if [ "$author" != Bot ]; then
    if [ "$icount" -eq 0 ]; then
      finding "no line starts implemented-by:, and the implemented-by: line must name claude, codex, or gemini, the agent that wrote the commits" "$implline"
    elif [ "$icount" -gt 1 ]; then
      finding "${icount} lines start implemented-by:, and exactly one may" "$implline"
    elif [ -z "$iagents" ]; then
      finding "the implemented-by: line must name claude, codex, or gemini, the agent that wrote the commits" "$implline"
    elif [ "$imodel" != yes ]; then
      finding "the implemented-by: line must name the implementing model in parentheses after the agent (model, effort pairs once a policy exists)" "implemented-by: ${iagents} (<model>${effort_hint})"
    fi
  fi
  if [ "$rcount" -eq 0 ]; then
    finding "no line starts reviewed-by:" "$wantline"
  elif [ "$rcount" -gt 1 ]; then
    finding "${rcount} lines start reviewed-by:, and exactly one may" "$wantline"
  elif [ "$ragents" != claude ] && [ "$ragents" != codex ] && [ "$ragents" != gemini ]; then
    finding "the reviewed-by: line must name exactly one of claude, codex, and gemini" "$wantline"
  elif [ -n "$impl_single" ] && [ "$ragents" = "$impl_single" ]; then
    finding "the reviewed-by: line names ${ragents}, which the implemented-by: line names; another agent (${others}) reviews" "$wantline"
  elif [ "$rmodel" != yes ]; then
    finding "the reviewed-by: line must name the reviewing model in parentheses after the agent" "reviewed-by: ${ragents} (<model>${effort_hint})"
  fi
  # Strength (ADR 0002 rule 2): each line's parenthesised group is read as
  # model, effort pairs and handed to ateam review compare as agent:model:effort
  # (an odd last model has an empty effort, which compare reports). The binary is
  # $ATEAM, else ateam on PATH; compare exits 0 pass, 1 finding, 2 undecided.
  if [ "$findings" -eq 0 ] && [ "$author" != Bot ] && [ -e "$policy" ]; then
    if ! runs=$(printf '%s' "$pull" | jq -r '
      def runs($p): ltrimstr($p)
        | capture("^[ \\t]*`?(?<a>(?i:claude|codex|gemini))`?[ \\t]*\\((?<g>[^)]*)\\)")
        | (.a | ascii_downcase) as $a
        | [.g | split(",")[] | gsub("^\\s+|\\s+$"; "") | select(. != "")] as $v
        | range(0; $v | length; 2) as $k | "\($a):\($v[$k]):\($v[$k + 1] // "")";
      [(.body // "") | split("\n")[] | rtrimstr("\r")] as $l
      | ($l[] | select(startswith("implemented-by:")) | "--implementer\t" + runs("implemented-by:")),
        ($l[] | select(startswith("reviewed-by:")) | "--reviewer\t" + runs("reviewed-by:"))'); then
      echo "merge-check: unavailable: jq failed reading the runs on the review lines of pull request #${pr}" >&2
      exit 2
    fi
    # A malformed group is a review finding here, not an undecided compare: an
    # empty group such as (,) yields no run (and bash 3.2 under set -u fails on
    # the empty array), a reviewer group with two pairs yields two --reviewer
    # flags, and a colon in a model or effort (claude:opus) shifts the fields.
    cargs=()
    icount_runs=0
    rcount_runs=0
    colon=0
    while IFS=$'\t' read -r flag value; do
      [ -n "$flag" ] || continue
      cargs+=("$flag" "$value")
      case "$flag" in
        --implementer) icount_runs=$((icount_runs + 1)) ;;
        --reviewer) rcount_runs=$((rcount_runs + 1)) ;;
      esac
      case "$value" in *:*:*:*) colon=1 ;; esac
    done <<<"$runs"
    if [ "$icount_runs" -lt 1 ] || [ "$rcount_runs" -ne 1 ] || [ "$colon" -ne 0 ]; then
      finding "the implemented-by: group must hold one or more model, effort pairs and the reviewed-by: group exactly one, with no colon in a model or effort" "<agent> (<model>, <effort>) on both lines"
      cargs=()
    fi
    compared=
    cstatus=0
    if [ "${#cargs[@]}" -gt 0 ]; then
      compared=$("${ATEAM:-ateam}" review compare "${cargs[@]}" 2>&1)
      cstatus=$?
    fi
    # Status 1 must come with a finding line, or it was not compare's verdict.
    if [ "$cstatus" -eq 1 ] && ! printf '%s\n' "$compared" | grep -q '^compare: finding: '; then
      cstatus=3
    fi
    case "$cstatus" in
      0 | 1)
        while IFS= read -r cline; do
          case "$cline" in
            "compare: finding: "*"names no effort"*) finding "${cline#compare: finding: }" "<agent> (<model>, <effort>) for every run on both lines" ;;
            "compare: finding: "*) finding "${cline#compare: finding: }" "${wantline}, not weaker by tier, then effort band, than every implementer run in ${policy}" ;;
            ?*) echo "merge-check: strength: ${cline#compare: }" ;;
          esac
        done <<<"$compared"
        ;;
      *)
        echo "merge-check: unavailable: ateam review compare exited ${cstatus} (set ATEAM to an ateam binary that has review compare): ${compared}" >&2
        exit 2
        ;;
    esac
  fi
  if [ "$findings" -eq 0 ]; then
    read_gh "the commits of pull request #${pr}" 'type == "array"' \
      gh api --paginate "repos/{owner}/{repo}/pulls/${pr}/commits?per_page=100"
    if ! commits=$(printf '%s' "$answer" | jq -c -s '
      if all(.[]; type == "array" and all(.[]; type == "object"
        and (.sha | type) == "string" and (.sha | length) >= 7
        and ((.parents // []) | type == "array" and all(.[]; type == "object" and (.sha | type) == "string"))
        and ((.commit.message // "") | type == "string")))
      then add // [] else error("not pages of commits") end'); then
      echo "merge-check: unavailable: the read of the commits of pull request #${pr} returned something other than what was asked for: ${answer}" >&2
      exit 2
    fi
    if ! printf '%s' "$commits" > "$commitsfile"; then
      echo "merge-check: unavailable: could not write the commit list of pull request #${pr} to a temp file" >&2
      exit 2
    fi
    reported_commits=$(printf '%s' "$pull" | jq '.commits')
    commit_entries=$(printf '%s' "$commits" | jq 'length')
    case "$commit_entries:$reported_commits" in
      [0-9]*:[0-9]*) ;;
      *) echo "merge-check: unavailable: jq failed reading the commit count of pull request #${pr}" >&2; exit 2 ;;
    esac
    if [ "$commit_entries" -eq 0 ]; then
      echo "merge-check: unavailable: pull request #${pr} has no commits" >&2
      exit 2
    fi
    if ! printf '%s' "$pull" | jq -e --argjson n "$commit_entries" --arg last "$(printf '%s' "$commits" | jq -r '.[-1].sha')" '
      .commits == $n and ((.head.sha | ascii_downcase) == ($last | ascii_downcase))' >/dev/null; then
      echo "merge-check: unavailable: the commit list of pull request #${pr} is incomplete or does not end at head: it has ${commit_entries} of ${reported_commits} commits" >&2
      exit 2
    fi
    read_gh "the comments of pull request #${pr}" 'type == "array"' \
      gh api --paginate "repos/{owner}/{repo}/issues/${pr}/comments?per_page=100"
    if ! comments=$(printf '%s' "$answer" | jq -c -s '
      if all(.[]; type == "array" and all(.[]; type == "object" and (.body == null or (.body | type) == "string")))
      then add // [] else error("not pages of comments") end'); then
      echo "merge-check: unavailable: the read of the comments of pull request #${pr} returned something other than what was asked for: ${answer}" >&2
      exit 2
    fi
    if ! record=$(printf '%s' "$comments" | jq -r --arg reviewer "$ragents" --slurpfile commits "$commitsfile" '
      $commits[0] as $commits |
      def is_record($b):
        ([ $b | scan("(?i)reviewed-by:[ \\t]*`?(claude|codex|gemini)(?=[^\\p{L}\\p{N}]|$)") | .[0] | ascii_downcase ] | unique) as $rb
        | ([ $b | split("\n")[] | rtrimstr("\r") | select(test("^[ \\t]*### Independent Cross-Agent Review\\b")) ]) as $hl
        | ([ $hl[] | scan("(?i)(?:^|[^\\p{L}\\p{N}])(claude|codex|gemini)(?=[^\\p{L}\\p{N}]|$)") | .[0] | ascii_downcase ] | unique) as $ha
        | ($rb == [$reviewer]) or (($rb | length) == 0 and ($hl | length) > 0 and (($ha | length) == 0 or $ha == [$reviewer]));
      def resolve_token($t):
        [ range(0; $commits | length) as $i
          | ($commits[$i].sha | ascii_downcase) as $csha
          | select(($csha == $t) or ($csha | startswith($t)))
          | $i ];
      def designated_commit_indices($b):
        ([ ( $b | scan("(?i)\\b(?:re-?)?reviewed[ \\t]+at[ \\t]*:?[ \\t]*`?([0-9a-f]{7,40})(?=[^0-9A-Za-z]|$)") | .[0] ),
           ( $b | scan("(?i)\\b(?:(?:re-?)?reviewed[ \\t]+(?:(?:head[ \\t]+)?commit|head)|(?:re-?)?review[ \\t]+of)(?:[ \\t]*\\([^)\\n]*\\))?[ \\t*_]*:?[ \\t*_]*`?([0-9a-f]{7,40})(?=[^0-9A-Za-z]|$)") | .[0] ),
           ( $b | scan("(?i)\\b(?:re-?)?reviewed[ \\t]+base[ \\t]+`?[0-9a-f]{7,40}`?[ \\t]+(?:->|→|to)[ \\t]+head[ \\t*_]*:?[ \\t*_]*`?([0-9a-f]{7,40})(?=[^0-9A-Za-z]|$)") | .[0] ),
           ( $b | scan("(?i)(?:^|\\n)[ \\t>*_-]*(?:reviewed-commit|head(?:[ \\t]+commit)?)(?:[ \\t]*\\([^)\\n]*\\))?[ \\t*_]*:[ \\t*_]*`?([0-9a-f]{7,40})(?=[^0-9A-Za-z]|$)") | .[0] )
         ] | map(ascii_downcase) | unique) as $toks
        | if ($toks | length) > 0 and all($toks[]; (resolve_token(.) | length) == 1)
          then [ $toks[] | resolve_token(.)[0] ] | unique
          else []
          end;
      [ .[]
        | (.body // "") as $b
        | select(is_record($b))
        | designated_commit_indices($b) as $idxs
        | select(($idxs | length) == 1)
        | $idxs[0] as $idx
        | "\($idx)|\($commits[$idx].sha)"
      ] | last // "none|"'); then
      echo "merge-check: unavailable: jq failed reading review records of pull request #${pr}" >&2
      exit 2
    fi
    IFS='|' read -r rec_idx rec_sha <<<"$record"
    case "$rec_idx" in
      none)
        finding "no review record comment names reviewed-by: ${ragents} (or ### Independent Cross-Agent Review) and a single designated reviewed commit in pull request #${pr}" \
          "post a review record comment with reviewed-by: ${ragents} and Reviewed at <commit-sha>"
        ;;
      '' | *[!0-9]*)
        echo "merge-check: unavailable: review record index of pull request #${pr} was read as '${record}'" >&2
        exit 2
        ;;
      *)
        if ! later_shas=$(printf '%s' "$commits" | jq -r --argjson idx "$rec_idx" '.[$idx + 1:][].sha'); then
          echo "merge-check: unavailable: jq failed reading commits after ${rec_sha} on pull request #${pr}" >&2
          exit 2
        fi
        while IFS= read -r c_sha; do
          [ -n "$c_sha" ] || continue
          read_gh "commit ${c_sha} of pull request #${pr}" 'type == "object"' \
            gh api --paginate "repos/{owner}/{repo}/commits/${c_sha}?per_page=100"
          if ! commit_detail=$(printf '%s' "$answer" | jq -c -s '
            (if length == 1 and (.[0] | type) == "array" then .[0] else . end)
            | if (length > 0 and all(.[]; type == "object"
                and (.sha | type) == "string"
                and (.files | type) == "array"
                and all(.files[]; type == "object" and (.filename | type) == "string" and (.previous_filename == null or (.previous_filename | type) == "string"))
                and ((.parents // []) | type == "array" and all(.[]; type == "object" and (.sha | type) == "string"))
                and ((.commit.message // "") | type == "string")))
              then
                (.[0].sha | ascii_downcase) as $s0
                | if all(.[]; (.sha | ascii_downcase) == $s0)
                  then .[0] + {page_count: length, first_page_files: (.[0].files | length), files: [.[].files[]]}
                  else error("mismatched commit page shas") end
              else error("not pages of a commit") end'); then
            echo "merge-check: unavailable: the read of commit ${c_sha} of pull request #${pr} returned something other than what was asked for: ${answer}" >&2
            exit 2
          fi
          if ! printf '%s' "$commit_detail" | jq -e '
            (.files | length) < 3000
            and ((.page_count == 1 and .first_page_files == 300) | not)
            and (.changed_files == null or ((.changed_files | type) == "number" and (.files | length) == .changed_files))
            and (.stats == null or ((.stats | type) == "object" and (.stats.additions == ([.files[].additions // 0] | add // 0)) and (.stats.deletions == ([.files[].deletions // 0] | add // 0))))' >/dev/null; then
            echo "merge-check: unavailable: the file list of commit ${c_sha} of pull request #${pr} has $(printf '%s' "$commit_detail" | jq '.files | length') files and may be incomplete" >&2
            exit 2
          fi
          if ! allowed=$(printf '%s' "$commit_detail" | jq -r --slurpfile pr_files "$filesfile" '
            def is_doc:
              (endswith(".md") or (startswith("openspec/") and (endswith(".yml") or endswith(".yaml") or endswith(".json"))))
              and (startswith(".claude/agents/") | not)
              and (startswith(".agents/contracts/") | not);
            ([$pr_files[0][] | .filename, (.previous_filename // empty)] | unique) as $prf
            | ([.files[] | .filename, (.previous_filename // empty)] | unique) as $cf
            | (((.parents // []) | length) >= 2
               and ((.commit.message // "") | split("\n")[0] | rtrimstr("\r") | test("(?i)^merge\\b.*\\b(origin/)?main\\b"))
               and all($cf[]; . as $f | ($prf | index($f)) == null))
              or (((.parents // []) | length) < 2
                  and ($cf | length) > 0
                  and all($cf[]; is_doc))'); then
            echo "merge-check: unavailable: jq failed classifying commit ${c_sha} of pull request #${pr}" >&2
            exit 2
          fi
          if [ "$allowed" != true ]; then
            finding "review record at ${rec_sha} is older than content commit ${c_sha}; request re-review" \
              "post a new review record comment from ${ragents} naming the current head commit"
            break
          fi
        done <<<"$later_shas"
        ;;
    esac
  fi
  if [ "$findings" -eq 0 ]; then
    echo "merge-check: the review check passes; reviewing agent: ${ragents} (record at ${rec_sha})"
  elif [ "$draft" = true ]; then
    echo "merge-check: pull request #${pr} is a draft: a review finding does not fail a draft (${findings} above)"
  else
    echo "merge-check: FAIL: the review check has ${findings} finding(s); editing the description starts no run, so run the merge-check job again after the edit"
    failed=1
  fi
fi

# 2. Known-flake check.
# The whole label list, not a search: gh 2.97 prints nothing at all, not [], for
# a search that matches no label, which would read as a failed read.
read_gh "labels" 'type == "array" and all(.[]; (.name | type) == "string")' \
  gh label list --json name --limit "$limit"
if ! printf '%s' "$answer" | jq -e --arg l "$label" 'map(.name) | index($l)' >/dev/null; then
  if [ "$(printf '%s' "$answer" | jq 'length')" -ge "$limit" ]; then
    echo "merge-check: unavailable: the label list returned ${limit} labels and may be incomplete" >&2
    exit 2
  fi
  echo "merge-check: FAIL: the label ${label} is missing; without it an empty issue list would read as no known flake"
  exit 1
fi

read_gh "open ${label} issues" 'type == "array" and all(.[]; (.url | type) == "string" and (.number | type) == "number")' \
  gh issue list --label "$label" --state open --json number,url,title --limit "$limit"
issues=$answer
count=$(printf '%s' "$issues" | jq 'length')
if [ "$count" -ge "$limit" ]; then
  echo "merge-check: FAIL: the list of open ${label} issues returned ${count} of ${limit} asked for and may be incomplete"
  exit 1
fi
if [ "$count" -eq 0 ]; then
  echo "merge-check: no open ${label} issue"
  [ "$failed" -eq 0 ] || { echo "merge-check: FAILED"; exit 1; }
  echo "merge-check: ok"
  exit 0
fi

read_gh "pull request #${pr}" '(.closingIssuesReferences | type == "array") and all(.closingIssuesReferences[]; (.url | type) == "string")' \
  gh pr view "$pr" --json closingIssuesReferences
# A failed jq leaves an empty result, which would read as "every flake is closed".
if ! closing=$(printf '%s' "$answer" | jq -c '[.closingIssuesReferences[].url]'); then
  echo "merge-check: unavailable: jq failed reading the closing references of pull request #${pr}" >&2
  exit 2
fi

# Compared by URL, so the same number in another repository does not count.
if ! open_not_closed=$(printf '%s' "$issues" | jq -r --argjson c "$closing" '.[] | select(.url as $u | $c | index($u) | not) | "#\(.number) \(.url) \(.title)"'); then
  echo "merge-check: unavailable: jq failed comparing the open ${label} issues with the closing references of pull request #${pr}" >&2
  exit 2
fi
if [ -n "$open_not_closed" ]; then
  echo "merge-check: FAIL: known flakes are open and not closed by this pull request:"
  printf '%s\n' "$open_not_closed" | sed 's/^/  /'
  echo "merge-check: fix them in this pull request (Closes #n for each), or wait for the pull request that does"
  exit 1
fi

exempting=$(printf '%s' "$issues" | jq -r '[.[] | "#\(.number)"] | join(", ")')
printf '%s' "$issues" | jq -r --arg pr "$pr" '.[] | "merge-check: closing #\(.number) exempts pull request #\($pr) (\(.url))"'
echo "::warning::pull request #${pr} is exempt from the known-flake check by closing ${exempting}; the reviewer confirms the reproduction before and after the fix"
[ "$failed" -eq 0 ] || { echo "merge-check: FAILED"; exit 1; }
echo "merge-check: ok"
