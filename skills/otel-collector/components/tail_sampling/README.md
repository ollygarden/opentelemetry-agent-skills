# `tail_sampling` processor

| | |
|-|-|
| Kind | processor |
| Type | `tail_sampling` |
| Signals | traces |
| Stability | Beta |
| Distributions | contrib, k8s |
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
- You **cannot guarantee all spans of a trace reach the same collector instance** (see [Known quirks](quirks.md)).

## Related components

- `probabilistic_sampler` — head sampling; stateless, trace-ID based, decided up front. Cheaper but content-blind.
- `loadbalancing` exporter — routes spans by `traceID` so all spans of a trace reach the same tail-sampling instance; required when scaling tail sampling to more than one collector.
- `groupbytrace` — groups spans by trace ID and waits before forwarding. Not needed in front of `tail_sampling`, which already groups by trace ID internally.

## Details

- [Configuration](configuration.md) — top-level config keys (`decision_wait`, `num_traces`, `decision_cache`, …), defaults, and the minimal pipeline. Open when wiring up the processor or checking a key/default.
- [Policy types](policies.md) — the full catalog of all ~17 policy types and their sub-fields. Open when choosing or configuring a sampling policy.
- [Verification](verification.md) — telemetrygen recipe to confirm sampling works. Open when you want to prove the config end-to-end.
- [Advanced use-cases](advanced.md) — `and`/`composite`/`drop` combinations and scaling out with `loadbalancing`. Open when building multi-condition policies or running more than one instance.
- [Known quirks](quirks.md) — same-instance/loadbalancing requirement, memory model, `decision_wait` latency, late spans, statefulness. Open when sizing memory or debugging missing/partial traces.
