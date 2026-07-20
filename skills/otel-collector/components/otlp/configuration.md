# `otlp` receiver: configuration

All keys live under the receiver instance (`receivers: { otlp: { … } }`). The only top-level key is `protocols:`, which holds two optional sub-blocks. Facts below trace to the core **v1.62.0 / v0.156.0** source (`receiver/otlpreceiver/config.go`, `config/configgrpc/configgrpc.go` `ServerConfig`, `config/confighttp/server.go` `ServerConfig`, and `config/configtls`).

## Top-level

| Key | Type | Default | Meaning |
|-----|------|---------|---------|
| `protocols.grpc` | object | enabled by factory default | OTLP over gRPC (see [grpc](#protocolsgrpc)). |
| `protocols.http` | object | enabled by factory default | OTLP over HTTP — protobuf and JSON (see [http](#protocolshttp)). |

**Factory default** enables **both** protocols: gRPC on `localhost:4317`, HTTP on `localhost:4318`. Once you write a `protocols:` block, only the sub-blocks you include are active; the common idiom

```yaml
receivers:
  otlp:
    protocols:
      grpc: {}     # accept per-protocol defaults
      http: {}
```

enables both at their defaults. Listing only `grpc:` disables HTTP, and vice versa.

**Validation:** if neither `grpc` nor `http` is set, startup fails with `must specify at least one protocol when using the OTLP receiver`.

> The default endpoints bind to `localhost`. In a container or pod you must set `endpoint: 0.0.0.0:4317` / `0.0.0.0:4318` — see [quirks.md](quirks.md).

## `protocols.grpc`

`configgrpc.ServerConfig`. Common keys:

| Key | Type | Default | Meaning |
|-----|------|---------|---------|
| `endpoint` | string | `localhost:4317` | Listen address. Set `0.0.0.0:4317` to accept non-loopback traffic. |
| `transport` | string | `tcp` | Socket type. gRPC servers accept only `tcp` or `unix`. |
| `tls` | object | — (plaintext) | Server TLS (see [TLS](#tls-server)). |
| `max_recv_msg_size_mib` | int | 4 (gRPC default) | Max received message size, in MiB. Raise for large batches. |
| `max_concurrent_streams` | uint32 | 0 (unlimited) | Max concurrent HTTP/2 streams per connection. |
| `read_buffer_size` | int | `524288` (512 KiB) | gRPC read buffer. |
| `write_buffer_size` | int | 0 (gRPC default) | gRPC write buffer. |
| `keepalive` | object | — | Server keepalive (see [keepalive](#grpc-keepalive)). |
| `auth` | object | — | `authenticator:` referencing an auth extension. |
| `include_metadata` | bool | `false` | Propagate per-request gRPC metadata (headers) into the downstream pipeline. Required for metadata routing and `from_context: metadata.*`; server authenticator extensions read transport headers directly. |
| `middlewares` | list | — | gRPC server middleware extensions. |

### grpc keepalive

| Key | Type | Meaning |
|-----|------|---------|
| `server_parameters.max_connection_idle` | duration | Close a connection after this much idle time. |
| `server_parameters.max_connection_age` | duration | Max total connection lifetime. |
| `server_parameters.max_connection_age_grace` | duration | Grace period after `max_connection_age` before forced close. |
| `server_parameters.time` | duration | Ping interval on idle connections. |
| `server_parameters.timeout` | duration | Ping ack timeout. |
| `enforcement_policy.min_time` | duration | Minimum client ping interval the server tolerates. |
| `enforcement_policy.permit_without_stream` | bool | Allow client pings with no active stream. |

## `protocols.http`

`confighttp.ServerConfig` (squashed) plus three OTLP URL-path overrides from `receiver/otlpreceiver/config.go`. Common keys:

| Key | Type | Default | Meaning |
|-----|------|---------|---------|
| `endpoint` | string | `localhost:4318` | Listen address. Set `0.0.0.0:4318` to accept non-loopback traffic. |
| `traces_url_path` | string | `/v1/traces` | Path for trace export. |
| `metrics_url_path` | string | `/v1/metrics` | Path for metric export. |
| `logs_url_path` | string | `/v1/logs` | Path for log export. |
| `tls` | object | — (plaintext) | Server TLS (see [TLS](#tls-server)). |
| `cors` | object | — | Browser CORS (see [cors](#cors)). |
| `auth` | object | — | `authenticator:` referencing an auth extension. |
| `max_request_body_size` | int | `20971520` (20 MiB) | Max accepted request body. Raise for large batches. |
| `include_metadata` | bool | `false` | Propagate per-request HTTP headers into the pipeline (see grpc note above). |
| `response_headers` | map | — | Static headers added to every response. |
| `compression_algorithms` | list | none, gzip, zstd, zlib, snappy, deflate, lz4, x-snappy-framed | Accepted request `Content-Encoding`s. |
| `read_timeout` | duration | 0 (none) | Max time to read the full request. |
| `read_header_timeout` | duration | 0 (none) | Max time to read request headers. |
| `write_timeout` | duration | 0 (none) | Max time to write the response. |
| `idle_timeout` | duration | 0 (none) | Max idle time on a keep-alive connection. |
| `keep_alives_enabled` | bool | `true` | Whether to allow HTTP keep-alives. |
| `middlewares` | list | — | HTTP server middleware extensions. |

There is **no profiles URL-path field** — the HTTP config exposes only traces/metrics/logs override keys. Profiles (Alpha) are still served over HTTP, but on a fixed, non-configurable path: `/v1development/profiles`.

The HTTP endpoint accepts both **OTLP/protobuf** and **OTLP/JSON** on the same paths; the encoding is selected by `Content-Type`.

### cors

| Key | Type | Meaning |
|-----|------|---------|
| `allowed_origins` | list of string | Origins permitted to make cross-origin OTLP requests (e.g. browser RUM). |
| `allowed_headers` | list of string | Additional request headers allowed cross-origin. |
| `exposed_headers` | list of string | Response headers browsers may expose to the caller (`Access-Control-Expose-Headers`). |
| `max_age` | int | Seconds browsers may cache the CORS preflight. |

## TLS (server)

Both `grpc.tls` and `http.tls` use `configtls.ServerConfig`. Common keys:

| Key | Type | Meaning |
|-----|------|---------|
| `cert_file` | string | Server certificate (enables TLS). |
| `key_file` | string | Server private key. |
| `ca_file` | string | CA bundle used to verify client certificates. |
| `client_ca_file` | string | CA bundle for mutual TLS — require and verify client certs. |

See `config/configtls` for the full set (min/max version, cipher suites, reload interval).

## Validation summary

| Condition | Error | When |
|-----------|-------|------|
| neither `grpc` nor `http` configured | `must specify at least one protocol when using the OTLP receiver` | config validation |
| `max_recv_msg_size_mib` out of range | `invalid max_recv_msg_size_mib value, must be between 1 and …` | startup |
