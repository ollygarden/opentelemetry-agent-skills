#!/usr/bin/env bash
# Resolve a Weaver registry to JSON and pretty-print a slice with jq.
#
# Use this for a quick raw dump of the resolved schema. Note: `weaver registry
# resolve` is deprecated (still works; errors under --future), and its raw shape
# is PRE-filter — it differs from the `ctx` a template receives via the
# semconv_grouped_* jq helpers. To see the exact template fields, render
# {{ ctx | tojson }} through the real filter (see references/template-authoring.md).
#
# Usage:
#   inspect-resolved.sh <registry-dir> [jq-filter]
#
# Examples:
#   inspect-resolved.sh ./telemetry/registry/
#   inspect-resolved.sh ./telemetry/registry/ '.groups[] | select(.type=="span")'
#   inspect-resolved.sh ./telemetry/registry/ 'semconv_grouped_attributes'   # not supported here; jq only

set -euo pipefail

if [ $# -lt 1 ]; then
  sed -n '2,12p' "$0"
  exit 2
fi

REGISTRY="$1"
FILTER="${2:-.}"

WEAVER="${WEAVER:-weaver}"
if ! command -v "$WEAVER" >/dev/null 2>&1; then
  echo "weaver binary not found on PATH (set WEAVER=... or install per https://github.com/open-telemetry/weaver#install)" >&2
  exit 1
fi

if ! command -v jq >/dev/null 2>&1; then
  echo "jq is required" >&2
  exit 1
fi

tmp="$(mktemp -t weaver-resolved.XXXXXX.json)"
trap 'rm -f "$tmp"' EXIT

"$WEAVER" registry resolve -r "$REGISTRY" -f json -o "$tmp" >/dev/null

jq "$FILTER" "$tmp"
