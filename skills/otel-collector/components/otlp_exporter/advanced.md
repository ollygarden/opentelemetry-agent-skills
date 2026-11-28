# `otlp_grpc` exporter: advanced use-cases

## Tuning the queue for throughput vs. latency

`num_consumers` sets how many exports run at once, `queue_size` how much a burst or outage can buffer, and, once batching is enabled, `batch.min_size` and `batch.flush_timeout` trade fewer, fuller RPCs against latency. Key meanings and validation: [`sending_queue`](../../references/exporterhelper.md#sending_queue).

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

## Persistent queue via `storage`

By default the queue is in-memory: buffered data is lost in a crash, and at a graceful shutdown each queued request gets one attempt. Set `sending_queue.storage` to a `file_storage` extension to keep it on disk across restarts — see the [`file_storage` example](../file_storage/advanced.md#persistent-exporter-queue-survives-restart) and how the queue then behaves in [Persistent queue](../../references/exporterhelper.md#persistent-queue-storage).

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
