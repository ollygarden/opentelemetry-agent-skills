# `otlp` receiver

| | |
|-|-|
| Kind | receiver |
| Type | `otlp` |
| Signals | traces (Stable), metrics (Stable), logs (Stable), profiles (Development) |
| Distributions | core, contrib, k8s, otlp |
| Go module | `go.opentelemetry.io/collector/receiver/otlpreceiver` |
| Upstream README | <https://github.com/open-telemetry/opentelemetry-collector/tree/main/receiver/otlpreceiver> |

## Description

Receives telemetry in the **OpenTelemetry Protocol (OTLP)** over gRPC and/or HTTP — the native wire format of the OpenTelemetry SDKs, the Collector's own `otlp` exporter, and most other OTLP-speaking agents. It is the default, canonical ingress for a Collector pipeline and supports traces, metrics, and logs at **Stable** stability (profiles are **Development**). The single config block is `protocols:`, holding optional `grpc:` and `http:` sub-blocks; you must enable at least one. The gRPC side listens on `4317`, the HTTP side on `4318` (accepting both OTLP/protobuf and OTLP/JSON on the same endpoint).

The most important operational fact is the **default endpoint**: each protocol binds to `localhost` (`localhost:4317` / `localhost:4318`), not `0.0.0.0`. That is a deliberate security default — inside a container or Kubernetes pod, traffic arrives on a non-loopback interface, so a `localhost`-bound receiver silently accepts nothing from outside. To receive external traffic you must set `endpoint: 0.0.0.0:4317` (and `:4318`). This is the single most common "the receiver gets no data in Docker/k8s" gotcha — see [quirks.md](quirks.md).

## Main use-cases

Use when:
- You need to ingest OTLP from OpenTelemetry SDKs, the OTel Collector `otlp` exporter, or any OTLP-native agent — the default and recommended Collector ingress.
- You want one receiver to handle traces, metrics, and logs over gRPC, HTTP/protobuf, and HTTP/JSON.
- You front a tier that needs per-request metadata (header-based auth, metadata routing, metadata-keyed load balancing) — set `include_metadata: true`.

Avoid when:
- The source speaks a non-OTLP protocol (Prometheus scrape, Jaeger, Zipkin, Kafka, syslog, filelog, …) — use the matching receiver instead.
- You only need OTLP over HTTP and want to be explicit — still this receiver, just configure only the `http:` sub-block.

## Related components

- [`otlp` exporter](../otlp_exporter/README.md) — the sending counterpart; a Collector-to-Collector hop is this receiver on one end and the `otlp_grpc`/`otlp` exporter on the other.
- [`memory_limiter`](../memory_limiter/README.md) — belongs **first** in every pipeline fed by this receiver, to apply backpressure before buffers grow.
- [`routing`](../routing/README.md) — to route by request metadata, this receiver must run with `include_metadata: true` so the headers survive into the pipeline.
- [`load_balancing`](../load_balancing/README.md) — its `routing_key: attributes` over client-metadata keys likewise needs `include_metadata: true` on the upstream `otlp` receiver.

## Details

- [Configuration](configuration.md) — the `protocols.grpc` and `protocols.http` sub-blocks with every key, default, the URL-path overrides, and the "at least one protocol" validation rule.
- [Verification](verification.md) — telemetrygen recipe sending OTLP/gRPC (and OTLP/HTTP) into a single collector with a `debug` exporter, binding `0.0.0.0` so containerized traffic is accepted.
- [Advanced use-cases](advanced.md) — enabling a single protocol, `include_metadata` for auth/routing/load_balancing, CORS for browser OTLP, mTLS, large-payload tuning, and named instances.
- [Known quirks](quirks.md) — the `localhost` vs `0.0.0.0` default (top item), the "at least one protocol" error, the 4317/4318 ports, OTLP/JSON on the HTTP port, `include_metadata` default, and per-signal stability.
