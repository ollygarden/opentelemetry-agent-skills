#!/usr/bin/env python3
"""Check that every skill is registered consistently across repository indexes.

Four places list skills, and a skill is only "registered" when it appears in
all of them: skills/ on disk, the plugins array in
.claude-plugin/marketplace.json, both the "Available Skills" table and the
Repository Structure tree in README.md, and the groupings in skills.sh.json.
Missing one is the most common defect in a new-skill PR, so this fails the build in both directions — a skill with no
entry, and an entry with no skill.

Spec conformance (frontmatter shape, name/description limits) is a separate
gate; see bin/validate-skill.sh.

Usage: bin/check-skill-inventory.py    (resolves the repository root itself,
so any path spelling works from any working directory)
"""

from __future__ import annotations

import json
import re
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
SKILLS = ROOT / "skills"
MARKETPLACE = ROOT / ".claude-plugin" / "marketplace.json"
README = ROOT / "README.md"
SKILLS_SH = ROOT / "skills.sh.json"


def frontmatter_value(text: str, key: str) -> str | None:
    match = re.match(r"^---\n(.*?)\n---(?:\n|$)", text, re.DOTALL)
    if not match:
        return None
    value = re.search(rf"^{re.escape(key)}:\s*(.+?)\s*$", match.group(1), re.MULTILINE)
    return value.group(1).strip("'\"") if value else None


def readme_tree_entries(readme: str) -> list[str]:
    """Skill directories listed under `skills/` in the Repository Structure tree.

    Anchored to the tree's actual line shape and scoped to the `skills/` block,
    rather than searching the whole README for the substring "  <name>/". A bare
    substring match is satisfied by a fenced example, a prose mention, or any
    other indented listing, so it can report a skill as present in the tree when
    the tree does not list it.
    """
    entries: list[str] = []
    in_skills_block = False
    for line in readme.splitlines():
        if line.rstrip() == "skills/":
            in_skills_block = True
            continue
        if not in_skills_block:
            continue
        # The block ends at the first non-blank line that is not indented —
        # a sibling top-level entry such as `bin/`, or the closing fence.
        if line.strip() and not line.startswith("  "):
            break
        # Exactly two spaces of indent is a skill directory; deeper lines are
        # its contents (`    references/`, `    SKILL.md`).
        match = re.match(r"^ {2}([a-z0-9-]+)/\s*(#.*)?$", line)
        if match:
            entries.append(match.group(1))
    return entries


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
    readme_label = README.relative_to(ROOT)

    # Both README indexes are diffed as sets, in both directions. Checking only
    # "every skill on disk appears in the README" would let a row for a deleted
    # skill live on forever, advertising something that is no longer shipped.
    table_rows = re.findall(
        r"^\| `([a-z0-9-]+)` \| `skills/\1/` \|", readme, re.MULTILINE
    )
    tree_entries = readme_tree_entries(readme)
    groupings = json.loads(SKILLS_SH.read_text())["groupings"]
    grouped = [name for group in groupings for name in group["skills"]]

    for source, label, found in (
        (readme_label, "Available Skills table", table_rows),
        (readme_label, "Repository Structure tree", tree_entries),
        (SKILLS_SH.relative_to(ROOT), "grouping", grouped),
    ):
        if len(found) != len(set(found)):
            duplicates = sorted({name for name in found if found.count(name) > 1})
            errors.append(f"{source}: {label} lists {duplicates} more than once")
        for name in sorted(expected_names - set(found)):
            errors.append(f"{source}: missing {label} entry for {name!r}")
        for name in sorted(set(found) - expected_names):
            errors.append(
                f"{source}: {label} lists {name!r}, which is not a directory under skills/"
            )

    if errors:
        print("Skill registration checks failed:", file=sys.stderr)
        for error in errors:
            print(f"- {error}", file=sys.stderr)
        return 1

    print(f"Validated registration for {len(skills)} skills.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
