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

Catches metric cardinality explosions by detecting abnormal per-`(metric, label key)` growth and stripping or tagging only the offending label, leaving the rest of the data point intact. A single runaway label — a request ID, a raw URL, a user-supplied tag — can multiply a metric's time-series count and blow up TSDB cost and query latency. When a label's new unique values in the current epoch exceed a threshold, the processor either:

- **strips** the offending label from the data point (`tag_only: false`, the default), or
- **tags** the data point with `otel.metric.overflow: true` and otherwise leaves it untouched (`tag_only: true`).

Because only the bad label is removed — not the whole data point — dashboards and alerts that rely on other labels keep working while the explosion is neutralized.

### How detection works

The processor measures cardinality **growth**, not absolute cardinality. For each `(metric, label key)` it keeps a HyperLogLog sketch and, at the end of every epoch, promotes the current sketch to "previous" and starts fresh. The enforcement check compares the current epoch's unique-value estimate against the previous epoch's — only the **delta** is tested against the threshold. A metric that has already reached a large but stable label space is left alone; only metrics whose label space is actively *exploding* trigger enforcement.

`MutatesData: true` — the processor takes ownership of the incoming `pmetric.Metrics`.

## Main use-cases

Use it when:

- A metrics pipeline feeds a per-series-priced backend (managed Prometheus, Datadog) where one bad deploy can balloon cost.
- You want a safety net that degrades gracefully (drop a label) instead of dropping whole metrics.
- You want to route overflow series to cheap storage rather than discard them (`tag_only: true` plus a routing connector).

Avoid it when:

- You need deterministic, explicit label add/drop — use `metricstransform` or `transform` instead.
- Your pipeline carries cumulative Sums/Histograms and you cannot run `tag_only: true` — stripping a label collapses series identity and silently corrupts `rate()`/`increase()` (see Known quirks).
- You need to bound *absolute* cardinality; this processor only reacts to per-epoch growth, so a one-time large-but-stable label space is never enforced.

## Typical config

```yaml
processors:
  cardinality_guardian:

service:
  pipelines:
    metrics:
      receivers: [otlp]
      processors: [memory_limiter, cardinality_guardian]
      exporters: [prometheusremotewrite]
```

### Configuration reference

| Key | Type | Default | Notes |
|-----|------|---------|-------|
| `max_cardinality_delta_per_epoch` | int | `100` | Max new unique values per `(metric, label key)` per epoch. Must be `> 0`. Global threshold; overridable per metric. |
| `epoch_duration_seconds` | int | `300` | Epoch (sliding window) rotation interval. Must be `>= 10`. Shorter = more sensitive but noisier; longer = smoother but slower to react. |
| `tag_only` | bool | `false` | `false` strips the offending label; `true` keeps it and adds `otel.metric.overflow: true`. `true` is the production-safe mode (see Known quirks). |
| `never_drop_labels` | []string | `[http.status_code, region]` | Label keys never stripped or tagged regardless of growth. O(1) exempt-list, read only at startup. |
| `metric_overrides` | map[string]int | `nil` | Per-metric delta overrides. Each value must be `> 0`; empty metric names are rejected at validation. Unspecified metrics use the global default. |
| `top_offenders_count` | int | `10` | Number of highest-delta `(metric, label)` pairs reported via gauge. `0` disables. Range `0`–`500`. Snapshot computed once per epoch (no hot-path cost). |
| `max_tracker_count` | int | `0` | Max concurrent trackers. `0` = unlimited. Range `0`–`10,000,000`. Once reached, new pairs pass through untracked and increment `..._trackers_rejected`. |
| `estimated_cost_per_metric_month` | float64 | `0.05` | Dollar value per series prevented, for the savings counter. `0` disables. Must be `>= 0`. Does not affect enforcement. |
| `drop_log_max_per_epoch` | int | `10` | Cap on per-epoch "dropping attribute" warning logs. `0` = no cap (logs every drop, not recommended at scale). Must be `>= 0`. |

Validation runs at pipeline construction; any out-of-range field prevents startup.

## Verification

`cardinality_guardian` is not bundled in any distribution — build a custom collector that includes `processor/cardinalityguardianprocessor` via the OpenTelemetry Collector Builder (OCB) first.

Config (`cardguard-verify.yaml`):

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
processors:
  cardinality_guardian:
    max_cardinality_delta_per_epoch: 5
    epoch_duration_seconds: 10
    tag_only: true
exporters:
  debug:
    verbosity: detailed
service:
  pipelines:
    metrics:
      receivers: [otlp]
      processors: [cardinality_guardian]
      exporters: [debug]
```

Emit metrics whose attribute set explodes — many unique values for one label key within an epoch. **`telemetrygen` cannot do this directly:** its `--telemetry-attributes key="value"` flag sets one fixed value, so every data point carries the *same* label value and no per-epoch growth is ever observed. The experimental `--unique-timeseries` flag varies the timeseries but does not let you target a specific named label key with a growing value space. To exercise the processor you need varying values on one label key, which means either:

- Prepend a `transform`/OTTL `metrics` statement upstream in the same pipeline that rewrites a label to an exploding value (e.g. derive it from a high-cardinality source), so the input shape grows over the epoch; see the `otel-ottl` skill for `set(...)` on metric attributes, or
- Use a small custom emitter (a few lines with the metrics SDK) that increments a counter while setting `runaway="value-N"` with `N` incrementing per export.

A constant-attribute run is still useful as a baseline — it should produce **no** overflow tagging, confirming stable label spaces are left alone:

```bash
telemetrygen metrics --otlp-insecure --otlp-endpoint localhost:4317 \
  --metric-type Sum --rate 50 --duration 30s --telemetry-attributes 'runaway="value-N"'
```

**What proves it worked (`tag_only: true`):** once a label crosses the per-epoch delta, the `debug` exporter shows the offending data points carrying `otel.metric.overflow: true` while other labels remain intact; the internal metric `otelcol_processor_cardinality_labels_stripped` increments. With a constant label value (the `telemetrygen` baseline above) nothing is tagged — that confirms growth, not absolute cardinality, is what triggers enforcement.

## Advanced use-cases

### Tag-and-route to cheap storage (production-safe)

Keep the offending series but split it off to cheap storage with the `routingconnector`, avoiding the single-writer problem:

```yaml
processors:
  cardinality_guardian:
    tag_only: true
connectors:
  routing:
    default_pipelines: [metrics/clean]
    table:
      - condition: 'attributes["otel.metric.overflow"] == true'
        pipelines: [metrics/overflow]
service:
  pipelines:
    metrics/in:
      receivers: [otlp]
      processors: [cardinality_guardian]
      exporters: [routing]
    metrics/clean:
      receivers: [routing]
      exporters: [prometheusremotewrite]
    metrics/overflow:
      receivers: [routing]
      exporters: [file]   # cheap cold storage
```

### Per-metric overrides

A legitimately wide metric (high but expected cardinality) can be given a looser threshold so it is not enforced at the global level:

```yaml
processors:
  cardinality_guardian:
    max_cardinality_delta_per_epoch: 100
    metric_overrides:
      http.server.request.duration: 5000
```

### Bounding memory with `max_tracker_count`

Each `(metric, label)` pair holds a HyperLogLog sketch. On a pipeline with a huge metric/label fan-out, set `max_tracker_count` to cap memory. Once the cap is hit, new pairs pass through untracked and `otelcol_processor_cardinality_trackers_rejected` increments — watch that counter to know whether the cap is too low.

### Interpreting the internal telemetry

| Metric | Type | Use |
|--------|------|-----|
| `otelcol_processor_cardinality_trackers_active` | Gauge | Active `(metric, label)` trackers across all shards. |
| `otelcol_processor_cardinality_labels_stripped` | Counter | Labels stripped or tagged. Alert on `rate(...[5m])` for spike detection. |
| `otelcol_processor_cardinality_top_offenders` | Gauge | Top-N highest-delta pairs (`metric_name`, `label_key` attributes) — find the exact pair exploding. |
| `otelcol_processor_cardinality_trackers_rejected` | Counter | New pairs ignored after hitting `max_tracker_count`. |
| `otelcol_processor_cardinality_savings_estimated` | Counter | Estimated dollar value of series prevented from reaching the TSDB. |

## Known quirks

### Single-writer violation in enforcement mode (`tag_only: false`)

This is the headline gotcha. In enforcement mode, stripping a label can collapse multiple data points onto the **same series identity**. Cumulative backends (Prometheus and similar) interpret the overlapping cumulative values as **counter resets**, silently corrupting `rate()` and `increase()` for affected Sum and Histogram metrics — the numbers look plausible but are wrong, with no error surfaced anywhere.

**Fix / recommendation:** run `tag_only: true` plus a routing connector (see Advanced use-cases) in production until a downstream spatial-reaggregation processor exists. Reserve `tag_only: false` for gauges or for pipelines you have verified are safe.

Add the labels your dashboards group by (`service.name`, `environment`, etc.) to `never_drop_labels` so enforcement can never break them.

### Nothing gets stripped despite an obvious explosion

The check is on per-epoch *delta*, not absolute cardinality. If the explosion happened before the processor started (steady state), no delta is observed. Lower `epoch_duration_seconds` or restart to re-baseline. Also confirm the label is not in `never_drop_labels`.

### Development stability

Introduced at **Development** stability in contrib v0.152.0. Configuration and behavior may change without a deprecation period — pin the collector version and re-check defaults on upgrade.

### Not in any distribution

Not bundled in the `contrib` or `k8s` distributions. You must build a custom collector that includes `processor/cardinalityguardianprocessor` via the OpenTelemetry Collector Builder (OCB).

## Related components

- [`metricstransform`](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/metricstransformprocessor) — deterministic label add/drop and aggregation across attribute combinations.
- [`transform`](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/transformprocessor) — OTTL-based per-metric attribute manipulation and filtering.
- [`filter`](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/filterprocessor) — drop whole metrics/data points by OTTL condition.
- [`routingconnector`](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/connector/routingconnector) — route the `otel.metric.overflow`-tagged stream to a separate pipeline when running in `tag_only` mode.
