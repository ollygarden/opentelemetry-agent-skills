# `probabilistic_sampler`: configuration

## Typical config

```yaml
processors:
  probabilistic_sampler:
    sampling_percentage: 25

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [probabilistic_sampler]
      exporters: [otlphttp]
```

## Configuration reference

| Key | Type | Default | Meaning |
|-----|------|---------|---------|
| `sampling_percentage` | float32 | (required) | Percentage sampled. `>= 100` keeps all; `0` keeps none. |
| `mode` | string | `hash_seed` | Sampling algorithm: `proportional`, `equalizing`, or `hash_seed`. See [Modes](#modes). |
| `hash_seed` | uint32 | `0` | Seed for the FNV hash. Only used in `hash_seed` mode. All collectors in a tier must share the same value (see [Known quirks](quirks.md)). |
| `fail_closed` | bool | `true` | Reject items with sampling-related errors (e.g. missing or zero randomness). When `false`, such items pass through. |
| `sampling_precision` | int | `4` | Number of hex digits used to encode the sampling threshold (range `1`–`14`). Higher precision is more exact but capped per mode (see [Known quirks](quirks.md)). |
| `attribute_source` | string | `traceID` | **Logs only.** Randomness source: `traceID` or `record`. |
| `from_attribute` | string | `""` | **Logs only.** Log attribute used for sampling when `attribute_source: record`. |
| `sampling_priority` | string | `""` | **Logs only.** Attribute name that, when present on a record, overrides the sampling decision for that record. |

## Modes

| Mode | Randomness | Behavior |
|------|------------|----------|
| `hash_seed` (default) | FNV hash of TraceID (or attribute), 14-bit | Hashes the source value with `hash_seed` and compares against a 14-bit threshold. The only mode that supports sampling logs by a record attribute. |
| `proportional` | W3C 56-bit randomness | Ignores any prior sampling decision and samples a predictable 1-in-N fraction. Best for a consistent ratio across tiers. |
| `equalizing` | W3C 56-bit randomness | Factors in prior sampling decisions and lowers probability to a floor across a pipeline, so a downstream sampler does not re-cut already-sampled data below the target. |

The 56-bit modes use the W3C Trace Context randomness (`tracestate` `ot=rv:`/`th:`), which makes their decisions composable across multiple samplers; `hash_seed` is self-contained and seed-dependent.

## Traces vs logs

| Aspect | Traces | Logs |
|--------|--------|------|
| Randomness source | Always the TraceID | TraceID (`attribute_source: traceID`) or a record attribute (`attribute_source: record` + `from_attribute`) |
| Per-item priority override | Fixed `sampling.priority` attribute | Configurable via `sampling_priority` |
| Threshold encoding (downstream) | W3C `tracestate` (`ot=th:...`) | Log attributes (`sampling.threshold` / `sampling.randomness`) |
| Stability | Beta | Alpha |

`attribute_source`, `from_attribute`, and `sampling_priority` are logs-only and have no effect on traces.
