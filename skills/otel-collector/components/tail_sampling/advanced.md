# `tail_sampling`: advanced use-cases

## `and` — require multiple conditions

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

`and` accepts all policy types except `composite`, `and`, and `drop`. A `not` sub-policy can be nested inside `and_sub_policy` (and inside `drop_sub_policy`), letting you AND a negated condition without a separate top-level policy.

## `composite` — tiered sampling with rate allocation

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

## Combining policies and dropping noise

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

## Scaling horizontally with the `loadbalancing` exporter

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
