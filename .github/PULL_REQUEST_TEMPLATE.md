## Summary

<!-- What does this PR change, and why? -->

## Agent involvement

<!-- We welcome PRs authored and implemented by AI coding agents (see CONTRIBUTING.md).
     Note here whether an agent wrote/implemented this change, and confirm a human reviewed it. -->

## Harness results (required for new or substantively changed skills)

<!-- Delete this whole section for changes that don't touch skill content,
     and say why (e.g. "Not applicable — no skill content changed").
     See CONTRIBUTING.md, "Proving the skill helps". -->

**Model + harness:** <!-- e.g. Claude Code X.Y driving claude-opus-N -->
**Prompt(s) used:** <!-- the representative prompts, or a link to them -->

| Arm | Target-skill revision / state | Cases × reps | Pass | Fail | Unknown |
| --- | --- | ---: | ---: | ---: | ---: |
| Target skill withheld | `Withheld` | | | | |
| Current `origin/main` skill | `<full SHA>` or `Not present` | | | | |
| Proposed PR skill | `<full SHA>` or `Not present` | | | | |

<!-- Every arm needs results. `Not present` / `Not run` go in the revision cell, with a
     reason. See CONTRIBUTING.md for what must be held identical across arms. -->

**What differed:** <!-- where the withheld arm was wrong, outdated, or wasteful; what the
skill fixed; and whether the proposed arm regressed anything against `origin/main` -->

**Failing or unknown runs kept, and why:**

**Per-case results by arm:** <!-- every case, for every arm, at least three repetitions each — or a
link to the detailed report -->

**Limitations:** <!-- model, harness, fixture, or coverage limits; anything the evidence does not
establish -->

**Transcripts or summaries:** <!-- links; a gist is fine. Sanitize transcripts first — redact
credentials, tokens, customer data, and private repository content. If safe sanitization would
destroy the evidence, give a summary instead and say that it replaces the transcript. -->

## Checklist

- [ ] I read and agree to follow the [Code of Conduct](https://github.com/ollygarden/.github/blob/main/CODE_OF_CONDUCT.md)
- [ ] `./bin/validate-skill.sh` passes ([Agent Skills spec](https://agentskills.io/specification) + the under-500-line rule)
- [ ] `./bin/check-skill-inventory.py` passes
- [ ] Skill registered in all four places (skill directory, `.claude-plugin/marketplace.json`, `README.md` table + tree, `skills.sh.json` grouping) — if adding/renaming a skill
- [ ] Content is vendor-neutral, non-opinionated, DRY, and token-efficient
- [ ] All three harness arms reported above, with any `Not run` arm given a reason — if skill content changed
- [ ] Commit messages follow Conventional Commits
- [ ] I have signed (or will sign via the CLA bot on this PR) the [OllyGarden CLA](https://github.com/ollygarden/.github/blob/main/CLA.md)
