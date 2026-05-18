# Telemetrygen Flag Reference

Complete flag reference for `telemetrygen` v0.147.0.

## Table of Contents

- [Common Flags](#common-flags)
- [Trace-Specific Flags](#trace-specific-flags)
- [Metric-Specific Flags](#metric-specific-flags)
- [Log-Specific Flags](#log-specific-flags)
- [Kubernetes Job Manifest](#kubernetes-job-manifest)

## Common Flags

These apply to all subcommands (`traces`, `metrics`, `logs`).

### Endpoint and Transport

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--otlp-endpoint` | string | `localhost:4317` (gRPC) / `localhost:4318` (HTTP) | Destination endpoint |
| `--otlp-http` | bool | `false` | Use HTTP exporter instead of gRPC |
| `--otlp-insecure` | bool | `false` | Disable transport security |

### TLS and mTLS

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--otlp-insecure-skip-verify` | bool | `false` | Skip server certificate verification |
| `--ca-cert` | string | `""` | Trusted CA certificate file |
| `--mtls` | bool | `false` | Enable mutual TLS |
| `--client-cert` | string | `""` | Client certificate file (requires `--mtls`) |
| `--client-key` | string | `""` | Client private key file (requires `--mtls`) |

### Generation Control

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--workers` | int | `1` | Concurrent worker goroutines |
| `--rate` | float64 | `0` | Items/sec/worker. `0` = no throttling |
| `--duration` | duration | `0` | How long to generate. Go durations (`5s`, `1m`) or `inf`. Overrides count flags |
| `--interval` | duration | `1s` | Progress reporting interval |

### Batching

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--batch` | bool | `true` | Batch records before sending |
| `--batch-size` | int | `100` | Records per batch |

### Identity and Attributes

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--service` | string | `"telemetrygen"` | Service name |
| `--otlp-header` | key="value" | -- | Custom OTLP request header. Repeatable |
| `--otlp-attributes` | key=value | -- | Resource attributes. Repeatable |
| `--telemetry-attributes` | key=value | -- | Telemetry-level attributes. Repeatable |

### Other

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--size` | int | `0` | Minimum payload size in MB per record |
| `--allow-export-failures` | bool | `false` | Continue when exports fail |

### Attribute Value Format

```
key="string-value"       # String (must be quoted)
key=true                  # Boolean
key=123                   # Integer
key=[val1,val2,val3]      # Slice (all elements same type)
```

## Trace-Specific Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--traces` | int | `0` | Traces per worker. Ignored if `--duration` set |
| `--child-spans` | int | `1` | Child spans per trace |
| `--span-duration` | duration | `123us` | Duration of each span |
| `--status-code` | string | `"0"` | `Unset`/`0`, `Error`/`1`, `Ok`/`2` |
| `--span-links` | int | `0` | Span links per span |
| `--marshal` | bool | `false` | Marshal trace context via HTTP headers |
| `--otlp-http-url-path` | string | `"/v1/traces"` | HTTP URL path |

## Metric-Specific Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--metrics` | int | `0` | Metrics per worker. Ignored if `--duration` set |
| `--otlp-metric-name` | string | `"gen"` | Metric name |
| `--metric-type` | string | `"Gauge"` | `Gauge`, `Sum`, `Histogram`, `ExponentialHistogram` |
| `--aggregation-temporality` | string | `"cumulative"` | `cumulative` or `delta` |
| `--trace-id` | string | `""` | Trace ID for exemplar linking |
| `--span-id` | string | `""` | Span ID for exemplar linking |
| `--unique-timeseries` | bool | `false` | Enforce unique timeseries (experimental) |
| `--unique-timeseries-duration` | duration | `1s` | Window for unique timeseries generation |
| `--otlp-http-url-path` | string | `"/v1/metrics"` | HTTP URL path |

### Metric Types

| Type | Description |
|------|-------------|
| `Gauge` | Point-in-time snapshot value |
| `Sum` | Cumulative or delta counter |
| `Histogram` | Distribution with explicit bucket boundaries |
| `ExponentialHistogram` | Distribution with exponential bucket boundaries |

## Log-Specific Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--logs` | int | `0` | Logs per worker. Ignored if `--duration` set |
| `--body` | string | `"the message"` | Log body content |
| `--severity-text` | string | `"Info"` | Severity text |
| `--severity-number` | int32 | `9` | Severity number (1-24) |
| `--trace-id` | string | `""` | Trace ID for log-trace correlation |
| `--span-id` | string | `""` | Span ID for log-trace correlation |
| `--otlp-http-url-path` | string | `"/v1/logs"` | HTTP URL path |

### Severity Number Reference

| Range | Level |
|-------|-------|
| 1-4 | Trace |
| 5-8 | Debug |
| 9-12 | Info |
| 13-16 | Warning |
| 17-20 | Error |
| 21-24 | Fatal |

## Kubernetes Job Manifest

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: telemetrygen-traces
spec:
  template:
    spec:
      containers:
      - name: telemetrygen
        image: ghcr.io/open-telemetry/opentelemetry-collector-contrib/telemetrygen:0.147.0
        args:
        - traces
        - --otlp-insecure
        - --otlp-endpoint=otel-collector.observability:4317
        - --duration=60s
        - --rate=10
        - --workers=4
        - --service=load-test
      restartPolicy: Never
  backoffLimit: 1
```
