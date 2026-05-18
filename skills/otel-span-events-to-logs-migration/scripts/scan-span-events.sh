#!/usr/bin/env bash
set -euo pipefail

# Scans a directory for deprecated Span Event API usage across common languages.
# Outputs each match with file, line number, matched pattern, and a classification hint.

usage() {
  cat <<'EOF' >&2
Usage:
  ./scripts/scan-span-events.sh <directory>
  ./scripts/scan-span-events.sh .

Scans for deprecated Span Event API calls (AddEvent, RecordException and
their language-specific variants) and prints matches grouped by category.
EOF
}

require_bin() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "missing dependency: $1" >&2
    exit 1
  }
}

scan_pattern() {
  local dir="$1"
  local label="$2"
  local pattern="$3"

  local results
  results="$(grep -rn --include='*.go' --include='*.java' --include='*.py' \
    --include='*.js' --include='*.ts' --include='*.tsx' --include='*.cs' \
    --include='*.rb' --include='*.rs' --include='*.kt' --include='*.scala' \
    -E "$pattern" "$dir" 2>/dev/null || true)"

  if [[ -n "$results" ]]; then
    printf '\n## %s\n\n' "$label"
    printf '%s\n' "$results" | while IFS= read -r line; do
      printf '  %s\n' "$line"
    done
    return 0
  fi
  return 1
}

main() {
  require_bin grep

  local dir="${1:-}"
  if [[ -z "$dir" || "$dir" == "-h" || "$dir" == "--help" ]]; then
    usage
    [[ -n "$dir" ]] && return 0
    return 1
  fi

  if [[ ! -d "$dir" ]]; then
    echo "error: '${dir}' is not a directory" >&2
    return 1
  fi

  local found=0

  printf '# Span Event API Usage Scan\n'
  printf 'directory: %s\n' "$dir"

  # RecordException variants
  if scan_pattern "$dir" "RecordException (exception recording)" \
    '(RecordException|record_exception|recordException|RecordError|RecordErr)'; then
    found=1
  fi

  # AddEvent variants
  if scan_pattern "$dir" "AddEvent (general events)" \
    '\.(AddEvent|add_event|addEvent)\b'; then
    found=1
  fi

  # Activity-based (.NET)
  if scan_pattern "$dir" "Activity span events (.NET)" \
    '(activity\??\.(AddEvent|RecordException)|ActivityEvent)'; then
    found=1
  fi

  if [[ "$found" -eq 0 ]]; then
    printf '\nNo deprecated Span Event API usage found.\n'
  else
    printf '\n---\n'
    printf 'total categories with matches: see sections above\n'
    printf 'classify each match using references/decision-tree.md\n'
  fi
}

main "$@"
