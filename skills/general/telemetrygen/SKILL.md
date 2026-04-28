---
name: otel-telemetrygen
description: Construct telemetrygen commands for generating synthetic OpenTelemetry traces, metrics, and logs via OTLP. Use this skill whenever the user wants to generate test telemetry, load test a collector or backend, create synthetic OTLP data, send sample traces/metrics/logs to an endpoint, test collector pipelines or processors, validate OTTL transforms, test tail sampling, or mentions telemetrygen in any context. Also trigger when the user asks how to simulate telemetry traffic, stress test an observability stack, or produce sample data for dashboards.
---

# Telemetrygen

Generate synthetic OpenTelemetry telemetry with `telemetrygen` from [opentelemetry-collector-contrib](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/cmd/telemetrygen).

## Quick orientation

`telemetrygen` has three subcommands -- `traces`, `metrics`, `logs` -- each exporting via OTLP to a collector or backend. The default transport is gRPC on port 4317; add `--otlp-http` to switch to HTTP on port 4318.

Every command needs at least a subcommand and typically `--otlp-insecure` for local development (TLS is on by default).

## Workflow

1. **Pick the signal** -- `traces`, `metrics`, or `logs`.
2. **Set the endpoint** -- defaults to `localhost:4317` (gRPC) or `localhost:4318` (HTTP). Use `--otlp-endpoint` to override.
3. **Choose count or duration** -- use `--traces`/`--metrics`/`--logs` for a fixed count per worker, or `--duration` for time-based generation. Duration overrides count when both are set.
4. **Control throughput** -- total rate = `--workers` x `--rate`. Without `--rate`, generation runs at max speed (dangerous against real backends).
5. **Add identity and attributes** -- `--service` sets the service name; `--otlp-attributes` adds resource-level attributes; `--telemetry-attributes` adds span/metric/log-level attributes.
6. **Review the anti-patterns** below before running against shared or production infrastructure.

## Common flags (all subcommands)

See `references/flags.md` for the full flag reference. Key flags:

| Flag | Default | Purpose |
|------|---------|---------|
| `--otlp-endpoint` | `localhost:4317` / `4318` | Target endpoint |
| `--otlp-http` | `false` | Switch to HTTP transport |
| `--otlp-insecure` | `false` | Disable TLS |
| `--workers` | `1` | Concurrent goroutines |
| `--rate` | `0` (unlimited) | Items/sec/worker |
| `--duration` | `0` | Time-based generation (`5s`, `1m`, `inf`) |
| `--service` | `"telemetrygen"` | Service name |
| `--otlp-attributes` | -- | Resource attributes (repeatable) |
| `--telemetry-attributes` | -- | Telemetry-level attributes (repeatable) |
| `--otlp-header` | -- | Custom request headers (repeatable) |
| `--batch` / `--batch-size` | `true` / `100` | Batching controls |

### Attribute value format

```
--otlp-attributes key="string-value"       # String (must be quoted)
--otlp-attributes key=true                  # Boolean
--otlp-attributes key=123                   # Integer
--otlp-attributes key=[val1,val2,val3]      # Slice
```

Resource attributes (`--otlp-attributes`) attach to the Resource; telemetry attributes (`--telemetry-attributes`) attach to the individual span, data point, or log record. Mixing them up is a common mistake.

## Traces

```bash
telemetrygen traces --otlp-insecure --traces 100
```

Key trace flags: `--child-spans` (default 1), `--span-duration` (default 123us), `--status-code` (Unset/Error/Ok), `--span-links`.

### Trace recipes

```bash
# Error spans for testing error-detection pipelines
telemetrygen traces --otlp-insecure --traces 50 --status-code Error

# Deep traces with 5 child spans
telemetrygen traces --otlp-insecure --traces 20 --child-spans 5

# Custom service with attributes
telemetrygen traces --otlp-insecure --traces 10 \
  --service "checkout-service" \
  --otlp-attributes deployment.environment="staging" \
  --telemetry-attributes http.method="POST"
```

## Metrics

```bash
telemetrygen metrics --otlp-insecure --metrics 100
```

Key metric flags: `--metric-type` (Gauge/Sum/Histogram/ExponentialHistogram), `--otlp-metric-name` (default "gen"), `--aggregation-temporality` (cumulative/delta), `--trace-id`/`--span-id` (exemplar linking).

### Metric recipes

```bash
# Histogram with a meaningful name
telemetrygen metrics --otlp-insecure --metrics 50 \
  --metric-type Histogram --otlp-metric-name "http.server.request.duration"

# Delta sums for 30 seconds
telemetrygen metrics --otlp-insecure --duration 30s \
  --metric-type Sum --aggregation-temporality delta

# Cardinality testing with unique timeseries
telemetrygen metrics --otlp-insecure --duration 10s \
  --unique-timeseries --unique-timeseries-duration 5s
```

## Logs

```bash
telemetrygen logs --otlp-insecure --logs 100
```

Key log flags: `--body` (default "the message"), `--severity-text` (default "Info"), `--severity-number` (1-24, default 9), `--trace-id`/`--span-id` (log-trace correlation).

Severity number ranges: 1-4 Trace, 5-8 Debug, 9-12 Info, 13-16 Warning, 17-20 Error, 21-24 Fatal.

### Log recipes

```bash
# Error logs with custom body
telemetrygen logs --otlp-insecure --logs 50 \
  --body "connection timeout: database unreachable" \
  --severity-text Error --severity-number 17

# Logs correlated with a trace
telemetrygen logs --otlp-insecure --logs 20 \
  --trace-id "0af7651916cd43dd8448eb211c80319c" \
  --span-id "b7ad6b7169203331"

# Continuous log stream
telemetrygen logs --otlp-insecure --duration inf --rate 10 \
  --body "heartbeat check"
```

## Load testing patterns

Total throughput = `workers` x `rate`.

```bash
# 100 traces/sec sustained
telemetrygen traces --otlp-insecure --duration inf --workers 10 --rate 10

# 1000 traces/sec burst for 60s
telemetrygen traces --otlp-insecure --duration 60s --workers 10 --rate 100

# Large payloads (~1MB per span) -- always pair --size with --rate
telemetrygen traces --otlp-insecure --duration 30s --size 1 --rate 1
```

## Multi-signal and multi-tenant

```bash
# Correlated signals sharing a trace ID
TRACE_ID="0af7651916cd43dd8448eb211c80319c"
SPAN_ID="b7ad6b7169203331"

telemetrygen traces --otlp-insecure --traces 1 --service "my-service"
telemetrygen metrics --otlp-insecure --metrics 10 \
  --metric-type Histogram --trace-id "$TRACE_ID" --span-id "$SPAN_ID"
telemetrygen logs --otlp-insecure --logs 5 \
  --trace-id "$TRACE_ID" --span-id "$SPAN_ID" --body "processing request"

# Multi-tenant via headers
telemetrygen traces --otlp-insecure --traces 100 \
  --otlp-header X-Scope-OrgID="tenant-a"
```

## TLS and mTLS

```bash
# TLS with custom CA
telemetrygen traces --ca-cert /path/to/ca.pem --traces 10

# Mutual TLS
telemetrygen traces --mtls \
  --ca-cert /path/to/ca.pem \
  --client-cert /path/to/client.pem \
  --client-key /path/to/client-key.pem \
  --traces 10
```

## Container usage

```bash
# Docker with host networking
docker run --rm --network host \
  ghcr.io/open-telemetry/opentelemetry-collector-contrib/telemetrygen:0.147.0 \
  traces --otlp-insecure --traces 100

# Kubernetes Job
# See references/flags.md for a complete Job manifest example
```

## Anti-patterns

These are the mistakes that cause real problems -- review before running against anything shared:

- **No rate limit against real backends**: without `--rate`, telemetrygen generates at max speed and can overwhelm a backend or exhaust resources.
- **Wrong transport**: forgetting `--otlp-http` when targeting port 4318 causes connection failures. gRPC uses 4317, HTTP uses 4318.
- **`--otlp-insecure-skip-verify` in production**: disables certificate validation entirely. Use `--ca-cert` instead.
- **Confusing attribute levels**: `--otlp-attributes` sets resource attributes (service-level); `--telemetry-attributes` sets span/metric/log attributes. Putting attributes at the wrong level makes them invisible to processors or queries that look at the correct level.
- **`--size` without `--rate`**: large payloads at unlimited speed exhaust memory.
- **`--duration` with count flags**: duration silently overrides `--traces`/`--metrics`/`--logs`. Pick one or the other.

## Installation

```bash
# go install (recommended, pin the version)
go install github.com/open-telemetry/opentelemetry-collector-contrib/cmd/telemetrygen@v0.147.0

# Container
docker pull ghcr.io/open-telemetry/opentelemetry-collector-contrib/telemetrygen:0.147.0
```
