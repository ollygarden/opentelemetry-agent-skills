---
name: telemetrygen
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

## Verifying a collector config

Pair telemetrygen with a short-lived `otelcol-contrib` container plus a `file` exporter to verify processor configs without leaving the laptop. The pattern is:

1. Write a minimal config: OTLP receiver → processor under test → file exporter writing to a host-mounted directory.
2. Run `otelcol-contrib` in Docker with `--network host` so telemetrygen can reach `localhost:4317`. Run as your host UID (`--user "$(id -u):$(id -g)"`) so the file exporter can write to the bind-mounted output directory.
3. Generate the *exact* shape of telemetry the processor is supposed to handle (matching service.name, attributes, span kind, etc.).
4. Stop the collector to flush, then read the JSON output back to confirm the transformation.

Minimal config example:

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317

processors:
  transform/under_test:
    error_mode: ignore
    log_statements:
      - context: log
        statements:
          - set(severity_text, "INFO") where IsMatch(severity_text, "(?i)^info$")

exporters:
  file:
    path: /output/result.json
    flush_interval: 200ms

service:
  pipelines:
    logs:
      receivers: [otlp]
      processors: [transform/under_test]
      exporters: [file]
```

Run it:

```bash
mkdir -p ./out
docker run -d --rm --name otelcol-verify \
  --network host \
  --user "$(id -u):$(id -g)" \
  -v "./config.yaml:/etc/otelcol-contrib/config.yaml:ro" \
  -v "./out:/output" \
  otel/opentelemetry-collector-contrib:0.142.0 \
  --config=/etc/otelcol-contrib/config.yaml

# wait for the collector to be ready, then send the telemetry shape under test
until ss -ltn | grep -q ':4317 '; do sleep 0.25; done
telemetrygen logs --otlp-insecure --logs 1 --severity-text Info

# stop the collector to flush, then inspect
docker stop otelcol-verify
cat ./out/result.json | python3 -c 'import sys,json; print(json.load(sys.stdin))'
```

Two recipes worth knowing for this pattern:

- **Verify a filter drops matching records**: send a matching record, expect output to be empty (file size 0). Then send a non-matching record, expect output to contain it. Both halves are needed: empty output alone could also mean the collector crashed.
- **Verify a transform that needs specific attributes**: telemetrygen alone can set resource and telemetry attributes but cannot rename spans or change kind. To exercise rules that depend on those, prepend a `transform/setup` processor in the same pipeline that sets the input shape, then chain the processor under test after it.

On SELinux systems (Fedora, RHEL), append `:z` to the bind mounts so they get relabeled. On rootless Podman, the `--user` flag may not be needed because the container already runs as the invoking user.

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
