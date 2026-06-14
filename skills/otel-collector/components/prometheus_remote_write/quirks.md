# `prometheus_remote_write` exporter: known quirks

## It pushes — it does not expose

This exporter **initiates** outbound Remote Write requests to a backend URL. It does **not** host a `/metrics` endpoint to be scraped — that is the separate [`prometheus`](../prometheus_exporter/README.md) exporter. The common mistake is reaching for one when you want the other: "push to a remote URL" is this component; "expose for a scraper" is the other. They are not interchangeable.

## The type was renamed in v0.154.0

The type string changed from `prometheusremotewrite` (no underscores) to `prometheus_remote_write` in **v0.154.0**. The old form still works as a **deprecated alias** and logs a deprecation warning at startup. Use the snake_case `prometheus_remote_write` in new configs.

## Non-cumulative monotonic, histogram, and summary metrics are dropped

Per upstream: **"Non-cumulative monotonic, histogram, and summary OTLP metrics are dropped by this exporter."** This is a Remote Write 1.0 limitation. If your pipeline produces delta sums, OTLP histograms, or summaries and you need them in the backend, convert them upstream (e.g. cumulative-ize deltas) or move to RW2 where applicable — under RW1 they silently disappear.

## TLS is on by default; snappy is the only compression

TLS is **enabled by default** — a plaintext backend needs `tls.insecure: true`, otherwise the connection fails. Compression must be `snappy` (the only value `Validate()` accepts; empty is allowed): the Remote Write protocol requires it, so you cannot switch to gzip/zstd.

## `remote_write_queue`, not `sending_queue`

This exporter does **not** use the standard `exporterhelper` `sending_queue`. Queueing is configured under `remote_write_queue` (`enabled`, `queue_size`, `num_consumers`). Configuring `sending_queue` has no effect here.

| Validation error | Fix |
|------------------|-----|
| `a 0 size queue will drop all the data` | Set `remote_write_queue.queue_size` > 0, or disable the queue (`enabled: false`). |
| `remote write queue size can't be negative` | Use a non-negative `queue_size`. |
| `remote write consumer number can't be negative` | Use a non-negative `num_consumers`. |
| `compression type must be snappy` | Remove the `compression` key, or set it to `snappy`. |
| `max_batch_byte_size must be greater than 0` | Use a non-negative `max_batch_size_bytes` (0 is auto-reset to `3000000`). |
| `max_batch_request_parallelism can't be set to below 1` | Set it to `1` or higher. |
| `remote write v2 is only supported with the feature gate …` | Enable the `enableSendingRW2` gate, or keep `protobuf_message: prometheus.WriteRequest`. |
| `invalid translation_strategy: <v>` | Use one of the four valid enum values. |
| `translation strategy <v> requires Prometheus Remote Write 2.0 (UTF-8 support)` | Switch to an escaping strategy, or enable RW2. |

## `endpoint` ships a placeholder, not a real default

The factory default for `endpoint` is `http://some.url:9411/api/prom/push` — a non-functional placeholder. `Validate()` does **not** catch it, so if you forget to override it the failure surfaces only when the exporter tries to send. Always set your real backend URL.

## `add_metric_suffixes` deprecation — version discrepancy

`add_metric_suffixes` (default `true`) is **deprecated** in favor of `translation_strategy`. The factory logs a deprecation warning when it is `false`. Note a documentation/source discrepancy: the `config.go` comment claims it will be removed in **v0.153.0**, but it is **still present in v0.154.0**. Treat it as live-but-deprecated; migrate to `translation_strategy` for new configs.

## RW2 is not production-ready

`protobuf_message: io.prometheus.write.v2.Request` (Remote Write 2.0) requires the `exporter.prometheusremotewritexporter.enableSendingRW2` feature gate, and per upstream is **"In Development", only partially implemented, and not ready for usage.** Stay on RW1 (`prometheus.WriteRequest`, the default) for production.

## The WAL is not the `file_storage` extension

The exporter's `wal` block is its **own** on-disk write-ahead log, written under `wal.directory`. It is unrelated to the [`file_storage`](../file_storage/README.md) extension and takes no `storage` ID — don't try to wire it to a storage extension.

## Name and label normalization mutates data

The exporter normalizes OTLP metric names and attributes to Prometheus naming rules (so `MutatesData` is effectively true). Details of the normalization are in the [`pkg/translator/prometheus`](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/pkg/translator/prometheus) module.

## Stability is per signal

Metrics are **Beta** — and metrics is the **only** signal this exporter supports (no traces, logs, or profiles).
