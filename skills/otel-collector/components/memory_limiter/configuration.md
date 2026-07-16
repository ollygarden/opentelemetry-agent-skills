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
| `limit_percentage` | uint32 | `0` | Hard limit as a percentage of total memory (`> 0`, `<= 100`). Used **only when `limit_mib` is unset/0**. On Linux reads the cgroup limit, falling back to `/proc/meminfo`; on non-Linux it reads total *system* memory (rarely a container limit). See [Known quirks](quirks.md). |
| `spike_limit_percentage` | uint32 | 20% of `limit_percentage` | Spike as a percentage of total memory. Must be `< limit_percentage` and `<= 100`. Only used with `limit_percentage`. |
| `min_gc_interval_when_soft_limited` | duration | `10s` | Minimum (floor) time between forced GCs while above the soft limit. Must be `>= min_gc_interval_when_hard_limited`. |
| `min_gc_interval_when_hard_limited` | duration | `0s` | Minimum (floor) time between forced GCs while above the hard limit (`0s` = GC on every over-limit check). Should be `<=` the soft-limited interval. |
| `max_gc_interval_when_soft_limited` | duration | `30s` | Caps the exponential backoff between forced GCs on the soft-limit path. When a forced GC reclaims `< 5%` of heap the interval doubles (from the min floor, or 95% of `check_interval`, whichever is larger) up to this cap, then resets once a GC is effective or heap drops. `0` disables backoff (GC stays at the min). Must be `>= min_gc_interval_when_soft_limited` unless `0`; and `>= max_gc_interval_when_hard_limited` when both caps are set. |
| `max_gc_interval_when_hard_limited` | duration | `30s` | Same backoff mechanism for the hard-limit path. Must be `>= min_gc_interval_when_hard_limited` unless `0`. |

One of `limit_mib` or `limit_percentage` is mandatory; `check_interval` is always mandatory.

## Soft limit vs hard limit

The processor derives two thresholds and acts on the current heap allocation:

| State | Threshold | Action |
|-------|-----------|--------|
| Below soft limit | heap `< limit - spike` | Normal operation; data accepted. |
| Above soft limit | heap `>= limit - spike` | Refuse data (`ErrDataRefused`); opportunistic GC throttled between `min_gc_interval_when_soft_limited` and `max_gc_interval_when_soft_limited`. Logs *"Memory usage is above soft limit. Refusing data."* |
| Above hard limit | heap `>= limit` | Refuse data; aggressive GC throttled between `min_gc_interval_when_hard_limited` and `max_gc_interval_when_hard_limited`. Logs *"Memory usage is above hard limit. Forcing a GC."* |

If a forced GC reclaims `< 5%` of heap (e.g. the memory is held by live references in exporter queues during a downstream outage), the processor treats it as ineffective and **exponentially backs off** the GC interval on that path — doubling up to the `max_gc_interval_*` cap — to avoid burning CPU on futile collections during recovery. The interval resets as soon as a GC frees memory or heap drops on its own.

Refusal is a **non-permanent** error, so receivers retry with backoff and clients can rate-limit. Recovery is automatic: once heap falls below the soft limit it logs *"Memory usage back within limits. Resuming normal operation."* The processor also reports its health via `componentstatus` — a **recoverable-error** status while refusing, and `StatusOK` once back within limits — visible through status-aware extensions (e.g. `healthcheckv2`).

## Validation rules

Startup fails (config error) when:

| Rule | Error message |
|------|---------------|
| `check_interval` not `> 0` | `'check_interval' must be greater than zero` |
| Neither limit set | `'limit_mib' or 'limit_percentage' must be greater than zero` |
| `spike_limit_mib >= limit_mib` | `'spike_limit_mib' must be smaller than 'limit_mib'` |
| `spike_limit_percentage >= limit_percentage` (percentage mode) | `'spike_limit_percentage' must be smaller than 'limit_percentage'` |
| `limit_percentage > 100` or `spike_limit_percentage > 100` | `'limit_percentage' and 'spike_limit_percentage' must be greater than zero and less than or equal to hundred` |
| `min_gc_interval_when_soft_limited < min_gc_interval_when_hard_limited` | `'min_gc_interval_when_soft_limited' should be larger than 'min_gc_interval_when_hard_limited'` |
| `max_gc_interval_when_soft_limited` set but `< min_gc_interval_when_soft_limited` | `'max_gc_interval_when_soft_limited' must be greater than or equal to 'min_gc_interval_when_soft_limited' (or 0 to disable)` |
| `max_gc_interval_when_hard_limited` set but `< min_gc_interval_when_hard_limited` | `'max_gc_interval_when_hard_limited' must be greater than or equal to 'min_gc_interval_when_hard_limited' (or 0 to disable)` |
| both max caps set and `max_gc_interval_when_soft_limited < max_gc_interval_when_hard_limited` | `'max_gc_interval_when_soft_limited' should be larger than or equal to 'max_gc_interval_when_hard_limited' (when both are set)` |

## Pipeline placement

`memory_limiter` must be the **first** processor in every pipeline so it refuses data before it enters allocating processors (`batch`, `transform`, `tail_sampling`) and so backpressure reaches receivers early:

```yaml
# Good — refuse before anything buffers
processors: [memory_limiter, tail_sampling, batch]

# Bad — memory limiting and sampling happen after batching
processors: [batch, memory_limiter, tail_sampling]
```
