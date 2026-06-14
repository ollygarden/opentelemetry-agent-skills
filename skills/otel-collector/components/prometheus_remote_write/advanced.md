# `prometheus_remote_write` exporter: advanced use-cases

## Write-Ahead Log (WAL)

The WAL persists outgoing metrics to disk before sending, so a crash or backend outage doesn't lose in-flight data. It is **off** unless `wal.directory` is set:

```yaml
exporters:
  prometheus_remote_write:
    endpoint: https://my-cortex:7900/api/v1/push
    wal:
      directory: ./prom_rw     # setting this enables the WAL
      buffer_size: 100         # default 300
      truncate_frequency: 45s  # default 1m
      lag_record_frequency: 15s
```

When the WAL is enabled, `max_batch_size_bytes` is **ignored** — the WAL `buffer_size` / `truncate_frequency` govern how much is read and how often it is truncated.

This WAL is the **exporter's own** on-disk buffer (backed by a `wal` library, stored under `<directory>/prom_remotewrite`). It is **not** the [`file_storage`](../file_storage/README.md) extension and does not reference a `storage` ID — don't confuse the two.

## RW2 / `protobuf_message` and the feature gate

`protobuf_message` selects the wire format:

- `prometheus.WriteRequest` — Remote Write 1.0 (the default).
- `io.prometheus.write.v2.Request` — Remote Write 2.0: more efficient, always includes metadata, and adds created-timestamp and native-histogram support.

RW2 is only honored when the feature gate `exporter.prometheusremotewritexporter.enableSendingRW2` (alpha, since v0.125.0) is enabled — otherwise `Validate()` errors. Per upstream, **PRW 2.0 is "In Development", only partially implemented, and not ready for usage.** Your backend must also support PRW 2.0.

RW2 also unlocks the UTF-8 `translation_strategy` values (`NoUTF8EscapingWithSuffixes`, `NoTranslation`), which are rejected under RW1.

## `resource_to_telemetry_conversion` vs `target_info`

By default, resource attributes are **not** copied onto each series — they are emitted on a separate `target_info` metric (`target_info.enabled` defaults to `true`). To use them in PromQL, join on the identity labels:

```promql
my_metric * on (job, instance) group_left target_info{k8s_namespace_name="my-namespace"}
```

Two ways to bring attributes onto the metric labels directly:

- **Copy the few you need** with the [`transform`](../transform/README.md) processor (datapoint context), keeping cardinality bounded:

  ```yaml
  processors:
    transform:
      metric_statements:
        - context: datapoint
          statements:
            - set(attributes["namespace"], resource.attributes["k8s.namespace.name"])
  ```

- **Copy them all** by enabling conversion on the exporter (simplest, but can explode label cardinality). Use `exclude_service_attributes` to keep `service.name` / `service.instance.id` / `service.namespace` off the labels (they already map to `job` / `instance`):

  ```yaml
  exporters:
    prometheus_remote_write:
      endpoint: https://my-cortex:7900/api/v1/push
      resource_to_telemetry_conversion:
        enabled: true
        exclude_service_attributes: true
  ```

## `external_labels`

`external_labels` attaches a fixed set of labels to **every** series — useful for tagging a cluster, region, or collector identity. Values may start with the reserved `__` prefix:

```yaml
exporters:
  prometheus_remote_write:
    endpoint: https://my-cortex:7900/api/v1/push
    external_labels:
      cluster: prod-eu
      collector: gateway-1
```

## Multiple workers: the `EnableMultipleWorkers` gate

The feature gate `exporter.prometheusremotewritexporter.EnableMultipleWorkers` (alpha, since v0.118.0) changes how `num_consumers` and `max_batch_request_parallelism` behave:

- **Gate off (default):** `remote_write_queue.num_consumers` defaults to `5` and drives **request** parallelism. The factory logs a migration warning if you set it to anything other than `5`.
- **Gate on:** `num_consumers` becomes the **queue-worker** count (defaulting to `1`), handling batches from the queue concurrently, while `max_batch_request_parallelism` drives parallelism for splitting a single batch larger than `max_batch_size_bytes`.

> **Out-of-order caveat.** With the gate on and `num_consumers > 1` (or `max_batch_request_parallelism > 1`), the temporal ordering of samples is no longer guaranteed. Vanilla Prometheus rejects out-of-order samples — enable `tsdb.out_of_order_time_window` on the backend, or set `max_batch_request_parallelism: 1`. Other backends (Thanos / Mimir / VictoriaMetrics) have their own settings.

```yaml
exporters:
  prometheus_remote_write:
    endpoint: https://my-cortex:7900/api/v1/push
    max_batch_request_parallelism: 1   # safe for backends that reject out-of-order samples
```

## Choosing a `translation_strategy`

`translation_strategy` supersedes the deprecated `add_metric_suffixes` when set (see [configuration.md](configuration.md#translation_strategy)).

```yaml
exporters:
  prometheus_remote_write:
    endpoint: https://my-cortex:7900/api/v1/push
    translation_strategy: UnderscoreEscapingWithSuffixes   # classic Prometheus compatibility
```

- `UnderscoreEscapingWithSuffixes` / `UnderscoreEscapingWithoutSuffixes` — escape names to classic Prometheus form, with or without type/unit suffixes. Both work under RW1.
- `NoUTF8EscapingWithSuffixes` / `NoTranslation` — keep UTF-8 / pass through unaltered. **Require PRW 2.0** (rejected under RW1).
