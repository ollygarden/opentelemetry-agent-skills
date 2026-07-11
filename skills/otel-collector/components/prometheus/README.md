# `prometheus` receiver

| | |
|-|-|
| Kind | receiver |
| Type | `prometheus` |
| Signals | metrics (Beta) |
| Distributions | core, contrib, k8s |
| Go module | `github.com/open-telemetry/opentelemetry-collector-contrib/receiver/prometheusreceiver` |
| Upstream README | <https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/receiver/prometheusreceiver> |

## Description

Scrapes metrics from Prometheus-format HTTP endpoints and converts them to OTLP metrics. It embeds the **scrape manager and parser from `prometheus/prometheus`**, so the configuration under the top-level `config:` key is the same YAML you would put in a `prometheus.yml` — `global`, `scrape_configs`, `scrape_config_files`, service discovery, and relabeling all behave exactly as in Prometheus. The receiver is a *scrape-only* drop-in: it pulls from targets, it does not run a Prometheus server. Anything omitted under `config:` falls back to Prometheus's own defaults for that setting.

Beyond the embedded Prometheus config, the receiver exposes a handful of extra top-level keys: `trim_metric_suffixes` (restore OTel-style metric names), `target_allocator` (fetch dynamically-assigned targets from the OpenTelemetry Operator's Target Allocator), `api_server` (host a local Prometheus agent-mode debug API), and three start-up/shutdown scrape tuning knobs added in v0.155.0 (`scrape_on_shutdown`, `discovery_reload_on_startup`, `initial_scrape_offset`). Scraped `target_info`/`otel_scope_*` series are consumed to populate the OTel Resource and Instrumentation Scope, and native histograms are auto-converted to OTel exponential histograms.

The two facts that catch people out: the receiver is **stateful and does not auto-shard** — running multiple replicas with the same config scrapes every target multiple times (duplicate metrics); use the Target Allocator or manual sharding instead. And the `$` character in the Prometheus config is interpreted by the Collector as an environment variable, so a literal `$` (in a relabel regex or replacement) must be escaped as `$$`. See [quirks.md](quirks.md).

## Main use-cases

Use when:
- You have applications, exporters, or infrastructure that expose metrics in Prometheus format and you want to pull them into an OTLP pipeline.
- You are migrating an existing `prometheus.yml` into the Collector — paste your `scrape_configs` under `config:` largely unchanged.
- You run on Kubernetes with the OpenTelemetry Operator and want dynamically-assigned, sharded scrape targets — use `target_allocator`.
- You need to scrape native histograms and forward them as OTel exponential histograms.

Avoid when:
- The source already speaks OTLP — use the [`otlp`](../otlp/README.md) receiver.
- The source pushes via Prometheus Remote Write — use the `prometheusremotewrite` receiver instead (push, not pull).
- You need Prometheus *server* features (`remote_write`, `remote_read`, `rule_files`, alerting) — this receiver rejects them; it only scrapes.

## Related components

- [`otlp` receiver](../otlp/README.md) — the OTLP-native ingress; `prometheus` is for Prometheus-format scrape targets.
- [`interval` processor](../interval/README.md) — downsample chatty scrapes by emitting only the latest value per series per interval.
- `prometheusremotewrite` receiver and `prometheus` / `prometheusremotewrite` exporters — the Prometheus-family counterparts (push vs pull); pair a scrape ingress here with a Prometheus-format egress.
- [`transform`](../transform/README.md) / [`filter`](../filter/README.md) — drop or mutate metrics after scrape, though Prometheus `metric_relabel_configs` does much of this at the source.

## Details

- [Configuration](configuration.md) — every top-level key (`config`, `trim_metric_suffixes`, `target_allocator`, `api_server`, plus the v0.155.0 `scrape_on_shutdown` / `discovery_reload_on_startup` / `initial_scrape_offset` knobs), every validation rule, scrape protocols, and representative `scrape_config` examples.
- [Verification](verification.md) — a self-scrape recipe (the collector scrapes its own `:8888` telemetry endpoint → `debug`), verified on contrib v0.154.0. Notes why `telemetrygen` cannot drive this receiver.
- [Advanced use-cases](advanced.md) — named instances, the Target Allocator, native histograms, the `api_server`, relabeling, and `scrape_config_files`.
- [Known quirks](quirks.md) — the no-auto-shard/duplicate-scrape warning, `$$` escaping, the `collector_id` `${...}` trap, removed `use_start_time_metric`, rejected server features, and stability.
