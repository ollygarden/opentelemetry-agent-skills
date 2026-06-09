# `tail_sampling` processor

| | |
|-|-|
| Kind | processor |
| Signals | traces |
| Stability | Beta |
| Distributions | contrib, k8s |
| `type` | `tail_sampling` |
| Go module | `github.com/open-telemetry/opentelemetry-collector-contrib/processor/tailsamplingprocessor` |
| Upstream README | <https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/tailsamplingprocessor> |

## Description

Tail sampling buffers **all spans of a trace** (grouped automatically by `trace_id` — no `groupbytrace` needed), waits a configurable `decision_wait` for the trace to complete, then evaluates a set of policies against the **whole trace** to make a single keep/drop decision. Because the decision is made after the trace is (mostly) assembled, it can act on trace-wide signals: presence of an error on any span, total latency, span count, specific attributes, etc.

This is the opposite of **head sampling** (e.g. `probabilistic_sampler`), which decides at the start of a trace based only on the trace ID — before any span content is known. Head sampling is cheap and stateless; tail sampling is content-aware but stateful, requires buffering, and adds latency.

## Main use-cases

Use it when:
- You want to keep **all error traces** regardless of volume.
- You want to keep **slow** traces (latency-based) while down-sampling fast successful ones.
- You need different sampling rates per service, endpoint, tenant, or attribute.
- You need sophisticated logic — combining conditions (`and`), tiered rate allocation (`composite`), or explicit noise removal (`drop`).

Avoid it when:
- Probabilistic head sampling already meets your needs (cheaper, no buffering, no co-location requirement).
- You need the sampling decision at the edge/SDK before data reaches a collector.
- You **cannot guarantee all spans of a trace reach the same collector instance** (see Known quirks).

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

### Configuration reference (top-level)

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

### Policy types

Each policy has `name` and `type`, plus a config block named after the type.

| `type` | Purpose |
|--------|---------|
| `always_sample` | Sample every trace (catch-all / debugging). No sub-config. |
| `latency` | Sample by total trace duration (earliest start to latest end). |
| `numeric_attribute` | Sample when a numeric span/resource attribute is within a range. |
| `probabilistic` | Sample a fixed percentage of traces via trace-ID hash. |
| `status_code` | Sample if any span has a matching status code. |
| `string_attribute` | Sample by string span/resource attribute value (exact or regex). |
| `rate_limiting` | Sample up to a maximum spans-per-second. |
| `span_count` | Sample by number of spans in the trace. |
| `trace_state` | Sample by W3C TraceState key/value. |
| `boolean_attribute` | Sample by boolean attribute value. |
| `ottl_condition` | Sample when OTTL conditions on spans/span-events match. |
| `and` | Combine sub-policies with AND — all must match to sample. |
| `composite` | Combine sub-policies with priority order and per-policy rate allocation. |
| `drop` | Force-drop traces where all sub-policies match; takes precedence over sampling. |

Key sub-fields for the common policies:

```yaml
# latency: minimum (and optional maximum) trace duration
- name: slow
  type: latency
  latency:
    threshold_ms: 1000
    upper_threshold_ms: 5000   # optional; 0 = no upper bound

# numeric_attribute: span or resource attribute within a range
- name: http-errors
  type: numeric_attribute
  numeric_attribute:
    key: http.response.status_code
    min_value: 400
    max_value: 599

# string_attribute: exact or regex match
- name: by-route
  type: string_attribute
  string_attribute:
    key: http.route
    values: ["/api/users", "^/api/.*"]
    enabled_regex_matching: false   # treat values as regex when true
    invert_match: false             # sample traces that do NOT match

# status_code: OK / ERROR / UNSET
- name: errors
  type: status_code
  status_code:
    status_codes: [ERROR]

# probabilistic: percentage via trace-ID hash
- name: baseline
  type: probabilistic
  probabilistic:
    sampling_percentage: 10.0

# rate_limiting: cap spans per second
- name: cap
  type: rate_limiting
  rate_limiting:
    spans_per_second: 1000
```

#### Decision precedence

All policies are evaluated (unless `sample_on_first_match: true`), then a single decision is chosen: a `drop` decision wins over everything; otherwise any `sample` decision keeps the trace; if no policy matches, the trace is **not** sampled.

> **`invert_match` note:** as of recent contrib releases the legacy "inverted decisions" are disabled — `invert_match: true` now yields a plain sample/not-sample on the negated condition and no longer vetoes other policies. To explicitly suppress traces, use a `drop` policy instead.

## Verification

`tail_sampling` ships in the `contrib` and `k8s` distributions.

Config (`tailsampling-verify.yaml`):

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
processors:
  tail_sampling:
    decision_wait: 5s
    num_traces: 1000
    policies:
      - name: errors-only
        type: status_code
        status_code:
          status_codes: [ERROR]
exporters:
  debug:
    verbosity: detailed
service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [tail_sampling]
      exporters: [debug]
```

Generate traces, some with error status (see the `otel-telemetrygen` skill):

```bash
# OK traces — expect these to be dropped by the errors-only policy
telemetrygen traces --otlp-insecure --otlp-endpoint localhost:4317 --traces 20
# Error traces — expect these to survive
telemetrygen traces --otlp-insecure --otlp-endpoint localhost:4317 --traces 20 --status-code Error
```

The `--status-code Error` flag is confirmed in the `otel-telemetrygen` skill (accepted values: `Unset`/`0`, `Error`/`1`, `Ok`/`2`). It sets the span status to ERROR, which the `status_code` policy matches.

**What proves it worked:** after `decision_wait`, the `debug` exporter shows the error traces and not the OK traces.

## Advanced use-cases

### `and` — require multiple conditions

Sample only traces that are **both** an error **and** slow:

```yaml
- name: slow-errors
  type: and
  and:
    and_sub_policy:
      - name: has-error
        type: status_code
        status_code:
          status_codes: [ERROR]
      - name: is-slow
        type: latency
        latency:
          threshold_ms: 1000
```

`and` accepts all policy types except `composite`, `and`, and `drop`.

### `composite` — tiered sampling with rate allocation

Allocate a total spans-per-second budget across policies in priority order:

```yaml
- name: tiered
  type: composite
  composite:
    max_total_spans_per_second: 1000
    policy_order: [errors, slow, sample-rest]
    composite_sub_policy:
      - name: errors
        type: status_code
        status_code:
          status_codes: [ERROR]
      - name: slow
        type: latency
        latency:
          threshold_ms: 1000
      - name: sample-rest
        type: always_sample
    rate_allocation:
      - policy: errors
        percent: 60
      - policy: slow
        percent: 30
      # sample-rest uses the remaining 10%
```

Each sub-policy gets `(percent/100) * max_total_spans_per_second`; the first matching policy under budget samples the trace. Put an `always_sample` last to use leftover capacity.

### Combining policies and dropping noise

Order policies by intent: force-drop noise first (`drop`), then keep errors, then service-specific rules, then a probabilistic baseline. Any `sample` decision keeps the trace unless a `drop` policy fired.

```yaml
policies:
  - name: drop-health
    type: drop
    drop:
      drop_sub_policy:
        - name: health-routes
          type: string_attribute
          string_attribute:
            key: http.route
            values: ["^/health$", "^/ready$", "^/metrics$"]
            enabled_regex_matching: true
  - name: errors
    type: status_code
    status_code:
      status_codes: [ERROR]
  - name: baseline
    type: probabilistic
    probabilistic:
      sampling_percentage: 1
```

### Scaling horizontally with the `loadbalancing` exporter

A single tail-sampling instance only works while every span of a trace lands on it. To run **more than one** tail-sampling collector you need a front layer of collectors using the `loadbalancing` exporter with `routing_key: traceID`, so all spans of a given trace are routed to the **same** downstream tail-sampling instance:

```yaml
# Layer 1 — routing collectors (any number of replicas)
exporters:
  loadbalancing:
    routing_key: traceID
    protocol:
      otlp:
        tls:
          insecure: true
    resolver:
      dns:
        hostname: tail-sampling-layer
        port: 4317
service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [loadbalancing]

# Layer 2 — tail-sampling collectors (behind the DNS name above)
processors:
  tail_sampling:
    decision_wait: 30s
    num_traces: 100000
    policies: [ ... ]
```

For high-volume systems also size `decision_cache` larger than `num_traces` so late-arriving spans inherit the prior decision instead of forming a new (partial) trace.

## Known quirks

### All spans of a trace must reach the SAME instance

The decision is made per-trace inside one process. If spans of one trace are spread across multiple tail-sampling collectors, each sees only a fragment and makes its own (wrong/partial) decision. Whenever you run more than one tail-sampling collector you **must** put a `loadbalancing` exporter layer in front that routes by `traceID`. A single instance needs no load balancer.

### Memory scales with `num_traces` and trace size

`num_traces` is the in-flight trace buffer: every trace awaiting a decision is held in memory. Rough estimate `num_traces * avg_spans_per_trace * ~1KB/span` (e.g. 50,000 traces × 20 spans ≈ 1 GB). Longer `decision_wait` means more traces resident at once. When the buffer fills, the oldest traces are evicted **before** their decision (surfacing as the `sampling_trace_dropped_too_early` metric) unless `block_on_overflow` is set. Size it as `traces_per_sec * decision_wait_seconds * safety_factor`.

### `decision_wait` adds latency and a fixed window

Sampled traces are only exported after `decision_wait` expires, so this delay is added before any trace leaves the processor. Set it long enough for distributed traces to assemble; too short and late spans miss the window. Spans that arrive **after** the decision are not retroactively folded into it — they either inherit the cached decision (if `decision_cache` is configured) or risk forming a separate/dropped partial trace.

### Stateful, single-process decision

Tail sampling is inherently stateful: the keep/drop choice is computed once, in one process, from whatever spans were buffered at decision time. There is no cross-instance coordination and no re-evaluation of a trace once decided. Place context-enriching processors (e.g. `k8sattributes`) **before** `tail_sampling` in the pipeline, since the processor re-batches spans and downstream context can be lost.

## Related components

- `probabilistic_sampler` — head sampling; stateless, trace-ID based, decided up front. Cheaper but content-blind.
- `loadbalancing` exporter — routes spans by `traceID` so all spans of a trace reach the same tail-sampling instance; required when scaling tail sampling to more than one collector.
- `groupbytrace` — groups spans by trace ID and waits before forwarding. Not needed in front of `tail_sampling`, which already groups by trace ID internally.
