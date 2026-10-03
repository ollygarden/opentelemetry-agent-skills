# `prometheus` exporter: configuration

All keys live under the exporter instance — `exporters: { prometheus: { … } }`. Facts below trace to contrib **v0.160.0** source (`exporter/prometheusexporter/config.go` + `factory.go`, with the embedded `confighttp.ServerConfig`).

## Top-level keys

| Key | Type | Default | Meaning |
|-----|------|---------|---------|
| `endpoint` | string | — (**required**) | Address the HTTP server binds to; metrics are exposed at the fixed path `/metrics`. From the embedded `confighttp.ServerConfig`. |
| `namespace` | string | — | If set, prefixes every exported series (e.g. `namespace: testns` → `testns_<metric>`). |
| `const_labels` | map[string]string | — | Key/value labels applied to **every** exported metric. |
| `send_timestamps` | bool | `false` | When `true`, appends the underlying sample timestamp to each exposed line. |
| `metric_expiration` | duration | `5m` | How long a series stays exposed without an update; after this it is dropped from `/metrics`. |
| `resource_constant_labels` | object | — | Select resource attributes to add as labels with `included` / `excluded` wildcard patterns. Added in v0.160.0. The legacy `enabled` and `exclude_service_attributes` fields are rejected in this block. |
| `resource_to_telemetry_conversion` | object | — | **Deprecated** in v0.160.0; use `resource_constant_labels`. Its legacy `enabled` / `exclude_service_attributes` fields remain accepted unless the migration gate is enabled. |
| `enable_open_metrics` | bool | `false` | When `true`, exposes using OpenMetrics format. Exemplars are exported **only** in OpenMetrics format, and **only** for histograms and monotonic sums (counters). |
| `without_scope_info` | bool | `false` | When `true`, omits instrumentation-scope labels (`otel_scope_name`, `otel_scope_version`, `otel_scope_schema_url`, and scope attributes). |
| `add_metric_suffixes` | bool | `true` | **Deprecated** (use `translation_strategy`). When `true`, appends type/unit suffixes (e.g. counters get `_total`). **Ignored** when `translation_strategy` is explicitly set. |
| `translation_strategy` | string (enum) | — | When set, takes precedence over `add_metric_suffixes`; controls how OTLP metric/attribute names are translated to Prometheus names. See [translation_strategy](#translation_strategy). |
| `sending_queue` | object | — | The standard `exporterhelper` queue block. Far less meaningful here than for a pushing exporter (see [advanced.md](advanced.md) and [quirks.md](quirks.md)). |

The exporter also accepts all standard HTTP **server** options from the embedded `confighttp.ServerConfig` (`tls`, `cors`, `auth`, `response_headers`, `compression_algorithms`, `max_request_body_size`, and others — cross-reference the confighttp documentation rather than enumerating exhaustively). Server defaults from `NewDefaultServerConfig()`:

| Server key | Default |
|-----|---------|
| `write_timeout` | `30s` |
| `read_header_timeout` | `1m` |
| `keepalive.idle_timeout` | `1m` |
| `keepalive.enabled` | `true` |

The former flat `idle_timeout` and `keep_alives_enabled` fields are deprecated in core v0.160.0.
Nonzero `idle_timeout` and `keep_alives_enabled: false` cannot be set with the new `keepalive` block;
the no-op values `idle_timeout: 0` and `keep_alives_enabled: true` may coexist with it.

## `translation_strategy`

When set, this enum takes precedence over the deprecated `add_metric_suffixes`. `Validate()` rejects any value other than the four below with `invalid translation_strategy: <v>`.

| Value | Meaning |
|-------|---------|
| `UnderscoreEscapingWithSuffixes` | Fully escape names for classic Prometheus compatibility **and** append type/unit suffixes. |
| `UnderscoreEscapingWithoutSuffixes` | Escape special characters to `_`, but no suffixes. |
| `NoUTF8EscapingWithSuffixes` | Keep UTF-8 (no `_` escaping); still append unit / `_total` suffixes. |
| `NoTranslation` | Bypass all translation; names/labels pass through unaltered. |

> **Content-negotiation caveat** (from the upstream README): Prometheus negotiates the exposition format when scraping, so underscore escaping may still be applied on the wire even when this is set to keep UTF-8.

## `sending_queue`

Optional. The standard `exporterhelper` queue block. Because this is a **pull** exporter that accumulates in memory and is scraped, queue/retry semantics are far less meaningful here than for a pushing exporter. Tune it only if you have a specific reason; for the full queue reference see the [`otlp_grpc` exporter's configuration](../otlp_exporter/configuration.md#sending_queue).

## Validation summary

| Condition | Error / rule | When |
|-----------|--------------|------|
| `translation_strategy` set to anything other than the four valid values | `invalid translation_strategy: <v>` | config validation |
| both resource-conversion blocks configured | rejected; use only `resource_constant_labels` | config validation |
| `resource_constant_labels.enabled` or `.exclude_service_attributes` set | rejected; use `included` / `excluded` | config validation |

An empty `endpoint` is not caught by `Config.Validate()`; `endpoint` has no default, so the missing-address error surfaces one step later, at exporter construction during pipeline build — which the `validate` subcommand triggers, reporting `expecting a non-blank address to run the Prometheus metrics handler`.

## Feature gate

| Gate | Stage | Since | Effect |
|------|-------|-------|--------|
| `exporter.prometheusexporter.DisableAddMetricSuffixes` | `beta` | v0.132.0 | When enabled, the deprecated `add_metric_suffixes` is ignored and `translation_strategy` is always used. (Spec PR 4533.) |
| `exporter.prometheus.DisableResourceToTelemetryConversion` | `alpha` | v0.160.0 | Rejects deprecated `resource_to_telemetry_conversion`; use `resource_constant_labels`. |
| `pkg.translator.prometheus.PermissiveLabelSanitization` | `beta` (since v0.162.0) | v0.55.0 | Shared with `prometheus_remote_write`. Enabled by default: labels with a single leading underscore (`_foo`) are kept instead of being rewritten to `key_foo`. Disable to restore the prefixing. |
