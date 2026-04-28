# Declarative Configuration YAML Schema Reference

## Root Structure

```yaml
file_format: "1.0"        # Required. Schema version string.
disabled: false            # Optional. Disable SDK for all signals (default: false).
attribute_limits:          # Optional. Global attribute constraints.
resource:                  # Optional. Resource identity for all signals.
propagator:                # Optional. Context propagation configuration.
tracer_provider:           # Optional. Trace signal configuration.
meter_provider:            # Optional. Metrics signal configuration.
logger_provider:           # Optional. Logs signal configuration.
instrumentation:           # Optional. Per-language instrumentation config (experimental).
```

## `attribute_limits`

```yaml
attribute_limits:
  attribute_value_length_limit: 4096  # Max attribute value size (int). No limit if unset.
  attribute_count_limit: 128          # Max number of attributes (int). Default: 128.
```

## `resource`

```yaml
resource:
  attributes:
    - name: service.name
      value: "my-service"              # String, int, float, or bool.
    - name: deployment.environment.name
      value: "${DEPLOY_ENV:-production}"
  detectors:                           # Experimental.
    - include: ["*"]
      exclude: []
  schema_url: "https://opentelemetry.io/schemas/1.26.0"
```

## `propagator`

```yaml
propagator:
  composite: [tracecontext, baggage]
```

Available: `tracecontext`, `baggage`, `b3`, `b3multi`, `xray`
Deprecated: `jaeger`, `ottrace`

## `tracer_provider`

### Processors

```yaml
tracer_provider:
  processors:
    - batch:                           # Recommended for production
        schedule_delay: 5000           # Delay between exports in ms. Default: 5000.
        export_timeout: 30000          # Max export time in ms. Default: 30000.
        max_queue_size: 2048           # Max spans queued. Default: 2048.
        max_export_batch_size: 512     # Max spans per batch. Default: 512.
        exporter:
          otlp: { ... }               # See Exporters section.
    - simple:                          # Development/debugging only
        exporter:
          console: {}
```

### Samplers

```yaml
tracer_provider:
  sampler:
    # Always sample
    always_on: {}

    # Never sample
    always_off: {}

    # Ratio-based
    trace_id_ratio_based:
      ratio: 0.1                       # Float in [0.0, 1.0].

    # Parent-based (recommended for production)
    parent_based:
      root:                            # Sampler for root spans (no parent).
        trace_id_ratio_based:
          ratio: 0.1
      remote_parent_sampled:           # Default: always_on.
        always_on: {}
      remote_parent_not_sampled:       # Default: always_off.
        always_off: {}
      local_parent_sampled:            # Default: always_on.
        always_on: {}
      local_parent_not_sampled:        # Default: always_off.
        always_off: {}
```

### Span Limits

```yaml
tracer_provider:
  limits:
    attribute_value_length_limit: 4096
    attribute_count_limit: 128
    event_count_limit: 128
    link_count_limit: 128
    event_attribute_count_limit: 128
    link_attribute_count_limit: 128
```

## `meter_provider`

### Readers

```yaml
meter_provider:
  readers:
    # Push-based (sends metrics at intervals)
    - periodic:
        interval: 60000               # Export interval in ms. Default: 60000.
        timeout: 30000                 # Export timeout in ms. Default: 30000.
        exporter:
          otlp: { ... }

    # Pull-based (Prometheus scrape endpoint)
    - pull:
        exporter:
          prometheus:
            host: "localhost"          # Default: "localhost".
            port: 9464                 # Default: 9464.
```

### Views

```yaml
meter_provider:
  views:
    - selector:
        instrument_name: "http.*"      # Glob pattern.
        instrument_type: histogram     # counter, histogram, gauge, etc.
        meter_name: "my-meter"
      stream:
        name: "new_name"              # Rename the metric.
        description: "..."
        aggregation:
          # One of:
          default: {}
          drop: {}                     # Drop the metric entirely.
          sum: {}
          last_value: {}
          explicit_bucket_histogram:
            boundaries: [0, 5, 10, 25, 50, 75, 100, 250, 500, 1000]
            record_min_max: true
          exponential_bucket_histogram:
            max_size: 160
            record_min_max: true
        attribute_keys: [key1, key2]   # Keep only these attribute keys.
```

## `logger_provider`

```yaml
logger_provider:
  processors:
    - batch:
        schedule_delay: 1000
        export_timeout: 30000
        max_queue_size: 2048
        max_export_batch_size: 512
        exporter:
          otlp: { ... }
  limits:
    attribute_value_length_limit: 4096
    attribute_count_limit: 128
```

## Exporters

### OTLP

```yaml
otlp:
  protocol: http/protobuf             # "http/protobuf" or "grpc"
  endpoint: "http://localhost:4318"
  headers:
    api-key: "${API_KEY}"
  compression: gzip                    # "gzip" or "none". Default: none.
  timeout: 10000                       # Export timeout in ms. Default: 10000.
  certificate: /path/to/ca.pem
  client_key: /path/to/key.pem
  client_certificate: /path/to/cert.pem
  # Metrics-specific:
  temporality_preference: cumulative   # "cumulative" or "delta"
  default_histogram_aggregation: explicit_bucket_histogram
```

| Protocol | Port | When to use |
|----------|------|-------------|
| `http/protobuf` | 4318 | Default. Works through proxies, load balancers. |
| `grpc` | 4317 | When gRPC infrastructure exists. Slightly more efficient. |

### Console

```yaml
console: {}    # Writes to stdout. Development/debugging only.
```

### Prometheus (metrics only, pull reader)

```yaml
prometheus:
  host: "localhost"
  port: 9464
```

## `instrumentation` (experimental)

Per-language instrumentation library configuration:

```yaml
instrumentation:
  go:
    net/http:
      request_captured_headers: ["X-Request-ID"]
  java:
    # Java-specific config
```
