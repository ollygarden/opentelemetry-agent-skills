# OTTL Contexts Reference

Context paths and enums for collector-contrib **v0.149.0**. Higher-level contexts are reachable from lower ones (a span statement can read `resource.attributes`); the reverse is not true. Always pick the most specific context for the work — using `datapoint` to set metric-point attributes is much cheaper than walking through `metric.data_points` from the metric context.

## Context hierarchy

```
Resource
  └── Scope (instrumentation library)
      ├── Span → Span Event
      ├── Metric → DataPoint
      ├── Log
      └── Profile → Profile Sample
```

## Resource context

```ottl
resource.attributes                    # map
resource.attributes["service.name"]
resource.dropped_attributes_count
resource.cache                         # transformation-scope scratchpad
resource.cache["key"]
resource.metadata["key"]               # client request metadata (v0.147+)
```

```ottl
set(resource.attributes["environment"], "production")
    where resource.attributes["environment"] == nil

set(resource.attributes["service.name"],
    ToLowerCase(resource.attributes["service.name"]))
```

## Scope (instrumentation scope) context

```ottl
scope.name
scope.version
scope.attributes
scope.attributes["key"]
scope.dropped_attributes_count
scope.cache
scope.cache["key"]
```

```ottl
where not IsMatch(scope.name, "opentelemetry-.*")

set(scope.attributes["quality"], "manual")
    where scope.name == "manual-instrumentation"
```

> Older configs may use `instrumentation_scope.*`. The current canonical prefix is `scope.*`.

## Span context (Beta)

### Identity & relationships
```ottl
span.trace_id                    # bytes
span.trace_id.string             # hex string
span.span_id                     # bytes
span.span_id.string              # hex string
span.parent_span_id              # bytes
span.parent_span_id.string       # hex string
span.flags                       # W3C trace flags, uint32 (v0.145+)
span.trace_state                 # W3C trace state
span.trace_state["key"]
```

### Metadata
```ottl
span.name
span.kind                        # int64
span.kind.string                 # "Server" | "Client" | "Internal" | …
span.status                      # status object
span.status.code                 # int64 (use STATUS_CODE_* enums)
span.status.message              # string
```

### Timing
```ottl
span.start_time_unix_nano        # int64
span.end_time_unix_nano          # int64
span.start_time                  # time.Time
span.end_time                    # time.Time
```

### Data
```ottl
span.attributes
span.attributes["http.method"]
span.events                      # collection
span.links                       # collection
span.dropped_attributes_count
span.dropped_events_count
span.dropped_links_count
span.metadata["key"]             # client request metadata (v0.147+)
span.cache["key"]                # transformation cache
```

### Examples
```ottl
# Duration in ms
set(span.attributes["duration_ms"],
    (span.end_time_unix_nano - span.start_time_unix_nano) / 1000000)

# Mark errored HTTP spans
set(span.status.code, STATUS_CODE_ERROR)
    where IsInt(span.attributes["http.status_code"])
      and Int(span.attributes["http.status_code"]) >= 400

# Identify root spans (preferred form, since v0.142 SpanID accepts hex strings too)
set(span.attributes["is_root"], true)
    where IsRootSpan()

# Latency-and-kind sampling decision
set(span.attributes["sampled"], true)
    where span.kind == SPAN_KIND_SERVER
      and (span.status.code == STATUS_CODE_ERROR
           or (span.end_time_unix_nano - span.start_time_unix_nano) > 1000000000)
```

## Span Event context (Beta)

```ottl
spanevent.name
spanevent.time_unix_nano
spanevent.time                       # time.Time
spanevent.attributes
spanevent.attributes["key"]
spanevent.dropped_attributes_count
spanevent.event_index                # 0-based position within span (v0.120+)
spanevent.cache["key"]
# Span fields are reachable: span.name, span.attributes, …
```

```ottl
where spanevent.name == "exception"

set(spanevent.attributes["span.name"], span.name)

set(spanevent.attributes["error.message"], "REDACTED")
    where IsMatch(spanevent.attributes["error.message"], ".*password.*")
```

> Older configs may use `span_event.*`. The current path prefix is `spanevent.*`.

## Metric context (Beta)

```ottl
metric.name
metric.description
metric.unit
metric.type                          # int64 (use METRIC_DATA_TYPE_* enums)
metric.aggregation_temporality       # int64
metric.is_monotonic                  # bool
metric.data_points                   # collection
metric.metadata
```

```ottl
# Normalize metric names to safe characters
set(metric.name,
    replace_all_patterns(metric.name, "value", "[^a-zA-Z0-9_]", "_"))

# Normalize legacy units
set(metric.unit, "s") where metric.unit == "seconds"
set(metric.unit, "B") where metric.unit == "bytes"

# Filter by type
where metric.type == METRIC_DATA_TYPE_GAUGE
```

## DataPoint context

```ottl
datapoint.start_time_unix_nano
datapoint.time_unix_nano
datapoint.start_time                 # time.Time
datapoint.time                       # time.Time
datapoint.value_double
datapoint.value_int
datapoint.attributes
datapoint.attributes["key"]
datapoint.flags
datapoint.exemplars                  # collection
# Metric, scope, resource paths are reachable.
```

```ottl
set(datapoint.attributes["value_range"], "high")
    where datapoint.value_double > 1000

# Rename an attribute
set(datapoint.attributes["host.name"], datapoint.attributes["hostname"])
delete_key(datapoint.attributes, "hostname")
```

## Log context (Beta)

```ottl
log.time_unix_nano
log.observed_time_unix_nano
log.time                             # time.Time
log.observed_time                    # time.Time
log.severity_number                  # int64 (use SEVERITY_NUMBER_* enums)
log.severity_text
log.body                             # any type
log.body.string                      # body coerced to string
log.body["key"]                      # if body is a map
log.body[0]                          # if body is a list
log.attributes
log.attributes["key"]
log.dropped_attributes_count
log.metadata["key"]                  # client request metadata (v0.147+)
log.flags
log.trace_id / log.trace_id.string
log.span_id  / log.span_id.string
log.event_name                       # for event-shaped logs
```

```ottl
# Parse JSON body into structured attributes
set(log.attributes, ParseJSON(log.body.string))
    where IsString(log.body) and IsMatch(log.body.string, "^\\s*\\{.*\\}\\s*$")

# Normalize severity text
set(log.severity_text, "ERROR")
    where log.severity_number >= SEVERITY_NUMBER_ERROR

# Trace correlation flag
set(log.attributes["has_trace"], true)
    where log.trace_id.string != "00000000000000000000000000000000"

# Body-level redaction
set(log.body, "REDACTED")
    where IsMatch(log.body.string, ".*(password|secret|token).*")
```

## Profile context (Development, v0.124+)

```ottl
profile.profile_id
profile.profile_id.string
profile.attributes
profile.attributes["key"]
profile.dropped_attributes_count
profile.time_unix_nano
profile.time                          # time.Time
profile.duration_unix_nano
profile.duration
profile.sample_type
profile.sample_type.type
profile.sample_type.unit
profile.sample
profile.period_type
profile.period_type.type
profile.period_type.unit
profile.period
profile.attribute_indices
profile.original_payload_format
profile.original_payload              # []byte
cache["key"]
```

## Profile Sample context (Development, v0.132+)

```ottl
profilesample.values                  # []int64
profilesample.attributes
profilesample.attributes["key"]
profilesample.link_index
profilesample.timestamps_unix_nano    # []int64
profilesample.timestamps              # []time.Time
profilesample.attribute_indices
cache["key"]
```

## Client request metadata (v0.147+)

All contexts expose a `metadata` map for reading metadata attached to the inbound request (gRPC metadata, HTTP headers). It is *not* part of the telemetry payload — values come from the receiver. Keys are case-sensitive and the available set depends on the receiver and transport.

```ottl
span.metadata["X-Tenant-ID"]
log.metadata["X-Scope-OrgID"]
resource.metadata["Authorization"]
datapoint.metadata["X-Custom-Header"]
```

Common uses: tenant routing, request-level enrichment, conditional drop based on caller identity.

## Enums

### Span kind
```ottl
SPAN_KIND_UNSPECIFIED  # 0
SPAN_KIND_INTERNAL     # 1
SPAN_KIND_SERVER       # 2
SPAN_KIND_CLIENT       # 3
SPAN_KIND_PRODUCER     # 4
SPAN_KIND_CONSUMER     # 5
```

### Span status
```ottl
STATUS_CODE_UNSET  # 0
STATUS_CODE_OK     # 1
STATUS_CODE_ERROR  # 2
```

### Log severity
```ottl
SEVERITY_NUMBER_UNSPECIFIED  # 0
SEVERITY_NUMBER_TRACE        # 1-4
SEVERITY_NUMBER_DEBUG        # 5-8
SEVERITY_NUMBER_INFO         # 9-12
SEVERITY_NUMBER_WARN         # 13-16
SEVERITY_NUMBER_ERROR        # 17-20
SEVERITY_NUMBER_FATAL        # 21-24
```

### Metric aggregation temporality
```ottl
AGGREGATION_TEMPORALITY_UNSPECIFIED  # 0
AGGREGATION_TEMPORALITY_DELTA        # 1
AGGREGATION_TEMPORALITY_CUMULATIVE   # 2
```

### Metric data type
```ottl
METRIC_DATA_TYPE_NONE                  # 0
METRIC_DATA_TYPE_GAUGE                 # 1
METRIC_DATA_TYPE_SUM                   # 2
METRIC_DATA_TYPE_HISTOGRAM             # 3
METRIC_DATA_TYPE_EXPONENTIAL_HISTOGRAM # 4
METRIC_DATA_TYPE_SUMMARY               # 5
```
