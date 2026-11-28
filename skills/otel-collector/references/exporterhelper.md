# Exporter queue, retry, and timeout (`exporterhelper`)

Exporters built on core's `exporterhelper` share the `sending_queue`, `retry_on_failure`, and top-level `timeout` keys. This page owns what those keys mean, the helper's own defaults, their validation, and how the in-memory and persistent queues behave. Facts trace to core [`exporter/exporterhelper` at v0.165.0](https://github.com/open-telemetry/opentelemetry-collector/tree/v0.165.0/exporter/exporterhelper) (the queue, batch, retry, and timeout code under `internal/`), [`config/configretry`](https://github.com/open-telemetry/opentelemetry-collector/blob/v0.165.0/config/configretry/generated_config.go), and [`config/configoptional`](https://github.com/open-telemetry/opentelemetry-collector/blob/v0.165.0/config/configoptional/optional.go); the exporter and receiver examples come from the same release's `exporter/otlpexporter`, `exporter/otlphttpexporter`, `exporter/debugexporter`, and `receiver/otlpreceiver`, and `file_storage` facts from contrib v0.165.1. Values hold for the named releases; check the current release before relying on them.

## Defaults belong to each exporter

The defaults on this page are the helper's (`NewDefaultQueueConfig`, `configretry.NewDefaultBackOffConfig`, `NewDefaultTimeoutConfig`). Each exporter's factory chooses its own: it may use them, override fields, disable a block, or not expose it at all. An exporter that passes the helper no retry settings never retries, and one that passes no queue settings has no queue. So an exporter's values come from its own page in the [Component index](../SKILL.md#component-index).

For an exporter without a page, read its factory's `createDefaultConfig` at the release tag. When its `metadata.yaml` declares `sending_queue`, its generated `documentation.md` has a `## Sending Queue` block that marks each overridden default `# OVERRIDE`. For example, core v0.165.0's [`debug` exporter](https://github.com/open-telemetry/opentelemetry-collector/blob/v0.165.0/exporter/debugexporter/documentation.md) disables its queue, and the `otlp_http` exporter switches the helper timeout off and applies its HTTP client's `timeout` (`30s`) instead.

`sending_queue` and `sending_queue.batch` are optional blocks: writing any of their keys enables a block whose default is disabled, and `enabled: false` disables it. `retry_on_failure` is a plain struct, so only its `enabled` key turns it on or off.

## `sending_queue`

| Key | Helper default | Meaning |
|-----|----------------|---------|
| `enabled` | `true` | `false` removes the queue; see [What the caller sees](#what-the-caller-sees). |
| `num_consumers` | `10` | Concurrent exports from the queue; see [Concurrency, order, overflow, and shutdown](#concurrency-order-overflow-and-shutdown). |
| `queue_size` | `1000` | Capacity in `sizer` units. A request counts from enqueue until its export finishes, including while it is being sent or retried. |
| `sizer` | `requests` | `requests` (each incoming request counts 1), `items` (spans, metric data points, log records), or `bytes` (serialized size). |
| `block_on_overflow` | `false` | On a full queue, `false` rejects the data at once with `sending queue is full`; `true` waits for space until the caller's context ends. |
| `wait_for_result` | `false` | The caller waits for the export to finish and receives its error. Not allowed with `storage`. |
| `storage` | none | A storage-extension ID. Replaces the in-memory queue with a [persistent queue](#persistent-queue-storage). |
| `batch` | disabled | Batching while draining; see [`sending_queue.batch`](#sending_queuebatch). |

### What the caller sees

What the exporter's consume call returns to the component in front of it (a processor or receiver):

| Setting | Returns |
|---------|---------|
| queue on (default) | `nil` once the request is enqueued; a later export failure is logged and counted in `otelcol_exporter_send_failed_*`, not returned |
| queue full, `block_on_overflow: false` | `sending queue is full` at once |
| queue full, `block_on_overflow: true` | `nil` once space frees, or the context error if the caller's context ends first; with no deadline it can wait indefinitely |
| `wait_for_result: true` | the export's final result, after retries |
| queue off (`enabled: false`) | the export's final result; with retry on, the caller blocks until success, `max_elapsed_time`, or its context deadline |

### Concurrency, order, overflow, and shutdown

- With batching off, `num_consumers` workers each take one request and export it, so at most that many exports run at once per exporter instance and signal, and a request waiting between retries keeps its worker busy. With batching on and no `partition.metadata_keys`, one reader feeds the batcher and up to `num_consumers` batches export at once.
- Both queues hand out requests first in, first out. With several workers, exports finish out of order.
- Data the queue rejects goes back up the pipeline as a retryable error and never reaches the retry logic; an `otlp` receiver passes it to its client as gRPC `Unavailable` or HTTP 503. It is counted in `otelcol_exporter_enqueue_failed_spans`, `_metric_points`, and `_log_records`. `otelcol_exporter_queue_size` and `otelcol_exporter_queue_capacity` report the fill level.
- With the `items` or `bytes` sizer, a single request larger than `queue_size` never fits, however often it is retried. The in-memory queue rejects it with `element size too large`, even with `block_on_overflow`. Without `block_on_overflow` the persistent queue rejects it with `sending queue is full`; with it, it waits until the caller's context ends.
- On shutdown the in-memory queue drains: each remaining request gets one attempt, because retries stop first, and requests that fail are dropped. A crash loses everything still in the in-memory queue.

### `sending_queue.batch`

Batching is disabled unless a `batch` block is present (`batch: {}` takes the defaults) or the alpha `pkg.exporterhelper.queueBatchEnabled` feature gate (since v0.158.0) is enabled, which turns it on for exporters whose factory keeps the helper default. Batching moves each queued request's resources into the current batch of its partition. A batch that merged requests is exported with a new context holding span links to them and, with `partition.metadata_keys`, those keys' values from the first request; other client metadata and auth context are dropped unless the exporter supplies its own context merge. Without `partition.metadata_keys` or an exporter-supplied partitioner there is one partition, so requests an exporter split before queuing (for example by a resource attribute) are merged back; `max_size` re-splits by size only. A `batch` processor in the same pipeline triggers a startup warning; see [Pipeline placement](../SKILL.md#pipeline-placement).

| Key | Helper default | Meaning |
|-----|----------------|---------|
| `flush_timeout` | `200ms` | Flush a partial batch after this long. |
| `min_size` | `8192` | Flush once the batch reaches this size. |
| `max_size` | `0` (no limit) | Split batches larger than this. |
| `sizer` | `items` | `items` or `bytes` only. When `sending_queue.sizer` is set explicitly and `batch` has keys but no `sizer`, the batch inherits the queue's sizer, so `sizer: requests` with a non-empty `batch` fails validation. |
| `partition.metadata_keys` | `[]` | One batcher per distinct combination of these client-metadata values; keys are case-insensitive. |
| `partition.cache_size` | `10000` | Maximum active partition batchers; the least recently used is flushed and removed. Reported as `otelcol_exporter_queue_batch_partition_cache_size` and `_capacity`. |
| `partition.idle_timeout` | `90s` | How long an empty partition lives before removal. |

## `retry_on_failure`

| Key | Helper default | Meaning |
|-----|----------------|---------|
| `enabled` | `true` | Retry retryable errors with exponential backoff. |
| `initial_interval` | `5s` | Wait before the first retry; `0` keeps every backoff interval at `0`. |
| `randomization_factor` | `0.5` | Each interval varies by ± this fraction (0–1). |
| `multiplier` | `1.5` | Interval growth per retry. |
| `max_interval` | `30s` | Cap on the interval before randomization, so with the defaults one wait can reach 45s. |
| `max_elapsed_time` | `5m` | Retry budget per request; `0` retries forever. |

- Permanent errors are not retried; the error is final.
- When the backend asks for a delay (gRPC `RetryInfo` in `otlp_grpc`, `Retry-After` in `otlp_http`), that delay replaces a shorter backoff interval; at core v0.165.0 `max_interval` does not cap it.
- When the next attempt would start after `max_elapsed_time`, the helper stops. With a queue the request is dropped, logged as `Exporting failed. Dropping data.` with `dropped_items`; without one the error goes back to the caller, logged as `Exporting failed. Rejecting data.`
- With `max_elapsed_time: 0`, a failing request keeps its worker. In a long outage every worker ends up retrying, the queue fills to `queue_size`, and new data is rejected, or waits with `block_on_overflow`. The queue never grows past `queue_size`.

## `timeout`

A top-level key, not nested under `sending_queue`. Helper default `5s`; it bounds each send attempt, not the whole retry loop. `0` disables it.

## Persistent queue (`storage`)

Set `sending_queue.storage` to a storage extension, usually [`file_storage`](../components/file_storage/README.md), and list that extension under `service.extensions`. Each request is written to storage on enqueue and deleted once its export finishes.

- **One queue per exporter instance and signal, on one Collector.** The queue asks the extension for a client keyed by exporter ID and signal; `file_storage` keeps one file per exporter and signal in its local `directory` ([naming](../components/file_storage/configuration.md#per-consumer-files-and-naming)). A backlog drains only through the Collector that wrote it: added replicas start with empty queues and take only new traffic. Renaming the exporter, including switching from a deprecated alias to the new type name (`otlp` → `otlp_grpc`, `otlphttp` → `otlp_http`), opens a new, empty queue and leaves the old backlog unread ([core#14988](https://github.com/open-telemetry/opentelemetry-collector/issues/14988)).
- **First in, first out.** Requests are read in write order, so data arriving after an outage waits behind the backlog. A request to prioritize new data is open as [core#12788](https://github.com/open-telemetry/opentelemetry-collector/issues/12788).
- **Drain concurrency.** Exports run as described in [Concurrency, order, overflow, and shutdown](#concurrency-order-overflow-and-shutdown). Every enqueue, read, and acknowledgement is a storage transaction made while holding the queue's lock, so raising `num_consumers` adds concurrent exports but not concurrent storage access.
- **Capacity.** As for the in-memory queue, `queue_size` includes in-flight requests. A failed storage write, such as a full disk or a `file_storage` file at its `max_size`, also rejects the data.
- **Retry budget.** Storage protects queued data across restarts, not past a spent retry budget: requests read during an outage are still dropped after `max_elapsed_time` ([`retry_on_failure`](#retry_on_failure)).
- **Restart.** On shutdown the queue stops reading, and queued requests stay in storage. Requests that were read but not finished — in flight when the process was killed, or with a retry interrupted by a graceful shutdown — go to the back of the queue at the next start and are sent again. A request the backend had already accepted is then delivered twice. Client metadata and span context are stored with each request; context set by auth extensions is not.

Upstream issues report the following with `file_storage`. They are observations from those reporters' environments, not guarantees:

- [contrib#41204](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/41204) (closed as not planned): draining slows sharply once the bbolt file no longer fits the OS page cache (about 19 GB: 3–4 minutes cached, about 40 minutes uncached).
- [contrib#52868](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/52868) (open): contrib v0.165.1 opens bbolt with `NoFreelistSync: true` ([`client.go`](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.165.1/extension/storage/filestorage/client.go)), so each open reads the file to rebuild its free list and open time grows with the backlog; a 5.6 GB queue on network storage outlasted a two-minute startup probe, so the Collector never started draining.
- [contrib#44469](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/44469) (open): memory grows with the memory-mapped file during an outage and spikes into OOM kills when a large backlog drains.

## Validation and startup errors

| Condition | Error | When |
|-----------|-------|------|
| `num_consumers` ≤ 0 | `` `num_consumers` must be positive `` | config validation |
| `queue_size` ≤ 0 | `` `queue_size` must be positive `` | config validation |
| `sizer` not `requests`/`items`/`bytes` | `invalid sizer: "<value>"` | config load |
| `wait_for_result: true` with `storage` | `` `wait_for_result` is not supported with a persistent queue configured with `storage` `` | config validation |
| batch on, sizers equal, `min_size` > `queue_size` (e.g. `sizer: items`, `queue_size` < 8192) | `` `min_size` must be less than or equal to `queue_size` `` | config validation |
| `batch.sizer` (set or inherited) not `items`/`bytes` | `` `batch` supports only `items` or `bytes` sizer, found "<value>" `` | config validation |
| `batch.flush_timeout` ≤ 0 | `` `flush_timeout` must be positive, found <value> `` | config validation |
| `batch.max_size` > 0 and < `min_size` | `` `max_size` (<max>) must be greater or equal to `min_size` (<min>) `` | config validation |
| duplicate `partition.metadata_keys` entry | `duplicate entry in metadata_keys: "<key>" (case-insensitive)` | config validation |
| `batch.min_size` or `batch.max_size` < 0 | `` `min_size` must be non-negative, found <value> `` (or `max_size`) | config validation |
| `partition.cache_size` or `partition.idle_timeout` ≤ 0 | `` `cache_size` must be positive, found <value> `` / `` `idle_timeout` must be positive, found <value> `` | config validation |
| `max_elapsed_time` > 0 but below `initial_interval` or `max_interval` | `'max_elapsed_time' must not be less than 'initial_interval'` (or `'max_interval'`) | config validation |
| `randomization_factor` > 1, or a negative retry value or `timeout` | `randomization_factor value must be less than or equal to 1` / `<key> value must be greater than or equal to 0` | config validation |
| `storage` names an extension that is not loaded | `no storage client extension found` | startup |
| `storage` names an extension that is not a storage extension | `requested extension is not a storage extension` | startup |
