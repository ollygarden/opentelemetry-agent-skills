# `otlp_grpc` exporter: configuration

All keys live under the exporter instance — `exporters: { otlp_grpc: { … } }` or, via the deprecated alias, `exporters: { otlp: { … } }`. Facts below trace to core **v1.66.0 / v0.160.0** source (`exporter/otlpexporter/config.go` + `factory.go`, `config/configgrpc/configgrpc.go` `ClientConfig`, `config/configtls`, `config/configretry/backoff.go`, and `exporter/exporterhelper/internal/queuebatch/config.go` + `queue_sender.go`); `sending_queue.batch.partition` rows re-checked at v1.68.0 / v0.162.0.

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

The buffer between the pipeline and the gRPC sender. **Enabled by default.** Its optional `batch`
sub-block uses defaults from `NewDefaultQueueConfig`, but batching itself is disabled by default
unless the `pkg.exporterhelper.queueBatchEnabled` feature gate is enabled.

| Key | Type | Default | Meaning |
|-----|------|---------|---------|
| `enabled` | bool | `true` | Whether to enqueue before sending. |
| `num_consumers` | int | `10` | Concurrent senders draining the queue. |
| `queue_size` | int | `1000` | Max items buffered, in `sizer` units. Must be > 0. |
| `sizer` | string | `requests` | Unit for `queue_size`: `requests`, `items`, or `bytes`. |
| `block_on_overflow` | bool | `false` | `false` → overflow returns a retryable error immediately; `true` → enqueue blocks until space frees. |
| `wait_for_result` | bool | `false` | Block the caller until the export result is known. **Not supported with a persistent `storage` queue.** |
| `storage` | component ID | — | A `file_storage` extension ID; turns the queue **persistent** (survives restarts). |
| `batch` | object | disabled | Built-in batching (see [batch](#sending_queuebatch)). Add `batch: {}` to enable it with defaults, or enable the migration gate globally. |

> `storage` references a [`file_storage`](../file_storage/README.md) extension. Persistence survives Collector restarts at the cost of disk I/O; see [advanced.md](advanced.md).

### `sending_queue.batch`

When enabled, flushes at `flush_timeout` or when `min_size` is reached, whichever comes first.
Add `batch: {}` beneath `sending_queue` to opt in with these defaults. The alpha
`pkg.exporterhelper.queueBatchEnabled` feature gate also enables batching by default for exporters
using `NewDefaultQueueConfig`; this is phase 1 of the Collector batching migration. A separate
`batch` processor remains supported for pipeline-level batching.

| Key | Type | Default | Meaning |
|-----|------|---------|---------|
| `flush_timeout` | duration | `200ms` | Max time a partial batch waits before flushing. Must be > 0. |
| `sizer` | string | `items` | Unit for the batch sizes: only `items` or `bytes`. |
| `min_size` | int | `8192` | Flush once the batch reaches this many items (the soft trigger). |
| `max_size` | int | `0` (unlimited) | Hard cap; when > 0, a batch is split to never exceed it. |
| `partition.metadata_keys` | list of string | — | One batcher per distinct combination of these client-metadata values. |
| `partition.cache_size` | int | `10000` | Max active partition batchers (LRU); must be > 0. Exposed as `otelcol_exporter_queue_batch_partition_cache_size` / `_capacity`. Core v0.162.0+. |
| `partition.idle_timeout` | duration | `90s` | How long an empty partition lives before removal; must be > 0. Configurable (and default raised to 90s) in core v0.162.0. |

## Validation summary

| Condition | Error / rule | When |
|-----------|--------------|------|
| empty `endpoint` | `requires a non-empty "endpoint"` | config validation |
| `sending_queue.batch.flush_timeout` ≤ 0 | rejected | config validation |
| `batch.sizer` not `items`/`bytes` | rejected | config validation |
| `batch.min_size` > `queue_size` (matching sizers) | rejected | config validation |
| `batch.max_size` < `min_size` (when `max_size` > 0) | rejected | config validation |
| `queue_size` ≤ 0 | `` `queue_size` must be positive `` | config validation |
| `num_consumers` ≤ 0 | `` `num_consumers` must be positive `` | config validation |
