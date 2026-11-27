#!/usr/bin/env python3
"""Check or bump the release version that the plugin manifests and package.json carry.

Marketplaces read the version from the plugin manifests at a release tag and npm
publishes package.json's, so all must carry one MAJOR.MINOR.PATCH value that the
tag names. See "Releases" in CONTRIBUTING.md.

Usage: bin/manifest-version.py check [--tag vX.Y.Z]
       bin/manifest-version.py bump patch|minor|major
Both print the resulting version. The script resolves the repository root
itself, so any path spelling works from any working directory.
"""

from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
MANIFESTS = {
    "plugin.json": ("version",),
    ".claude-plugin/plugin.json": ("version",),
    ".claude-plugin/marketplace.json": ("metadata", "version"),
    "package.json": ("version",),
}
SEMVER = re.compile(r"(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)")


def load(path: str) -> dict:
    return json.loads((ROOT / path).read_text(encoding="utf-8"))


def version_of(data: dict, keys: tuple[str, ...]) -> str | None:
    for key in keys:
        data = data.get(key) if isinstance(data, dict) else None
    return data if isinstance(data, str) else None


def check(tag: str | None = None) -> str:
    versions = {
        f"{path} {'.'.join(keys)}": version_of(load(path), keys)
        for path, keys in MANIFESTS.items()
    }
    values = set(versions.values())
    version = values.pop() if len(values) == 1 else None
    if not version or not SEMVER.fullmatch(version):
        listed = ", ".join(f"{field} = {value!r}" for field, value in versions.items())
        sys.exit(f"Manifests must carry one MAJOR.MINOR.PATCH version: {listed}")
    if tag is not None and tag != f"v{version}":
        sys.exit(f"Tag {tag} does not match manifest version {version}. Tag the commit that bumped it.")
    return version


def bump(part: str) -> str:
    major, minor, patch = map(int, check().split("."))
    version = {
        "major": f"{major + 1}.0.0",
        "minor": f"{major}.{minor + 1}.0",
        "patch": f"{major}.{minor}.{patch + 1}",
    }[part]
    for path, keys in MANIFESTS.items():
        data = load(path)
        parent = data
        for key in keys[:-1]:
            parent = parent[key]
        parent[keys[-1]] = version
        (ROOT / path).write_text(json.dumps(data, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
    return check()


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    commands = parser.add_subparsers(dest="command", required=True)
    commands.add_parser("check").add_argument("--tag")
    commands.add_parser("bump").add_argument("part", choices=("patch", "minor", "major"))
    args = parser.parse_args()
    print(check(args.tag) if args.command == "check" else bump(args.part))


if __name__ == "__main__":
    main()
