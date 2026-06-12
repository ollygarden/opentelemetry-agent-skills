# `memory_limiter`: configuration

## Typical config

```yaml
processors:
  memory_limiter:
    check_interval: 1s
    limit_mib: 4000
    spike_limit_mib: 800

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [memory_limiter, batch]
      exporters: [otlp]
```

Hard limit 4000 MiB, soft limit 3200 MiB, checked every second. `memory_limiter` is first in the processor list (see [Pipeline placement](#pipeline-placement)).

## Configuration reference

| Key | Type | Default | Meaning |
|-----|------|---------|---------|
| `check_interval` | duration | `0s` (**required**, must be `> 0`) | Time between memory measurements. `1s` is recommended; `500ms` for spiky traffic. |
| `limit_mib` | uint32 | `0` | Hard limit, in MiB of **heap**. Required unless `limit_percentage` is set; **takes precedence** over `limit_percentage` if both are set. |
| `spike_limit_mib` | uint32 | 20% of `limit_mib` | Expected max heap increase between checks. Must be `< limit_mib`. Soft limit = `limit_mib - spike_limit_mib`. |
| `limit_percentage` | uint32 | `0` | Hard limit as a percentage of total memory (`> 0`, `<= 100`). Used **only when `limit_mib` is unset/0**. Needs Linux cgroups, else falls back to `/proc/meminfo`. |
| `spike_limit_percentage` | uint32 | 20% of `limit_percentage` | Spike as a percentage of total memory. Must be `< limit_percentage` and `<= 100`. Only used with `limit_percentage`. |
| `min_gc_interval_when_soft_limited` | duration | `10s` | Minimum time between forced GCs while above the soft limit. Must be `>= min_gc_interval_when_hard_limited`. |
| `min_gc_interval_when_hard_limited` | duration | `0s` | Minimum time between forced GCs while above the hard limit (`0s` = GC on every over-limit check). Should be `<=` the soft-limited interval. |

One of `limit_mib` or `limit_percentage` is mandatory; `check_interval` is always mandatory.

## Soft limit vs hard limit

The processor derives two thresholds and acts on the current heap allocation:

| State | Threshold | Action |
|-------|-----------|--------|
| Below soft limit | heap `< limit - spike` | Normal operation; data accepted. |
| Above soft limit | heap `>= limit - spike` | Refuse data (`ErrDataRefused`); opportunistic GC, throttled by `min_gc_interval_when_soft_limited`. Logs *"Memory usage is above soft limit. Refusing data."* |
| Above hard limit | heap `>= limit` | Refuse data; aggressive GC, throttled by `min_gc_interval_when_hard_limited`. Logs *"Memory usage is above hard limit. Forcing a GC."* |

Refusal is a **non-permanent** error, so receivers retry with backoff and clients can rate-limit. Recovery is automatic: once heap falls below the soft limit it logs *"Memory usage back within limits. Resuming normal operation."*

## Validation rules

Startup fails (config error) when:

| Rule | Error message |
|------|---------------|
| `check_interval` not `> 0` | `'check_interval' must be greater than zero` |
| Neither limit set | `'limit_mib' or 'limit_percentage' must be greater than zero` |
| `spike_limit_mib >= limit_mib` | `'spike_limit_mib' must be smaller than 'limit_mib'` |
| `min_gc_interval_when_soft_limited < min_gc_interval_when_hard_limited` | `'min_gc_interval_when_soft_limited' should be larger than 'min_gc_interval_when_hard_limited'` |

## Pipeline placement

`memory_limiter` must be the **first** processor in every pipeline so it refuses data before it enters allocating processors (`batch`, `transform`, `tail_sampling`) and so backpressure reaches receivers early:

```yaml
# Good — refuse before anything buffers
processors: [memory_limiter, batch, tail_sampling]

# Bad — data is already batched and processed before refusal
processors: [batch, tail_sampling, memory_limiter]
```
