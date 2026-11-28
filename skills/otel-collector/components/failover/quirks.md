# `failover`: known quirks

Release basis: [configuration.md](configuration.md).

## Async stages in a priority pipeline hide failures

The connector sees only what the priority pipeline's first consumer returns synchronously. A stage that hands the batch to another goroutine returns before the export result exists. Rows are core v0.165.0 behavior ([`memory_queue.go`](https://github.com/open-telemetry/opentelemetry-collector/blob/v0.165.0/exporter/exporterhelper/internal/queue/memory_queue.go), [`retry_sender.go`](https://github.com/open-telemetry/opentelemetry-collector/blob/v0.165.0/exporter/exporterhelper/internal/retry_sender.go), [`batch_processor.go`](https://github.com/open-telemetry/opentelemetry-collector/blob/v0.165.0/processor/batchprocessor/batch_processor.go)).

| Stage in the priority pipeline (destination down) | What the connector receives | When it fails over |
|---------------------------------------------------|-----------------------------|--------------------|
| Exporter, `sending_queue` and `retry_on_failure` disabled | The export error from the first attempt. | On the first failed batch. |
| Exporter, `sending_queue` disabled, `retry_on_failure` enabled | The error after retries end (`no more retries left`), or at once for a permanent error. | After up to `max_elapsed_time` per batch. |
| Exporter, `sending_queue` enabled (default), `block_on_overflow: false` (default) | `nil` for every batch the queue accepts. `sending queue is full` once it is full. Requests still being retried count toward `queue_size`. | Only once the queue is full, which needs consumers held by retries or slow sends; with `retry_on_failure` disabled and a fast-failing destination (connection refused) it never fills. Accepted batches are retried there (if `retry_on_failure` is enabled), then dropped with `Exporting failed. Dropping data.`. |
| Exporter, `sending_queue` enabled, `block_on_overflow: true` | Nothing: the enqueue blocks until space frees or the caller's context ends. | Not while blocked. |
| Exporter, `sending_queue.wait_for_result: true` | The export result, after any retries. | As for a disabled queue. |
| `batch` processor | `nil`: `consume` hands the batch to the processor's own goroutine. | Never for export errors behind it. |
| A processor that returns an error | That error. | On that batch. |

Behind the default connector queue the downstream call has no deadline, so a stage that blocks or retries holds a connector queue consumer until it finishes. The [typical config](configuration.md#typical-config) queues in the connector and disables the exporters' queues.

## Recovery after a total outage waits for a probe

Recovery takes up to one `retry_interval` after a destination returns, plus the wait for the next batch; until then, with every level failed, batches are rejected untried ([failover](configuration.md#failover) step 5, [failback](configuration.md#failback)).

## Failovers are not logged at `info`

The connector logs a failover only at `debug` level, and only when `condition` is set. A failover is visible in the failing exporter's log (`Exporting failed. Rejecting data.` when its queue is disabled) and in its `otelcol_exporter_send_failed_*` metrics. Failed probes appear the same way.

## `condition.error.contains` is ignored in v0.162.x

Contrib v0.162.x accepts the key but fails over on every error. Matching is honored, and empty strings are rejected, since v0.163.0.

## Validation errors → fixes

| Error | Cause | Fix |
|-------|-------|-----|
| `No pipelines are defined in the priority list` | `priority_levels` missing or empty | List at least one level. |
| `Retry interval must be positive` | `retry_interval` is `0` or negative | Use a positive duration, or omit it for `10m`. |
| `no condition is defined` | `condition: {}` | Add `error.contains`, or remove `condition`. |
| `error condition must define non-empty 'contains' strings` | `contains` empty, or holds `""` | List non-empty substrings. |
| `'failoverconnector.Config' has invalid keys: <key>` | `retry_on_failure`, `timeout`, or `retry_gap`/`max_retries` (removed in v0.157.0) | Remove the key. |
| `failed to build pipelines: Error registering consumer` | A `priority_levels` ID is not a pipeline of the connector's signal that lists the connector under `receivers` | Fix the ID, add the connector to that pipeline's `receivers`, or use one [instance per signal](advanced.md#one-named-instance-per-signal). |

A pipeline that lists the connector under `receivers` but is missing from `priority_levels` passes validation. It receives nothing from the connector.
