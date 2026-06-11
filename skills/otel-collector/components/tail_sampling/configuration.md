# `tail_sampling`: configuration

## Typical config

```yaml
processors:
  tail_sampling:
    decision_wait: 10s
    num_traces: 50000
    policies:
      # Keep all error traces
      - name: errors
        type: status_code
        status_code:
          status_codes: [ERROR]
      # Sample 10% of everything else
      - name: baseline
        type: probabilistic
        probabilistic:
          sampling_percentage: 10

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [tail_sampling]
      exporters: [otlphttp]
```

## Configuration reference (top-level)

| Key | Type | Default | Notes |
|-----|------|---------|-------|
| `decision_wait` | duration | `30s` | Time from first span arrival before the decision is made. Buffers spans for this long. |
| `decision_wait_after_root_received` | duration | `0s` | Decide this long after the root span arrives; `0` disables (only `decision_wait` is used). |
| `num_traces` | int | `50000` | Max traces kept in memory. When full, the oldest are evicted (before decision) unless `block_on_overflow`. |
| `expected_new_traces_per_sec` | int | `0` | Hint for pre-allocating the trace buffer; `0` disables pre-allocation. |
| `sample_on_first_match` | bool | `false` | Stop evaluating and sample as soon as one policy matches. |
| `block_on_overflow` | bool | `false` | Block ingest instead of dropping the oldest traces when `num_traces` is reached. |
| `drop_pending_traces_on_shutdown` | bool | `false` | On shutdown, drop pending traces instead of deciding with partial data. |
| `maximum_trace_size_bytes` | int | `0` | Traces larger than this are dropped immediately; `0` disables. |
| `decision_cache.sampled_cache_size` | int | `0` | LRU cache of sampled trace IDs so late spans inherit the decision; `0` disables. |
| `decision_cache.non_sampled_cache_size` | int | `0` | LRU cache of dropped trace IDs; `0` disables. |
| `policies` | list | (required) | Sampling policies. At least one is required. |

For the full catalog of policy types and their sub-fields, see [Policy types](policies.md).
