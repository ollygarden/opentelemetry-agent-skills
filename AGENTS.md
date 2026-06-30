# AGENTS.md

This file provides guidance to AI coding agents when working with code in this repository.

## What this repository is

A catalog of **Agent Skills** (per the [agentskills.io](https://agentskills.io/specification) spec) for OpenTelemetry, published by OllyGarden. The skills are **non-opinionated and vendor-neutral by design** — there are many valid ways to use OpenTelemetry, so prescribing conventions is out of scope. They exist to give an agent **token-efficient, agent-friendly retrieval**: small fetch tables, lookup indexes, and scripts that point at upstream sources of truth instead of copying docs into context, so answers stay current as OpenTelemetry evolves.

Most changes are to Markdown and YAML files that AI agents consume. The exception is `tools/otel-agent-tools/`, a Go module that generates and validates some of the bundled reference data (see below). "Correctness" means a skill is well-scoped, accurate, points at the maintained source of truth, and is registered in all the right places.

## Architecture

Each skill is a self-contained directory under `skills/<skill-name>/`:

- `SKILL.md` (required) — YAML frontmatter (`name`, `description`, optional `license`, `compatibility`, `metadata`) followed by the instruction body.
- `references/` (optional) — task-focused docs the SKILL.md links to for detail it doesn't inline (e.g. `otel-go/references/` splits setup, API, instrumentation libraries, performance, breaking changes).
- `components/` (skill-specific) — `otel-collector` uses one file per Collector component for progressive disclosure.
- `scripts/` (optional) — helper or lookup scripts (e.g. `otel-semantic-conventions` ships a query script).

Two hard rules that are easy to get wrong:

1. **The directory name must equal the `name:` field** in its `SKILL.md` (spec directory rule).
2. Skill `name:` fields are **unprefixed** (`otel-go`, `otel-collector`, …). This is the upstream *facts* package — the companion [`skills`](https://github.com/ollygarden/skills) repo holds OllyGarden's *opinions* under an `ollygarden-` prefix and references these skills for facts. Keep facts here; don't fold opinions in.

## The `tools/otel-agent-tools` Go module

A small Go CLI under `tools/otel-agent-tools/` (wired into the workspace via `go.work`) fetches upstream OpenTelemetry data and renders the generated reference index consumed by `otel-sdk-versions`. CI lints, builds, and tests it, and link-checks the generated index. When changing it:

- `go build ./cmd/otel-agent-tools` and `go test ./...` from `tools/otel-agent-tools/`.
- Generated output (e.g. `skills/otel-sdk-versions/references/generated/otel-version-index.md`) is produced by the tool — regenerate it rather than hand-editing, so it stays consistent and the link check passes.

## Adding or renaming a skill — keep three places in sync

A new skill is only "registered" when it appears in **all** of these. Missing any one is the most common defect:

1. The directory `skills/<name>/` with a `SKILL.md`.
2. The `plugins` array in `.claude-plugin/marketplace.json` (`name` + `source: ./skills/<name>` + `description`).
3. The "Available Skills" table **and** the Repository Structure layout tree in `README.md`.

## Conventions

- Commits follow [Conventional Commits](https://www.conventionalcommits.org/) (`feat`, `fix`, `docs`, `chore`, `refactor`), with an optional scope naming the skill (`docs(otel-go): …`). New skills are typically `feat`.
- **Keep skills DRY.** Prefer referencing official docs, examples, and source that are already maintained over copying large amounts of knowledge into a skill. Exceptions exist, but the default is to link to the source of truth.
- **Design for token efficiency.** Avoid dumping large files or broad context when a targeted lookup, focused reference, or small generated artifact will do.
- **Stay vendor neutral and non-opinionated.** Opinions belong in the companion `skills` repo.
- A skill `description` is the trigger surface: it should enumerate concrete user phrasings so agents activate it reliably. Mirror the existing skills' description style.
- `local/` is gitignored — used for scratch/research notes, never published.
