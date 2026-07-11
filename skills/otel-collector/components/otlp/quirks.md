# `otlp` receiver: known quirks

## The default endpoint is `localhost`, not `0.0.0.0`

This is the single most common "the receiver gets no data in Docker/k8s" problem. Each protocol defaults to `localhost:4317` / `localhost:4318` — a deliberate security-hardening default so a Collector started with no explicit endpoint doesn't expose an open OTLP port on every interface. The consequence: inside a container or pod, traffic arrives on a **non-loopback** interface, and a `localhost`-bound listener silently accepts nothing from outside — no error, just no data.

To receive external traffic, set the endpoint explicitly:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318
```

If a containerized receiver "gets no data," check this first.

## Must specify at least one protocol

A `protocols:` block with neither `grpc` nor `http` fails at config validation with `must specify at least one protocol when using the OTLP receiver`. Note the inverse trap: once you write `protocols:`, only the sub-blocks you include are active — listing only `grpc:` **disables** the HTTP listener that the bare factory default would have enabled. Use `grpc: {}` / `http: {}` (empty maps) to keep a protocol on at its defaults.

## gRPC is 4317, HTTP is 4318

The two protocols listen on different ports: gRPC `4317`, HTTP `4318`. Sending OTLP/HTTP to `:4317` (or gRPC to `:4318`) fails. With `telemetrygen`, gRPC is the default and `--otlp-http` switches to the HTTP exporter — point each at the matching port.

## OTLP/JSON works on the HTTP port

The HTTP endpoint accepts both OTLP/protobuf and **OTLP/JSON** on the same paths (`/v1/traces`, `/v1/metrics`, `/v1/logs`); the encoding is chosen by `Content-Type`. There is no separate JSON port or path. gRPC is protobuf only.

## `include_metadata` is off by default

Per-request headers/metadata are dropped unless `include_metadata: true`. Header-based auth extensions, metadata [`routing`](../routing/README.md), and [`load_balancing`](../load_balancing/README.md) keyed on client-metadata attributes all silently see no metadata without it. There is no error — the feature just behaves as if the headers weren't sent.

## No configurable profiles URL path

The HTTP config exposes `traces_url_path`, `metrics_url_path`, and `logs_url_path` overrides only. There is no `profiles_url_path` key — profiles (Alpha) are still served over HTTP, but on a fixed path the receiver hardcodes: `/v1development/profiles`. (Note: the upstream README mentions a `profiles_url_path` and a `/v1/profiles` default; neither exists in the config struct or the handler — the path is fixed.)

## Stability is per signal

Traces, metrics, and logs are **Stable**; **profiles** are **Alpha**. Treat profile ingestion as still-evolving and subject to breaking change, even though the same receiver serves it.
