# Design: Standardized component pages for the `otel-collector` skill

**Date:** 2026-06-09
**Status:** Approved (pending implementation plan)
**Author:** Juraci Paixão Kröhling (with Claude)

## Problem

The `otel-collector` skill (`skills/otel-collector/`) documents individual Collector
components as self-contained pages under `components/`. Today only two exist
(`log_dedup`, `interval`) and they follow an ad-hoc structure. We want to:

1. Establish a **canonical page template** so every component page is consistent and
   agent-friendly.
2. Add four high-value cost/quality components: `cardinality_guardian`, `tail_sampling`,
   `drain`, `redaction`.
3. Produce a prioritized backlog for the next wave of components.

The skill's stated design goals (see repo `README.md`) are **non-opinionated** and
**token-efficient, agent-friendly retrieval** — small fetch tables and pointers to
upstream sources of truth rather than copied docs. The template must preserve those goals.

## Canonical component-page template

Every `components/<type>.md` page MUST contain these sections, in this order. No
frontmatter on component pages (only `SKILL.md` carries frontmatter).

1. **Header metadata table** — kind, signals, per-signal stability, distributions,
   `type` name, Go module, upstream README link, and a rename note if the component was
   renamed (snake_case migration). Load-bearing for production-readiness advice.
2. **Description** — 1–3 paragraphs: what the component does and the mechanism behind it.
3. **Main use-cases** — "Use when" / "Avoid when" bullet lists.
4. **Typical config** — a minimal working YAML snippet shown inside a `service.pipelines`
   block, **plus** the full configuration reference table (key, type, default, validation
   rules).
5. **Verification** — *new required section.* A runnable recipe using `telemetrygen` (cross-
   reference the `otel-telemetrygen` skill) plus a `debug` or `file` exporter that proves
   the component behaves as documented. The recipe is tailored per component (e.g.
   `tail_sampling`: emit traces with and without error spans, confirm the sampling
   decision; `cardinality_guardian`: emit metrics with a runaway label, confirm the label
   is stripped/tagged).
6. **Advanced use-cases** — named instances (`type/name`), multi-pipeline routing,
   combining with other components, notable edge configurations.
7. **Known quirks** — gotchas, stability caveats, memory model, a validation-error → fix
   table, anti-patterns, and symptom → fix troubleshooting.
8. **Related components** — cross-links to adjacent components.

Sections 4 and 7 absorb the rich content the current pages already carry (config-reference
tables, validation errors, troubleshooting, anti-patterns) — **no information is lost** in
the refactor.

### `SKILL.md` changes

- Rewrite the "Adding a new component to this skill" section to point at this canonical
  template (the 8 sections above) instead of the looser current guidance.
- Keep the existing workflow, conventions, stability table, and rename guidance.
- Update the component index table as pages are added.

## Phases

Each phase is one Linear issue in the **Engineering** team, with a branch created from
**Linear's auto-suggested branch name** for that issue (not hand-picked names).

### Phase 1 — Refactor existing pages to the template
- Restructure `components/log_dedup.md` and `components/interval.md` to the 8-section
  canonical template.
- Primary additive change: a **Verification** section for each (currently missing).
- Reorganize existing content into the new section order; no information loss.
- Outcome: the two pages become the reference implementation of the template.

### Phase 2 — Four new cost/quality components
One new `components/<type>.md` per component + an index row in `SKILL.md`:

| Type | Signals | Stability | Primary source |
|------|---------|-----------|----------------|
| `cardinality_guardian` | metrics | Development (contrib v0.152.0) | ai-context (`telemetrydrops/ai-context`, current) |
| `tail_sampling` | traces | Beta | ai-context + upstream README |
| `drain` | logs | Alpha | upstream README (not yet in ai-context) |
| `redaction` | traces (Beta), logs/metrics (Alpha) | upstream README (not yet in ai-context) |

### Phase 3 — Backlog for the next wave
Deliverable is a **prioritized backlog** (a planning doc plus Linear sub-issues), **not**
implementation. Candidate components, ranked by OllyGarden's cost/quality lens:

1. `filter` processor
2. `transform` processor (OTTL-based; complements the `otel-ottl` skill)
3. `probabilistic_sampler` processor
4. `attributes` / `resource` processors
5. `k8s_attributes` processor
6. `routing` connector
7. `memory_limiter` processor
8. `loadbalancing` exporter — prerequisite for scaling `tail_sampling` horizontally

Explicitly **out of scope** for the backlog: `deltatocumulative` and the `span_metrics`
connector.

## Source-of-truth strategy

- **`cardinality_guardian`, `tail_sampling`**: `telemetrydrops/ai-context` (updated June
  2026) is the primary, current source.
- **`drain`, `redaction`**: fetch the upstream contrib README directly (WebFetch); not yet
  mirrored in ai-context.
- The local `opentelemetry-collector-contrib` checkout is stale (April 2025) — secondary
  reference only; do not treat its component list as authoritative.
- Verification recipes draw on the `otel-telemetrygen` skill.

## Non-goals

- No OTTL language reference (lives in `otel-ottl`).
- No declarative SDK config (lives in `otel-declarative-config`).
- No end-to-end pipeline-design guidance.
- No opinionated "best practice" prescriptions beyond documenting component behavior and
  documented gotchas.
