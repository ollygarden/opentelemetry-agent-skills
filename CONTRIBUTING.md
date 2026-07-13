# Contributing to OpenTelemetry Agent Skills

Thank you for your interest in contributing!

## Community expectations

Participation in this project is governed by OllyGarden's
[Code of Conduct](https://github.com/ollygarden/.github/blob/main/CODE_OF_CONDUCT.md).
For help choosing the right channel, see [SUPPORT.md](SUPPORT.md). Report suspected
security vulnerabilities privately under [SECURITY.md](SECURITY.md), not in a public issue.

Project roles and decision making are documented in [GOVERNANCE.md](GOVERNANCE.md).

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
Install it in a virtual environment from its pinned upstream source:

```bash
python3 -m venv .venv
source .venv/bin/activate
python -m pip install "git+https://github.com/agentskills/agentskills.git@38a2ff82958afee88dadf4831509e6f7e9d8ef4e#subdirectory=skills-ref"
```

Go is only required when changing `tools/otel-agent-tools/` or its generated output.

## Adding a Skill

Prefer using the [`skill-creator`](https://github.com/anthropics/skills/tree/main/skill-creator) skill to scaffold and refine new skills rather than authoring them by hand — it walks you through the structure and helps keep skills well-scoped.

Skills live under `skills/<skill-name>/` and must follow the [Agent Skills specification](https://agentskills.io/specification). Each must include a `SKILL.md` with YAML frontmatter (`name` and `description`), where the directory name matches the `name` field. Optional subdirectories: `references/`, `scripts/`, `assets/`.

Validate spec conformance locally with the [`skills-ref`](https://github.com/agentskills/agentskills/tree/main/skills-ref) reference tool before opening a PR:

```bash
skills-ref validate skills/<skill-name>
python .github/scripts/check-skill-registration.py
```

These skills are **non-opinionated and vendor neutral by design** — they describe how OpenTelemetry works, not how you should use it. Keep them DRY and token efficient: prefer linking to official docs, examples, and source code that are already maintained over copying large amounts of knowledge into a skill, and prefer a targeted lookup or small generated artifact over dumping broad context. OllyGarden's opinionated guidance lives in the companion [`skills`](https://github.com/ollygarden/skills) repo.

When you add or rename a skill, keep all three registration points in sync: the `skills/<skill-name>/SKILL.md` directory, the `plugins` entry in `.claude-plugin/marketplace.json`, and the "Available Skills" table and Repository Structure layout tree in `README.md`.

## Proving the skill helps: harness results

Every PR that adds a skill or substantively changes one must include evaluation results demonstrating that the skill actually improves agent output. A skill that doesn't measurably help is context-window cost with no benefit.

A substantive change is one that can alter when a skill triggers or what an agent does,
retrieves, recommends, or generates. Typo-only, formatting-only, link-only, and equivalent
wording changes normally do not require a harness comparison. When in doubt, include the
comparison or ask in an issue before opening the pull request.

The required evidence is an A/B comparison from an agent harness (Claude Code, or a comparable harness driving a frontier model):

1. Pick one or more representative prompts a user would realistically ask — ideally prompts that exercise the part of the skill you added or changed.
2. Run each prompt **without** the skill installed, on a frontier model (e.g. the current Claude Opus/Sonnet generation), in a fresh session.
3. Run the **same prompt, same model, same harness** with the skill installed, in a fresh session.
4. Include the results in the PR description: the prompts used, the model and harness versions, and a summary of how the outputs differed — where the baseline was wrong, outdated, or wasteful, and what the skill fixed. Attach or link the transcripts (a gist is fine) so reviewers can verify.

What we look for: the baseline getting facts wrong (stale versions, renamed packages, invalid config keys) that the skill corrects; the skill reaching the right answer with fewer tokens or fewer wrong turns; and no regressions on prompts the skill shouldn't affect. If the comparison shows no meaningful difference, that's a signal the skill (or the change) isn't earning its place — rework it rather than submitting the results anyway.

The [`skill-creator`](https://github.com/anthropics/skills/tree/main/skill-creator) skill can help you set up and run these evals.

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

Before we can merge your first pull request, you must sign the OllyGarden [Contributor License Agreement](CLA.md). Signing is handled automatically in the PR: the CLA bot will comment with instructions, and you sign by replying with the requested confirmation. You only need to sign once; the signature covers all your future contributions to this repository.
