# `interval`: configuration

## Typical config

```yaml
processors:
  interval:

service:
  pipelines:
    metrics:
      receivers: [otlp]
      processors: [memory_limiter, interval]
      exporters: [otlphttp]
```

## Configuration reference

| Key | Type | Default | Notes |
|-----|------|---------|-------|
| `interval` | duration | `60s` | Emission cadence. After each emission, internal state is cleared. |
| `pass_through.gauge` | bool | `false` | If `true`, gauges are forwarded unchanged instead of buffered. |
| `pass_through.summary` | bool | `false` | If `true`, summaries are forwarded unchanged instead of buffered. |

Validation: `interval` must be greater than `0`. A zero or negative value fails startup with `invalid interval value` (`ErrInvalidIntervalValue` in source). `pass_through` booleans have no validation.

> **Doc typo, upstream:** the README's configuration block shows `summary: <boo>l`. It's a typo for `<bool>` — the parameter is a plain boolean.

## What it does to each metric type

| Metric type | Default behavior | Pass-through option |
|-------------|------------------|---------------------|
| Cumulative monotonic sum | Aggregated (only latest value per series emitted per interval) | — |
| Cumulative monotonic histogram | Aggregated | — |
| Cumulative monotonic exponential histogram | Aggregated | — |
| Gauge | Aggregated (lossy — see below) | `pass_through.gauge: true` |
| Summary | Aggregated (lossy — see below) | `pass_through.summary: true` |
| Delta monotonic sum, delta histogram, delta exponential histogram | Accumulated: the interval's points are **added** per series and emitted once, with the earliest start and latest timestamp | — |
| Non-monotonic sum (any temporality) | Passed through unchanged, immediately | — (always pass-through) |

> **Doc drift, upstream:** as of v0.162.0 the upstream README still says all delta metrics pass through unchanged; the code (`processor.go`, PR #50922) aggregates them as above.

## What "lossy" means here

- Monotonic cumulative series: you lose **precision** (intermediate values), not totals — the final cumulative value still represents the full count.
- Delta series: per-point granularity is merged into one delta per interval; the total is preserved.
- Gauges and summaries: actual **data loss**. A value that rose and fell back inside the interval is reduced to whatever the last reading happened to be. If the up-and-down matters (latency spikes, queue depth bursts), set `pass_through.gauge: true` / `pass_through.summary: true` to keep them flowing as-is.

## Behavior example

Monotonic sums arriving into the processor:

| Time | Metric | Temporality | Attributes | Value |
|------|--------|-------------|------------|------:|
| 0 | `test_metric` | Cumulative | `labelA: foo` | 4.0 |
| 2 | `test_metric` | Cumulative | `labelA: bar` | 3.1 |
| 4 | `other_metric` | Delta | `fruitType: orange` | 77.4 |
| 6 | `test_metric` | Cumulative | `labelA: foo` | 8.2 |
| 8 | `test_metric` | Cumulative | `labelA: foo` | 12.8 |
| 10 | `test_metric` | Cumulative | `labelA: bar` | 6.4 |

At the next interval boundary, the processor forwards the latest value of each cumulative series and the sum of each delta series (here a single point):

| Time | Metric | Temporality | Attributes | Value |
|------|--------|-------------|------------|------:|
| 4 | `other_metric` | Delta | `fruitType: orange` | 77.4 |
| 8 | `test_metric` | Cumulative | `labelA: foo` | 12.8 |
| 10 | `test_metric` | Cumulative | `labelA: bar` | 6.4 |

Then state is cleared.
