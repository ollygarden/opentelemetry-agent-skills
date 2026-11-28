# `otlp_grpc` exporter: configuration

All keys live under the exporter instance — `exporters: { otlp_grpc: { … } }` or, via the deprecated alias, `exporters: { otlp: { … } }`. Top-level and `tls` facts trace to core **v1.66.0 / v0.160.0** source (`exporter/otlpexporter/config.go` + `factory.go`, `config/configgrpc/configgrpc.go` `ClientConfig`, `config/configtls`). The `timeout`, `retry_on_failure`, and `sending_queue` defaults trace to core **v0.165.0** [`factory.go`](https://github.com/open-telemetry/opentelemetry-collector/blob/v0.165.0/exporter/otlpexporter/factory.go) and [`metadata.yaml`](https://github.com/open-telemetry/opentelemetry-collector/blob/v0.165.0/exporter/otlpexporter/metadata.yaml).

## Top-level (gRPC client) keys

| Key | Type | Default | Meaning |
|-----|------|---------|---------|
| `endpoint` | string | — (**required**) | gRPC target (`host:port`, gRPC naming syntax). Accepts `http://`, `https://`, `dns://` prefixes (stripped internally for validation). |
| `compression` | string | `gzip` | `gzip`, `snappy`, `zstd`, or none/empty to disable. Default set by the factory. |
| `tls` | object | — | gRPC client TLS (see [tls](#tls)). |
| `headers` | map[string]string | — | Static headers added to every gRPC request. |
| `timeout` | duration | `5s` | Per-attempt send timeout ([`timeout`](../../references/exporterhelper.md#timeout)). |
| `keepalive` | object | — | gRPC client keepalive: `time`, `timeout`, `permit_without_stream`. |
| `read_buffer_size` | int | 0 (gRPC default) | gRPC read buffer. The factory leaves this unset (the exporter reads almost nothing). |
| `write_buffer_size` | int | `524288` (512 KiB) | gRPC write buffer. |
| `wait_for_ready` | bool | `false` | Block RPCs until the connection is ready instead of failing fast. Core v0.157.0 fixed this option being parsed but not applied on gRPC client calls. |
| `balancer_name` | string | `round_robin` | gRPC client-side load-balancing policy across resolved addresses. |
| `authority` | string | — | Overrides the `:authority` pseudo-header. |
| `user_agent` | string | — (build-info default) | Overrides the default gRPC user-agent header. Empty keeps the build-derived default. |
| `auth` | object | — | `authenticator:` referencing an auth extension (e.g. bearer/OAuth2). |
| `middlewares` | list | — | gRPC client middleware extensions. |

**Validation:** an empty `endpoint` fails with `requires a non-empty "endpoint"`.

## `tls`

`configtls.ClientConfig` (squashed). Common keys:

| Key | Type | Default | Meaning |
|-----|------|---------|---------|
| `insecure` | bool | `false` | Use **plaintext** gRPC (no TLS). Set `true` for an internal/test backend with no TLS. |
| `insecure_skip_verify` | bool | `false` | Use TLS but skip server-cert verification. |
| `ca_file` | string | — | CA bundle to verify the server certificate. |
| `cert_file` | string | — | Client certificate (for mTLS). |
| `key_file` | string | — | Client private key (for mTLS). |
| `server_name_override` | string | — | Override the SNI / verified server name. |

`insecure` (plaintext) and `insecure_skip_verify` (TLS without verification) are different: the first disables TLS entirely, the second keeps TLS but trusts any certificate.

## `retry_on_failure`

`otlp_grpc` uses the helper default unchanged (`configretry.NewDefaultBackOffConfig`): `enabled: true`, `initial_interval: 5s`, `randomization_factor: 0.5`, `multiplier: 1.5`, `max_interval: 30s`, `max_elapsed_time: 5m`. Backoff, the drop after `max_elapsed_time`, and validation: [`retry_on_failure`](../../references/exporterhelper.md#retry_on_failure).

## `sending_queue`

`metadata.yaml` declares `sending_queue: support: default`, so `otlp_grpc` uses the helper queue default unchanged: `enabled: true`, `num_consumers: 10`, `queue_size: 1000`, `sizer: requests`, `block_on_overflow: false`, `wait_for_result: false`, no `storage` (in-memory), and batching off unless a `batch` block or the alpha `pkg.exporterhelper.queueBatchEnabled` gate turns it on (`flush_timeout: 200ms`, `sizer: items`, `min_size: 8192`, `max_size: 0`). What each key does, the `batch` and `partition` sub-keys, the persistent queue via [`file_storage`](../file_storage/README.md), and validation: [`sending_queue`](../../references/exporterhelper.md#sending_queue).

## Validation summary

| Condition | Error / rule | When |
|-----------|--------------|------|
| empty `endpoint` | `requires a non-empty "endpoint"` | config validation |

Queue, batch, retry, and timeout errors are in the shared [validation table](../../references/exporterhelper.md#validation-and-startup-errors).
