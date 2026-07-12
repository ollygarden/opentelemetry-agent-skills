# `otlp_grpc` exporter: advanced use-cases

## Tuning the queue for throughput vs. latency

The defaults (`num_consumers: 10`, `queue_size: 1000`, `batch.min_size: 8192`, `batch.flush_timeout: 200ms`) suit a general gateway. The levers:

- **Throughput** — raise `num_consumers` (more parallel senders) and `queue_size` (deeper buffer to absorb bursts). Larger `batch.min_size` packs more per request, reducing per-RPC overhead.
- **Latency** — lower `batch.flush_timeout` (flush sooner) and `batch.min_size` (smaller batches leave the queue faster). The trade-off is more, smaller RPCs.

```yaml
exporters:
  otlp_grpc:
    endpoint: gateway:4317
    tls:
      insecure: true
    sending_queue:
      num_consumers: 20
      queue_size: 10000
      batch:
        flush_timeout: 5s        # tolerate latency for fewer, fuller batches
        min_size: 16384
```

`batch.min_size` must be ≤ `queue_size` when their sizers match — see [quirks.md](quirks.md).

## Persistent queue via `storage`

By default the queue is in-memory: data buffered at a crash or restart is lost. Point `sending_queue.storage` at a `file_storage` extension to persist the queue to disk so it survives restarts:

```yaml
extensions:
  file_storage:
    directory: /var/lib/otelcol/queue

exporters:
  otlp_grpc:
    endpoint: backend:4317
    sending_queue:
      storage: file_storage      # queue persists across restarts

service:
  extensions: [file_storage]
```

> The [`file_storage`](../file_storage/README.md) extension backs persistent queues. Persistence trades durability for disk I/O. **`wait_for_result` is not supported with a persistent `storage` queue** (see below).

## `wait_for_result` trade-off

`sending_queue.wait_for_result: true` makes the caller block until the export result is known, so the pipeline learns about failures synchronously instead of fire-and-forget. It costs throughput (the producer waits) and is **incompatible with a persistent `storage` queue** — enabling both is a config error. Use it only when you need synchronous delivery confirmation and an in-memory queue.

## mTLS to a secure backend

Drop `insecure` and supply client credentials to authenticate to a TLS backend:

```yaml
exporters:
  otlp_grpc:
    endpoint: backend.example.com:4317
    tls:
      ca_file: /certs/backend-ca.crt
      cert_file: /certs/client.crt        # mTLS
      key_file: /certs/client.key
    headers:
      x-tenant-id: team-a                 # static per-request header
```

## Multiple named exporters

The `type/name` pattern fans a pipeline out to several backends, each with its own queue/retry/TLS:

```yaml
exporters:
  otlp_grpc/primary:
    endpoint: primary:4317
    tls: { insecure: true }
  otlp_grpc/backup:
    endpoint: backup:4317
    tls: { insecure: true }

service:
  pipelines:
    traces:
      exporters: [otlp_grpc/primary, otlp_grpc/backup]
```

## Relationship to `load_balancing`

[`load_balancing`](../load_balancing/README.md) wraps a per-backend `otlp` exporter under `protocol.otlp` — one sub-exporter per resolved backend. Everything on this page (TLS, `headers`, `compression`, `sending_queue`, `retry_on_failure`, `timeout`) applies **per backend** there. The one difference: under `load_balancing` you do **not** set `protocol.otlp.endpoint` — the load balancer populates it from the resolver. Reach for `load_balancing` when you need trace-ID / service affinity across a pool (to scale `tail_sampling`/`span_metrics`); reach for a plain `otlp_grpc` exporter when you just send to one backend (or fan out to a fixed few via named instances).
