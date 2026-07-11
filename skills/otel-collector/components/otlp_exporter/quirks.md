# `otlp_grpc` exporter: known quirks

## The type is now `otlp_grpc` — and there's a separate `otlphttp`

The canonical type was renamed from `otlp` to **`otlp_grpc`** in core **v1.50.0** (≈ contrib v0.144.0). The old name **`otlp` still works** as a deprecated alias (it emits a deprecation warning) and is what most existing configs use, so both `exporters: { otlp: … }` and `exporters: { otlp_grpc: … }` configure this same gRPC exporter. The rename disambiguates it from the **`otlphttp`** exporter, which is a *different* component speaking OTLP over **HTTP**. If your backend has no gRPC endpoint, you want `otlphttp`, not this one — they are not interchangeable.

## Don't add a `batch` processor — batching is built in

This exporter's `sending_queue` is enabled by default with a `batch` sub-block that flushes at **200ms** or **8192 items**. So a pipeline ending in this exporter **already batches**. Adding the legacy `batch` processor (now deprecated) double-batches and just adds latency. Tune `sending_queue.batch` instead. This matches the skill-wide pipeline-placement rule.

## gzip compression is on by default

The factory sets `compression: gzip`. To send uncompressed gRPC (e.g. a CPU-constrained agent, or a backend that rejects gzip), set `compression: none` (or empty) explicitly. The other accepted values are `snappy` and `zstd`.

## `insecure: true` is needed for plaintext

A backend with no TLS (a sibling Collector on a trusted network, a local test) requires `tls: { insecure: true }`. Without it the exporter attempts a TLS handshake and the connection fails. Note the distinction from `insecure_skip_verify: true`, which keeps TLS but skips certificate verification — different setting, different behavior.

## `wait_for_result` is incompatible with a persistent `storage` queue

`sending_queue.wait_for_result: true` and `sending_queue.storage` (a `file_storage`-backed persistent queue) cannot both be set. Pick synchronous delivery confirmation **or** restart-durable persistence, not both.

## `min_size` must be ≤ `queue_size`

When their sizers match, `sending_queue.batch.min_size` (default 8192) must not exceed `sending_queue.queue_size` (default 1000 in `requests` units — but if you switch `queue_size` to `items`, the 8192 default `min_size` can exceed it and fail validation). If you set `queue_size` in items, raise it above `min_size`, or lower `min_size`. Also: `batch.flush_timeout` must be > 0, and `batch.max_size` (when > 0) must be ≥ `min_size`.

## Data is dropped after `max_elapsed_time`

`retry_on_failure` retries with exponential backoff, but only until `max_elapsed_time` (default **5m**). After that budget is exhausted the data is **dropped** — retries are not infinite by default. Set `max_elapsed_time: 0` to retry forever (at the risk of unbounded queue growth), or pair with a persistent `storage` queue to survive longer outages.

## Stability is per signal

Traces, metrics, and logs are **Stable**; **profiles** are **Alpha**. Treat profile export as experimental and subject to breaking change, even though the same exporter serves it.
