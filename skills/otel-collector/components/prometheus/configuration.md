# `prometheus`: configuration

## Typical config

```yaml
receivers:
  prometheus:
    config:
      scrape_configs:
        - job_name: 'my-service'
          scrape_interval: 5s
          static_configs:
            - targets: ['my-service:9090']

service:
  pipelines:
    metrics:
      receivers: [prometheus]
      processors: [memory_limiter]
      exporters: [otlp]
```

## Configuration reference

These are the receiver's **own** top-level keys. The `config:` block holds the standard Prometheus
configuration — `global`, `scrape_configs`, `scrape_config_files`, service discovery, relabeling —
documented in the upstream [Prometheus configuration reference][promcfg]; it is not reproduced here.

| Key | Type | Default | Notes |
|-----|------|---------|-------|
| `config` | embedded Prometheus config | — | The Prometheus configuration, same YAML structure as `prometheus.yml`. Holds `global`, `scrape_configs`, `scrape_config_files`, and related sections. Any field you omit takes Prometheus's own default for that setting. |
| `trim_metric_suffixes` | bool | `false` | [**Experimental**] Trims unit and counter-type suffixes from metric names, e.g. `singing_duration_seconds_total` → `singing_duration`. Useful to restore OpenTelemetry-style names. |
| `target_allocator` | object | — | Optional. Client config to fetch dynamically-assigned scrape targets from the OpenTelemetry Operator's Target Allocator. See table below and [advanced.md](advanced.md). |
| `api_server` | object | — | Optional. Hosts a local Prometheus agent-mode API server for debugging. See table below and [advanced.md](advanced.md). |
| `scrape_on_shutdown` | bool | `false` | Perform one final scrape before the receiver closes. This last scrape ignores the configured scrape interval. (v0.155.0) |
| `discovery_reload_on_startup` | bool | `false` | Discover targets immediately on start-up instead of waiting for the configured interval before initializing the scrape pools. (v0.155.0) |
| `initial_scrape_offset` | duration | `0s` | Fixed delay added before the initial scrape of targets, on top of the receiver's internal load-balancing offset. Avoids readiness races and backend rate-limits right after start-up. (v0.155.0) |

[promcfg]: https://prometheus.io/docs/prometheus/latest/configuration/configuration/

### `target_allocator` keys

The block squashes the Collector `confighttp` [client configuration][confighttp] (so `endpoint`,
`tls`, `proxy_url`, etc. are available) and adds:

| Key | Type | Default | Notes |
|-----|------|---------|-------|
| `endpoint` | string | — | **Required.** Target Allocator URL. Must be a valid request URI. |
| `interval` | duration | — | How often to refresh the target list. Must be a **positive** duration. |
| `collector_id` | string | — | **Required.** Identifies this collector instance to the allocator. Must be non-empty. A **literal** `${` is rejected, but `${ENV}` references are env-expanded by the Collector before validation (see [quirks.md](quirks.md)). |
| `http_sd_config` | Prometheus HTTP SD config | — | HTTP service-discovery config used to fetch targets. |
| `http_scrape_config` | Prometheus HTTP client config | — | HTTP client config applied when scraping the discovered targets. |
| `tls`, `proxy_url`, … | (from `confighttp.ClientConfig`) | — | Standard confighttp client options. |

[confighttp]: https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/confighttp/README.md#client-configuration

### `api_server` keys

| Key | Type | Default | Notes |
|-----|------|---------|-------|
| `enabled` | bool | `false` | Turn the debug API server on. |
| `max_connections` | int | `512` | Maximum number of simultaneous HTTP connections the server accepts. |
| `server_config` | `confighttp.ServerConfig` | endpoint `127.0.0.1:9090`, `read_timeout: 10m` | Standard confighttp [server config][confighttp-server] (`endpoint`, `tls`, CORS, timeouts). The endpoint is populated by default, so enabling the server does not require setting it. |

[confighttp-server]: https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/confighttp/README.md#server-configuration

## Validation rules

| Rule | Error message |
|------|---------------|
| At least one of `config.scrape_configs`, `config.scrape_config_files`, or `target_allocator` must be set. | `no Prometheus scrape_configs or target_allocator set` |
| Prometheus server-only features are rejected: `remote_write`, `remote_read`, `rule_files`, `alert_config.relabel_configs`, `alert_config.alertmanagers`. | `unsupported features: …` (the offending features, one per line) |
| `target_allocator.endpoint` must parse as a valid request URI. | `TargetAllocator endpoint is not valid: …` |
| `target_allocator.collector_id` must be non-empty and must not contain `${`. | `CollectorID is not a valid ID` |
| `target_allocator.interval` must be a positive duration. | `interval must be a positive duration, got …` |

## Scrape protocols and exposition formats

The receiver supports the protocols/exposition formats accepted by the Prometheus scrape config:

- `PrometheusProto`
- `OpenMetricsText1.0.0`
- `OpenMetricsText0.0.1`
- `PrometheusText1.0.0`
- `PrometheusText0.0.4`

Since Prometheus 3.0, scrapes need a proper `Content-Type` header. For back-compat, the receiver
defaults each scrape config's `fallback_scrape_protocol` to `PrometheusText0.0.4` when it is unset,
so targets that do not return a content type still parse.

## How scraped series map to OTel

- `target_info` is dropped; its attributes populate the OTel **Resource**.
- `otel_scope_name` / `otel_scope_version` labels are dropped and populate the Instrumentation **Scope** name/version.
- `otel_scope_info` is dropped; its remaining attributes populate Scope **attributes**.
- The `job` target label (its value comes from `job_name`) maps to `service.name`, and `instance` maps to `service.instance.id` (plus `server.address`/`server.port` when discernible). Kubernetes SD meta-labels map to `k8s.*` resource attributes — see the upstream [resource attribute mapping][resmap].
- Native histograms are converted to OTel exponential histograms (see [advanced.md](advanced.md)).

[resmap]: https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/receiver/prometheusreceiver/resource_attribute_mapping.md
