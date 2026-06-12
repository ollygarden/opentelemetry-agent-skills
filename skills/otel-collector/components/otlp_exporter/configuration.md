# `otlp_grpc` exporter: configuration

All keys live under the exporter instance — `exporters: { otlp_grpc: { … } }` or, via the deprecated alias, `exporters: { otlp: { … } }`. Facts below trace to core **v1.60.0** source (`exporter/otlpexporter/config.go` + `factory.go`, `config/configgrpc/configgrpc.go` `ClientConfig`, `config/configtls`, `config/configretry/backoff.go`, and `exporter/exporterhelper/internal/queuebatch/config.go` + `queue_sender.go`).

## Top-level (gRPC client) keys

| Key | Type | Default | Meaning |
|-----|------|---------|---------|
| `endpoint` | string | — (**required**) | gRPC target (`host:port`, gRPC naming syntax). Accepts `http://`, `https://`, `dns://` prefixes (stripped internally for validation). |
| `compression` | string | `gzip` | `gzip`, `snappy`, `zstd`, or none/empty to disable. Default set by the factory. |
| `tls` | object | — | gRPC client TLS (see [tls](#tls)). |
| `headers` | map[string]string | — | Static headers added to every gRPC request. |
| `timeout` | duration | `5s` | Per-attempt send timeout. |
| `keepalive` | object | — | gRPC client keepalive: `time`, `timeout`, `permit_without_stream`. |
| `read_buffer_size` | int | 0 (gRPC default) | gRPC read buffer. The factory leaves this unset (the exporter reads almost nothing). |
| `write_buffer_size` | int | `524288` (512 KiB) | gRPC write buffer. |
| `wait_for_ready` | bool | `false` | Block RPCs until the connection is ready instead of failing fast. |
| `balancer_name` | string | `round_robin` | gRPC client-side load-balancing policy across resolved addresses. |
| `authority` | string | — | Overrides the `:authority` pseudo-header. |
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

`configretry.BackOffConfig` — exponential backoff with jitter. After `max_elapsed_time` the data is **dropped**.

| Key | Type | Default | Meaning |
|-----|------|---------|---------|
| `enabled` | bool | `true` | Whether to retry failed sends. |
| `initial_interval` | duration | `5s` | Wait before the first retry. |
| `randomization_factor` | float | `0.5` | Jitter applied to each interval. |
| `multiplier` | float | `1.5` | Growth factor per retry. |
| `max_interval` | duration | `30s` | Cap on the per-retry interval. |
| `max_elapsed_time` | duration | `5m` | Total retry budget; after this, the data is dropped. Set `0` to retry forever. |

## `sending_queue`

The buffer between the pipeline and the gRPC sender. **Enabled by default**, and it carries the built-in batching (`batch` below). Defaults from `NewDefaultQueueConfig`.

| Key | Type | Default | Meaning |
|-----|------|---------|---------|
| `enabled` | bool | `true` | Whether to enqueue before sending. |
| `num_consumers` | int | `10` | Concurrent senders draining the queue. |
| `queue_size` | int | `1000` | Max items buffered, in `sizer` units. Must be > 0. |
| `sizer` | string | `requests` | Unit for `queue_size`: `requests`, `items`, or `bytes`. |
| `block_on_overflow` | bool | `false` | `false` → overflow returns a retryable error immediately; `true` → enqueue blocks until space frees. |
| `wait_for_result` | bool | `false` | Block the caller until the export result is known. **Not supported with a persistent `storage` queue.** |
| `storage` | component ID | — | A `file_storage` extension ID; turns the queue **persistent** (survives restarts). |
| `batch` | object | present by default | Built-in batching (see [batch](#sending_queuebatch)). |

> `storage` references a `file_storage` extension, which this skill does not yet have its own page for. Persistence survives Collector restarts at the cost of disk I/O; see [advanced.md](advanced.md).

### `sending_queue.batch`

Present by default — flushes at `flush_timeout` or when `min_size` is reached, whichever comes first. **This is why you don't add a separate `batch` processor.**

| Key | Type | Default | Meaning |
|-----|------|---------|---------|
| `flush_timeout` | duration | `200ms` | Max time a partial batch waits before flushing. Must be > 0. |
| `sizer` | string | `items` | Unit for the batch sizes: only `items` or `bytes`. |
| `min_size` | int | `8192` | Flush once the batch reaches this many items (the soft trigger). |
| `max_size` | int | `0` (unlimited) | Hard cap; when > 0, a batch is split to never exceed it. |
| `partition.metadata_keys` | list of string | — | One batcher per distinct combination of these client-metadata values. |

## Validation summary

| Condition | Error / rule | When |
|-----------|--------------|------|
| empty `endpoint` | `requires a non-empty "endpoint"` | config validation |
| `sending_queue.batch.flush_timeout` ≤ 0 | rejected | config validation |
| `batch.sizer` not `items`/`bytes` | rejected | config validation |
| `batch.min_size` > `queue_size` (matching sizers) | rejected | config validation |
| `batch.max_size` < `min_size` (when `max_size` > 0) | rejected | config validation |
| `queue_size` ≤ 0 | rejected | config validation |
