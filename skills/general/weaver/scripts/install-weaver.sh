#!/usr/bin/env bash
# Install a pinned OpenTelemetry Weaver binary to ~/.local/bin/weaver.
#
# Why this exists:
#   `brew install weaver` resolves to the unrelated Scribd `weaver` tool.
#   Always install from open-telemetry/weaver GitHub releases.
#
# Usage:
#   install-weaver.sh                # install the pinned version
#   install-weaver.sh --force        # overwrite existing binary
#   install-weaver.sh --version 0.22.1
#
# Releases: https://github.com/open-telemetry/weaver/releases

set -euo pipefail

VERSION="0.22.1"
FORCE=0

while [ $# -gt 0 ]; do
  case "$1" in
    --force) FORCE=1; shift ;;
    --version) VERSION="$2"; shift 2 ;;
    -h|--help)
      sed -n '2,16p' "$0"
      exit 0
      ;;
    *) echo "unknown arg: $1" >&2; exit 2 ;;
  esac
done

INSTALL_DIR="${HOME}/.local/bin"
BINARY="${INSTALL_DIR}/weaver"

if [ -e "$BINARY" ] && [ "$FORCE" -eq 0 ]; then
  echo "weaver already installed at $BINARY (use --force to overwrite)"
  "$BINARY" --version || true
  exit 0
fi

uname_s="$(uname -s)"
uname_m="$(uname -m)"

case "${uname_s}-${uname_m}" in
  Darwin-arm64)  ASSET="weaver-aarch64-apple-darwin.tar.xz" ;;
  Darwin-x86_64) ASSET="weaver-x86_64-apple-darwin.tar.xz" ;;
  Linux-x86_64)  ASSET="weaver-x86_64-unknown-linux-gnu.tar.xz" ;;
  Linux-aarch64) ASSET="weaver-aarch64-unknown-linux-gnu.tar.xz" ;;
  *)
    echo "unsupported platform: ${uname_s}-${uname_m}" >&2
    echo "see https://github.com/open-telemetry/weaver/releases for available assets" >&2
    exit 1
    ;;
esac

URL="https://github.com/open-telemetry/weaver/releases/download/v${VERSION}/${ASSET}"

mkdir -p "$INSTALL_DIR"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

echo "downloading $URL"
curl -fsSL "$URL" -o "${tmpdir}/weaver.tar.xz"

tar -xJf "${tmpdir}/weaver.tar.xz" -C "$tmpdir"

# Archive layout: <asset-stem>/weaver
extracted="$(find "$tmpdir" -type f -name weaver | head -n1)"
if [ -z "$extracted" ]; then
  echo "could not locate weaver binary inside archive" >&2
  exit 1
fi

install -m 0755 "$extracted" "$BINARY"

echo "installed: $BINARY"
"$BINARY" --version

case ":$PATH:" in
  *":${INSTALL_DIR}:"*) ;;
  *) echo "note: $INSTALL_DIR is not on \$PATH" >&2 ;;
esac
