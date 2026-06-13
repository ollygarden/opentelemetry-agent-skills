# `file_storage` extension

| | |
|-|-|
| Kind | extension (storage) |
| Type | `file_storage` |
| Stability | beta |
| Distributions | contrib, k8s |
| Go module | `github.com/open-telemetry/opentelemetry-collector-contrib/extension/storage/filestorage` |
| Upstream README | <https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/extension/storage/filestorage> |

## Description

A **storage extension** that persists other components' state to local disk, backed by **bbolt** (an mmap-based on-disk B+tree key/value store). It gives the Collector durable, restart-surviving state: a receiver's read offsets, an exporter's persistent send queue, or a stateful processor's decision cache all live in files under a single `directory`. The extension writes **one bbolt file per consuming component**, named `<kind>_<type>_<name>` inside that directory.

Unlike a processor or receiver, `file_storage` does **not** sit in a pipeline. It is loaded by listing it under `service.extensions:`, and is then referenced by other components — receivers via their `storage:` key, exporters via `sending_queue.storage:`. Without it, those components hold their state in memory and lose it on every restart.

## Main use-cases

Use when:
- A receiver (e.g. [`file_log`](../file_log/README.md)) must resume reading where it left off after a restart instead of re-reading or skipping data — persist its offsets via `storage:`.
- An exporter needs a **persistent send queue** that survives a restart or crash — set `sending_queue.storage: file_storage` (see the [`otlp` exporter](../otlp_exporter/README.md)).
- A stateful processor needs durable state across restarts — e.g. the [`tail_sampling`](../tail_sampling/README.md) decision cache or [`drain`](../drain/README.md).

Avoid when:
- You are running a stateless pipeline where losing in-memory buffers on restart is acceptable — the extension adds disk I/O and a writable directory requirement for no benefit.
- The Collector has no stable local disk (e.g. a fully ephemeral container with no volume) — state would not actually survive a restart.

## Related components

- [`file_log`](../file_log/README.md) receiver — its `storage:` key points here to persist read offsets/checkpoints across restarts.
- [`otlp` exporter](../otlp_exporter/README.md) — set `sending_queue.storage: file_storage` for a restart-durable persistent queue (otherwise the queue is in-memory).
- [`tail_sampling`](../tail_sampling/README.md) / [`drain`](../drain/README.md) — stateful processors whose decision/aggregation caches can be made durable via this extension.

## Details

- [Configuration](configuration.md) — full top-level and `compaction` config tables, the bbolt per-consumer file naming, and the directory-permissions/umask detail.
- [Verification](verification.md) — offset-persistence-across-restart recipe (no telemetrygen) proving the extension survives a Collector restart.
- [Advanced use-cases](advanced.md) — persistent exporter queues, multiple named storage instances, rebound-compaction tuning, and `create_directory` for ephemeral/container environments.
- [Known quirks](quirks.md) — directory-must-exist gotcha, the validation-error→fix table, per-consumer file naming and the `receiver_filelog_` surprise, bbolt corruption + `recreate`, mmap reclamation, fsync trade-off, and container permission gotchas.
