# `otlp_grpc` exporter

| | |
|-|-|
| Kind | exporter |
| Type | `otlp_grpc` (deprecated alias `otlp`) |
| Signals | traces (Stable), metrics (Stable), logs (Stable), profiles (Alpha) |
| Distributions | core, contrib, k8s, otlp |
| Go module | `go.opentelemetry.io/collector/exporter/otlpexporter` |
| Upstream README | <https://github.com/open-telemetry/opentelemetry-collector/tree/main/exporter/otlpexporter> |
| Rename | type renamed from `otlp` to `otlp_grpc` in core **v1.50.0** (≈ contrib v0.144.0); the old name `otlp` is preserved as a deprecated alias and still works, emitting a deprecation warning. |

## Description

Sends telemetry in **OTLP over gRPC** to a downstream OTLP endpoint — another Collector's `otlp` receiver, or any OTLP/gRPC backend. It is the standard egress for a pipeline and supports traces, metrics, and logs at **Stable** stability (profiles are **Alpha**). The canonical type is **`otlp_grpc`**, renamed from `otlp` in core v1.50.0 to disambiguate it from the separate `otlphttp` exporter (OTLP over HTTP). The old name **`otlp` still works** as a deprecated alias — and is what the vast majority of existing configs use — so both `exporters: { otlp: … }` and `exporters: { otlp_grpc: … }` configure this component. Only `endpoint` is required; gRPC requests are **gzip-compressed by default**.

The exporter's `sending_queue` and `retry_on_failure` are enabled by default; queue **batching is
opt-in** ([configuration.md](configuration.md#sending_queue)). The separate `batch` processor
remains Beta and supported for pipeline-level batching; see [quirks.md](quirks.md).

## Main use-cases

Use when:
- You send telemetry from one Collector to another (a gateway hop, an agent→gateway tier) over OTLP/gRPC.
- You export to an OTLP/gRPC-native backend or vendor endpoint.
- You want built-in queuing and retry, with optional exporter-side batching.

Avoid when:
- The backend only speaks OTLP over **HTTP** — use the `otlphttp` exporter instead.
- You need trace-ID / service affinity across a pool of backends (to scale `tail_sampling`/`span_metrics`) — use [`load_balancing`](../load_balancing/README.md), which wraps this exporter per backend.
- The destination isn't OTLP at all (Prometheus remote-write, Kafka, a vendor's proprietary protocol) — use that exporter.

## Related components

- [`otlp` receiver](../otlp/README.md) — the receiving counterpart; a Collector-to-Collector hop is this exporter on one end and the `otlp` receiver on the other.
- `otlphttp` exporter — the sibling that speaks OTLP over **HTTP** instead of gRPC; pick it when the backend has no gRPC endpoint.
- [`load_balancing`](../load_balancing/README.md) — wraps a per-backend `otlp` exporter under `protocol.otlp`; its queue/retry defaults and TLS settings are this exporter's.
- [`memory_limiter`](../memory_limiter/README.md) — the back-pressure guard at the **front** of the pipeline this exporter terminates.
- The **`batch` processor** — still supported for pipeline-level batching; distinct from the exporter's opt-in `sending_queue.batch`.

## Details

- [Configuration](configuration.md) — `endpoint`, `compression`, `tls`, `headers`, `timeout`, and this exporter's `retry_on_failure` and `sending_queue` (+ `batch`) defaults; key semantics and queue validation are in the [shared exporterhelper reference](../../references/exporterhelper.md).
- [Verification](verification.md) — a two-collector chain (A exports to B) proving spans sent to A arrive in B's `debug` log.
- [Advanced use-cases](advanced.md) — a queue-tuning example, a persistent queue via `storage`, mTLS to a secure backend, named instances, and the relationship to `load_balancing`.
- [Known quirks](quirks.md) — the `otlp`→`otlp_grpc` rename and `otlphttp` distinction (top item), exporter batching vs. the `batch` processor, default gzip, `insecure: true` for plaintext, the pre-v0.157.0 `wait_for_ready` bug, and per-signal stability.
