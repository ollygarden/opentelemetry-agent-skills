#!/usr/bin/env python3
"""Check that every skill is registered consistently across repository indexes."""

from __future__ import annotations

import json
import re
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]
SKILLS = ROOT / "skills"
MARKETPLACE = ROOT / ".claude-plugin" / "marketplace.json"
README = ROOT / "README.md"


def frontmatter_value(text: str, key: str) -> str | None:
    match = re.match(r"^---\n(.*?)\n---(?:\n|$)", text, re.DOTALL)
    if not match:
        return None
    value = re.search(rf"^{re.escape(key)}:\s*(.+?)\s*$", match.group(1), re.MULTILINE)
    return value.group(1).strip("'\"") if value else None


def main() -> int:
    errors: list[str] = []
    skills: dict[str, Path] = {}

    for skill_file in sorted(SKILLS.glob("*/SKILL.md")):
        directory_name = skill_file.parent.name
        text = skill_file.read_text()
        declared_name = frontmatter_value(text, "name")
        description = frontmatter_value(text, "description")

        if declared_name != directory_name:
            errors.append(
                f"{skill_file.relative_to(ROOT)}: name {declared_name!r} does not match "
                f"directory {directory_name!r}"
            )
        if not description:
            errors.append(f"{skill_file.relative_to(ROOT)}: missing frontmatter description")
        if declared_name in skills:
            errors.append(f"duplicate skill name: {declared_name}")
        elif declared_name:
            skills[declared_name] = skill_file.parent

    marketplace = json.loads(MARKETPLACE.read_text())
    plugins = marketplace.get("plugins", [])
    plugin_names = [plugin.get("name") for plugin in plugins]
    if len(plugin_names) != len(set(plugin_names)):
        errors.append(f"{MARKETPLACE.relative_to(ROOT)}: duplicate plugin names")

    expected_names = set(skills)
    actual_names = set(plugin_names)
    for name in sorted(expected_names - actual_names):
        errors.append(f"{MARKETPLACE.relative_to(ROOT)}: missing plugin {name!r}")
    for name in sorted(actual_names - expected_names):
        errors.append(f"{MARKETPLACE.relative_to(ROOT)}: unknown plugin {name!r}")

    for plugin in plugins:
        name = plugin.get("name")
        if name in skills and plugin.get("source") != f"./skills/{name}":
            errors.append(
                f"{MARKETPLACE.relative_to(ROOT)}: plugin {name!r} has source "
                f"{plugin.get('source')!r}; expected './skills/{name}'"
            )
        if not plugin.get("description"):
            errors.append(f"{MARKETPLACE.relative_to(ROOT)}: plugin {name!r} has no description")

    readme = README.read_text()
    for name in sorted(expected_names):
        table_entry = f"| `{name}` | `skills/{name}/` |"
        tree_entry = f"  {name}/"
        if table_entry not in readme:
            errors.append(f"{README.relative_to(ROOT)}: missing Available Skills row for {name!r}")
        if tree_entry not in readme:
            errors.append(f"{README.relative_to(ROOT)}: missing Repository Structure entry for {name!r}")

    if errors:
        print("Skill registration checks failed:", file=sys.stderr)
        for error in errors:
            print(f"- {error}", file=sys.stderr)
        return 1

    print(f"Validated registration for {len(skills)} skills.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
