# `prometheus` exporter: known quirks

## It's a server (pull), not a pusher

This exporter **hosts** an HTTP server and exposes metrics at `/metrics` for a scraper to pull. It does **not** push metrics anywhere. The common mistake is reaching for it when you actually want to **send** metrics out via Prometheus Remote Write — that is the separate [`prometheus_remote_write`](../prometheus_remote_write/README.md) exporter. They are not interchangeable: this one waits to be scraped; that one initiates pushes.

## Same `prometheus` type string as the receiver

The `prometheus` **receiver** (a scrape ingress) and this `prometheus` **exporter** share the type string `prometheus` but are different component classes. They coexist in one config — `receivers: { prometheus: … }` and `exporters: { prometheus: … }` are unrelated components — so don't assume a reference to "the prometheus component" means either one.

## `endpoint` is required, and the path is fixed at `/metrics`

`endpoint` has **no default**; without it the HTTP server has nothing to bind to. The exposition path is always `/metrics` — it is not configurable. `Config.Validate()` does not catch a missing `endpoint` (it only validates `translation_strategy`), so the failure surfaces when the exporter is **constructed during pipeline build**, not from `Config.Validate()`. Because the `validate` subcommand builds pipelines, it does report this — `failed to build pipelines: failed to create "prometheus" exporter for data type "metrics": expecting a non-blank address to run the Prometheus metrics handler` — rather than waiting for server startup.

## Stale series after `metric_expiration`; counters reset on restart

The exporter holds an in-memory accumulator. A series stays exposed only as long as it keeps getting updates within `metric_expiration` (default **5m**); after that it is dropped from `/metrics`. Because the state is in-memory, all accumulated values — including counters — **reset when the collector restarts**, which a scraping Prometheus will see as a counter reset.

## `add_metric_suffixes` is deprecated — prefer `translation_strategy`

`add_metric_suffixes` (default `true`) appends type/unit suffixes (counters get `_total`), but it is **deprecated** and is **ignored** whenever `translation_strategy` is explicitly set. The `exporter.prometheusexporter.DisableAddMetricSuffixes` feature gate (stage `beta`, since v0.132.0) makes the collector ignore `add_metric_suffixes` entirely and always use `translation_strategy`. Migrate to `translation_strategy` for new configs.

## Exemplars need OpenMetrics, and only on histograms/counters

Exemplars are exported **only** when `enable_open_metrics: true`, and even then **only** for histograms and monotonic sums (counters). Other metric types never carry exemplars, and native histograms get none at all.

## Horizontal scaling: scrape every replica, no aggregation

Each collector replica exposes only the metrics it itself received — there is no cross-replica aggregation. If you run several replicas, the scraping Prometheus must target **every** replica to get the full picture, and you must handle the resulting per-replica series accordingly. There is no built-in sharding or merging on the exposition side.

## Stability is per signal

Metrics are **Beta** — and metrics is the **only** signal this exporter supports (no traces, logs, or profiles).
