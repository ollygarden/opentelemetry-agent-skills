# AGENTS.md

This file provides guidance to AI coding agents when working with code in this repository.

## What this repository is

A catalog of **Agent Skills** (per the [agentskills.io](https://agentskills.io/specification) spec) for OpenTelemetry, published by OllyGarden. The skills are **non-opinionated and vendor-neutral by design** — there are many valid ways to use OpenTelemetry, so prescribing conventions is out of scope. They exist to give an agent **token-efficient, agent-friendly retrieval**: small fetch tables, lookup indexes, and scripts that point at upstream sources of truth instead of copying docs into context, so answers stay current as OpenTelemetry evolves.

Most changes are to Markdown and YAML files that AI agents consume. The exception is `tools/otel-agent-tools/`, a Go module that generates and validates some of the bundled reference data (see below). "Correctness" means a skill is well-scoped, accurate, points at the maintained source of truth, and is registered in all the right places.

## Preferred workflow

For any change that adds, renames, moves, or removes a skill, or that alters what a skill triggers on or recommends, follow [`docs/preferred-workflow.md`](docs/preferred-workflow.md). Typo and prose fixes go straight to a PR — unless they touch `SKILL.md` frontmatter, since a `description:` edit changes when the skill activates.

## Running the gates locally

The two skill gates are scripts in `bin/`. Both resolve the repository root themselves, so any path spelling works from any working directory — the `./` form below assumes you are at the root:

```bash
./bin/validate-skill.sh            # spec conformance + house rules; a path checks one skill
./bin/check-skill-inventory.py     # skills/, marketplace.json, README, and skills.sh.json in sync
```

CI enforces one further check that is not a `bin/` script: the `Link Check` workflow. Reproduce it locally with the `lychee` command in [`docs/preferred-workflow.md`](docs/preferred-workflow.md#5-run-the-gates).

`validate-skill.sh` needs `skills-ref`, pinned in `bin/skills-ref.requirement` — that file is the single source CI, Renovate, and `CONTRIBUTING.md` all read, so never paste a revision anywhere else. Install with `uv tool install "$(cat bin/skills-ref.requirement)"`.

## Skill constraints

- The directory name equals the `name:` field — **automated** by `bin/validate-skill.sh` (via `skills-ref`).
- Frontmatter parses as YAML, `description` fits in 1024 characters, no unknown keys — **automated**, same script.
- `SKILL.md` stays under 500 lines — **automated**. Past that, move detail into `references/`.
- Registration across `skills/`, `marketplace.json`, `README.md`, and `skills.sh.json` — **automated** by `bin/check-skill-inventory.py`.
- Links resolve — **automated** by the `Link Check` workflow, which scans the whole repository weekly and on every PR.
- Keeping content vendor-neutral, DRY, and token-efficient — **review-enforced**, not automated. A green build says nothing about it.

## Architecture

Each skill is a self-contained directory under `skills/<skill-name>/`:

- `SKILL.md` (required) — YAML frontmatter (`name`, `description`, optional `license`, `compatibility`, `metadata`) followed by the instruction body.
- `references/` (optional) — task-focused docs the SKILL.md links to for detail it doesn't inline (e.g. `otel-go/references/` splits setup, API, instrumentation libraries, performance, breaking changes).
- `components/` (skill-specific) — `otel-collector` uses one directory per Collector component (`README.md` plus `configuration.md`, `advanced.md`, `quirks.md`, `verification.md`) for progressive disclosure.
- `scripts/` (optional) — helper or lookup scripts (e.g. `otel-semantic-conventions` ships a query script).

Two hard rules that are easy to get wrong:

1. **The directory name must equal the `name:` field** in its `SKILL.md` (spec directory rule).
2. Skill `name:` fields are **unprefixed** (`otel-go`, `otel-collector`, …). This is the upstream *facts* package — the companion [`skills`](https://github.com/ollygarden/skills) repo holds OllyGarden's *opinions* under an `ollygarden-` prefix and references these skills for facts. Keep facts here; don't fold opinions in.

## The `tools/otel-agent-tools` Go module

A small Go CLI under `tools/otel-agent-tools/` (wired into the workspace via `go.work`) fetches upstream OpenTelemetry data and renders the generated reference index consumed by `otel-sdk-versions`. CI lints, builds, and tests it, and link-checks the generated index. When changing it:

- `go build ./cmd/otel-agent-tools` and `go test ./...` from `tools/otel-agent-tools/`.
- Generated output (e.g. `skills/otel-sdk-versions/references/generated/otel-version-index.md`) is produced by the tool — regenerate it rather than hand-editing, so it stays consistent and the link check passes.

## Adding or renaming a skill — keep four places in sync

A new skill is only "registered" when it appears in **all** of these. Missing any one is the most common defect:

1. The directory `skills/<name>/` with a `SKILL.md`.
2. The `plugins` array in `.claude-plugin/marketplace.json` (`name` + `source: ./skills/<name>` + `description`).
3. The "Available Skills" table **and** the Repository Structure layout tree in `README.md`.
4. A grouping in `skills.sh.json`, which sets the sections on the skills.sh repository page.

## Contribution requirements (external PRs)

`CONTRIBUTING.md` is the source of truth; the parts an agent preparing a PR must know:

- **Agent-authored PRs are accepted** and expected — but a human must own the PR, and agent involvement should be disclosed in the description.
- **Harness evidence is required** for any PR that adds or substantively changes a skill: the same representative prompt(s) run on a frontier model, in fresh sessions, at least three times per case, under identical conditions across three arms — target skill withheld, current `origin/main` skill, and the proposed PR skill. The `origin/main` arm is the one that catches a regression in a skill that already ships; for a brand-new skill it has no revision to name (`Not present`) and is the same configuration as the withheld arm, so it needs no extra runs. Report the arms in the table in `.github/PULL_REQUEST_TEMPLATE.md`, with links to **sanitized** transcripts — redact credentials, tokens, customer data, and private repository content before posting, or summarize the run instead. **Read [`CONTRIBUTING.md`](CONTRIBUTING.md#proving-the-skill-helps-harness-results) before running the evals** — it is the source of truth for what must be held identical across arms and how an absent or unrun arm is recorded, and this bullet is a summary of it, not a substitute.
- **Spec conformance**: validate with `./bin/validate-skill.sh` ([agentskills.io spec](https://agentskills.io/specification)).
- **CLA**: first-time contributors sign the organization-wide
  [OllyGarden CLA](https://github.com/ollygarden/.github/blob/main/CLA.md) via the CLA bot on the PR
  (`.github/workflows/cla.yml`).

## Conventions

- Commits follow [Conventional Commits](https://www.conventionalcommits.org/) (`feat`, `fix`, `docs`, `chore`, `refactor`), with an optional scope naming the skill (`docs(otel-go): …`). New skills are typically `feat`.
- **Keep skills DRY.** Prefer referencing official docs, examples, and source that are already maintained over copying large amounts of knowledge into a skill. Exceptions exist, but the default is to link to the source of truth.
- **Design for token efficiency.** Avoid dumping large files or broad context when a targeted lookup, focused reference, or small generated artifact will do.
- **Stay vendor neutral and non-opinionated.** Opinions belong in the companion `skills` repo.
- A skill `description` is the trigger surface: it should enumerate concrete user phrasings so agents activate it reliably. Mirror the existing skills' description style.
- `local/` is gitignored (see `.gitignore`) — used for scratch/research notes, never published. The link checker honours `.gitignore`, so scratch files cannot redden CI either.
