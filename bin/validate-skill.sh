#!/usr/bin/env bash
# Validate skills against the Agent Skills spec plus this repository's house rules.
#
# Spec conformance is delegated to `skills-ref`, the reference validator from
# the Agent Skills project. It parses the frontmatter with a real YAML parser
# rather than by pattern-matching lines, so it catches what a regex cannot: an
# unquoted `description:` containing a bare `: ` breaks the document, and a
# line-based check happily reports a plausible character count for frontmatter
# that no spec client can load at all.
#
# Delegated to skills-ref:
#   - the frontmatter is parseable YAML
#   - `name` present, lowercase, <= 64 chars, letters/digits/hyphens, no
#     leading, trailing, or doubled hyphen, and equal to the directory name
#   - `description` present and <= 1024 chars
#   - `compatibility` <= 500 chars
#   - no frontmatter keys outside the spec's allowed set
#
# Checked here:
#   - SKILL.md exists. Done locally so a missing file reports as itself rather
#     than as a spec violation. This is stricter than skills-ref, which also
#     accepts a lowercase `skill.md`; AGENTS.md requires `SKILL.md`.
#   - SKILL.md is under 500 lines, per AGENTS.md "Skill constraints". A skill
#     that has outgrown that is a sign it should split detail into
#     `references/` instead of loading it all up front.
#   - No verification dates or "unchanged through" wording, per AGENTS.md
#     "Version facts name their release".
#
# Registration across README.md and marketplace.json is a separate gate; see
# bin/check-skill-inventory.py.
#
# Usage: bin/validate-skill.sh [skill-directory ...]
#        With no arguments, validates every skill under skills/.
#
# Exits 0 when every target passes, 1 on any failure.
set -euo pipefail

MAX_LINES=500
# `[^_]` skips the generated `last_verified` frontmatter key.
DATED_PATTERN='(^|[^_])[Vv]erified:? 20[0-9]{2}-[0-9]{2}|[Aa]s of 20[0-9]{2}-[0-9]{2}|[Uu]nchanged (through|since) v?[0-9]'

# Resolve arguments against the caller's directory BEFORE cd'ing to the
# repository root, so a relative path means what the caller typed.
REQUESTED=()
for arg in "$@"; do
  case "$arg" in
    /*) REQUESTED+=("${arg%/}") ;;
    *) REQUESTED+=("${PWD}/${arg%/}") ;;
  esac
done

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

if ! command -v skills-ref >/dev/null 2>&1; then
  echo "FAIL: skills-ref not found on PATH." >&2
  echo "" >&2
  echo "Install the pinned revision:" >&2
  echo "" >&2
  echo "  uv tool install '$(cat bin/skills-ref.requirement)'" >&2
  echo "" >&2
  echo "See CONTRIBUTING.md for the pip alternative." >&2
  exit 1
fi

failures=0
fail() {
  echo "FAIL: $*" >&2
  failures=$((failures + 1))
}

if [ "${#REQUESTED[@]}" -gt 0 ]; then
  TARGETS=("${REQUESTED[@]}")
else
  TARGETS=()
  # macOS ships bash 3.2, which has no mapfile.
  while IFS= read -r dir; do
    TARGETS+=("$dir")
  done < <(find skills -mindepth 1 -maxdepth 1 -type d | sort)
  [ "${#TARGETS[@]}" -gt 0 ] || {
    echo "FAIL: no skills found under skills/" >&2
    exit 1
  }
fi

for target in "${TARGETS[@]}"; do
  # Canonicalize so output does not depend on how the caller spelled the path.
  if [ -d "$target" ]; then
    target="$(cd "$target" && pwd -P)"
  fi

  SKILL_DIR="${target#"$REPO_ROOT"/}"
  SKILL_FILE="${SKILL_DIR}/SKILL.md"

  if [ ! -f "$SKILL_FILE" ]; then
    fail "$SKILL_FILE not found"
    continue
  fi

  # Count logical lines, not newline terminators: `wc -l` reports 499 for a
  # 500-line file that lacks a trailing newline, which would slip past the cap.
  LINE_COUNT="$(awk 'END { print NR }' "$SKILL_FILE")"
  house_rules_ok=true
  if [ "$LINE_COUNT" -ge "$MAX_LINES" ]; then
    fail "$SKILL_FILE is $LINE_COUNT lines, must be under $MAX_LINES"
    house_rules_ok=false
  fi

  DATED="$(grep -rnE "$DATED_PATTERN" "$SKILL_DIR" --include='*.md' --include='*.json' || true)"
  if [ -n "$DATED" ]; then
    fail "$SKILL_DIR has verification dates or history wording; state facts for the release they hold for (AGENTS.md):
$DATED"
    house_rules_ok=false
  fi

  # skills-ref exits 1 for validation failures, so anything higher is a crash
  # and must not be reported as a spec violation.
  set +e
  SPEC_OUTPUT="$(skills-ref validate "$SKILL_DIR" 2>&1)"
  SPEC_STATUS=$?
  set -e

  if [ "$SPEC_STATUS" -gt 1 ]; then
    fail "skills-ref crashed on $SKILL_DIR (exit $SPEC_STATUS):
$SPEC_OUTPUT"
  elif [ "$SPEC_STATUS" -ne 0 ]; then
    fail "$SKILL_DIR does not conform to the Agent Skills spec:
$SPEC_OUTPUT"
  elif [ "$house_rules_ok" = true ]; then
    echo "OK: $SKILL_DIR ($LINE_COUNT lines)"
  fi
done

if [ "$failures" -gt 0 ]; then
  echo "" >&2
  echo "$failures validation problem(s)." >&2
  exit 1
fi

echo "OK: ${#TARGETS[@]} skill(s) validated"
