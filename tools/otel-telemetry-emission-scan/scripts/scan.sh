#!/usr/bin/env bash
set -euo pipefail

# One headless pi agent per component in config.json, then rebuild index.md.
# The prompt is SKILL.md itself, rendered per component with envsubst.
# Versions that already have a file are skipped unless options.force is true.
# SCAN_COMPONENT selects one exact <repo>/<path>; SCAN_TAG selects one exact
# release tag and requires SCAN_COMPONENT. SCAN_FORCE overrides options.force.

SKILL_DIR="$(cd "$(dirname "$0")/.." && pwd)"
OUT_DIR="$(cd "$SKILL_DIR/../.." && pwd)/skills/otel-telemetry-emissions"
CONFIG="$SKILL_DIR/config.json"
PI_CMD="${PI_CMD:-pi -p --provider openai-codex --model gpt-5.6-luna --thinking medium}"
WORK_DIR="/tmp/otel-component-telemetry"
mkdir -p "$WORK_DIR"

parallel_agents="$(jq -r '.options.parallel_agents' "$CONFIG")"
last_versions="$(jq -r '.options.last_versions' "$CONFIG")"
force="$(jq -r '.options.force // false' "$CONFIG")"
cleanup="$(jq -r '.options.cleanup // false' "$CONFIG")"
scan_component="${SCAN_COMPONENT:-}"
scan_tag="${SCAN_TAG:-}"
force="${SCAN_FORCE:-$force}"

if [ -n "$scan_tag" ] && [ -z "$scan_component" ]; then
  echo "SCAN_TAG requires SCAN_COMPONENT" >&2
  exit 1
fi

components() {
  jq -c --arg selected "$scan_component" \
    '.components[] | select($selected == "" or (.repo + "/" + .path) == $selected)' "$CONFIG"
}

if [ -n "$scan_component" ] && [ "$(components | wc -l | tr -d ' ')" -ne 1 ]; then
  echo "SCAN_COMPONENT must match exactly one configured component: $scan_component" >&2
  exit 1
fi

if [ "$cleanup" = true ]; then
  trap 'rm -rf "$WORK_DIR"' EXIT
fi

# Clone each unique repo once (blobless; file contents are fetched lazily on
# checkout). Valid cached clones are refreshed; invalid cached paths are
# replaced automatically.
components | jq -r .repo | sort -u | while read -r repo; do
  repo_name="$(basename "$repo")"
  if [[ ! "$repo_name" =~ ^[A-Za-z0-9][A-Za-z0-9._-]*$ ]]; then
    echo "invalid repository name: $repo" >&2
    exit 1
  fi
  dir="$WORK_DIR/$repo_name"
  if git -C "$dir" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    git -C "$dir" fetch --tags --force --prune
  else
    if [ -e "$dir" ]; then
      echo "replace invalid repository cache $dir" >&2
      rm -rf -- "$dir"
    fi
    git clone --filter=blob:none "https://github.com/$repo" "$dir"
  fi
done

components | {
  while read -r entry; do
    repo="$(jq -r .repo <<<"$entry")"
    path="$(jq -r .path <<<"$entry")"
    tag_prefix="$(jq -r '.tag_prefix // ""' <<<"$entry")"
    version_request="$last_versions"
    if [ -n "$scan_tag" ]; then
      git -C "$WORK_DIR/$(basename "$repo")" rev-parse --verify --quiet "refs/tags/$scan_tag" >/dev/null || {
        echo "tag not found for $repo: $scan_tag" >&2
        exit 1
      }
      if [ -n "$tag_prefix" ] && [[ "$scan_tag" != "$tag_prefix"* ]]; then
        echo "tag $scan_tag does not match prefix $tag_prefix" >&2
        exit 1
      fi
      version_request="$scan_tag"
    fi

    # Skip when all of the last N matching tags already have a file. A literal
    # tag prefix supports independently released monorepo packages. When no
    # usable tags match, let the agent resolve package versions itself.
    if [ "$force" != true ]; then
      missing=0
      matched=0
      if [ -n "$scan_tag" ]; then
        matched=1
        version=""
        if [ -z "$tag_prefix" ]; then
          version="$(git -C "$WORK_DIR/$(basename "$repo")" show "$scan_tag:$path/package.json" 2>/dev/null |
            jq -r '.version // empty' || true)"
        fi
        version="${version:-${scan_tag#"$tag_prefix"}}"
        [ -f "$OUT_DIR/$(basename "$repo")/$path/v${version#v}.md" ] || missing=1
      elif [ -n "$tag_prefix" ]; then
        while IFS= read -r tag; do
          [ -n "$tag" ] || continue
          matched=1
          version="${tag#"$tag_prefix"}"
          [ -f "$OUT_DIR/$(basename "$repo")/$path/v$version.md" ] || missing=1
        done < <(git -C "$WORK_DIR/$(basename "$repo")" tag --list "$tag_prefix*" | sort -V | tail -n "$last_versions")
      else
        while IFS= read -r tag; do
          [ -n "$tag" ] || continue
          matched=1
          [ -f "$OUT_DIR/$(basename "$repo")/$path/$tag.md" ] || missing=1
        done < <(git -C "$WORK_DIR/$(basename "$repo")" tag | grep -E '^v[0-9]+\.[0-9]+\.[0-9]+$' | sort -V | tail -n "$last_versions")
      fi
      [ "$matched" = 1 ] || missing=1
      if [ "$missing" = 0 ]; then
        echo "skip $(basename "$repo")/$path — all versions scanned" >&2
        continue
      fi
    fi

    # Strip the frontmatter — pi would parse a prompt starting with "---" as a
    # CLI option — then fill in the placeholders.
    prompt="$(awk 'NR==1 && $0=="---" {skip=1; next} skip && $0=="---" {skip=0; next} !skip' "$SKILL_DIR/SKILL.md" |
      Repo="$repo" Path="$path" Version="$version_request" TagPrefix="$tag_prefix" Force="$force" OutDir="$OUT_DIR" \
      envsubst '${Repo} ${Path} ${Version} ${TagPrefix} ${Force} ${OutDir}')"
    $PI_CMD "$prompt" &

    while [ "$(jobs -rp | wc -l)" -ge "$parallel_agents" ]; do sleep 1; done
  done
  wait
}

{
  echo "# Telemetry emissions index"
  echo
  echo "| File | Version | Last verified |"
  echo "|---|---|---|"
  find "$OUT_DIR" -name 'v*.md' | sort | while read -r f; do
    fm() { awk -F': *' "/^$1:/{print \$2; exit}" "$f"; }
    echo "| ${f#"$OUT_DIR"/} | $(fm version) | $(fm last_verified) |"
  done
} >"$OUT_DIR/index.md"
