# `probabilistic_sampler`: advanced use-cases

## Choosing a mode

| Goal | Mode |
|------|------|
| A predictable, reproducible ratio across multiple tiers/collectors | `proportional` — uses W3C 56-bit randomness and ignores prior decisions, so a 1-in-N stays 1-in-N everywhere. |
| Enforce a sampling **floor** without re-cutting already-sampled data | `equalizing` — factors in prior decisions and only lowers probability to the target, so a downstream sampler does not double-sample. |
| Sample **logs by a record attribute**, or stay on the legacy seed-based hash | `hash_seed` — the only mode that supports `attribute_source: record`. |

```yaml
processors:
  probabilistic_sampler:
    mode: proportional
    sampling_percentage: 10
```

## Consistent `hash_seed` across a fleet (hash_seed mode)

In `hash_seed` mode the keep/drop decision depends on `hash_seed`. Every collector in the same sampling tier must use the **same** seed, or two collectors will make different decisions for the same TraceID and the effective rate becomes unpredictable. Pin the seed explicitly rather than relying on the `0` default when more than one collector samples the same stream:

```yaml
processors:
  probabilistic_sampler:
    mode: hash_seed
    hash_seed: 22
    sampling_percentage: 25
```

Use a **different** seed at each tier if you intentionally want independent decisions (e.g. cascading samplers); use the same seed when you want them to agree.

## Sampling logs by attribute

Sample logs on a record attribute instead of the TraceID — useful when logs carry a high-cardinality key (such as a request or session ID) that should drive a consistent decision:

```yaml
processors:
  probabilistic_sampler:
    mode: hash_seed
    sampling_percentage: 20
    attribute_source: record
    from_attribute: request.id
```

`attribute_source: record` requires `from_attribute`; with `attribute_source: traceID` (the default) the log's TraceID is hashed.

## `sampling_priority` overrides (logs)

Set `sampling_priority` to the name of an attribute that overrides the per-record decision. When a record carries that attribute, its value is read as a sampling percentage (`0`–`100`) that replaces `sampling_percentage` for that record — `0` never samples, `>= 100` always samples — handy for always keeping flagged records (set the attribute to `100`):

```yaml
processors:
  probabilistic_sampler:
    sampling_percentage: 5
    sampling_priority: log.priority
```

For traces, the equivalent override is the fixed `sampling.priority` attribute (not configurable), and it is binary rather than a percentage: `0` drops the span, any non-zero value keeps it.

## Combining head + tail sampling

Probabilistic head sampling at the **edge** and `tail_sampling` at a **gateway** compose well: cut bulk volume cheaply and statelessly near the source, then keep the survivors that matter (errors, slow traces) at the gateway. Prefer `proportional`/`equalizing` at the edge so the W3C threshold is carried in `tracestate` and the gateway stays consistent rather than re-cutting blindly.

```yaml
# Edge collector — cheap stateless head sampling
processors:
  probabilistic_sampler:
    mode: proportional
    sampling_percentage: 30

# Gateway collector — content-aware tail sampling on the survivors
processors:
  tail_sampling:
    decision_wait: 10s
    policies:
      - name: errors
        type: status_code
        status_code:
          status_codes: [ERROR]
```

## The `fail_closed` trade-off

`fail_closed: true` (default) **drops** items that have sampling-related errors — for example, missing or zero randomness (no usable TraceID / W3C randomness). This is the safe choice: it never lets un-decidable data skew the rate. Set `fail_closed: false` only when you would rather keep such items than lose them, accepting that the effective sampling rate becomes less precise.
