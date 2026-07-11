# `tail_sampling`: policy types

Each policy has `name` and `type`, plus a config block named after the type.

| `type` | Purpose |
|--------|---------|
| `always_sample` | Sample every trace (catch-all / debugging). No sub-config. |
| `latency` | Sample by total trace duration (earliest start to latest end). |
| `numeric_attribute` | Sample when a numeric span/resource attribute is within a range. |
| `probabilistic` | Sample a fixed percentage of traces via trace-ID hash. |
| `status_code` | Sample if any span has a matching status code. |
| `string_attribute` | Sample by string span/resource attribute value (exact or regex). |
| `rate_limiting` | Sample up to a maximum spans-per-second via a token bucket; `spans_per_second` (required) + `burst_capacity` (optional, default 2× the rate). A trace with more spans than the burst never passes. |
| `bytes_limiting` | Sample up to a maximum bytes-per-second via a token bucket; `bytes_per_second` (required) + `burst_capacity` (optional, default 2× the rate). |
| `span_count` | Sample by number of spans in the trace. |
| `trace_state` | Sample by W3C TraceState key/value. |
| `trace_flags` | Sample if the W3C `sampled` trace flag is set on any span in the trace. No sub-config. |
| `boolean_attribute` | Sample by boolean attribute value. |
| `ottl_condition` | Sample when OTTL conditions on spans/span-events match (`span:` and `spanevent:` lists, plus `error_mode`). Prefer path-qualified context (e.g. `span.attributes[...]`, `resource.attributes[...]`, `spanevent.name`) over bare `attributes[...]` to avoid future breaking changes. |
| `and` | Combine sub-policies with AND — all must match to sample. |
| `composite` | Combine sub-policies with priority order and per-policy rate allocation. |
| `not` | Invert the decision of a single wrapped sub-policy via `not_sub_policy`. |
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

# rate_limiting: cap spans per second (token bucket)
- name: cap
  type: rate_limiting
  rate_limiting:
    spans_per_second: 1000
    burst_capacity: 2000   # optional; default 2× spans_per_second
```

## Decision precedence

All policies are evaluated (unless `sample_on_first_match: true`), then a single decision is chosen: a `drop` decision wins over everything; otherwise any `sample` decision keeps the trace; if no policy matches, the trace is **not** sampled.

> **`invert_match` note:** the legacy "inverted decisions" behavior is gone — the `disableinvertdecisions` feature gate was stabilized and then removed, so as of v0.156.0 `invert_match: true` always yields a plain sample/not-sample on the negated condition and no longer vetoes other policies. `invert_match` still exists on the `numeric_attribute`, `string_attribute`, and `boolean_attribute` policies. To explicitly suppress traces, use a `drop` policy; to sample on the opposite of a wrapped policy's decision, use a `not` policy instead.

The `and`, `composite`, `not`, and `drop` policy types compose other policies; see [Advanced use-cases](advanced.md) for worked examples.
