---
name: otel-ottl
description: OpenTelemetry Transformation Language (OTTL) expert for writing and debugging telemetry transformations in the OpenTelemetry Collector. Use when authoring or reviewing `transform`, `filter`, `routing`, or `tail_sampling` processor configs, debugging OTTL syntax or semantics, transforming traces, metrics, logs, or profiles, or converting data-processing requirements into OTTL statements.
---

# OpenTelemetry Transformation Language (OTTL)

OTTL is a domain-specific language for transforming telemetry inside the OpenTelemetry Collector. It is consumed by the `transform`, `filter`, `routing`, and `tail_sampling` processors (and a few others) in [opentelemetry-collector-contrib](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/pkg/ottl).

This skill targets `pkg/ottl` as of collector-contrib **v0.149.0**. Function and path availability before that version differs; check the upstream `pkg/ottl/ottlfuncs/README.md` and `pkg/ottl/contexts/*/README.md` for the exact set in any older release.

## Statement syntax

```ottl
function(arguments) [where condition]
```

Every statement has exactly one **editor** (lowercase: `set`, `delete_key`, `append`, …) optionally guarded by a `where` clause whose body is a boolean expression. Conditions can call **converters** (uppercase: `Concat`, `IsMatch`, `ParseJSON`, …) which return values but do not mutate telemetry.

```ottl
set(span.attributes["env"], "prod") where resource.attributes["env"] == nil
```

## Workflow

1. **Pick the processor.** `transform` rewrites; `filter` drops; `routing` fans out by pipeline; `tail_sampling` keeps/drops traces. The processor decides which contexts and function set are usable.
2. **Pick the context.** `resource`, `scope`, `span`, `spanevent`, `metric`, `datapoint`, `log`, `profile`, `profilesample`. Operate at the lowest level that gives you the data — using `datapoint` to set attributes is much cheaper than walking through `metric.data_points` from the metric context.
3. **Write statements.** Reach for `references/quick-reference.md` for common recipes; `references/contexts.md` for paths/enums; `references/functions.md` for the editor and converter catalog.
4. **Set `error_mode`.** `ignore` (default) keeps the pipeline running and logs errors; `silent` does the same but quietly; `propagate` aborts on first failure (use only when you want a bad config to fail loud in tests).
5. **Verify.** OTTL gotchas are the kind that pass the eye test (see [Common gotchas](#common-gotchas)). Use the [telemetrygen verification recipe](../telemetrygen/SKILL.md#verifying-a-collector-config) — `otelcol-contrib` + file exporter + telemetrygen — to confirm the snippet does what the prose claims before shipping.

## Contexts at a glance

OTTL paths are scoped by signal. Higher levels are reachable from lower ones (e.g., `resource.attributes` from a span statement); the reverse is not true.

| Context | Common paths |
|---------|--------------|
| Resource | `resource.attributes["service.name"]`, `resource.metadata["X-Tenant-ID"]` |
| Scope | `scope.name`, `scope.version`, `scope.attributes["…"]` |
| Span | `span.name`, `span.kind`, `span.status.code`, `span.attributes["…"]`, `span.flags` |
| Span Event | `spanevent.name`, `spanevent.attributes["…"]`, `spanevent.event_index` |
| Metric | `metric.name`, `metric.unit`, `metric.type`, `metric.aggregation_temporality` |
| DataPoint | `datapoint.value_double`, `datapoint.value_int`, `datapoint.attributes["…"]` |
| Log | `log.body`, `log.body.string`, `log.severity_number`, `log.attributes["…"]` |
| Profile | `profile.profile_id`, `profile.attributes["…"]` (Development) |

Full path inventory plus enums in `references/contexts.md`.

## Essential functions

```ottl
# Editors (mutate telemetry)
set(target, value)
delete_key(target, key)               # delete by exact key
delete_matching_keys(target, regex)   # delete by regex
delete_index(target, index)           # remove from a slice (v0.145+)
keep_keys(target, [k1, k2])           # keep only these keys
merge_maps(target, source, "upsert")  # "insert" | "update" | "upsert"
truncate_all(target, max_len)         # UTF-8 safe by default in v0.148+
replace_pattern(target, regex, replacement)

# Converters (return values)
Concat([a, b], "-")
Split(s, ",")
ToLowerCase(s) / ToUpperCase(s)
IsMatch(s, "pattern")                 # bool
String(v) / Int(v) / Double(v) / Bool(v)
ParseJSON(s) / ParseKeyValue(s, "&", "=")
URL(s)                                # parse URL components (v0.127+)
ExtractPatterns(s, "(?P<name>…)")     # named captures → map
ExtractGrokPatterns(s, "%{IP:client}") # Grok (v0.130+)
IsInCIDR(ip, ["10.0.0.0/8"])          # CIDR membership (v0.146+)
SHA256(v) / Murmur3Hash(v) / XXH3(v)  # hashing
UUID() / UUIDv7()
```

Full catalog with signatures in `references/functions.md`.

## Common patterns

```ottl
# Conditional set
set(span.attributes["sampled"], true)
    where (span.end_time_unix_nano - span.start_time_unix_nano) > 1000000000

# Normalize a value
set(span.attributes["http.method"],
    ToUpperCase(String(span.attributes["http.method"])))

# Parse a JSON log body into structured attributes
set(log.attributes, ParseJSON(log.body.string))
    where IsString(log.body) and IsMatch(log.body.string, "^\\s*\\{.*\\}\\s*$")

# Mark errored HTTP spans
set(span.status.code, STATUS_CODE_ERROR)
    where IsInt(span.attributes["http.status_code"])
      and Int(span.attributes["http.status_code"]) >= 400

# Redact secrets by key pattern
delete_matching_keys(span.attributes, "(?i).*(password|secret|token|apikey).*")

# Hash PII rather than dropping it (preserves cardinality for analytics)
set(span.attributes["user.email_hash"], SHA256(span.attributes["user.email"]))
delete_key(span.attributes, "user.email")
```

More recipes (sampling, redaction, time math, parsing) in `references/quick-reference.md`.

## Processor wiring

```yaml
processors:
  transform:
    error_mode: ignore        # ignore | silent | propagate
    trace_statements:
      - context: span
        statements:
          - set(span.attributes["processed"], true)
    log_statements:
      - context: log
        statements:
          - set(log.attributes["source"], "collector")
    metric_statements:
      - context: datapoint
        statements:
          - set(datapoint.attributes["env"], "prod")

  filter:
    error_mode: ignore
    traces:
      span:
        - 'IsMatch(span.name, "^/health.*")'
    logs:
      log_record:
        - 'log.severity_number < SEVERITY_NUMBER_WARN'

  routing:
    default_pipelines: [traces/default]
    table:
      - statement: 'route() where resource.attributes["env"] == "prod"'
        pipelines: [traces/prod]
      - statement: 'route() where span.status.code == STATUS_CODE_ERROR'
        pipelines: [traces/errors]

  tail_sampling:
    policies:
      - name: errors
        type: ottl_condition
        ottl_condition:
          span:
            - 'span.status.code == STATUS_CODE_ERROR'
```

## Common gotchas

These mistakes pass YAML validation but break OTTL semantics. Most have cost real time in production rollouts.

### `replace_pattern` backreferences need `$${1}` in YAML

OTTL uses `${1}`, `${2}`, … for regex backreferences. The collector's YAML loader treats `$` as an env-var marker, so YAML `$$` becomes OTTL `$`. To produce `${1}` at the OTTL level, write `$${1}` in YAML. Writing `"$1REDACTED"` produces literal `$1REDACTED` with no replacement — silent failure.

### Go RE2 has a repeat-count ceiling

Patterns like `(.{1024}).*` fail to compile with "invalid repeat count". For length-based truncation, prefer `Substring` + `Len`:

```ottl
set(attributes["db.statement"], Substring(attributes["db.statement"], 0, 1024))
    where attributes["db.statement"] != nil and Len(attributes["db.statement"]) > 1024
```

Or use `truncate_all` for whole maps (UTF-8 safe by default since v0.148):

```ottl
truncate_all(span.attributes, 1024)
```

### `attributes` processor ≠ `resource` processor

The OTel `attributes` processor only operates on span/log/metric attributes. To touch a *resource* attribute use the `resource` processor or a `transform` processor with `context: resource`. A config like `attributes/strip_resource: actions: [...delete os.description...]` runs without error but doesn't change resource attributes — silent no-op.

### `logdedup` paths use dot-notation only

The processor accepts `include_fields` / `exclude_fields` (not `fields`). Paths must start with `attributes.` or `body.` and use dot-notation. Bracket notation (`attributes["service.name"]`) is rejected, and `resource[...]` paths are not addressable. Default behavior dedups on the full record, which is usually what's wanted.

### `k8sattributes` cannot extract `k8s.cluster.name`

The processor's `metadata` list is restricted to pod-level identity. Cluster name has to come from `resourcedetection` or a static `resource` processor that reads it from an env var.

### Path syntax changed in v0.120

In older configs you may see `span_event.*` — current syntax is `spanevent.*`. Cache paths now require the context prefix: write `span.cache["x"]` not just `cache["x"]`. Plain `cache` is only valid in profile/profilesample contexts where it's the documented path. Paste-from-old-config is the most common source of regressions.

### `Base64Decode` is deprecated

Use `Decode(value, "base64")` instead. The same `Decode` converter handles `base64-raw`, `base64-url`, `base64-raw-url`, and IANA character set encodings. Keep `Base64Decode` only if pinned to a pre-v0.141 collector.

### `Bool` converter coercion is loose

`Bool("true")`, `Bool("1")`, `Bool(1)` all return `true`; `Bool("false")`, `Bool("0")`, `Bool(0)`, `Bool(0.0)` return `false`. Anything else errors. Don't assume Python-like truthiness for arbitrary strings.

### Verify before publishing

YAML/OTTL gotchas like the above pass the eye test. Use the [telemetrygen verification recipe](../telemetrygen/SKILL.md#verifying-a-collector-config) (otelcol-contrib + file exporter + telemetrygen) to confirm the snippet does what the surrounding prose claims, especially before shipping to a customer or production.

## Best practices

1. **Validate inputs** before conversion: `where IsString(x)`, `where x != nil`.
2. **Order conditions by selectivity** (cheap and most-selective first). `span.kind == SPAN_KIND_SERVER and IsMatch(...)` lets the kind check short-circuit before the regex runs.
3. **Cache expensive operations** in `<context>.cache`: `set(span.cache["url_parts"], Split(span.attributes["http.url"], "/"))`, then read from cache.
4. **Use the most specific context.** `datapoint` for metric-attribute work beats walking `metric.data_points` from the metric context.
5. **Escape regex correctly.** Use `\\d+`, `\\.`, `\\s+` in OTTL strings (single backslash in YAML becomes a literal).
6. **Prefer `keep_keys` to a long list of `delete_key`** when shaping output — easier to read, fails closed.

## Versioning notes

Recently added (still useful to know which release introduced them when supporting users on older collectors):

| Feature | Since |
|---------|-------|
| Profile / ProfileSample contexts | v0.124 / v0.132 (Development) |
| Cache paths require context prefix; `spanevent` rename | v0.120 (breaking) |
| `delete_index` editor | v0.145 |
| `span.flags` path | v0.145 |
| `<context>.metadata` for client request metadata | v0.147 |
| `truncate_all` UTF-8 safe default (`utf8_safe` parameter) | v0.148 (behavior change) |
| `SpanID` / `TraceID` accept hex strings | v0.142 |
| `flatten` `resolveConflicts` parameter | v0.139 |
| `Base64Encode` | v0.147 |
| `Decode`, deprecates `Base64Decode` | v0.141 |
| `Bool` converter | v0.143 |
| `ExtractGrokPatterns` | v0.130 |
| `URL`, `UserAgent` | v0.127, v0.134 |
| `IsInCIDR` | v0.146 |
| `Murmur3Hash*`, `XXH3`, `XXH128` | v0.129, v0.135 |
| `Sort`, `Index`, `SliceToMap` | v0.125, v0.126, v0.128 |
| `UUIDv7`, `ParseSeverity`, `CommunityID` | v0.138, v0.133, v0.131 |

## References

- `references/contexts.md` — context paths and enums
- `references/functions.md` — editors and converters with signatures
- `references/quick-reference.md` — recipes, regex patterns, troubleshooting
- Upstream: <https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/pkg/ottl>
