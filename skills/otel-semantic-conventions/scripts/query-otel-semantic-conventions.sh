#!/usr/bin/env bash
set -euo pipefail

readonly RELEASE_API_URL="https://api.github.com/repos/open-telemetry/semantic-conventions/releases/latest"
readonly SEMCONV_REPO="${OTEL_SEMCONV_REPO:-}"
readonly SEMCONV_TAG="${OTEL_SEMCONV_TAG:-}"

blob_url() {
  local tag="$1"
  local path="$2"
  local line="$3"
  printf 'https://github.com/open-telemetry/semantic-conventions/blob/%s/%s#L%s' "$tag" "$path" "$line"
}

group_url() {
  local tag="$1"
  local group="$2"
  printf 'https://github.com/open-telemetry/semantic-conventions/tree/%s/model/%s' "$tag" "$group"
}

usage() {
  cat <<'EOF' >&2
Usage:
  ./scripts/query-otel-semantic-conventions.sh --groups
  ./scripts/query-otel-semantic-conventions.sh <group>
  ./scripts/query-otel-semantic-conventions.sh <group> <kind>
  ./scripts/query-otel-semantic-conventions.sh <group> <id>

Examples:
  ./scripts/query-otel-semantic-conventions.sh --groups
  ./scripts/query-otel-semantic-conventions.sh http
  ./scripts/query-otel-semantic-conventions.sh http spans
  ./scripts/query-otel-semantic-conventions.sh http span.http.client
  ./scripts/query-otel-semantic-conventions.sh http http.request.method
EOF
}

require_bin() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "missing dependency: $1" >&2
    exit 1
  }
}

normalize_group() {
  printf '%s' "$1" |
    tr '[:upper:]' '[:lower:]' |
    tr ' ' '-' |
    tr -d '.'
}

is_supported_kind() {
  case "$1" in
    common|entities|events|logs|metrics|registry|spans) return 0 ;;
    *) return 1 ;;
  esac
}

validate_release_tag() {
  local tag="$1"
  if [[ ! "$tag" =~ ^v[0-9]+(\.[0-9]+)*$ ]]; then
    echo "not a stable semantic conventions release tag: ${tag}" >&2
    return 1
  fi
}

fetch_latest_tag() {
  if [[ -n "$SEMCONV_TAG" ]]; then
    validate_release_tag "$SEMCONV_TAG" || return 1
    printf '%s\n' "$SEMCONV_TAG"
    return 0
  fi

  if [[ -n "$SEMCONV_REPO" ]]; then
    local tag
    tag="$(
      git -C "$SEMCONV_REPO" tag --list 'v[0-9]*' --sort=-version:refname |
        grep -E '^v[0-9]+(\.[0-9]+)*$' |
        sed -n '1p' ||
        true
    )"
    if [[ -z "$tag" ]]; then
      echo "failed to find a stable semantic conventions release tag in ${SEMCONV_REPO}" >&2
      return 1
    fi
    validate_release_tag "$tag" || return 1
    printf '%s\n' "$tag"
    return 0
  fi

  local tag
  tag="$(curl -fsSL "$RELEASE_API_URL" | jq -er '.tag_name')"
  validate_release_tag "$tag" || return 1
  printf '%s\n' "$tag"
}

fetch_model_paths() {
  local tag="$1"

  if [[ -n "$SEMCONV_REPO" ]]; then
    if ! git -C "$SEMCONV_REPO" ls-tree -r --name-only "$tag" model 2>/dev/null; then
      echo "failed to read semantic convention model tree from ${SEMCONV_REPO} at ${tag}" >&2
      return 1
    fi
    return 0
  fi

  local url="https://api.github.com/repos/open-telemetry/semantic-conventions/git/trees/${tag}?recursive=1"
  if ! curl -fsSL "$url" 2>/dev/null | jq -r '.tree[] | select(.type == "blob") | .path'; then
    echo "failed to fetch semantic convention model tree from ${url}" >&2
    return 1
  fi
}

fetch_group_file() {
  local tag="$1"
  local path="$2"

  if [[ -n "$SEMCONV_REPO" ]]; then
    if ! git -C "$SEMCONV_REPO" show "${tag}:${path}" 2>/dev/null; then
      echo "failed to read semantic convention file ${path} from ${SEMCONV_REPO} at ${tag}" >&2
      return 1
    fi
    return 0
  fi

  local url="https://raw.githubusercontent.com/open-telemetry/semantic-conventions/${tag}/${path}"
  if ! curl -fsSL "$url" 2>/dev/null; then
    echo "failed to fetch semantic convention file from ${url}" >&2
    return 1
  fi
}

model_records() {
  local model_paths="$1"

  awk '
    function kind_for_file(file) {
      if (index(file, "registry")) return "registry"
      if (index(file, "spans")) return "spans"
      if (index(file, "events")) return "events"
      if (index(file, "metrics")) return "metrics"
      if (index(file, "entities")) return "entities"
      if (index(file, "apphub")) return "entities"
      if (index(file, "common")) return "common"
      if (index(file, "logs")) return "logs"
      return ""
    }

    $0 ~ "^model/[^/]+/.+\\.ya?ml$" {
      path = $0
      if (path ~ /\/deprecated\//) next
      count = split(path, parts, "/")
      file = tolower(parts[count])
      if (index(file, "deprecated")) next
      kind = kind_for_file(file)
      if (kind == "") next
      printf "%s\t%s\t%s\t%s\n", parts[2], kind, path, parts[count]
    }
  ' <<<"$model_paths"
}

list_available_groups() {
  local records="$1"
  awk -F '\t' '{ print $1 }' <<<"$records" | sort -u
}

group_records() {
  local records="$1"
  local normalized_group="$2"
  awk -F '\t' -v group="$normalized_group" '
    function comparable(value) {
      gsub(/_/, "-", value)
      return value
    }

    $1 == group || comparable($1) == comparable(group)
  ' <<<"$records"
}

list_group_kinds() {
  local records="$1"
  awk -F '\t' '
    {
      key = $2
      if (files[key] == "") {
        files[key] = $4
      } else {
        files[key] = files[key] ", " $4
      }
    }
    END {
      for (key in files) {
        printf "%s\t%s\n", key, files[key]
      }
    }
  ' <<<"$records" | sort -t $'\t' -k1,1
}

find_kind_files() {
  local records="$1"
  local requested_kind="$2"
  awk -F '\t' -v kind="$requested_kind" '$2 == kind { printf "%s\t%s\n", $3, $4 }' <<<"$records"
}

find_registry_path() {
  local records="$1"
  awk -F '\t' '$2 == "registry" { print $3; exit }' <<<"$records"
}

print_available_groups() {
  local version="$1"
  local groups="$2"

  printf 'version: %s\n' "$version"
  printf 'groups:\n'
  sed 's/^/- /' <<<"$groups"
}

print_kinds_section() {
  local kinds="$1"
  printf 'kinds:\n'
  awk -F '\t' '{ printf "- %s (%s)\n", $1, $2 }' <<<"$kinds"
}

extract_entries() {
  local content="$1"
  local id_prefix="$2"
  local field_prefix="$3"
  local width="$4"

  awk \
    -v id_prefix="$id_prefix" \
    -v field_prefix="$field_prefix" \
    -v width="$width" '
    function trim(value) {
      sub(/^[[:space:]]+/, "", value)
      sub(/[[:space:]]+$/, "", value)
      return value
    }

    function squish(value) {
      gsub(/[[:space:]]+/, " ", value)
      return trim(value)
    }

    function unquote(value) {
      value = trim(value)
      if (length(value) >= 2) {
        if ((substr(value, 1, 1) == "\"" && substr(value, length(value), 1) == "\"") ||
            (substr(value, 1, 1) == "'"'"'" && substr(value, length(value), 1) == "'"'"'")) {
          value = substr(value, 2, length(value) - 2)
        }
      }
      return value
    }

    function indentation(value) {
      match(value, /[^ ]/)
      return RSTART == 0 ? length(value) : RSTART - 1
    }

    function emit_record() {
      local_brief = squish(brief)
      local_stability = stability == "" ? "-" : stability
      if (id != "") {
        printf "%-*s %-12s %s\n", width, id, local_stability, local_brief
      }
      id = ""
      stability = ""
      brief = ""
      collecting_brief = 0
    }

    index($0, id_prefix) == 1 {
      emit_record()
      id = $0
      sub("^" id_prefix, "", id)
      next
    }

    id == "" {
      next
    }

    collecting_brief && $0 !~ /^[[:space:]]*$/ && indentation($0) < length(field_prefix) {
      collecting_brief = 0
      next
    }

    collecting_brief && $0 ~ ("^" field_prefix "[A-Za-z_][A-Za-z0-9_]*:") {
      collecting_brief = 0
    }

    collecting_brief {
      line = $0
      sub(/^[[:space:]]+/, "", line)
      brief = brief " " line
      next
    }

    index($0, field_prefix "stability: ") == 1 {
      stability = $0
      sub("^" field_prefix "stability: ", "", stability)
      sub(/[[:space:]]+#.*/, "", stability)
      stability = unquote(stability)
      next
    }

    index($0, field_prefix "brief:") == 1 {
      line = $0
      sub("^" field_prefix "brief:[[:space:]]*", "", line)
      if (line == "" || line == ">" || line == "|" || line == ">-" || line == "|-" || line == ">+" || line == "|+") {
        collecting_brief = 1
        next
      }
      brief = brief " " unquote(line)
      next
    }

    END {
      emit_record()
    }
  ' <<<"$content"
}

extract_exact_entry() {
  local content="$1"
  local id_prefix="$2"
  local entry_id="$3"
  local tag="$4"
  local path="$5"

  awk -v id_prefix="$id_prefix" -v entry_id="$entry_id" -v tag="$tag" -v path="$path" '
    BEGIN {
      entry_indent = match(id_prefix, /[^ ]/) - 1
    }

    index($0, id_prefix) == 1 {
      if (found) {
        exit
      }
      if ($0 == id_prefix entry_id) {
        found = 1
        start_line = NR
        print "source: https://github.com/open-telemetry/semantic-conventions/blob/" tag "/" path "#L" start_line
      }
    }

    found {
      if (NR > start_line && $0 !~ /^[[:space:]]*$/) {
        current_indent = match($0, /[^ ]/) - 1
        if (current_indent <= entry_indent) {
          exit
        }
      }
      print
    }

    END {
      if (!found) {
        exit 1
      }
    }
  ' <<<"$content"
}

print_group_summary() {
  local requested_group="$1"
  local version="$2"
  local kinds="$3"
  local source_url="$4"
  local registry_entries="${5:-}"

  printf 'group: %s\n' "$requested_group"
  printf 'version: %s\n' "$version"
  printf 'source: %s\n' "$source_url"
  print_kinds_section "$kinds"
  printf '\n'
  if [[ -n "$registry_entries" ]]; then
    printf '%s' "$registry_entries"
  else
    printf 'note: no registry file available for this group in the latest release\n'
  fi
}

print_kind_listing() {
  local requested_group="$1"
  local version="$2"
  local kinds="$3"
  local source_url="$4"
  local requested_kind="$5"
  local kind_entries="$6"

  printf 'group: %s\n' "$requested_group"
  printf 'version: %s\n' "$version"
  printf 'source: %s\n' "$source_url"
  print_kinds_section "$kinds"
  printf '\nkind: %s\n\n%s' "$requested_kind" "$kind_entries"
}

main() {
  require_bin awk
  if [[ -n "$SEMCONV_REPO" ]]; then
    require_bin git
  else
    require_bin curl
    require_bin jq
  fi

  case "${1:-}" in
    ""|-h|--help)
      usage
      [[ $# -gt 0 ]] && return 0
      return 1
      ;;
  esac

  local tag version model_paths records groups

  tag="$(fetch_latest_tag)"
  version="${tag#v}"
  model_paths="$(fetch_model_paths "$tag")"
  records="$(model_records "$model_paths")"

  case "${1:-}" in
    --groups)
      [[ $# -eq 1 ]] || { usage; return 1; }
      groups="$(list_available_groups "$records")"
      print_available_groups "$version" "$groups"
      return 0
      ;;
  esac

  [[ $# -le 2 ]] || { usage; return 1; }

  local requested_group="$1"
  local second_arg="${2:-}"
  local normalized_group resolved_group group_records_output kinds registry_path registry_content registry_entries source_url
  local requested_kind kind_files file_path file_content kind_entries matched_entry

  normalized_group="$(normalize_group "$requested_group")"
  group_records_output="$(group_records "$records" "$normalized_group")"
  [[ -n "$group_records_output" ]] || {
    echo "no supported semantic convention files found for group '${requested_group}'" >&2
    return 1
  }

  resolved_group="$(awk -F '\t' 'NR == 1 { print $1 }' <<<"$group_records_output")"
  kinds="$(list_group_kinds "$group_records_output")"
  registry_path="$(find_registry_path "$group_records_output")"
  source_url="$(group_url "$tag" "$resolved_group")"

  if [[ -z "$second_arg" ]]; then
    if [[ -n "$registry_path" ]]; then
      registry_content="$(fetch_group_file "$tag" "$registry_path")"
      registry_entries="$(extract_entries "$registry_content" "      - id: " "        " 32)"
    else
      registry_entries=""
    fi
    print_group_summary "$resolved_group" "$version" "$kinds" "$source_url" "$registry_entries"
    return 0
  fi

  if is_supported_kind "$second_arg"; then
    requested_kind="$second_arg"
    kind_files="$(find_kind_files "$group_records_output" "$requested_kind")"
    [[ -n "$kind_files" ]] || {
      echo "semantic convention kind '${requested_kind}' is not available for group '${requested_group}'" >&2
      return 1
    }

    if [[ "$requested_kind" == "registry" ]]; then
      [[ -n "$registry_path" ]] || {
        echo "semantic convention kind 'registry' is not available for group '${requested_group}'" >&2
        return 1
      }
      registry_content="$(fetch_group_file "$tag" "$registry_path")"
      registry_entries="$(extract_entries "$registry_content" "      - id: " "        " 32)"
      print_kind_listing "$resolved_group" "$version" "$kinds" "$source_url" "$requested_kind" "$registry_entries"
      return 0
    fi

    kind_entries=""
    while IFS=$'\t' read -r file_path _; do
      [[ -n "$file_path" ]] || continue
      file_content="$(fetch_group_file "$tag" "$file_path")"
      kind_entries+="$(extract_entries "$file_content" "  - id: " "    " 36)"$'\n'
    done <<<"$kind_files"

    print_kind_listing "$resolved_group" "$version" "$kinds" "$source_url" "$requested_kind" "$kind_entries"
    return 0
  fi

  if [[ -n "$registry_path" ]]; then
    registry_content="$(fetch_group_file "$tag" "$registry_path")"
    if matched_entry="$(extract_exact_entry "$registry_content" "      - id: " "$second_arg" "$tag" "$registry_path")"; then
      printf '%s\n' "$matched_entry"
      return 0
    fi
  fi

  while IFS=$'\t' read -r requested_kind _; do
    [[ -n "$requested_kind" ]] || continue
    [[ "$requested_kind" == "registry" ]] && continue
    kind_files="$(find_kind_files "$group_records_output" "$requested_kind")"
    while IFS=$'\t' read -r file_path _; do
      [[ -n "$file_path" ]] || continue
      file_content="$(fetch_group_file "$tag" "$file_path")"
      if matched_entry="$(extract_exact_entry "$file_content" "  - id: " "$second_arg" "$tag" "$file_path")"; then
        printf '%s\n' "$matched_entry"
        return 0
      fi
    done <<<"$kind_files"
  done <<<"$kinds"

  echo "semantic convention entry not found: ${second_arg}" >&2
  return 1
}

main "$@"
