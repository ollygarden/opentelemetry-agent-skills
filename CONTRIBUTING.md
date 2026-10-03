# Contributing to OpenTelemetry Agent Skills

Thank you for your interest in contributing!

## Community expectations

Participation in this project is governed by OllyGarden's
[Code of Conduct](https://github.com/ollygarden/.github/blob/main/CODE_OF_CONDUCT.md).
For help choosing the right channel, see [SUPPORT.md](SUPPORT.md). Report suspected
security vulnerabilities privately under the inherited
[security policy](https://github.com/ollygarden/opentelemetry-agent-skills/security/policy), not in a
public issue.

Project roles and decision making are documented in OllyGarden's organization-wide
[governance policy](https://github.com/ollygarden/.github/blob/main/GOVERNANCE.md).

## Contributions from AI coding agents

We accept — and encourage — pull requests that were authored and implemented by AI coding agents (Claude Code, Codex, Cursor, etc.). This repository is itself a set of Agent Skills, and most of it was built that way.

Agent-authored PRs are held to the same bar as any other PR:

- A human contributor must open the PR (or take ownership of it), review the agent's output before submitting, and be able to respond to review feedback. You are responsible for what you submit.
- Disclose agent involvement in the PR description (e.g. a `Co-Authored-By` trailer or a short note). This is for transparency, not gatekeeping — it will not count against the PR.
- The evaluation requirement below applies regardless of who or what wrote the change.

## Getting Started

1. Search existing issues and pull requests for related work.
2. For a new skill or another large change, open a
   [proposal](https://github.com/ollygarden/opentelemetry-agent-skills/issues/new?template=new-skill.yml)
   before investing in implementation.
3. Fork and clone the repository.
4. Create a feature branch from `main`.
5. Make and validate your changes.
6. Open a focused pull request using the repository template.

Skill validation requires Python 3.11 or newer and the Agent Skills reference validator.
The pinned upstream revision lives in `bin/skills-ref.requirement`, which is the single
source CI, Renovate, and the commands below all read — so install from that file rather
than pasting a revision:

```bash
uv tool install "$(cat bin/skills-ref.requirement)"
```

Without [`uv`](https://docs.astral.sh/uv/), a virtual environment works the same way:

```bash
python3 -m venv .venv
source .venv/bin/activate
python -m pip install "$(cat bin/skills-ref.requirement)"
```

Go is only required when changing `tools/otel-agent-tools/` or its generated output.

## Adding a Skill

Prefer using the [`skill-creator`](https://github.com/anthropics/skills/tree/main/skills/skill-creator) skill to scaffold and refine new skills rather than authoring them by hand — it walks you through the structure and helps keep skills well-scoped.

Skills live under `skills/<skill-name>/` and must follow the [Agent Skills specification](https://agentskills.io/specification). Each must include a `SKILL.md` with YAML frontmatter (`name` and `description`), where the directory name matches the `name` field. Optional subdirectories: `references/`, `scripts/`, `assets/`.

Run the same gates CI runs, before opening a PR. They take no setup beyond the validator
install above. Both scripts locate the repository root themselves, so any path spelling
works from any working directory; the `./` form below assumes you are at the root:

```bash
./bin/validate-skill.sh            # all skills; pass a path to check just one
./bin/check-skill-inventory.py     # README, marketplace, skills.sh.json, and skills/ in sync
```

`validate-skill.sh` delegates spec conformance to the
[`skills-ref`](https://github.com/agentskills/agentskills/tree/main/skills-ref) reference
tool and adds this repository's own rule that a `SKILL.md` stays under 500 lines. A skill
that outgrows the cap should move detail into `references/` rather than load it all up
front.

These skills are **non-opinionated and vendor neutral by design** — they describe how OpenTelemetry works, not how you should use it. Keep them DRY and token efficient: prefer linking to official docs, examples, and source code that are already maintained over copying large amounts of knowledge into a skill, and prefer a targeted lookup or small generated artifact over dumping broad context. OllyGarden's opinionated guidance lives in the companion [`skills`](https://github.com/ollygarden/skills) repo.

When you add or rename a skill, keep all four registration points in sync: the `skills/<skill-name>/SKILL.md` directory, the `plugins` entry in `.claude-plugin/marketplace.json`, the "Available Skills" table and Repository Structure layout tree in `README.md`, and a grouping in `skills.sh.json`.

## Proving the skill helps: harness results

Every PR that adds a skill or substantively changes one must include evaluation results demonstrating that the skill actually improves agent output. A skill that doesn't measurably help is context-window cost with no benefit.

A substantive change is one that can alter when a skill triggers or what an agent does,
retrieves, recommends, or generates. Typo-only, formatting-only, link-only, and equivalent
wording changes normally do not require a harness comparison. When in doubt, include the
comparison or ask in an issue before opening the pull request.

The required evidence comes from an agent harness (Claude Code, or a comparable harness driving a frontier model), run in **three arms**:

1. **Target skill withheld** — the skill is not installed.
2. **Current `origin/main` skill** — the skill exactly as it ships today.
3. **Proposed PR skill** — the skill as this PR would ship it.

Arms 1 and 3 are the A/B comparison this repo has always asked for: does this skill help at all? Arm 2 is what catches a **regression in a skill that already ships** — neither of the others can, because neither is the current baseline. For a brand-new skill there is no revision to name, so its revision cell is `Not present` — but the arm still reports results. It is then the same configuration as the withheld arm, so it needs no extra runs. For a change to an existing skill it is the arm that matters most.

1. Pick one or more representative prompts a user would realistically ask — ideally prompts that exercise the part of the skill you added or changed.
2. Run every arm with the **same** cases, repetitions, model, harness, grading rules, and tool access, each in a fresh session. Name the model and harness once, above the table.
3. In the withheld arm, withhold **only** the target skill. Leave everything else in place.
4. Run each case **at least three times** per arm. A single run cannot distinguish a real improvement from a lucky sample.
5. Report the results in the PR description using the table in the pull request template, and attach or link the transcripts (a gist is fine) so reviewers can verify.

**Sanitize a transcript before you link it.** A harness transcript records everything the agent saw: environment variables, tokens pasted into a session, customer names, paths and file contents from private repositories. This repository is public and a linked gist usually is too, so redact before posting — and if a transcript cannot be sanitized without destroying the evidence, summarize the run instead and say that is what you did. A reviewer can work with a summary; neither of us can unpublish a leaked credential.

Recording the arms honestly matters more than a clean-looking table:

- **`Not present`** goes in the target-skill revision cell only — for a new skill, on the `origin/main` arm; for a removal, on the proposed arm. The arm's *results* are still required either way.
- **An arm you did not run, or that was invalidated, is `Not run`, with the reason.** Incomplete evidence is not an improvement claim, and a gap named plainly costs a reviewer far less than one they have to find.
- **Preserve genuine misses.** Never retry a failing repetition until it passes, and never report a designed-but-unrun case as passing.

What we look for: the baseline getting facts wrong (stale versions, renamed packages, invalid config keys) that the skill corrects; the skill reaching the right answer with fewer tokens or fewer wrong turns; and no regression against the shipping version. If the comparison shows no meaningful difference, that's a signal the skill (or the change) isn't earning its place — rework it rather than submitting the results anyway.

A pure **efficiency** improvement — trimming a skill so it costs less context while behaving identically — is a legitimate result here. Say so, state how you measured it, and show the required behavior still passing in the proposed arm.

The [`skill-creator`](https://github.com/anthropics/skills/tree/main/skills/skill-creator) skill can help you set up and run these evals.

## The `otel-agent-tools` Module

Some bundled reference data is generated by the Go CLI under `tools/otel-agent-tools/` (e.g. the version index used by `otel-sdk-versions`). If your change touches that tool or its generated output, run its checks from `tools/otel-agent-tools/` before opening a PR:

```bash
go build ./cmd/otel-agent-tools
go test ./...
```

Regenerate generated files with the tool rather than hand-editing them. CI lints, builds, and tests the module and link-checks the generated index.

## Commit Messages

We use [Conventional Commits](https://www.conventionalcommits.org/), with an optional scope naming the affected skill. Format:

```
<type>(<optional scope>): <short description>

<optional body>
```

For example: `docs(otel-go): document independently-versioned module groups`.

Common types:

- `feat` — new skill or feature
- `fix` — bug fix
- `docs` — documentation only
- `chore` — maintenance, CI, tooling
- `refactor` — restructuring without behavior change

## Pull Requests

- Keep PRs focused on a single change
- Include a summary and test plan in the PR description
- For skill additions or substantive skill changes, include the harness comparison results described above
- Update `README.md` and `.claude-plugin/marketplace.json` if adding, renaming, or removing a skill

Code owners review pull requests for correctness, scope, and maintainability. Maintainers
may ask contributors to update a branch, split unrelated changes, or permit maintainer
edits. Pull requests are squash-merged after required checks, review feedback, and CLA
requirements are satisfied. There is no guaranteed response time, but contributors are
welcome to leave a concise follow-up if a pull request has had no maintainer response for
two weeks.

## Contributor License Agreement

Before we can merge your first pull request, you must sign the organization-wide OllyGarden
[Contributor License Agreement](https://github.com/ollygarden/.github/blob/main/CLA.md). Signing is
handled automatically in the PR: the CLA bot will comment with instructions, and you sign by
replying with the requested confirmation. You only need to sign once; the signature covers all
your future contributions to this repository.
