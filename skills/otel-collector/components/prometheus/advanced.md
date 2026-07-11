# `prometheus`: advanced use-cases

## Named instances

Run more than one scrape configuration as independent receiver instances with the `prometheus/<name>` syntax:

```yaml
receivers:
  prometheus/apps:
    config:
      scrape_configs:
        - job_name: 'apps'
          static_configs:
            - targets: ['app-a:9090', 'app-b:9090']
  prometheus/infra:
    config:
      scrape_configs:
        - job_name: 'node'
          static_configs:
            - targets: ['node-exporter:9100']

service:
  pipelines:
    metrics:
      receivers: [prometheus/apps, prometheus/infra]
      exporters: [otlp]
```

## Target Allocator (OpenTelemetry Operator)

Instead of static jobs, fetch a dynamically-assigned, sharded target list from the OpenTelemetry
Operator's Target Allocator (or a compatible endpoint). The block embeds the full `confighttp`
client configuration (so `tls`, `proxy_url`, etc. are available) plus the allocator-specific keys:

```yaml
receivers:
  prometheus:
    target_allocator:
      endpoint: http://my-targetallocator-service
      interval: 30s
      collector_id: collector-1     # literal id — must NOT contain ${...}
```

`endpoint` must be a valid request URI, `interval` must be a positive duration, and `collector_id`
is required and must not contain `${` (see [quirks.md](quirks.md)). Optional `http_sd_config` and
`http_scrape_config` tune the discovery and scrape HTTP clients respectively.

The Target Allocator is also how you avoid duplicate scrapes when running multiple collector
replicas — it shards targets across the collectors rather than every replica scraping everything.

## Native histograms

The receiver auto-converts scraped Prometheus native histograms to OTel exponential histograms.
This is now unconditional — the old `receiver.prometheusreceiver.EnableNativeHistograms` gate
graduated to stable and has been removed, so there is no gate to toggle. You still need **two**
things in the Prometheus scrape config:

1. `scrape_native_histograms: true` (globally under `global`, or per-job)
2. `PrometheusProto` included in `scrape_protocols` (required until Prometheus supports native histograms over text formats)

```yaml
receivers:
  prometheus:
    config:
      global:
        scrape_native_histograms: true
      scrape_configs:
        - job_name: 'native-hist'
          scrape_protocols: [PrometheusProto, OpenMetricsText1.0.0, OpenMetricsText0.0.1, PrometheusText0.0.4]
          static_configs:
            - targets: ['app:9090']
```

If a metric has both classic buckets and native histogram buckets, only the native histogram buckets are used.

## Debug API server (`api_server`)

Host a local Prometheus agent-mode API server to inspect targets, config, and service discovery for
debugging. The `server_config` is the standard Collector `confighttp` server config:

```yaml
receivers:
  prometheus:
    api_server:
      enabled: true
      server_config:
        endpoint: "localhost:9090"   # defaults to 127.0.0.1:9090 if omitted
```

`server_config` is the standard confighttp server config (defaults: endpoint `127.0.0.1:9090`,
`read_timeout: 10m`), and `max_connections` (default `512`) caps simultaneous HTTP connections.

Endpoints mirror the Prometheus agent-mode API:

- `/api/v1/targets`
- `/api/v1/targets/metadata`
- `/api/v1/status/config`
- `/api/v1/scrape_pools`
- `/metrics`

## Relabeling and metric filtering

`relabel_configs` and `metric_relabel_configs` behave exactly as in Prometheus — do most of your
target selection and metric dropping at the source rather than downstream. Note the `$$` escaping
rule for literal `$` in regex/replacement values (see [quirks.md](quirks.md)).

```yaml
receivers:
  prometheus:
    config:
      scrape_configs:
        - job_name: 'k8s-pods'
          kubernetes_sd_configs:
            - role: pod
          relabel_configs:
            - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
              regex: "true"
              action: keep
          metric_relabel_configs:
            - source_labels: [__name__]
              regex: "(request_duration_seconds.*|response_duration_seconds.*)"
              action: keep
```

## Splitting scrape configs into files (`scrape_config_files`)

Keep large or shared scrape definitions out of the main config with Prometheus's
`scrape_config_files` (glob paths to files holding `scrape_configs`). This counts as a valid scrape
source for validation (you do not also need inline `scrape_configs`):

```yaml
receivers:
  prometheus:
    config:
      scrape_config_files:
        - /etc/otel/scrape_configs/*.yaml
```

## Feature gates

- `receiver.prometheusreceiver.EnableCreatedTimestampZeroIngestion` (from v0.113.0) — **alpha, off by default**. Injects created-timestamps as 0-valued samples. Off by default due to higher CPU cost at high metric volume.
- `receiver.prometheusreceiver.IgnoreScopeInfoMetric` (from v0.148.0) — **beta, on by default since v0.156.0**. The `otel_scope_info` metric is now ignored for scope-attribute extraction by default; scope attributes come from `otel_scope_<name>` labels. To temporarily restore the old behavior, disable it with `--feature-gates=-receiver.prometheusreceiver.IgnoreScopeInfoMetric`.
