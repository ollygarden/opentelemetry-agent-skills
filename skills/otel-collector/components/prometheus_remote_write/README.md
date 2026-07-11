# `prometheus_remote_write` exporter

| | |
|-|-|
| Kind | exporter |
| Type | `prometheus_remote_write` |
| Signals | metrics (Beta) |
| Distributions | core, contrib |
| Go module | `github.com/open-telemetry/opentelemetry-collector-contrib/exporter/prometheusremotewriteexporter` |
| Upstream README | <https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/exporter/prometheusremotewriteexporter> |

> **Renamed in v0.154.0.** The type string was `prometheusremotewrite` (no underscores); it is now `prometheus_remote_write`. The old `prometheusremotewrite` still works as a deprecated alias and logs a deprecation warning — migrate to the snake_case type.

## Description

This is a **push** exporter — it sends OTLP metrics out as Prometheus Remote Write requests to any Remote-Write-compatible backend (Cortex, Mimir, Thanos, Prometheus itself with the receiver enabled, and similar). This is the mirror image of the [`prometheus`](../prometheus_exporter/README.md) exporter, which **hosts** a `/metrics` endpoint to be scraped: this one **initiates** outbound writes. If you are unsure which you want, "push to a remote URL" is this component; "expose for a scraper" is the other.

By default it speaks **Remote Write 1.0** (`protobuf_message: prometheus.WriteRequest`), **requires TLS** (set `tls.insecure: true` for plaintext), and uses **snappy** compression (the only value the protocol — and this exporter's `Validate()` — accepts). Outgoing requests are queued through `remote_write_queue` (this exporter does **not** use the standard `sending_queue`). The full key list, defaults, and validation rules are in [configuration.md](configuration.md).

## Main use-cases

Use when:
- Your metrics backend speaks Prometheus Remote Write (Cortex, Mimir, Thanos, VictoriaMetrics, Prometheus with `--web.enable-remote-write-receiver`).
- You want the collector to **push** metrics out rather than be scraped.
- You are bridging an OTLP pipeline into a Prometheus-protocol world without running a separate scraper.

Avoid when:
- You want to be **scraped** — use the [`prometheus`](../prometheus_exporter/README.md) exporter (hosts `/metrics`).
- The backend speaks OTLP — use the [`otlp_grpc`](../otlp_exporter/README.md) or `otlphttp` exporter.
- You rely heavily on non-cumulative monotonic sums, OTLP histograms, or summaries under RW1 — these are **dropped** by this exporter (see [quirks.md](quirks.md)).

## Related components

- [`prometheus` exporter](../prometheus_exporter/README.md) — the **pull / expose** sibling (hosts `/metrics`); same family, opposite direction. This one pushes.
- [`otlp_grpc`](../otlp_exporter/README.md) / `otlphttp` exporter — OTLP egress when the backend speaks OTLP.
- [`prometheus` receiver](../prometheus/README.md) — scrape **ingress** that pulls Prometheus-format metrics into a pipeline.
- [`resource`](../resource/README.md) / [`transform`](../transform/README.md) — shape resource attributes into labels before export (relates to `resource_to_telemetry_conversion` / `target_info`).
- [`file_storage`](../file_storage/README.md) — only loosely related: the WAL here is the exporter's **own** on-disk buffer, **not** the `file_storage` extension (see [quirks.md](quirks.md)).

## Details

- [Configuration](configuration.md) — `endpoint`, `namespace`, `external_labels`, `add_metric_suffixes`, `translation_strategy`, `send_metadata`, `remote_write_queue`, `resource_to_telemetry_conversion`, `wal`, `target_info`, `disable_scope_info`, `max_batch_size_bytes`, `max_batch_request_parallelism`, `protobuf_message`, the embedded confighttp / TLS / retry / timeout options, and validation rules.
- [Verification](verification.md) — push to a Prometheus container with the remote-write receiver enabled and query the series back, proving `namespace` and `external_labels`. Verified on contrib v0.154.0; config/behavior unchanged through v0.156.0.
- [Advanced use-cases](advanced.md) — the WAL, RW2 / `protobuf_message` and its feature gate, `resource_to_telemetry_conversion` vs `target_info`, `external_labels`, the multi-worker feature gate with `num_consumers` / `max_batch_request_parallelism`, the `RetryOn429` gate, and `translation_strategy` choices.
- [Known quirks](quirks.md) — push-not-pull, the type rename, required `endpoint`, TLS-on-by-default, snappy-only, dropped metric types, the `remote_write_queue` (not `sending_queue`) distinction, the `add_metric_suffixes` deprecation discrepancy, RW2 readiness, and per-signal stability.
