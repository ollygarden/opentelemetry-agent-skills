# Backlog: next-wave `otel-collector` component pages

**Date:** 2026-06-09
**Parent:** Phase 3 of `2026-06-09-otel-collector-component-skills-design.md`
**Status:** Backlog (planning only — no pages written in this phase)

This is the prioritized list of OpenTelemetry Collector components to document next in the
`otel-collector` skill, after the two refactored pages (`log_dedup`, `interval`) and the
four cost/quality pages (`cardinality_guardian`, `tail_sampling`, `drain`, `redaction`).

Every page follows the canonical 8-section template documented in `SKILL.md`: header
metadata table · description · main use-cases · typical config · verification · advanced
use-cases · known quirks · related components. Each candidate below has a Linear sub-issue
under the Phase 3 parent (E-2201).

Ranking reflects OllyGarden's cost/quality lens: components that directly cut telemetry
volume/cardinality, or that are load-bearing prerequisites for the components already
documented, come first.

## Candidates (in priority order)

### 1. `filter` processor
- **Kind / signals:** processor / traces, metrics, logs
- **Why high-value:** the most direct volume lever — drops spans/metrics/logs by OTTL
  condition before they hit the backend. Pairs with the static-asset / health-check
  noise patterns OllyGarden's detectors surface.
- **Primary source:** ai-context `collector/components/filterprocessor/`.

### 2. `transform` processor
- **Kind / signals:** processor / traces, metrics, logs
- **Why high-value:** OTTL-based in-place mutation — strip high-cardinality attributes,
  normalize resources, redact, downsample. The general-purpose quality knob. Complements
  the existing `otel-ottl` skill (this page covers the *processor* config surface; OTTL
  language stays in `otel-ottl`).
- **Primary source:** ai-context `collector/components/transformprocessor/`.

### 3. `probabilistic_sampler` processor
- **Kind / signals:** processor / traces, logs
- **Why high-value:** head-sampling volume control; the cheaper counterpart to
  `tail_sampling`. Documenting both lets the skill advise on the head-vs-tail trade-off.
- **Primary source:** ai-context (add if missing) / upstream contrib README.

### 4. `attributes` / `resource` processors
- **Kind / signals:** processor / traces, metrics, logs (resource: all)
- **Why high-value:** the everyday tools for adding/removing/hashing attributes and
  resource keys — cardinality control and PII handling at the simplest level. Two closely
  related components; document together or as adjacent pages.
- **Primary source:** ai-context `collector/components/attributesprocessor/` and
  `resourceprocessor/`.

### 5. `k8s_attributes` processor
- **Kind / signals:** processor / traces, metrics, logs
- **Why high-value:** enriches telemetry with Kubernetes metadata; ubiquitous in customer
  pipelines and a common source of both useful context and cardinality. Quirks around RBAC
  and pod association are worth capturing.
- **Primary source:** ai-context `collector/components/k8sattributesprocessor/`.

### 6. `routing` connector
- **Kind / signals:** connector / traces, metrics, logs
- **Why high-value:** routes telemetry to different pipelines by attribute/OTTL — the
  mechanism behind tiered storage (e.g. send `cardinality_guardian`'s
  `otel.metric.overflow`-tagged stream to cheap storage). Enables several advanced
  cost patterns referenced in the Phase 2 pages.
- **Primary source:** ai-context `collector/components/routingconnector/`.

### 7. `memory_limiter` processor
- **Kind / signals:** processor / traces, metrics, logs
- **Why high-value:** the standard backpressure/safety component; the conventions doc
  already tells authors to place it first in every pipeline. A reference page makes that
  guidance concrete (soft/hard limits, GC behavior, placement).
- **Primary source:** ai-context `collector/components/memorylimiterprocessor/`.

### 8. `loadbalancing` exporter
- **Kind / signals:** exporter / traces (and logs/metrics by routing key)
- **Why high-value:** **prerequisite for scaling `tail_sampling`** — trace-ID-aware routing
  ensures all spans of a trace reach the same downstream collector. The Phase 2
  `tail_sampling` page references it as a hard requirement; this page closes that loop.
- **Primary source:** ai-context (add if missing) / upstream contrib README.

## Explicitly out of scope (for now)

- **`deltatocumulative`** — temporality conversion; not a priority for the cost/quality
  focus of this wave.
- **`span_metrics` connector** — useful, but deferred; not part of this backlog.

## Notes for whoever picks these up

- Confirm each component is present in `telemetrydrops/ai-context`; if not, add it there
  first (that mirror is the current source of truth) or fall back to the upstream contrib
  README via WebFetch.
- Each page needs a runnable `telemetrygen` verification recipe — verify every
  `telemetrygen` flag against the `otel-telemetrygen` skill before publishing; do not
  assert a flag that the skill doesn't document.
- Add an index row to `SKILL.md` and update the frontmatter trigger phrases for any
  component that introduces a distinct user-facing keyword.
