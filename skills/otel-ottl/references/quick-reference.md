# OTTL Quick Reference

Recipes, regex patterns, and a troubleshooting table. Pair with `contexts.md` for paths and `functions.md` for full signatures.

## Common patterns

### Attribute management

```ottl
# Set attribute
set(span.attributes["environment"], "production")

# Set if missing
set(span.attributes["version"], "unknown")
    where span.attributes["version"] == nil

# Copy from resource
set(span.attributes["service"], resource.attributes["service.name"])

# Rename
set(span.attributes["host.name"], span.attributes["hostname"])
delete_key(span.attributes, "hostname")

# Drop sensitive keys
delete_matching_keys(span.attributes, "(?i).*(password|secret|token|apikey).*")

# Whitelist (fail-closed)
keep_keys(span.attributes, ["http.method", "http.status_code", "http.route"])
```

### String manipulation

```ottl
set(span.attributes["method"], ToUpperCase(String(span.attributes["http.method"])))
set(span.attributes["short_id"], Substring(span.span_id.string, 0, 8))
set(span.attributes["service"], Split(resource.attributes["service.name"], "-")[0])
set(log.body, Concat([log.severity_text, ": ", log.body.string], ""))
set(log.body, Trim(log.body.string, " \t\n"))
```

### Data parsing

```ottl
# JSON body -> attributes
set(log.attributes, ParseJSON(log.body.string))
    where IsString(log.body) and IsMatch(log.body.string, "^\\s*\\{.*\\}\\s*$")

# Query string -> map
set(span.attributes["params"],
    ParseKeyValue(span.attributes["http.query"], "&", "="))

# URL parts (v0.127+)
set(span.attributes["http.host"], URL(span.attributes["http.url"])["domain"])

# User agent (v0.134+)
set(span.attributes["client.os"],
    UserAgent(span.attributes["http.user_agent"])["os.name"])

# Grok (v0.130+) — much cleaner than the equivalent regex
set(log.attributes,
    ExtractGrokPatterns(log.body.string,
                        "%{IP:client_ip} %{WORD:method} %{URIPATHPARAM:path}"))
```

### Error / status detection

```ottl
# HTTP errors
set(span.status.code, STATUS_CODE_ERROR)
    where IsInt(span.attributes["http.status_code"])
      and Int(span.attributes["http.status_code"]) >= 400

# Exception spans
set(span.status.code, STATUS_CODE_ERROR)
set(span.status.message, span.attributes["exception.message"])
    where span.attributes["exception.type"] != nil

# Slow spans
set(span.attributes["slow"], true)
    where (span.end_time_unix_nano - span.start_time_unix_nano) > 1000000000
```

### Filter conditions

```ottl
# Skip health checks
where not IsMatch(span.name, ".*(health|ready|alive).*")

# Errors only
where log.severity_number >= SEVERITY_NUMBER_ERROR

# Server spans only
where span.kind == SPAN_KIND_SERVER

# Drop traffic from internal CIDRs
where IsInCIDR(span.attributes["client.address"],
               ["10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"])
```

### Sampling

```ottl
# Hash-based 10% — XXH3 is faster than FNV on long inputs
set(span.attributes["sampled"], true)
    where XXH3(span.trace_id.string) % 100 < 10

# Always-sample errors
set(span.attributes["sampled"], true)
    where span.status.code == STATUS_CODE_ERROR

# Latency-based
set(span.attributes["sampled"], true)
    where (span.end_time_unix_nano - span.start_time_unix_nano) > 1000000000
```

### Time operations

```ottl
# Duration in ms
set(span.attributes["duration_ms"],
    (span.end_time_unix_nano - span.start_time_unix_nano) / 1000000)

# Format timestamp
set(log.attributes["formatted_time"], FormatTime(log.time, "%Y-%m-%d %H:%M:%S"))

# Stamp processing time
set(span.attributes["processed_at"], Now())

# Hour bucket
set(log.attributes["hour"], Hour(log.time))
```

### Redaction

```ottl
# Credit cards
replace_all_patterns(log.attributes, "value",
                     "\\b(?:\\d{4}[\\s-]?){3}\\d{4}\\b",
                     "****-****-****-****")

# Emails
replace_all_patterns(log.attributes, "value",
                     "[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}",
                     "[REDACTED_EMAIL]")

# IPv4
replace_all_patterns(log.attributes, "value",
                     "\\b\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\b",
                     "[REDACTED_IP]")

# Hash, don't drop — preserves cardinality for analytics
set(span.attributes["user.email_hash"], SHA256(span.attributes["user.email"]))
delete_key(span.attributes, "user.email")
```

## Processor configuration

### `transform`

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
```

### `filter`

```yaml
processors:
  filter:
    error_mode: ignore
    traces:
      span:
        - 'IsMatch(span.name, "^/health.*")'
        - 'span.kind == SPAN_KIND_INTERNAL'
    logs:
      log_record:
        - 'log.severity_number < SEVERITY_NUMBER_WARN'
```

### `routing`

```yaml
processors:
  routing:
    default_pipelines: [traces/default]
    table:
      - statement: 'route() where resource.attributes["env"] == "prod"'
        pipelines: [traces/prod]
      - statement: 'route() where span.status.code == STATUS_CODE_ERROR'
        pipelines: [traces/errors]
```

### `tail_sampling`

```yaml
processors:
  tail_sampling:
    policies:
      - name: errors
        type: ottl_condition
        ottl_condition:
          span:
            - 'span.status.code == STATUS_CODE_ERROR'
      - name: slow
        type: ottl_condition
        ottl_condition:
          span:
            - '(span.end_time_unix_nano - span.start_time_unix_nano) > 1000000000'
```

## Regular expressions

### Common patterns

```ottl
"[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}"                          # email
"\\b\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\b"                          # IPv4
"\\b(?:\\d{4}[\\s-]?){3}\\d{4}\\b"                                          # credit card
"\\b\\d{3}-\\d{3}-\\d{4}\\b"                                                # US phone
"https?://[^\\s]+"                                                          # URL
"[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}"  # UUID
"\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}(?:\\.\\d{3})?Z?"                 # ISO 8601
"^\\{.*\\}$"                                                                # JSON object
"^\\d+$"                                                                    # digits only
```

### Escape rules

In OTTL strings, escape regex metacharacters with double backslash. Single-backslash YAML chars get eaten before OTTL sees them.

| You want in regex | Write in OTTL string |
|-------------------|---------------------|
| `\d` | `\\d` |
| `\s` | `\\s` |
| `\w` | `\\w` |
| `\.` | `\\.` |
| `\*` | `\\*` |
| `\+` | `\\+` |

**Backreference escape (YAML).** OTTL `${1}` must be written as `$${1}` in YAML — the loader treats `$` as an env-var marker. `"$1REDACTED"` becomes a literal string at the OTTL level, no replacement.

**RE2 repeat-count limit.** `(.{1024}).*` fails to compile. Use `Substring` + `Len` for length truncation:

```ottl
set(attributes["x"], Substring(attributes["x"], 0, 1024))
    where attributes["x"] != nil and Len(attributes["x"]) > 1024
```

## Debugging

### Enable debug logging

```yaml
service:
  telemetry:
    logs:
      level: debug
```

### Add probes

```ottl
# Track that a statement ran
set(span.attributes["ottl.processed"], true)
set(span.attributes["ottl.timestamp"], Now())

# 0.1% sampled debug snapshot
set(span.attributes["debug.original_name"], span.name)
    where XXH3(span.trace_id.string) % 1000 == 0
```

### Validate before transforming

```ottl
where IsString(span.attributes["count"])
  and IsMatch(span.attributes["count"], "^\\d+$")

where span.attributes["value"] != nil

where Len(String(log.body)) > 0
```

## Performance

**Do**

- Order conditions by selectivity (cheap and most-discriminating first).
- Use early filtering to drop work at the smallest scope.
- Cache expensive function results in `<context>.cache`.
- Pick the most specific context (operate on `datapoint`, not on `metric.data_points`).
- Batch attribute pruning with `keep_keys` rather than many `delete_key` statements.

**Don't**

- Repeat expensive function calls inside the same statement set.
- Run complex regex on large bodies.
- Process telemetry that a cheaper filter could drop earlier in the pipeline.
- Suppress error_mode noise (`silent`) without first verifying behavior.

### Cache pattern

```ottl
set(span.cache["url_parts"], Split(span.attributes["http.url"], "/"))
set(span.attributes["api_version"], span.cache["url_parts"][2])
```

### Early-filter ordering

```ottl
# Cheap kind check fails first; regex only runs for SERVER spans.
where span.kind == SPAN_KIND_SERVER and IsMatch(span.name, "expensive.*")
```

## Troubleshooting

### Common errors

| Symptom | Cause | Fix |
|---------|-------|-----|
| `nil value` | Path resolves to a missing field | Add `where path != nil` (or `IsString(path)` etc.) |
| `type mismatch` | Wrong types in arithmetic / function args | Convert with `Int`, `Double`, `String` after type-checking |
| `invalid regex` | Unescaped metacharacter | Double-escape: `\\d`, `\\.`, `\\s` |
| `division by zero` | `/` with a zero divisor | Add `where divisor != 0` |
| Backreferences appear literal | YAML ate the `$` | Use `$${1}` in YAML for OTTL `${1}` |
| `attributes` processor doesn't touch resource | Wrong processor | Use `resource` processor or `transform` with `context: resource` |
| `cache["x"]` errors in span context | Cache requires a context prefix since v0.120 | Write `span.cache["x"]` |
| `Bool("yes")` errors | `Bool` only accepts true/false/1/0 inputs | Use `IsMatch` with an explicit pattern, or check with `where` |
| `Base64Decode` deprecation warning | v0.141+ moved to generic decoder | Use `Decode(value, "base64")` |

### Checklist before shipping

1. Syntax: parens balanced, strings closed, regex escapes doubled.
2. Types: type-check before conversion (`IsString`, `IsInt`).
3. Existence: nil-guard paths that may be absent.
4. Logic: dry-run with the [telemetrygen verification recipe](../../telemetrygen/SKILL.md#verifying-a-collector-config) — generate a known input, capture the file-exporter output, confirm the transformation. The eye test misses too many gotchas.
5. Performance: most-selective `where` clause first.

### Safe transformation skeletons

```ottl
# Safe JSON parse
set(log.cache["parsed"], ParseJSON(log.body.string))
    where IsString(log.body)
      and IsMatch(log.body.string, "^\\s*[\\{\\[].*[\\}\\]]\\s*$")

# Safe numeric conversion
set(span.attributes["count_int"], Int(span.attributes["count"]))
    where IsString(span.attributes["count"])
      and IsMatch(span.attributes["count"], "^-?\\d+$")

# Safe duration calc
set(span.attributes["duration_ns"],
    span.end_time_unix_nano - span.start_time_unix_nano)
    where span.end_time_unix_nano > span.start_time_unix_nano
```
