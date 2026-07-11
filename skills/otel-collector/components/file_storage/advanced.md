# `file_storage`: advanced use-cases

## Persistent exporter queue (survives restart)

By default an exporter's `sending_queue` is in-memory: data still queued at shutdown (or lost in a crash) is gone. Point the queue at `file_storage` and it becomes a **persistent queue** that survives a Collector restart:

```yaml
extensions:
  file_storage:
    directory: /var/lib/otelcol/storage
    create_directory: true
exporters:
  otlp:
    endpoint: backend:4317
    tls:
      insecure: true
    sending_queue:
      enabled: true
      storage: file_storage        # <- queue persisted on disk
service:
  extensions: [file_storage]
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [otlp]
```

If the backend is unreachable and the queue fills disk during an outage, the still-undelivered batches are replayed after a restart instead of being dropped. See the [`otlp` exporter](../otlp_exporter/README.md) for the queue's own sizing knobs.

## Multiple named storage instances

You can run several `file_storage` instances (`type/name`) so different consumers get their own directories — handy to isolate a large, spiky exporter queue from small, latency-sensitive receiver offsets:

```yaml
extensions:
  file_storage/queue:
    directory: /var/lib/otelcol/queue
    create_directory: true
  file_storage/offsets:
    directory: /var/lib/otelcol/offsets
    create_directory: true
receivers:
  file_log:
    include: [/var/log/app/*.log]
    storage: file_storage/offsets
exporters:
  otlp:
    endpoint: backend:4317
    sending_queue:
      enabled: true
      storage: file_storage/queue
service:
  extensions: [file_storage/queue, file_storage/offsets]
  pipelines:
    logs:
      receivers: [file_log]
      exporters: [otlp]
```

Each instance still writes one bbolt file per consuming component inside its own `directory`.

## Tuning rebound compaction for persistent-queue spikes

The classic case for online compaction: a persistent exporter queue balloons during a network outage, then drains once connectivity returns. bbolt is mmap-backed, so the freed pages stay **allocated** — the file does not shrink on its own. Rebound compaction reclaims that space, but only after the spike has clearly passed:

```text
allocated MiB
   │            ┌──────────┐   <- spike: queue fills during outage
   │            │          │
100├─ ─ ─ ─ ─ ─ ┘          │   <- rebound_needed_threshold_mib: arms the "needed" flag
   │                        ╲
 10├─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ╲▼ <- rebound_trigger_threshold_mib: drops below -> compact now
   │                          └────────
   └──────────────────────────────────▶ time
```

- `rebound_needed_threshold_mib` (default `100`) — allocation must exceed this for the extension to consider compaction needed at all (avoids compacting trivial usage).
- `rebound_trigger_threshold_mib` (default `10`) — once "needed" is armed, compaction fires when allocation falls back below this, i.e. when the heavy period is over and the rewrite will be cheap.
- `check_interval` (default `5s`) — how often these conditions are evaluated; **must be > 0** when `on_rebound: true`.

Raise the thresholds if your steady-state working set is large (so normal operation never trips compaction); lower them if you want space reclaimed more eagerly after smaller spikes. `compaction.max_transaction_size` (default `65536`) bounds how many items move per compaction transaction.

To hard-cap growth instead of just reclaiming after the fact, set `max_size` (bytes, default `0` = unlimited). It bounds **each** per-consumer bbolt file: once a file is at the cap, a write that needs to grow it is rejected with a storage-full error (writes that fit existing free space still succeed), so a runaway persistent queue can't fill the disk. If you also enable `on_rebound`, both rebound thresholds (×1,048,576) must be ≤ `max_size` or validation fails — see [quirks.md](quirks.md).

## `create_directory` / `directory_permissions` for ephemeral and container environments

In containers or fresh hosts the data directory often does not exist yet. Rather than pre-creating it in an init step, let the extension do it:

```yaml
extensions:
  file_storage:
    directory: /var/lib/otelcol/storage
    create_directory: true
    directory_permissions: "0700"   # only honored when create_directory is true
```

`directory_permissions` defaults to `"0750"` and is applied minus the process umask. It is **ignored** unless `create_directory: true`. Remember the mounted/host directory must be writable by the uid the Collector runs as (the contrib image is non-root) — see [quirks.md](quirks.md).
