# `otlp` receiver: advanced use-cases

## Enabling only one protocol

Once `protocols:` is written, only the sub-blocks present are active. To accept gRPC only (and refuse to even open the HTTP port):

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
```

HTTP-only is the mirror image — list only `http:`. Listing neither fails startup (`must specify at least one protocol when using the OTLP receiver`).

## `include_metadata` for downstream metadata consumers

Per-request gRPC/HTTP headers (and other client metadata) are **not propagated into the downstream pipeline** by default. Downstream features that read them need the receiver to carry them into the pipeline. Server authenticator extensions are different: they read the transport headers directly and do not require `include_metadata`.

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
        include_metadata: true     # headers survive into the pipeline
```

`include_metadata: true` is the prerequisite for:
- [`routing`](../routing/README.md) by request metadata.
- `attributes` / `resource` actions that use `from_context: metadata.*`.

Without it, those downstream consumers see no metadata and behave as if the headers were absent. The [`load_balancing`](../load_balancing/README.md) exporter's `routing_key: attributes` reads telemetry resource/scope/record attributes, not transport metadata; copy a header into a telemetry attribute first if it must become a load-balancing key.

## CORS for browser OTLP

Browser RUM agents send OTLP/HTTP cross-origin and require CORS preflight handling:

```yaml
receivers:
  otlp:
    protocols:
      http:
        endpoint: 0.0.0.0:4318
        cors:
          allowed_origins:
            - https://app.example.com
          allowed_headers:
            - "*"
          max_age: 7200
```

## mTLS (require client certificates)

Set `client_ca_file` on the protocol's `tls` block to require and verify client certificates:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
        tls:
          cert_file: /certs/server.crt
          key_file: /certs/server.key
          client_ca_file: /certs/client-ca.crt   # mutual TLS
```

## Large-payload tuning

Reject-on-size limits differ per protocol. Raise them when senders batch aggressively:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
        max_recv_msg_size_mib: 16        # gRPC message cap (default 4 MiB)
      http:
        endpoint: 0.0.0.0:4318
        max_request_body_size: 67108864  # 64 MiB (default 20 MiB)
```

## Named instances

The `type/name` pattern lets one Collector run several OTLP listeners — e.g. an internal-only plaintext listener and an external mTLS one:

```yaml
receivers:
  otlp/internal:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
  otlp/external:
    protocols:
      grpc:
        endpoint: 0.0.0.0:5317
        tls:
          cert_file: /certs/server.crt
          key_file: /certs/server.key
          client_ca_file: /certs/client-ca.crt

service:
  pipelines:
    traces/internal:
      receivers: [otlp/internal]
    traces/external:
      receivers: [otlp/external]
```

Give each named instance a distinct `endpoint` — two listeners cannot share a port.
