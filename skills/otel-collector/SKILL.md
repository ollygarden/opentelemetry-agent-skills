---
name: otel-collector
description: OpenTelemetry Collector component configuration. Use when authoring, reviewing, or debugging Collector YAML for a specific receiver, processor, exporter, connector, or extension — config keys, defaults, validation rules, signal support, stability levels, and component-level gotchas. Triggers on questions about specific components such as `log_dedup` / `logdedup`, `interval` (metric aggregation), `tail_sampling`, `cardinality_guardian`, `drain`, `redaction`, and other Collector components covered in `components/`.
---

# OpenTelemetry Collector

This skill covers the configuration surface of individual OpenTelemetry Collector components. It targets [opentelemetry-collector](https://github.com/open-telemetry/opentelemetry-collector) and [opentelemetry-collector-contrib](https://github.com/open-telemetry/opentelemetry-collector-contrib).

It does **not** cover OTTL expressions (see `otel-ottl`), declarative SDK configuration (`otel-declarative-config`), or end-to-end pipeline design choices. Reach for those skills when the question is about transformation language, SDK setup, or pipeline composition.

## Workflow

1. **Identify the component.** Find the `type` in the user's config or question (`log_dedup`, `interval`, `otlp`, …). Note that several components were renamed to snake_case in v0.150.0–v0.151.0 with deprecated aliases preserved — see [Recent renames](#recent-renames).
2. **Load the component page.** If the component is in the [Component index](#component-index), read `components/<name>.md` for the full config reference, examples, gotchas, and anti-patterns. Do not load pages you don't need.
3. **If the component is not indexed**, say so explicitly and fall back to the upstream README under `processor/<name>/`, `receiver/<name>/`, `exporter/<name>/`, `connector/<name>/`, or `extension/<name>/` in [opentelemetry-collector-contrib](https://github.com/open-telemetry/opentelemetry-collector-contrib). Don't invent config keys from memory — Collector components evolve quickly.
4. **Apply Collector-wide conventions.** Named instances (`type/name`), stability levels, and pipeline placement rules in [Collector-wide conventions](#collector-wide-conventions) apply to every component.
5. **Verify.** Run the component page's **Verification** recipe — `telemetrygen` (see the `otel-telemetrygen` skill) plus a `debug` or `file` exporter — to confirm the component behaves as the docs claim. Alpha- and Development-stability components are common here, and behavior changes between releases. See [Verification harness](#verification-harness) for how to run a recipe end-to-end.

## Component index

Pages live in `components/`. Each page is self-contained: when to use / when not, full config reference, examples, troubleshooting, anti-patterns.

| Type | File | Kind | Signals | Stability | Summary |
|------|------|------|---------|-----------|---------|
| `log_dedup` | `components/log_dedup.md` | processor | logs | Alpha | Deduplicates identical log records over a time window; emits one aggregated log with a count. Renamed from `logdedup` in v0.151.0; alias preserved. |
| `interval` | `components/interval.md` | processor | metrics | Alpha | Buffers cumulative monotonic metrics (and optionally gauges/summaries) and emits the latest value once per interval. Delta and non-monotonic sums pass through unchanged. |
| `cardinality_guardian` | `components/cardinality_guardian.md` | processor | metrics | Development | Catches metric cardinality explosions by detecting abnormal per-label growth and stripping or tagging the offending label. Not in any distribution — build via OCB. |
| `tail_sampling` | `components/tail_sampling.md` | processor | traces | Beta | Buffers whole traces and makes a single keep/drop decision after a wait window via policies. Requires a loadbalancing layer to scale across instances. |
| `drain` | `components/drain.md` | processor | logs | Alpha | Clusters log bodies with the Drain algorithm and annotates each record with a derived template string. |
| `redaction` | `components/redaction.md` | processor | traces, logs, metrics | Beta (traces), Alpha (logs/metrics) | Allow/block-list masking or removal of sensitive attribute keys and values, with hashing and URL/DB sanitizers. |

## Collector-wide conventions

### Named instances

Every component type supports the `type/name` pattern so the same type can be configured more than once. The pipeline references the named form:

```yaml
processors:
  log_dedup/health-checks:
    interval: 30s
    conditions:
      - 'attributes["log.type"] == "health_check"'
  log_dedup/access-logs:
    interval: 10s

service:
  pipelines:
    logs/health:
      processors: [log_dedup/health-checks]
    logs/access:
      processors: [log_dedup/access-logs]
```

### Stability levels

Components publish a stability level per signal. Treat these as load-bearing when recommending production use:

| Level | Use in |
|-------|--------|
| Development | Tests and prototypes only — breaking changes expected. |
| Alpha | Limited, non-critical workloads — config keys can still change. |
| Beta | Production viable — breaking changes rare. |
| Stable | Production — backward compatibility guaranteed. |

Stability now varies per component — and per signal for multi-signal components (e.g. `redaction` is Beta for traces but Alpha for logs/metrics). Don't assume a single level for the indexed set: check each page's header metadata table for the authoritative stability and surface it when the user asks about production readiness.

### Recent renames

Many components were renamed to snake_case in v0.150.0–v0.151.0. The legacy names remain as deprecated aliases — old configs keep working but new configs should use the new names. Check the upstream component README for the exact rename version before flagging a config as broken.

Examples: `logdedup` → `log_dedup`, `hostmetrics` → `host_metrics`, `spanmetrics` → `span_metrics`, `servicegraph` → `service_graph`, `k8sattributes` → `k8s_attributes`, plus several `_log` and `_check` receivers.

### Pipeline placement

Two rules of thumb that apply across components:

- `memory_limiter` belongs first in any processor list, before anything that allocates buffers (`log_dedup`, `transform`, `tail_sampling`, …).
- Batching is now done by the exporter's `sending_queue.batch`, not by a separate `batch` processor. Don't add `batch` to new pipelines.

### Verification harness

Each component page's **Verification** section gives a config, a `telemetrygen` command, and the expected output. To run any of them:

1. Save the YAML to a file and start a collector that bundles the component — for components in the `contrib`/`k8s` distributions, `otelcol-contrib --config <file>.yaml`; for components not in any distribution, build a custom collector with the [OpenTelemetry Collector Builder (OCB)](https://github.com/open-telemetry/opentelemetry-collector/tree/main/cmd/builder) first.
2. Send telemetry with `telemetrygen` (see the `otel-telemetrygen` skill).
3. Watch the `debug` exporter's stdout (or the `file` exporter's output) for the expected result.

The Verification configs are **minimal repros**: they omit `memory_limiter` and other production scaffolding on purpose, to isolate the component under test. Don't copy them verbatim into production.

## Adding a new component to this skill

When extending coverage:

1. **Create `components/<type>.md`.** Component pages carry no frontmatter — only `SKILL.md` has frontmatter.
2. **Follow the canonical 8-section template, in this order:**
   1. **Header metadata table** — kind, signals, per-signal stability, distributions, `type` name, Go module, upstream README link, and a rename note if the component was renamed.
   2. **Description** — what the component does and the mechanism behind it.
   3. **Main use-cases** — "Use when" / "Avoid when".
   4. **Typical config** — minimal working YAML inside a `service.pipelines` block, plus the full config-reference table (key, type, default, validation).
   5. **Verification** — a `telemetrygen` + `debug`/`file` exporter recipe that proves the documented behavior; cross-reference the `otel-telemetrygen` skill. **Verify every `telemetrygen` flag against that skill — never assert a flag that doesn't exist.** If `telemetrygen` can't produce the input the component needs, say so and point to an alternative (OTTL/`transform`, a custom emitter). Keep the config a minimal repro (see [Verification harness](#verification-harness)).
   6. **Advanced use-cases** — named instances, multi-pipeline setups, combinations, and edge configs.
   7. **Known quirks** — gotchas, stability caveats, memory model, a validation-error→fix table, anti-patterns, and troubleshooting.
   8. **Related components** — cross-links to related pages.
3. **Use `components/log_dedup.md` and `components/interval.md` as reference implementations** of this template.
4. **Add a row to the [Component index](#component-index)** above.
5. **Update the description trigger phrases** in this file's frontmatter if the new component introduces a clearly distinct user-facing keyword.
