# `failover`: configuration

Facts trace to contrib **v0.165.1** [`connector/failoverconnector`](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.165.1/connector/failoverconnector) and core **v0.165.0** [`exporter/exporterhelper/internal`](https://github.com/open-telemetry/opentelemetry-collector/tree/v0.165.0/exporter/exporterhelper/internal). Version notes come from the [contrib v0.165.1 changelog](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/v0.165.1/CHANGELOG.md). Values hold for the named releases; check the current release before relying on them.

## Typical config

This is the upstream README layout, with the queue in the connector and the exporters' `sending_queue` disabled:

```yaml
connectors:
  failover:
    priority_levels:
      - [traces/primary]
      - [traces/secondary]
    retry_interval: 1m

exporters:
  otlp_grpc/primary:
    endpoint: primary.example:4317
    sending_queue:
      enabled: false
  otlp_grpc/secondary:
    endpoint: secondary.example:4317
    sending_queue:
      enabled: false

service:
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [failover]          # connector as exporter of the input pipeline
    traces/primary:
      receivers: [failover]          # connector as receiver of each priority pipeline
      exporters: [otlp_grpc/primary]
    traces/secondary:
      receivers: [failover]
      exporters: [otlp_grpc/secondary]
```

The exporters keep the default [`retry_on_failure`](../otlp_exporter/configuration.md#retry_on_failure), so a batch fails over only after its retries end (up to `max_elapsed_time`). Disable retry or shorten `max_elapsed_time` to fail over sooner ([why](quirks.md#async-stages-in-a-priority-pipeline-hide-failures)).

## Configuration reference

| Key | Type | Default | Meaning |
|-----|------|---------|---------|
| `priority_levels` | list of lists of pipeline IDs | — (**required**, non-empty) | Ordered levels, highest priority first. Each inner list is one level whose pipelines all receive every batch. Each ID must name a pipeline of the same signal that lists the connector under `receivers`. |
| `retry_interval` | duration | `10m` | Period of the probe ticker ([failback](#failback)). Must be positive. |
| `sending_queue` | exporterhelper queue block | enabled | Queue in front of the router ([below](#connector-sending_queue)). Keys, defaults and validation are those of the exporter [`sending_queue`](../otlp_exporter/configuration.md#sending_queue). `enabled: false` removes it. |
| `condition.error.contains` | list of non-empty strings | unset — every error fails over | Case-insensitive substrings matched against the downstream error message. Only a matching error fails over ([ignored in v0.162.x](quirks.md#conditionerrorcontains-is-ignored-in-v0162x)). |

Error strings and fixes, including unsupported keys: [validation errors](quirks.md#validation-errors--fixes).

## Failover

1. Each batch goes to the current level, initially level 0. The level's consume call sends the batch to every pipeline in the level and joins their errors ([`fanoutconsumer`](https://github.com/open-telemetry/opentelemetry-collector/blob/v0.165.0/internal/fanoutconsumer/traces.go)). Pipelines that accept the batch keep it, even when another pipeline in the level fails.
2. `nil` means delivered. With no `condition`, any error marks the level failed; with `condition`, only a matching one does. The current level then moves down by one, and the **same batch** goes to the new current level. This repeats until a level accepts it.
3. A non-matching error is returned to the caller unchanged, and the level stays current.
4. Concurrent failures on one level move it down only once.
5. When the last level fails, the batch gets `All provided pipelines return errors`. So does every later batch, **with no export attempt**, until a probe succeeds.
6. Since v0.164.0, if the level's consumer reports that it mutates data, the connector copies the batch before each attempt. Later levels and probes then receive the original batch.

## Failback

1. A failover while no ticker is running starts one at `retry_interval`. That includes a failover right after level 0 recovers (since v0.164.0). Each tick arms one probe. A tick that arrives while a probe is already armed is dropped.
2. The next batch after a tick is the probe. It goes to level 0, then 1, and so on up to the level above the current one. The first level that returns `nil` becomes current. With `condition` set, so does the first level whose error does not match. That error is returned, and the batch goes nowhere else.
3. If no higher level accepts the probe, the batch goes to the current level, or is rejected when every level has failed.
4. Restoring level 0 stops the ticker. Restoring an intermediate level keeps it running.
5. Probes need traffic: with no incoming batches, nothing is probed. A probe pays the latency of every failing higher level, including its timeouts and synchronous retries, before it is delivered.

## Connector `sending_queue`

The factory wraps the router in an exporterhelper queue built from `NewDefaultQueueConfig`. It keeps the exporter [`sending_queue`](../otlp_exporter/configuration.md#sending_queue) defaults, adds no retry, and sets no deadline on the downstream call (`timeout: 0`, since v0.155.0).

| With the queue enabled (default) | With `sending_queue.enabled: false` |
|----------------------------------|--------------------------------------|
| The input pipeline gets `nil` at enqueue. Routing runs in the queue's `num_consumers` goroutines. | Routing runs in the caller's goroutine, under the caller's context deadline. |
| A routing error (`All provided pipelines return errors`, or a non-matching `condition` error) drops the batch. It is logged as `Exporting failed. Dropping data.` with `otelcol.component.id: failover`. | The error returns to the input pipeline: the OTLP receiver counts it in `otelcol_receiver_refused_*`, and a gRPC client sees `Unavailable desc = All provided pipelines return errors`. |
| A full queue returns `sending queue is full` to the input pipeline. | — |
| Telemetry uses `exporter="failover"`: `otelcol_exporter_queue_size`, `_queue_capacity`, `_sent_*`, `_send_failed_*`. | No `exporter="failover"` series. |

`sending_queue.storage` makes the queue persistent ([advanced](advanced.md#persistent-connector-queue)).
