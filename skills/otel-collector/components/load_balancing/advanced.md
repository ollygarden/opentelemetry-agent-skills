# `load_balancing`: advanced use-cases

## Scaling `tail_sampling` (the canonical topology)

`tail_sampling` must see **every span of a trace** to make one keep/drop decision, so it can't be sharded by chance. Put a load-balancing tier in front:

```
agents ──▶ [ LB Collectors: otlp recv → load_balancing(routing_key: traceID) ]
                                  │  consistent hash on trace ID
                                  ▼
           [ sampling Collectors: otlp recv → tail_sampling → otlp/backend ]
```

- The **LB tier** does no sampling — it only fans traces out by trace ID.
- Each **sampling Collector** receives whole traces, so its policies are correct.
- Scale the sampling tier up/down; the resolver picks up the change and the ring rebalances with minimal churn.

The same shape applies to `span_metrics`/`service_graph` downstream, but key on `service` (or `resource`) so all of a service's spans — not each trace — converge on one instance, keeping its RED/series math whole.

## Dynamic discovery

Pick the resolver that matches how backends come and go:

- **`static`** — fixed lab/edge setups. The ring only changes on restart; a dead backend stays in the ring (data to it is lost until you fix the list).
- **`dns`** against a headless Service — backends scale, DNS A-records change, re-resolved every `interval`. Subject to DNS caching delay.
- **`k8s`** EndpointSlice watch — reacts to pod churn faster than DNS and skips DNS caching; needs RBAC (see [configuration.md](configuration.md)). Use `return_hostnames: true` only with a headless Service backing a StatefulSet (stable pod identities).
- **`aws_cloud_map`** — ECS/EC2 fleets registered in Cloud Map; filter by `health_status` and remember the 100-host cap.

## Two-level resiliency model

There are **two** independent queue/retry/timeout layers:

1. **Per-backend** (`protocol.otlp.sending_queue` / `retry_on_failure` / `timeout`) — handles a single backend's transient failures (network blip, brief unavailability). Each sub-exporter has its own.
2. **Load-balancer level** (top-level `sending_queue` / `retry_on_failure` / `timeout`) — when a sub-exporter exhausts its own retries and bounces the data back, the LB level can re-queue and re-dispatch, which on the **next** resolver update may pick a healthy backend. Valuable in elastic environments (K8s scale events) where a backend disappears entirely.

Flow: sub-exporter exhausts → data returns to the LB → LB queue/retry (if enabled) re-dispatches → if all exhausted, data can be dropped. With a `static` resolver and **all** backends down, loss is unavoidable — there is nowhere to re-route.

> Persistent (file-storage-backed) queues are not supported at the per-backend sub-exporter level — the storage config is shared. Put persistence on the LB-level `sending_queue` instead.

## Multiple deterministic front-ends

The hash ring is a pure function of the routing key and the **resolved backend set**. Two LB Collectors with the same `resolver` (hence the same backend list) and same `routing_key` compute the **identical** key→backend mapping. So you can run several LB front-ends behind a plain round-robin L4 load balancer for HA, and a given trace still always reaches the same sampling backend regardless of which front-end handled it — no shared state or coordinator needed. Keep their configs identical, or the rings diverge.

## Named instances

Standard `type/name` works, e.g. one tier keyed by trace ID for sampling and another keyed by service for metrics:

```yaml
exporters:
  load_balancing/traces:
    routing_key: traceID
    resolver: { k8s: { service: sampling-collectors.observability } }
    protocol: { otlp: { tls: { insecure: true } } }
  load_balancing/metrics:
    routing_key: service
    resolver: { k8s: { service: spanmetrics-collectors.observability } }
    protocol: { otlp: { tls: { insecure: true } } }
```
