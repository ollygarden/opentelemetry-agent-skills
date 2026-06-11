# `cardinality_guardian` processor

| | |
|-|-|
| Kind | processor |
| Type | `cardinality_guardian` |
| Signals | metrics |
| Stability | Development (contrib v0.152.0) |
| Distributions | none — build into a custom collector via OCB |
| Go module | `github.com/open-telemetry/opentelemetry-collector-contrib/processor/cardinalityguardianprocessor` |
| Upstream README | <https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/cardinalityguardianprocessor> |

## Description

Catches metric cardinality explosions by detecting abnormal per-`(metric, label key)` growth and stripping or tagging only the offending label, leaving the rest of the data point intact. A single runaway label — a request ID, a raw URL, a user-supplied tag — can multiply a metric's time-series count and blow up TSDB cost and query latency. When a label's new unique values in the current epoch exceed a threshold, the processor either **strips** the offending label (`tag_only: false`, the default) or **tags** the data point with `otel.metric.overflow: true` and otherwise leaves it untouched (`tag_only: true`).

Because only the bad label is removed — not the whole data point — dashboards and alerts that rely on other labels keep working while the explosion is neutralized. The processor measures cardinality *growth* per epoch, not absolute cardinality, so a large-but-stable label space is never enforced (see [Configuration](configuration.md) for the mechanism).

## Main use-cases

Use it when:

- A metrics pipeline feeds a per-series-priced backend (managed Prometheus, Datadog) where one bad deploy can balloon cost.
- You want a safety net that degrades gracefully (drop a label) instead of dropping whole metrics.
- You want to route overflow series to cheap storage rather than discard them (`tag_only: true` plus a routing connector).

Avoid it when:

- You need deterministic, explicit label add/drop — use `metricstransform` or `transform` instead.
- Your pipeline carries cumulative Sums/Histograms and you cannot run `tag_only: true` — stripping a label collapses series identity and silently corrupts `rate()`/`increase()` (see Known quirks).
- You need to bound *absolute* cardinality; this processor only reacts to per-epoch growth, so a one-time large-but-stable label space is never enforced.

## Related components

- [`metricstransform`](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/metricstransformprocessor) — deterministic label add/drop and aggregation across attribute combinations.
- [`transform`](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/transformprocessor) — OTTL-based per-metric attribute manipulation and filtering.
- [`filter`](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/filterprocessor) — drop whole metrics/data points by OTTL condition.
- [`routingconnector`](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/connector/routingconnector) — route the `otel.metric.overflow`-tagged stream to a separate pipeline when running in `tag_only` mode.

## Details

- [Configuration](configuration.md) — config keys, defaults, validation, how detection works, and internal telemetry metrics.
- [Verification](verification.md) — build-via-OCB note plus a recipe to confirm growth-based enforcement.
- [Advanced use-cases](advanced.md) — tag-and-route to cheap storage, per-metric overrides, memory bounding, reading internal telemetry.
- [Known quirks](quirks.md) — the single-writer hazard in enforcement mode, Development stability, OCB-only, troubleshooting.
