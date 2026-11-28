# `prometheus_remote_write` exporter: configuration

All keys live under the exporter instance — `exporters: { prometheus_remote_write: { … } }` (the deprecated alias `prometheusremotewrite` also works). Facts below trace to contrib **v0.160.0** source (`exporter/prometheusremotewriteexporter/config.go` + `factory.go` + `wal.go`, with `confighttp.ClientConfig`, `exporterhelper.TimeoutConfig`, and `configretry.BackOffConfig`).

## Top-level keys

| Key | Type | Default | Meaning |
|-----|------|---------|---------|
| `http` | object | standard HTTP client defaults | Nested `confighttp.ClientConfig`. Added in v0.159.0; preferred over the deprecated flat HTTP client settings. |
| `http.endpoint` | string | `http://some.url:9411/api/prom/push` (**must override**) | Remote-write URL to push samples to. The factory ships a placeholder — set your real backend URL. |
| `namespace` | string | `""` | Prefix prepended to every exported metric name. |
| `external_labels` | map[string]string | `{}` | Labels attached to **every** series. Values may start with the reserved `__` prefix. |
| `add_metric_suffixes` | bool | `true` | **Deprecated** (use `translation_strategy`). When `false`, no type/unit suffixes are added; the factory logs a deprecation warning. **Ignored** when `translation_strategy` is explicitly set. See note below on the removal-version discrepancy. |
| `translation_strategy` | string (enum) | `""` (unset) | When set, takes precedence over `add_metric_suffixes`. See [translation_strategy](#translation_strategy). When unset, `add_metric_suffixes` governs. |
| `send_metadata` | bool | `false` | When `true`, generate and send Prometheus metadata. **Ignored under PRW 2.0**, which always includes metadata. |
| `disable_scope_info` | bool | `false` | When `true`, omits instrumentation-scope labels (`otel_scope_name`, `otel_scope_version`, and `otel_scope_*` attributes). |
| `max_batch_size_bytes` | int | `3000000` (≈ 2.86 MB) | Batches larger than this are split into multiple requests. **Ignored when the WAL is enabled** (the WAL `buffer_size` / `truncate_frequency` govern instead). |
| `max_batch_request_parallelism` | int (pointer) | unset (documented as `5`) | Parallel requests when splitting a single oversized batch. When unset (the default), request parallelism is governed by `remote_write_queue.num_consumers`; this key only takes over once the `EnableMultipleWorkers` gate is on. Set to `1` if the backend cannot ingest out-of-order samples. See validation below. |
| `protobuf_message` | string | `prometheus.WriteRequest` (RW1) | `prometheus.WriteRequest` = Remote Write 1.0; `io.prometheus.write.v2.Request` = Remote Write 2.0. RW2 requires the `enableSendingRW2` feature gate (see [advanced.md](advanced.md)). |
| `include_metadata_keys` | []string | `[]` | Forward selected incoming client-metadata values as HTTP request headers. Reserved Remote Write protocol headers are rejected. WAL-replayed requests cannot forward this metadata because the WAL stores only serialized request bytes. Added in v0.158.0. |
| `convert_explicit_histograms_to_nhcb` | bool | `false` | Convert explicit-bucket histograms to Native Histograms with Custom Buckets (schema -53), for RW1 or RW2. Added in v0.158.0. |
| `keep_classic_histograms` | bool | `false` | Also emit classic histogram series during NHCB migration; no effect unless conversion is enabled. |
| `remote_write_queue` | object | see below | Outgoing-request queue. This exporter does **not** use `sending_queue`. |
| `resource_constant_labels` | object | see below | Select resource attributes to convert to labels. Added in v0.160.0. |
| `resource_to_telemetry_conversion` | object | see below | **Deprecated** in v0.160.0; use `resource_constant_labels`. |
| `wal` | object | off unless `directory` set | Write-ahead log. See [wal](#wal). |
| `target_info` | object | `enabled: true` | Emit a `target_info` metric per resource for resource-attribute joins. |
| `retry_on_failure` | object | `configretry.BackOffConfig` | The standard [`retry_on_failure` keys](../../references/exporterhelper.md#retry_on_failure); the factory overrides `InitialInterval` to `50ms`. The exporter runs its own backoff loop with these settings, not the helper's retry sender. Transport errors and 5xx retry until `max_elapsed_time`, as does `429` with the [`RetryOn429` gate](advanced.md#retrying-on-http-429); other non-2xx responses fail on the first attempt. |
| `timeout` | duration | `5s` | From the embedded `exporterhelper.TimeoutConfig` (squashed). Since v0.164.0 the factory switches the helper timeout off and the exporter applies `timeout` to each send attempt, as the HTTP client applies `http.timeout`. When only one of the two keys is written and its value is above `0`, the other takes that value, and `0` disables only the one it is set on, so with an `http` block `timeout: 0` leaves `http.timeout` at `5s` ([`factory.go` and `config.go` at v0.165.1](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.165.1/exporter/prometheusremotewriteexporter)). Through v0.163.0 the factory passes it to the exporterhelper, so it bounds the whole push, including the exporter's own retries ([v0.163.0 `factory.go`](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.163.0/exporter/prometheusremotewriteexporter/factory.go)). |

The `http` block accepts the standard HTTP **client** options from `confighttp.ClientConfig`, such
as `endpoint`, `headers`, `tls`, `compression`, and `auth`. Deprecated flat HTTP settings still
work when `http` is absent; when `http` is present it takes precedence. The alpha
`exporter.prometheusremotewritexporter.removeTopLevelHTTPSettings` gate rejects flat HTTP client
settings so migrations can be tested before their eventual removal. For `http.timeout`, see the
[`timeout` row](#top-level-keys). Cross-reference rather than enumerating:

- [HTTP client settings (confighttp)](https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/confighttp/README.md) — note only `snappy` compression is accepted (required by the Remote Write protocol). Headers `Content-Encoding`, `Content-Type`, `X-Prometheus-Remote-Write-Version`, and `User-Agent` cannot be overridden via `headers`.
- [TLS / mTLS settings (configtls)](https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/configtls/README.md) — **TLS is on by default**; set `http.tls.insecure: true` for plaintext.
- [`retry_on_failure` keys (exporterhelper)](../../references/exporterhelper.md#retry_on_failure) — key names and validation only, since this exporter runs its own retry loop; it does **not** support `sending_queue` and provides `remote_write_queue` instead.

## `remote_write_queue`

| Key | Type | Default | Meaning |
|-----|------|---------|---------|
| `enabled` | bool | `true` | When `false`, export requests run synchronously (no queue). |
| `queue_size` | int | `10000` | Maximum number of OTLP metric batches queued at once. Ignored if `enabled: false`. |
| `num_consumers` | int | `5` (or `1` when `EnableMultipleWorkers` gate is on) | Workers used to fan out outgoing requests. See [advanced.md](advanced.md) for how the feature gate changes its meaning. |

## Resource labels

| Key | Type | Default | Meaning |
|-----|------|---------|---------|
| `resource_constant_labels.included` | []string | `[]` | Wildcard patterns selecting resource attributes to convert to labels. |
| `resource_constant_labels.excluded` | []string | `[]` | Wildcard patterns removed from the included set. |

The deprecated `resource_to_telemetry_conversion` block retains `enabled` and
`exclude_service_attributes` for migration. Do not configure both blocks. The alpha
`exporter.prometheusremotewrite.DisableResourceToTelemetryConversion` gate rejects the deprecated
block. `enabled` and `exclude_service_attributes` are not valid under `resource_constant_labels`.

## `wal`

Optional and **off unless `directory` is set**. When enabled, `max_batch_size_bytes` is ignored.

| Key | Type | Default | Meaning |
|-----|------|---------|---------|
| `directory` | string | `""` | Directory to store the WAL in. Setting this enables the WAL. |
| `buffer_size` | int | `300` | Count of elements read from the WAL before truncating. |
| `truncate_frequency` | duration | `1m` | How often the WAL is truncated. |
| `lag_record_frequency` | duration | `15s` | How often the exporter records WAL lag. |

## `target_info`

| Key | Type | Default | Meaning |
|-----|------|---------|---------|
| `enabled` | bool | `true` | When `true`, emits a `target_info` metric per resource metric, carrying resource attributes for PromQL joins. |

## `translation_strategy`

When set, this enum takes precedence over the deprecated `add_metric_suffixes`. `Validate()` rejects any value other than the four below with `invalid translation_strategy: <v>`.

| Value | Meaning |
|-------|---------|
| `UnderscoreEscapingWithSuffixes` | Escape special characters to `_` **and** append type/unit suffixes. |
| `UnderscoreEscapingWithoutSuffixes` | Escape special characters to `_`, but no suffixes. |
| `NoUTF8EscapingWithSuffixes` | Keep UTF-8 (no `_` escaping); still append unit / `_total` suffixes. **Requires PRW 2.0.** |
| `NoTranslation` | Pass names/labels through unaltered. **Requires PRW 2.0.** |

> The upstream README prose calls `UnderscoreEscapingWithSuffixes` the "default", but the **code** default is the empty string, with `add_metric_suffixes: true` governing when `translation_strategy` is unset. Be precise: unset means `add_metric_suffixes` decides.

## Validation summary

| Condition | Error / rule |
|-----------|--------------|
| `max_batch_request_parallelism` set and `< 1` | `max_batch_request_parallelism can't be set to below 1` |
| `remote_write_queue.queue_size < 0` | `remote write queue size can't be negative` |
| `remote_write_queue.enabled` and `queue_size == 0` | `a 0 size queue will drop all the data` |
| `remote_write_queue.num_consumers < 0` | `remote write consumer number can't be negative` |
| `max_batch_size_bytes < 0` | `max_batch_size_bytes must be greater than 0` |
| `max_batch_size_bytes == 0` | not an error — **reset to `3000000`** |
| `http.compression` set and not `snappy` | `compression type must be snappy` (empty is allowed) |
| `protobuf_message` = `io.prometheus.write.v2.Request` without the `enableSendingRW2` gate | `remote write v2 is only supported with the feature gate exporter.prometheusremotewritexporter.enableSendingRW2` |
| `translation_strategy` not one of the four valid values | `invalid translation_strategy: <v>` |
| `translation_strategy` = `NoUTF8EscapingWithSuffixes` or `NoTranslation` under RW1 | `translation strategy <v> requires Prometheus Remote Write 2.0 (UTF-8 support)` |
| `include_metadata_keys` contains `Content-Encoding`, `Content-Type`, `User-Agent`, or `X-Prometheus-Remote-Write-Version` (case-insensitive) | rejected because it collides with a reserved Remote Write header |
| both resource-label blocks configured | `cannot configure both resource_to_telemetry_conversion and resource_constant_labels; resource_to_telemetry_conversion is deprecated` |
| legacy fields under `resource_constant_labels` | `enabled and exclude_service_attributes are not supported under resource_constant_labels; use included and excluded instead` |

> Note: `Validate()` does **not** catch an unset `http.endpoint` — the factory ships a non-empty placeholder default, so the failure surfaces at send time, not config validation.
