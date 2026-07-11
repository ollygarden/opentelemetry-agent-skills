# `load_balancing` exporter

| | |
|-|-|
| Kind | exporter |
| Type | `load_balancing` |
| Signals | traces (Beta), logs (Beta), metrics (Development) |
| Distributions | contrib, k8s |
| Go module | `github.com/open-telemetry/opentelemetry-collector-contrib/exporter/loadbalancingexporter` |
| Upstream README | <https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/exporter/loadbalancingexporter> |
| Rename | type renamed from `loadbalancing` to `load_balancing` in **v0.153.0**; the old name is preserved as a deprecated alias. |

## Description

Distributes telemetry across a set of downstream Collector backends so that all records sharing a **routing key** (the trace ID, the `service.name`, etc.) always land on the **same** backend. It wraps a nested `otlp` exporter — one sub-exporter per backend — and picks the target with a **consistent-hash ring** keyed on the routing value. The backend set comes from a `resolver` (`static`, `dns`, `k8s`, or `aws_cloud_map`); when the resolver reports a change, the ring is rebuilt and only ~`R/N` keys (R = routes, N = backends) move, so most traffic keeps its backend.

The point is **affinity, not raw spreading**: stateful downstream processors — `tail_sampling` (needs every span of a trace) and `span_metrics` (needs every span of a service) — only work correctly if a whole trace, or a whole service, reaches one instance. `load_balancing` is the standard way to scale those processors horizontally. Because the ring is deterministic, every front-end Collector running this exporter with the same resolver output computes the **same** key→backend mapping, so you can run several identical front-ends without a coordinator. The routing key is signal-specific (and, for logs, version-specific) — see [Configuration](configuration.md). This page targets **v0.156.0**; the type only exists as `load_balancing` from v0.153.0 onward.

## Main use-cases

Use when:
- You run `tail_sampling` (or `groupbytrace`) on more than one downstream Collector and need every span of a trace to reach the same instance.
- You generate span/service metrics (`span_metrics`, `service_graph`) downstream and need a service's spans pinned to one instance to avoid split/duplicated series.
- You want to fan telemetry across a horizontally-scaled, dynamically-discovered pool of backend Collectors (DNS/headless-service/CloudMap) with stable affinity.

Avoid when:
- You just need plain round-robin/failover to a couple of static endpoints with no affinity requirement — a single `otlp` exporter (optionally with multiple DNS-resolved targets) is simpler.
- You need to route by attribute to **different** pipelines/backends by rule — that is `routing` (a connector), not hash-based distribution.
- Your only goal is throughput batching/retry to one backend — that is the `otlp` exporter's `sending_queue`/`retry_on_failure`.

## Related components

- `tail_sampling` — the primary reason this exporter exists; place `load_balancing` in the tier **in front of** the tail-sampling Collectors so each trace is whole when it arrives.
- `routing` — rule-based routing to named pipelines (a connector); use it for "send tenant X here", not for hash-based affinity.
- `otlp` exporter — the per-backend transport `load_balancing` wraps under `protocol.otlp`; its queue/retry/timeout semantics apply per backend.
- `groupbytrace` / `span_metrics` — downstream processors that depend on the per-trace / per-service affinity this exporter provides.

## Details

- [Configuration](configuration.md) — full config tables for `routing_key` (and its per-signal support), `routing_attributes`, `protocol.otlp`, the four resolver types with every sub-key and default, and the load-balancer-level queue/retry/timeout.
- [Verification](verification.md) — telemetrygen recipe with a `static` resolver fanning four services across two backend Collectors, proving per-service affinity and stable distribution.
- [Advanced use-cases](advanced.md) — the tail_sampling scaling topology, DNS/k8s/aws_cloud_map dynamic discovery, the two-level resiliency model, and running multiple deterministic front-ends.
- [Known quirks](quirks.md) — `protocol.otlp.endpoint` must be unset, logs ignored `routing_key` before v0.154.0, traces reject `resource`/`metric`/`streamID`, resolver count validation, static-resolver data-loss window, and stability caveats.
