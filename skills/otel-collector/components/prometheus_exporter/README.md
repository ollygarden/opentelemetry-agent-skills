# `prometheus` exporter

| | |
|-|-|
| Kind | exporter |
| Type | `prometheus` |
| Signals | metrics (Beta) |
| Distributions | core, contrib |
| Go module | `github.com/open-telemetry/opentelemetry-collector-contrib/exporter/prometheusexporter` |
| Upstream README | <https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/exporter/prometheusexporter> |

## Description

This is a **pull / expose** exporter — it does **not push anywhere**. It hosts an HTTP server that exposes your pipeline's metrics in Prometheus exposition format at the fixed path `/metrics`, for a Prometheus server (or any scraper) to scrape. This is the single most common point of confusion: if you want to **push** metrics out via Prometheus Remote Write, you want the `prometheusremotewrite` exporter, not this one. It embeds `confighttp.ServerConfig`, so it accepts the standard HTTP server options (`tls`, `cors`, `auth`, `max_request_body_size`, and so on); `MutatesData` is `true`.

Internally it keeps an in-memory **accumulator**: it stores the latest aggregated value per series and re-exposes that value on every scrape, dropping a series after `metric_expiration` (default `5m`) elapses with no update. Metric naming is controlled by `add_metric_suffixes` (default `true`, **deprecated**) or the newer `translation_strategy` enum, which takes precedence when set. See the detail files below.

> **Same type string as the `prometheus` receiver.** The `prometheus` *receiver* (a scrape ingress) and this `prometheus` *exporter* share the type string `prometheus` but are different component classes; they coexist in one config without conflict. Don't confuse the two.

## Main use-cases

Use when:
- You want to expose OTLP-pipeline metrics for a Prometheus server (or any scraper) to pull.
- You want a local, human-scrapable `/metrics` endpoint for debugging a metrics pipeline.
- You are bridging an OTLP pipeline into a pull-based Prometheus world.

Avoid when:
- You need to **push** metrics out — use the `prometheusremotewrite` exporter.
- The backend speaks OTLP — use the [`otlp_grpc`](../otlp_exporter/README.md) or `otlphttp` exporter.
- You run many horizontally-scaled collector replicas — each exposes only its own subset of the data, so the scraper must target every replica and there is no cross-replica aggregation.

## Related components

- [`prometheus` receiver](../prometheus/README.md) — the scrape-ingress counterpart (same type string, different class); pulls Prometheus-format metrics *into* a pipeline.
- `prometheusremotewrite` exporter — the **push** sibling; pick it when you need to send metrics out via Remote Write rather than be scraped.
- [`otlp_grpc`](../otlp_exporter/README.md) / `otlphttp` exporter — OTLP egress when the backend speaks OTLP.
- [`transform`](../transform/README.md) — copy selected resource attributes onto datapoint attributes so they become Prometheus labels for grouping.

## Details

- [Configuration](configuration.md) — `endpoint`, `namespace`, `const_labels`, `send_timestamps`, `metric_expiration`, `resource_to_telemetry_conversion`, `enable_open_metrics`, `without_scope_info`, `add_metric_suffixes`, `translation_strategy`, the embedded HTTP server options, and validation rules.
- [Verification](verification.md) — drive an OTLP pipeline with `telemetrygen` and `curl` the exporter's own `/metrics`, proving namespace, suffixing, labels, timestamps, and latest-value accumulation. Verified on contrib v0.154.0.
- [Advanced use-cases](advanced.md) — `target_info` vs `resource_to_telemetry_conversion`, OpenMetrics + exemplars, native histograms, `translation_strategy` choices, and the queue caveat.
- [Known quirks](quirks.md) — pull-not-push, the shared `prometheus` type string, required `endpoint`, stale series and counter resets, `add_metric_suffixes` deprecation, exemplar limits, horizontal-scaling behavior, and per-signal stability.
