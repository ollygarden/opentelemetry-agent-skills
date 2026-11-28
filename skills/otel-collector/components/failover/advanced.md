# `failover`: advanced use-cases

Release basis: [configuration.md](configuration.md).

## Several pipelines in one level

One failing pipeline fails the whole level, so the next level also receives a batch the other pipelines already kept ([failover](configuration.md#failover) step 1).

```yaml
connectors:
  failover:
    priority_levels:
      - [traces/backend_a, traces/backend_b]
      - [traces/archive]
```

## One named instance per signal

Levels cannot mix signals ([`Error registering consumer`](quirks.md#validation-errors--fixes)), so each signal needs its own instance:

```yaml
connectors:
  failover/traces:
    priority_levels: [[traces/primary], [traces/backup]]
  failover/logs:
    priority_levels: [[logs/primary], [logs/backup]]

service:
  pipelines:
    traces:         { receivers: [otlp], exporters: [failover/traces] }
    traces/primary: { receivers: [failover/traces], exporters: [otlp_grpc/primary] }
    traces/backup:  { receivers: [failover/traces], exporters: [otlp_grpc/backup] }
    # logs, logs/primary, logs/backup: same shape with failover/logs
```

## Fail over only on some errors

```yaml
connectors:
  failover:
    priority_levels: [[traces/primary], [traces/backup]]
    condition:
      error:
        contains: ["connection refused", "Unavailable"]
```

Other errors, such as a rejected payload, do not fail over ([failover](configuration.md#failover) step 3; probes: [failback](configuration.md#failback) step 2).

## Persistent connector queue

`sending_queue.storage` takes a [`file_storage`](../file_storage/advanced.md#persistent-exporter-queue-survives-restart) extension ID, as on an exporter. Batches waiting in the connector queue then survive a restart.

```yaml
connectors:
  failover:
    priority_levels: [[logs/primary], [logs/backup]]
    sending_queue:
      storage: file_storage   # declared under extensions and service.extensions
```
