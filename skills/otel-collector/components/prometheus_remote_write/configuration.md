# `prometheus_remote_write` exporter: configuration

All keys live under the exporter instance — `exporters: { prometheus_remote_write: { … } }` (the deprecated alias `prometheusremotewrite` also works). Facts below trace to contrib **v0.156.0** source (`exporter/prometheusremotewriteexporter/config.go` + `factory.go` + `wal.go`, with the embedded `confighttp.ClientConfig`, `exporterhelper.TimeoutConfig`, and `configretry.BackOffConfig`). The config surface is unchanged since v0.154.0 (only a 5xx-classification bugfix and dependency bumps landed in between).

## Top-level keys

| Key | Type | Default | Meaning |
|-----|------|---------|---------|
| `endpoint` | string | `http://some.url:9411/api/prom/push` (**must override**) | Remote-write URL to push samples to. From the embedded `confighttp.ClientConfig`. The factory ships a placeholder — set your real backend URL. |
| `namespace` | string | `""` | Prefix prepended to every exported metric name. |
| `external_labels` | map[string]string | `{}` | Labels attached to **every** series. Values may start with the reserved `__` prefix. |
| `add_metric_suffixes` | bool | `true` | **Deprecated** (use `translation_strategy`). When `false`, no type/unit suffixes are added; the factory logs a deprecation warning. **Ignored** when `translation_strategy` is explicitly set. See note below on the removal-version discrepancy. |
| `translation_strategy` | string (enum) | `""` (unset) | When set, takes precedence over `add_metric_suffixes`. See [translation_strategy](#translation_strategy). When unset, `add_metric_suffixes` governs. |
| `send_metadata` | bool | `false` | When `true`, generate and send Prometheus metadata. **Ignored under PRW 2.0**, which always includes metadata. |
| `disable_scope_info` | bool | `false` | When `true`, omits instrumentation-scope labels (`otel_scope_name`, `otel_scope_version`, and `otel_scope_*` attributes). |
| `max_batch_size_bytes` | int | `3000000` (≈ 2.86 MB) | Batches larger than this are split into multiple requests. **Ignored when the WAL is enabled** (the WAL `buffer_size` / `truncate_frequency` govern instead). |
| `max_batch_request_parallelism` | int (pointer) | unset (documented as `5`) | Parallel requests when splitting a single oversized batch. When unset (the default), request parallelism is governed by `remote_write_queue.num_consumers`; this key only takes over once the `EnableMultipleWorkers` gate is on. Set to `1` if the backend cannot ingest out-of-order samples. See validation below. |
| `protobuf_message` | string | `prometheus.WriteRequest` (RW1) | `prometheus.WriteRequest` = Remote Write 1.0; `io.prometheus.write.v2.Request` = Remote Write 2.0. RW2 requires the `enableSendingRW2` feature gate (see [advanced.md](advanced.md)). |
| `remote_write_queue` | object | see below | Outgoing-request queue. This exporter does **not** use `sending_queue`. |
| `resource_to_telemetry_conversion` | object | see below | Convert resource attributes to metric labels. |
| `wal` | object | off unless `directory` set | Write-ahead log. See [wal](#wal). |
| `target_info` | object | `enabled: true` | Emit a `target_info` metric per resource for resource-attribute joins. |
| `retry_on_failure` | object | `configretry.BackOffConfig` | Standard retry block; the factory overrides `InitialInterval` to `50ms`. Cross-reference the [exporterhelper retry docs](https://github.com/open-telemetry/opentelemetry-collector/blob/main/exporter/exporterhelper/README.md). |
| `timeout` | duration | per exporterhelper | From the embedded `exporterhelper.TimeoutConfig` (squashed). Cross-reference exporterhelper. |

The exporter also accepts all standard HTTP **client** options from the embedded `confighttp.ClientConfig` — squashed at the top level, so `endpoint`, `headers`, `tls`, `timeout`, `compression`, `auth`, etc. are top-level keys. Cross-reference rather than enumerating:

- [HTTP client settings (confighttp)](https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/confighttp/README.md) — note only `snappy` compression is accepted (required by the Remote Write protocol). Headers `Content-Encoding`, `Content-Type`, `X-Prometheus-Remote-Write-Version`, and `User-Agent` cannot be overridden via `headers`.
- [TLS / mTLS settings (configtls)](https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/configtls/README.md) — **TLS is on by default**; set `tls.insecure: true` for plaintext.
- [Retry and timeout settings (exporterhelper)](https://github.com/open-telemetry/opentelemetry-collector/blob/main/exporter/exporterhelper/README.md) — note the exporter does **not** support `sending_queue`; it provides `remote_write_queue` instead.

## `remote_write_queue`

| Key | Type | Default | Meaning |
|-----|------|---------|---------|
| `enabled` | bool | `true` | When `false`, export requests run synchronously (no queue). |
| `queue_size` | int | `10000` | Maximum number of OTLP metric batches queued at once. Ignored if `enabled: false`. |
| `num_consumers` | int | `5` (or `1` when `EnableMultipleWorkers` gate is on) | Workers used to fan out outgoing requests. See [advanced.md](advanced.md) for how the feature gate changes its meaning. |

## `resource_to_telemetry_conversion`

| Key | Type | Default | Meaning |
|-----|------|---------|---------|
| `enabled` | bool | `false` | When `true`, **all** resource attributes are converted to metric labels (otherwise they land on `target_info`). |
| `exclude_service_attributes` | bool | `false` | When `true`, excludes `service.name`, `service.instance.id`, and `service.namespace` (already mapped to `job` / `instance`) from the converted labels. |

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
| `max_batch_size_bytes < 0` | `max_batch_byte_size must be greater than 0` |
| `max_batch_size_bytes == 0` | not an error — **reset to `3000000`** |
| `compression` set and not `snappy` | `compression type must be snappy` (empty is allowed) |
| `protobuf_message` = `io.prometheus.write.v2.Request` without the `enableSendingRW2` gate | `remote write v2 is only supported with the feature gate exporter.prometheusremotewritexporter.enableSendingRW2` |
| `translation_strategy` not one of the four valid values | `invalid translation_strategy: <v>` |
| `translation_strategy` = `NoUTF8EscapingWithSuffixes` or `NoTranslation` under RW1 | `translation strategy <v> requires Prometheus Remote Write 2.0 (UTF-8 support)` |

> Note: `Validate()` does **not** catch an unset `endpoint` — the factory ships a non-empty placeholder default, so the failure surfaces at send time, not config validation.
