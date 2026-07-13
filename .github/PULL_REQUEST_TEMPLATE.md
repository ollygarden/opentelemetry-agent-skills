## Summary

<!-- What does this PR change, and why? -->

## Agent involvement

<!-- We welcome PRs authored and implemented by AI coding agents (see CONTRIBUTING.md).
     Note here whether an agent wrote/implemented this change, and confirm a human reviewed it. -->

## Harness results (required for new or substantively changed skills)

<!-- A/B comparison proving the skill helps, per CONTRIBUTING.md:
     - Prompt(s) used
     - Model + harness (e.g. Claude Code X.Y with claude-opus-N)
     - Baseline run (without the skill): what it got wrong, outdated, or wasteful
     - Skill run (same prompt, same model): what improved
     - Links to transcripts (gist is fine)
     Delete this section for changes that don't touch skill content. -->

## Checklist

- [ ] I read and agree to follow the [Code of Conduct](https://github.com/ollygarden/.github/blob/main/CODE_OF_CONDUCT.md)
- [ ] `skills-ref validate skills/<skill-name>` passes ([Agent Skills spec](https://agentskills.io/specification))
- [ ] `python .github/scripts/check-skill-registration.py` passes
- [ ] Skill registered in all three places (skill directory, `.claude-plugin/marketplace.json`, `README.md` table + tree) — if adding/renaming a skill
- [ ] Content is vendor-neutral, non-opinionated, DRY, and token-efficient
- [ ] Harness comparison results included above — if skill content changed
- [ ] Commit messages follow Conventional Commits
- [ ] I have signed (or will sign via the CLA bot on this PR) the [OllyGarden CLA](../CLA.md)
